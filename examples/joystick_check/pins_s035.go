//go:build esp32_3248s035c

package main

import "machine"

// ESP32-3248S035C: connector CN1 has GND, IO22, IO21 and 3.3V. I2C1, so
// that I2C0 stays for the GT911 touch controller (IO32/IO33).
//
// Some sources say IO21 is the GT911's INT line; if it is, the touch
// controller drives it and the joystick cannot use it.
//
//	module G   -> CN1 GND
//	module V   -> CN1 3.3V (not 5V)
//	module SDA -> CN1 IO21
//	module SCL -> CN1 IO22
var (
	bus    = machine.I2C1
	sdaPin = machine.Pin(21)
	sclPin = machine.Pin(22)
	wiring = "CN1: SDA IO21, SCL IO22, 3.3V"
)
