//go:build !esp32_2432s028r && !esp32_3248s035c

package main

import (
	"github.com/sat0ken/tinygo-cyd/games/fifteen2"
	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/picopad"
)

// The Fifteen puzzle with pictures has 1.1 MB of pictures: with them all
// the games do not fit in the 4 MB of flash of the ESP32 boards (S028,
// S035), which leave it out (nofifteen2.go).
const hasFifteen2 = true

func startFifteen2(d hal.Display, t hal.Touch, seed uint32) (*picopad.Console, func()) {
	g := fifteen2.New(d, t, seed)
	return g.Console, g.Run
}
