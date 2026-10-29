// 22_eggs runs the Eggs of PicoLibSDK (package games/eggs), ported from
// the PicoPad. A game like Reversi: a chicken put next to a line of stones
// that ends with a chicken turns them, hen -> chicken -> egg for player 1,
// egg -> chicken -> hen for player 2. The first screen chooses: LEFT plays
// the eggs against the computer, RIGHT the hens, DOWN is two players, UP a
// demo; the arrows move the cursor, A puts a chicken, X shows the
// computer's choice, Y ends the game; Y on the first screen ends it and it
// starts again (on the PicoPad: back to the boot loader).
//
// The mini joystick module (platform.InitPad) has the PicoPad's buttons:
// the stick is the d-pad. Without it, touch the display: it is split into
// 3 x 3 buttons (package picopad). In the browser, the arrow keys and Z
// (A), A (X), S (Y).
//
//	make flash-s028 PKG=./examples/22_eggs   # ESP32-2432S028R
//	make wasm PKG=./examples/22_eggs && make serve
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/games/eggs"
	"github.com/sat0ken/tinygo-cyd/platform"
)

func main() {
	time.Sleep(500 * time.Millisecond)
	d, t, err := platform.Init()
	if err != nil {
		println("platform.Init:", err.Error())
		platform.Halt()
	}
	g := eggs.New(d, t, uint32(time.Now().UnixNano()))
	g.Pad = platform.InitPad() // the mini joystick module, if it is connected
	println("22_eggs: started")
	for {
		g.Run()
		println("22_eggs: Y, again")
	}
}
