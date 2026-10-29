// 25_pacman runs the PacMan of PicoLibSDK (package games/pacman), ported
// from the PicoPad. The open screen shows the ghosts, then the game
// starts: the arrows steer, A pauses, Y goes back to the open screen; Y on
// the open screen ends Run.
//
// The mini joystick module (platform.InitPad) has the PicoPad's buttons:
// the stick is the d-pad. Without it, touch the display: it is split into
// 3 x 3 buttons (package picopad). In the browser, the arrow keys and Z
// (A), S (Y).
//
//	make flash-s028 PKG=./examples/25_pacman   # ESP32-2432S028R
//	make wasm PKG=./examples/25_pacman && make serve
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/games/pacman"
	"github.com/sat0ken/tinygo-cyd/platform"
)

func main() {
	time.Sleep(500 * time.Millisecond)
	d, t, err := platform.Init()
	if err != nil {
		println("platform.Init:", err.Error())
		platform.Halt()
	}
	g := pacman.New(d, t, uint32(time.Now().UnixNano()))
	g.Pad = platform.InitPad() // the mini joystick module, if it is connected
	println("25_pacman: started")
	for {
		g.Run()
		println("25_pacman: Y, again")
	}
}
