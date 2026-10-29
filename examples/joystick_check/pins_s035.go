//go:build esp32_3248s035c

package main

import "machine"

// ESP32-3248S035C: connector P4 has GND, 3V3, IO17 and IO18 (as printed on
// the board), in the same order as the module's G, V, SDA and SCL, so a
// straight cable fits. I2C1, so that I2C0 stays for the GT911 touch
// controller (IO32/IO33).
//
// IO18 is also the microSD slot's SCK, so the SD card cannot be used at the
// same time. IO17 is the blue of the RGB LED according to some sources,
// which then glows a little while the bus is busy. CN1 (IO21, IO22) is not
// used: IO21 is said to be the GT911's INT line.
//
//	module G   -> P4 GND
//	module V   -> P4 3V3
//	module SDA -> P4 IO17
//	module SCL -> P4 IO18
var (
	bus    = machine.I2C1
	sdaPin = machine.Pin(17)
	sclPin = machine.Pin(18)
	wiring = "P4: SDA IO17, SCL IO18, 3V3"
)
