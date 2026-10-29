// 28_raptor runs the Raptor of PicoLibSDK (package games/raptor), ported
// from the PicoPad. A shooting game over a scrolling landscape: the arrows
// move the ship, A shoots, B selects the weapon (a new one each level), Y
// ends Run.
//
// The mini joystick module (platform.InitPad) has the PicoPad's buttons:
// the stick is the d-pad. Without it, touch the display: it is split into
// 3 x 3 buttons (package picopad). In the browser, the arrow keys and Z
// (A), X (B), S (Y).
//
//	make flash-s028 PKG=./examples/28_raptor   # ESP32-2432S028R
//	make wasm PKG=./examples/28_raptor && make serve
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/games/raptor"
	"github.com/sat0ken/tinygo-cyd/platform"
)

func main() {
	time.Sleep(500 * time.Millisecond)
	d, t, err := platform.Init()
	if err != nil {
		println("platform.Init:", err.Error())
		platform.Halt()
	}
	g := raptor.New(d, t, uint32(time.Now().UnixNano()))
	g.Pad = platform.InitPad() // the mini joystick module, if it is connected
	println("28_raptor: started")
	for {
		g.Run()
		println("28_raptor: Y, again")
	}
}
