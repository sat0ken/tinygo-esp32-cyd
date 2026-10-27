//go:build esp32_2432s028r && !e32r28t

package main

import "machine"

// ESP32-2432S028R: connector CN1 has GND, IO22, IO27 and 3.3V.
//
//	module G   -> CN1 GND
//	module V   -> CN1 3.3V (not 5V)
//	module SDA -> CN1 IO27
//	module SCL -> CN1 IO22
var (
	bus    = machine.I2C0
	sdaPin = machine.Pin(27)
	sclPin = machine.Pin(22)
	wiring = "CN1: SDA IO27, SCL IO22, 3.3V"
)
