// 29_ants runs the Ants card game of PicoLibSDK (package games/ants),
// ported from the PicoPad. The black and the red ants build castles:
// LEFT/RIGHT and A choose two players, a player against the computer or
// the computer against itself; in the game LEFT/RIGHT choose a card, A
// plays it, B discards it, X shows the computer's choice, Y goes back; Y
// on the open screen ends Run.
//
// The mini joystick module (platform.InitPad) has the PicoPad's buttons:
// the stick is the d-pad. Without it, touch the display: it is split into
// 3 x 3 buttons (package picopad). In the browser, the arrow keys and Z
// (A), X (B), A (X), S (Y).
//
//	make flash-s028 PKG=./examples/29_ants   # ESP32-2432S028R
//	make wasm PKG=./examples/29_ants && make serve
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/games/ants"
	"github.com/sat0ken/tinygo-cyd/platform"
)

func main() {
	time.Sleep(500 * time.Millisecond)
	d, t, err := platform.Init()
	if err != nil {
		println("platform.Init:", err.Error())
		platform.Halt()
	}
	g := ants.New(d, t, uint32(time.Now().UnixNano()))
	g.Pad = platform.InitPad() // the mini joystick module, if it is connected
	println("29_ants: started")
	for {
		g.Run()
		println("29_ants: Y, again")
	}
}
