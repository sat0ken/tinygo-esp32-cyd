//go:build esp32s3

package platform

import (
	"device/esp"
	"errors"
)

// SerialInput returns the serial console's input. TinyGo 0.44's ESP32-S3
// UART only sends (machine_esp32s3.go has no receive interrupt), so
// machine.Serial never has a byte; this reads UART0's 128-byte receive
// FIFO directly. Read it often enough that the FIFO does not fill up.
func SerialInput() SerialReader { return uart0RX{} }

type uart0RX struct{}

var errNoByte = errors.New("uart0: no byte")

func (uart0RX) Buffered() int {
	return int(esp.UART0.STATUS.Get() & esp.UART_STATUS_RXFIFO_CNT_Msk)
}

func (u uart0RX) ReadByte() (byte, error) {
	if u.Buffered() == 0 {
		return 0, errNoByte
	}
	return byte(esp.UART0.FIFO.Get() & esp.UART_FIFO_RXFIFO_RD_BYTE_Msk), nil
}
