// 16_maze runs the Maze of PicoLibSDK (package games/maze), ported from
// the PicoPad: find the way out of the maze with the arrows; only the
// places next to the way gone are shown. A shows the door for a moment.
// At the door the whole maze is shown and any button starts a new one. Y
// ends the game, and it starts again (on the PicoPad: back to the boot
// loader).
//
// The mini joystick module (platform.InitPad) has the PicoPad's buttons:
// the stick is the d-pad. Without it, touch the display: it is split into
// 3 x 3 buttons (package picopad). In the browser, the arrow keys and Z
// (A), S (Y).
//
//	make flash-s028 PKG=./examples/16_maze   # ESP32-2432S028R
//	make flash-s035 PKG=./examples/16_maze   # ESP32-3248S035C
//	make wasm PKG=./examples/16_maze && make serve
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/games/maze"
	"github.com/sat0ken/tinygo-cyd/platform"
)

func main() {
	time.Sleep(500 * time.Millisecond)
	d, t, err := platform.Init()
	if err != nil {
		println("platform.Init:", err.Error())
		platform.Halt()
	}
	g := maze.New(d, t, uint32(time.Now().UnixNano()))
	g.Pad = platform.InitPad() // the mini joystick module, if it is connected
	println("16_maze: started")
	for {
		g.Run()
		println("16_maze: Y, again")
	}
}
