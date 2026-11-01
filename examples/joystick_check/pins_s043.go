//go:build esp32_4827s043

package main

import (
	"machine"

	"github.com/sat0ken/tinygo-cyd/board"
)

// ESP32-4827S043: IO17 and IO18 with 3.3V and GND. IO18 is also the touch
// controller's IRQ line, which platform.InitTouch turns off.
//
//	module G   -> GND
//	module V   -> 3.3V
//	module SDA -> IO17
//	module SCL -> IO18
var (
	bus    = machine.I2C0
	sdaPin = machine.Pin(board.PinJoySDA)
	sclPin = machine.Pin(board.PinJoySCL)
	wiring = "SDA IO17, SCL IO18, 3.3V"
)
