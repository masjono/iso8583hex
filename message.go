package iso8583

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

const (
	TAG_FIELD  string = "field"
	TAG_ENCODE string = "encode"
	TAG_LENGTH string = "length"
)

type fieldInfo struct {
	Index     int
	Encode    int
	LenEncode int
	Length    int
	Field     Iso8583Type
}

type Message struct {
	Mti          string
	MtiEncode    int
	SecondBitmap bool
	Data         interface{}
}

func NewMessage(mti string, data interface{}) *Message {
	return &Message{mti, ASCII, false, data}
}

func (m *Message) Bytes() ([]byte, error) {
	ret := make([]byte, 0)

	// generate MTI:
	mtiBytes, err := m.encodeMti()
	if err != nil {
		return nil, err
	}
	ret = append(ret, mtiBytes...)

	// generate bitmap and fields:
	fields := parseFields(m.Data)

	byteNum := 8
	if m.SecondBitmap {
		byteNum = 16
	}
	bitmap := make([]byte, byteNum)
	data := make([]byte, 0, 512)

	for byteIndex := 0; byteIndex < byteNum; byteIndex++ {
		for bitIndex := 0; bitIndex < 8; bitIndex++ {
			i := byteIndex*8 + bitIndex + 1

			if m.SecondBitmap && i == 1 {
				step := uint(7 - bitIndex)
				bitmap[byteIndex] |= (0x01 << step)
			}

			if info, ok := fields[i]; ok {
				// mark 1 in bitmap:
				step := uint(7 - bitIndex)
				bitmap[byteIndex] |= (0x01 << step)
				// append data:
				d, err := info.Field.Bytes(info.Encode, info.LenEncode, info.Length)
				if err != nil {
					return nil, err
				}
				data = append(data, d...)
			}
		}
	}

	// modified by MJ to make bitmap to hex
	str := fmt.Sprintf("%X", bitmap)

	bitmap2 := []byte(str)

	ret = append(ret, bitmap2...)
	ret = append(ret, data...)

	return ret, nil
}

func (m *Message) encodeMti() ([]byte, error) {
	if m.Mti == "" {
		panic("MTI is required")
	}
	if len(m.Mti) != 4 {
		panic("MTI is invalid")
	}

	switch m.MtiEncode {
	case BCD:
		return bcd([]byte(m.Mti)), nil
	default:
		return []byte(m.Mti), nil
	}
}

func parseFields(msg interface{}) map[int]*fieldInfo {
	fields := make(map[int]*fieldInfo)

	v := reflect.Indirect(reflect.ValueOf(msg))
	if v.Kind() != reflect.Struct {
		panic("data must be a struct")
	}
	for i := 0; i < v.NumField(); i++ {
		if isPtrOrInterface(v.Field(i).Kind()) && v.Field(i).IsNil() {
			continue
		}

		sf := v.Type().Field(i)
		if sf.Tag == "" || sf.Tag.Get(TAG_FIELD) == "" {
			continue
		}

		index, err := strconv.Atoi(sf.Tag.Get(TAG_FIELD))
		if err != nil {
			panic("value of field must be numeric")
		}

		encode := 0
		lenEncode := 0
		if raw := sf.Tag.Get(TAG_ENCODE); raw != "" {
			enc := strings.Split(raw, ",")
			if len(enc) == 2 {
				lenEncode = parseEncodeStr(enc[0])
				encode = parseEncodeStr(enc[1])
			} else {
				encode = parseEncodeStr(enc[0])
			}
		}

		length := -1
		if l := sf.Tag.Get(TAG_LENGTH); l != "" {
			length, err = strconv.Atoi(l)
			if err != nil {
				panic("value of length must be numeric")
			}
		}

		field, ok := v.Field(i).Interface().(Iso8583Type)
		if !ok {
			panic("field must be Iso8583Type")
		}
		fields[index] = &fieldInfo{index, encode, lenEncode, length, field}
	}
	return fields
}

func isPtrOrInterface(k reflect.Kind) bool {
	return k == reflect.Interface || k == reflect.Ptr
}

func parseEncodeStr(str string) int {
	switch str {
	case "ascii":
		return ASCII
	case "bcd":
		return BCD
	}
	return -1
}

func (m *Message) Load(raw []byte) (err error) {
	if m.Mti == "" {
		m.Mti, err = decodeMti(raw, m.MtiEncode)
		if err != nil {
			return err
		}
	}
	start := 4
	if m.MtiEncode == BCD {
		start = 2
	}

	fields := parseFields(m.Data)

	byteNum := 16 // diubah mj jadi baca hex dulu
	if hexToInt(string(raw[start])) & 0x8 == 0x8 {
		// 1st bit == 1
		m.SecondBitmap = true
		byteNum = 32 // diubah mj jadi baca hex dulu
	}
	bitByte := raw[start : start+byteNum]
	start += byteNum
	
	// convert bitbyte dari hex ke byte
	bitByte, err = convertHexStringToByte(string(bitByte))
	if err != nil{
		return err
	}	
	
	fmt.Println("byte to load:", bitByte)

	for byteIndex := 0; byteIndex < byteNum/2; byteIndex++ {
		for bitIndex := 0; bitIndex < 8; bitIndex++ {
			step := uint(7 - bitIndex)
			if (bitByte[byteIndex] & (0x01 << step)) == 0 {
				continue
			}

			i := byteIndex*8 + bitIndex + 1
			if i == 1 {
				// field 1 is the second bitmap
				continue
			}
			f, ok := fields[i]
			if !ok {
				return errors.New(fmt.Sprintf("field %d not defined", i))
			}
			if start > len(raw) {
				continue
			}
			l, err := f.Field.Load(raw[start:], f.Encode, f.LenEncode, f.Length)
			if err != nil {
				return err
			}
			start += l
		}
	}
	return nil
}

// add by mj to convert hex string representation into byte representation
func convertHexStringToByte(hex string) ([]byte, error) {
	var res []byte
	
	if len(hex) % 2 != 0{
		return res, errors.New("Error: odd length hex")
	}
	
	bytestr := []byte(hex)
	//loop per 2 byte
	for i:=0 ; i < len(hex); i+=2 {
		b1 := hexToInt(string(bytestr[i:i+1])) << 4
		b2 := hexToInt(string(bytestr[i+1:i+2]))
		res = append(res, byte(b1+b2))
	}
	
	return res, nil
}

// add by mj
func hexToInt(bs string) int {
	switch bs {
		case "0":
			return 0x0
		case "1":
			return 0x1
		case "2":
			return 0x2
		case "3":
			return 0x3
		case "4":
			return 0x4
		case "5":
			return 0x5
		case "6":
			return 0x6
		case "7":
			return 0x7
		case "8":
			return 0x8
		case "9":
			return 0x9
		case "A":
			return 0xA
		case "B":
			return 0xB
		case "C":
			return 0xC
		case "D":
			return 0xD
		case "E":
			return 0xE
		case "F":
			return 0xF
	}
	return 0x0
}
