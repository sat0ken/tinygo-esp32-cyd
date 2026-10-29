// 32_pictor runs Pictor (Picopad Collector) of PicoLibSDK (package
// games/pictor), ported from the PicoPad. Bara runs through 12 scenes and
// collects the parts of a PicoPad. A starts on the title; the arrows move
// (UP jumps for the walking ones), A shoots, B/X transform Bara into the
// helpers, Y opens the game menu; X in the menu of the title ends Run.
//
// The mini joystick module (platform.InitPad) has the PicoPad's buttons:
// the stick is the d-pad. Without it, touch the display: it is split into
// 3 x 3 buttons (package picopad). In the browser, the arrow keys and Z
// (A), X (B), A (X), S (Y).
//
//	make flash-s028 PKG=./examples/32_pictor   # ESP32-2432S028R
//	make wasm PKG=./examples/32_pictor && make serve
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/games/pictor"
	"github.com/sat0ken/tinygo-cyd/platform"
)

func main() {
	time.Sleep(500 * time.Millisecond)
	d, t, err := platform.Init()
	if err != nil {
		println("platform.Init:", err.Error())
		platform.Halt()
	}
	g := pictor.New(d, t, uint32(time.Now().UnixNano()))
	g.Pad = platform.InitPad() // the mini joystick module, if it is connected
	println("32_pictor: started")
	for {
		g.Run()
		println("32_pictor: Y, again")
	}
}
