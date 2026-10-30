//go:build baremetal

package main

import "machine"

// openInput returns the serial port of the USB-UART bridge (115200 baud).
func openInput() input { return machine.Serial }
