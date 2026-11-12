//go:build baremetal

package main

import "github.com/sat0ken/tinygo-cyd/platform"

// openInput returns the serial port of the USB-UART bridge (115200 baud):
// machine.Serial on the ESP32, UART0's receive FIFO on the ESP32-S3,
// whose UART in TinyGo 0.44 only sends (see platform.SerialInput).
func openInput() input { return platform.SerialInput() }
