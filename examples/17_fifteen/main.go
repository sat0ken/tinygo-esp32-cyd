// 17_fifteen runs the Fifteen puzzle of PicoLibSDK (package
// games/fifteen), ported from the PicoPad: sort the tiles from 1 to 15.
// The arrows move a tile into the hole, A shuffles a new game, X lets the
// solver play (any button stops it), Y ends the game and it starts again
// (on the PicoPad: back to the boot loader).
//
// The mini joystick module (platform.InitPad) has the PicoPad's buttons:
// the stick is the d-pad. Without it, touch the display: it is split into
// 3 x 3 buttons (package picopad). In the browser, the arrow keys and Z
// (A), A (X), S (Y).
//
//	make flash-s028 PKG=./examples/17_fifteen   # ESP32-2432S028R
//	make flash-s035 PKG=./examples/17_fifteen   # ESP32-3248S035C
//	make wasm PKG=./examples/17_fifteen && make serve
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/games/fifteen"
	"github.com/sat0ken/tinygo-cyd/platform"
)

func main() {
	time.Sleep(500 * time.Millisecond)
	d, t, err := platform.Init()
	if err != nil {
		println("platform.Init:", err.Error())
		platform.Halt()
	}
	g := fifteen.New(d, t, uint32(time.Now().UnixNano()))
	g.Pad = platform.InitPad() // the mini joystick module, if it is connected
	println("17_fifteen: started")
	for {
		g.Run()
		println("17_fifteen: Y, again")
	}
}
