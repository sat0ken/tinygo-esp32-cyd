// 31_fifteen2 runs the Fifteen puzzle with pictures of PicoLibSDK (package
// games/fifteen2), ported from the PicoPad. UP/DOWN choose the picture, A
// starts; the arrows move a tile into the hole, A shows the numbers, X
// lets the solver play, Y goes back to the pictures; Y there ends Run.
//
// The mini joystick module (platform.InitPad) has the PicoPad's buttons:
// the stick is the d-pad. Without it, touch the display: it is split into
// 3 x 3 buttons (package picopad). In the browser, the arrow keys and Z
// (A), A (X), S (Y).
//
//	make flash-s028 PKG=./examples/31_fifteen2   # ESP32-2432S028R
//	make wasm PKG=./examples/31_fifteen2 && make serve
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/games/fifteen2"
	"github.com/sat0ken/tinygo-cyd/platform"
)

func main() {
	time.Sleep(500 * time.Millisecond)
	d, t, err := platform.Init()
	if err != nil {
		println("platform.Init:", err.Error())
		platform.Halt()
	}
	g := fifteen2.New(d, t, uint32(time.Now().UnixNano()))
	g.Pad = platform.InitPad() // the mini joystick module, if it is connected
	println("31_fifteen2: started")
	for {
		g.Run()
		println("31_fifteen2: Y, again")
	}
}
