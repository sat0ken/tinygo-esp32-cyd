// 30_ghostrac runs the Ghost Racing of PicoLibSDK (package
// games/ghostrac), ported from the PicoPad. A car race on a track seen in
// perspective: UP or X speeds up, DOWN or A brakes, LEFT/RIGHT turn, Y
// ends Run. The ghost drives the best lap again.
//
// The mini joystick module (platform.InitPad) has the PicoPad's buttons:
// the stick is the d-pad. Without it, touch the display: it is split into
// 3 x 3 buttons (package picopad). In the browser, the arrow keys and Z
// (A), A (X), S (Y).
//
//	make flash-s028 PKG=./examples/30_ghostrac   # ESP32-2432S028R
//	make wasm PKG=./examples/30_ghostrac && make serve
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/games/ghostrac"
	"github.com/sat0ken/tinygo-cyd/platform"
)

func main() {
	time.Sleep(500 * time.Millisecond)
	d, t, err := platform.Init()
	if err != nil {
		println("platform.Init:", err.Error())
		platform.Halt()
	}
	g := ghostrac.New(d, t)
	g.Pad = platform.InitPad() // the mini joystick module, if it is connected
	println("30_ghostrac: started")
	for {
		g.Run()
		println("30_ghostrac: Y, again")
	}
}
