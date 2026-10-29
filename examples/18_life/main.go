// 18_life runs the Life cell simulator of PicoLibSDK (package games/life),
// ported from the PicoPad: the arrows move the cursor, A flips a cell, B
// runs life until a button, X goes to the next slot (patterns: Life,
// penta-decathlon, R-pentomino, Diehard, Acorn, glider, spaceship, pulsar,
// glider guns), Y ends the game and it starts again (on the PicoPad: back
// to the boot loader).
//
// The mini joystick module (platform.InitPad) has the PicoPad's buttons:
// the stick is the d-pad. Without it, touch the display: it is split into
// 3 x 3 buttons (package picopad). In the browser, the arrow keys and Z
// (A), X (B), A (X), S (Y).
//
//	make flash-s028 PKG=./examples/18_life   # ESP32-2432S028R
//	make flash-s035 PKG=./examples/18_life   # ESP32-3248S035C
//	make wasm PKG=./examples/18_life && make serve
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/games/life"
	"github.com/sat0ken/tinygo-cyd/platform"
)

func main() {
	time.Sleep(500 * time.Millisecond)
	d, t, err := platform.Init()
	if err != nil {
		println("platform.Init:", err.Error())
		platform.Halt()
	}
	g := life.New(d, t)
	g.Pad = platform.InitPad() // the mini joystick module, if it is connected
	println("18_life: started")
	for {
		g.Run()
		println("18_life: Y, again")
	}
}
