//go:build esp32_3248s035c

package main

import (
	"machine"

	"github.com/sat0ken/tinygo-cyd/board/s035"
)

// ESP32-3248S035C: connector CN1 has GND, IO22, IO21 and 3.3V from the
// top; the module's order is G, V, SDA, SCL, so the wires cross. I2C1, so
// that I2C0 stays for the GT911 touch controller (IO32/IO33).
//
//	module G   -> CN1 GND
//	module V   -> CN1 3.3V
//	module SDA -> CN1 IO21
//	module SCL -> CN1 IO22
var (
	bus    = machine.I2C1
	sdaPin = machine.Pin(s035.PinJoySDA)
	sclPin = machine.Pin(s035.PinJoySCL)
	wiring = "CN1: SDA IO21, SCL IO22, 3.3V"
)
