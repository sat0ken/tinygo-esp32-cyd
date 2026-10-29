// 15_tetris runs the Tetris of PicoLibSDK (package games/tetris), ported
// from the PicoPad: LEFT and RIGHT move, UP or X turns, DOWN or A drops, B
// pauses, Y ends the game. A new game starts paused; after 10 seconds
// without a button the demo plays. Holding Y in the demo ends it and the
// game starts again (on the PicoPad: back to the boot loader).
//
// The mini joystick module (platform.InitPad) has the PicoPad's buttons:
// the stick is the d-pad. Without it, touch the display: it is split into
// 3 x 3 buttons (package picopad). In the browser, the arrow keys and Z
// (A), X (B), A (X), S (Y).
//
//	make flash-s028 PKG=./examples/15_tetris   # ESP32-2432S028R
//	make flash-s035 PKG=./examples/15_tetris   # ESP32-3248S035C
//	make wasm PKG=./examples/15_tetris && make serve
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/games/tetris"
	"github.com/sat0ken/tinygo-cyd/platform"
)

func main() {
	time.Sleep(500 * time.Millisecond)
	d, t, err := platform.Init()
	if err != nil {
		println("platform.Init:", err.Error())
		platform.Halt()
	}
	g := tetris.New(d, t, uint32(time.Now().UnixNano()))
	g.Pad = platform.InitPad() // the mini joystick module, if it is connected
	println("15_tetris: started")
	for {
		g.Run()
		println("15_tetris: Y in the demo, again")
	}
}
