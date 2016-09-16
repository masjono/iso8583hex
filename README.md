# iso8583hex
A golang implementation to marshal and unmarshal iso8583 message.
develop based on github.com/ideazxy/iso8583 and modified to support hex format bitmap

### Example

```go
package main

import (
	"fmt"

	"github.com/masjon/iso8583hex" iso8583
)

type Data struct {
	No   *iso8583.Numeric      `field:"3" length:"6" encode:"bcd"`
	Oper *iso8583.Numeric      `field:"26" length:"2" encode:"ascii"`
	Ret  *iso8583.Alphanumeric `field:"39" length:"2"`
	Sn   *iso8583.Llvar        `field:"45" length:"23" encode:"bcd,ascii"`
	Info *iso8583.Lllvar       `field:"46" length:"42" encode:"bcd,ascii"`
	Mac  *iso8583.Binary       `field:"64" length:"8"`
}

func main() {
	data := &Data{
		No:   iso8583.NewNumeric("001111"),
		Oper: iso8583.NewNumeric("22"),
		Ret:  iso8583.NewAlphanumeric("ok"),
		Sn:   iso8583.NewLlvar([]byte("abc001")),
		Info: iso8583.NewLllvar([]byte("你好 golang!")),
		Mac:  iso8583.NewBinary([]byte("a1s2d3f4")),
	}
	msg := iso8583.NewMessage("0800", data)
	msg.MtiEncode = iso8583.BCD
	b, err := msg.Bytes()
	if err != nil {
		log.Println(err.Error())
	}
	fmt.Printf("% x\n", b)
}
```
