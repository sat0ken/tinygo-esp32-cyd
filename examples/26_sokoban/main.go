// 26_sokoban runs the Sokoban of PicoLibSDK (package games/sokoban),
// ported from the PicoPad. 2,020 levels of seven authors that let them be
// distributed freely, with their solutions. A demo plays the solutions
// until a key; then the arrows choose the author, the collection and the
// level, A enters, Y goes back. In the game the arrows move, X shows the
// solution, B starts again, A+RIGHT/LEFT go to the next/previous level. Y
// held in the demo ends Run.
//
// The mini joystick module (platform.InitPad) has the PicoPad's buttons:
// the stick is the d-pad. Without it, touch the display: it is split into
// 3 x 3 buttons (package picopad). In the browser, the arrow keys and Z
// (A), X (B), A (X), S (Y).
//
//	make flash-s028 PKG=./examples/26_sokoban   # ESP32-2432S028R
//	make wasm PKG=./examples/26_sokoban && make serve
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/games/sokoban"
	"github.com/sat0ken/tinygo-cyd/platform"
)

func main() {
	time.Sleep(500 * time.Millisecond)
	d, t, err := platform.Init()
	if err != nil {
		println("platform.Init:", err.Error())
		platform.Halt()
	}
	g := sokoban.New(d, t, uint32(time.Now().UnixNano()))
	g.Pad = platform.InitPad() // the mini joystick module, if it is connected
	println("26_sokoban: started")
	for {
		g.Run()
		println("26_sokoban: Y, again")
	}
}
