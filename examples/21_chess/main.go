// 21_chess runs the Chess of PicoLibSDK (package games/chess), ported from
// the PicoPad. The first screen chooses: LEFT plays white against the
// computer, RIGHT black, DOWN is two players, UP a demo, A the level (2 to
// 5 moves ahead). The arrows move the cursor, A picks up a piece and puts
// it down, B puts it back, X lets the computer move, Y ends the game; Y on
// the first screen ends it and it starts again (on the PicoPad: back to
// the boot loader).
//
// The mini joystick module (platform.InitPad) has the PicoPad's buttons:
// the stick is the d-pad. Without it, touch the display: it is split into
// 3 x 3 buttons (package picopad). In the browser, the arrow keys and Z
// (A), X (B), A (X), S (Y).
//
//	make flash-s028 PKG=./examples/21_chess   # ESP32-2432S028R
//	make wasm PKG=./examples/21_chess && make serve
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/games/chess"
	"github.com/sat0ken/tinygo-cyd/platform"
)

func main() {
	time.Sleep(500 * time.Millisecond)
	d, t, err := platform.Init()
	if err != nil {
		println("platform.Init:", err.Error())
		platform.Halt()
	}
	g := chess.New(d, t, uint32(time.Now().UnixNano()))
	g.Pad = platform.InitPad() // the mini joystick module, if it is connected
	println("21_chess: started")
	for {
		g.Run()
		println("21_chess: Y, again")
	}
}
