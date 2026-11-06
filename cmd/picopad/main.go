// Command picopad is a menu of the games ported from the PicoPad
// (PicoLibSDK, examples 15 to 33) and Baseball (35, an original game on
// package picopad): choose one with the arrows and A, or
// tap it twice. A game comes back to the menu at its end (game over, the
// end of a match) or with Y (see the bottom line of the menu); puzzles go
// on to the next puzzle until Y.
//
// The ESP32 boards (ESP32-2432S028R, ESP32-3248S035C) have 4 MB of flash:
// all the games but the Fifteen puzzle with pictures (1.1 MB of pictures,
// examples/31_fifteen2 alone). The ESP32-4827S043 and the browser have
// all 20.
//
// The PicoPad's screen is 320x240. On the ESP32-3248S035C (480x320) it is
// drawn 4/3 bigger, 427x320 (package picopad); on the other boards in the
// middle of the display.
//
// The mini joystick module (platform.InitPad) has the PicoPad's buttons:
// the stick is the d-pad. Without it, touch the display: in the menu tap
// the games, in a game it is split into 3 x 3 buttons (package picopad).
// In the browser, the arrow keys and Z (A), X (B), A (X), S (Y).
//
//	make flash-s028 PKG=./cmd/picopad   # ESP32-2432S028R
//	make wasm PKG=./cmd/picopad && make serve
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/platform"
)

func main() {
	time.Sleep(500 * time.Millisecond)
	d, t, err := platform.InitFull() // on the S035, the screen 4/3 bigger
	if err != nil {
		println("platform.InitFull:", err.Error())
		platform.Halt()
	}
	n := uint32(0)
	l := NewLauncher(d, t, func() uint32 {
		n++
		return uint32(time.Now().UnixNano()) ^ n*0x9E3779B9
	})
	l.Pad = platform.InitPad() // the mini joystick module, if it is connected
	println("picopad: started")
	l.Run()
}
