//go:build esp32_2432s028r && e32r28t

package main

import "machine"

// E32R28T (LCDwiki 2.8inch ESP32-32E Display): no connector has power and
// two free pins together, so the I2C lines come from the SPI peripheral
// connector (IO27 is its CS, IO18 its SCK, shared with the microSD slot)
// and the power from the expansion connector.
//
//	module G   -> expansion GND
//	module V   -> expansion 3.3V (not 5V)
//	module SDA -> SPI connector IO27
//	module SCL -> SPI connector IO18
//
// Build with: make flash-s028 PKG=./examples/joystick_check TAGS=e32r28t
var (
	bus    = machine.I2C0
	sdaPin = machine.Pin(27)
	sclPin = machine.Pin(18)
	wiring = "SPI conn.: SDA IO27, SCL IO18; exp. 3.3V"
)
