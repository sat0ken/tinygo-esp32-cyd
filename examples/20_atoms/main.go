// 20_atoms runs the Exploding Atoms of PicoLibSDK (package games/atoms),
// ported from the PicoPad. Up to five players, human or computer, put
// atoms on a 10 x 10 board in turn; a place with as many atoms as
// neighbours explodes into them and takes them over. The first screen sets
// the players (LEFT, UP, RIGHT, DOWN, B) and A starts; in the game the
// arrows move the cursor, A puts an atom, Y goes back. Y on the first
// screen ends the game and it starts again (on the PicoPad: back to the
// boot loader).
//
// The mini joystick module (platform.InitPad) has the PicoPad's buttons:
// the stick is the d-pad. Without it, touch the display: it is split into
// 3 x 3 buttons (package picopad). In the browser, the arrow keys and Z
// (A), X (B), S (Y).
//
//	make flash-s028 PKG=./examples/20_atoms   # ESP32-2432S028R
//	make wasm PKG=./examples/20_atoms && make serve
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/games/atoms"
	"github.com/sat0ken/tinygo-cyd/platform"
)

func main() {
	time.Sleep(500 * time.Millisecond)
	d, t, err := platform.Init()
	if err != nil {
		println("platform.Init:", err.Error())
		platform.Halt()
	}
	g := atoms.New(d, t, uint32(time.Now().UnixNano()))
	g.Pad = platform.InitPad() // the mini joystick module, if it is connected
	println("20_atoms: started")
	for {
		g.Run()
		println("20_atoms: Y, again")
	}
}
