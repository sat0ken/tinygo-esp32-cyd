// 23_train runs the Train of PicoLibSDK (package games/train), ported from
// the PicoPad. The train collects the things on the board, each one a new
// wagon, and leaves through the gate, which opens when everything is
// collected. The arrows steer, A enters a password, X shows the solution,
// Y starts the level again; a key ends the demo, Y held there ends Run.
//
// The mini joystick module (platform.InitPad) has the PicoPad's buttons:
// the stick is the d-pad. Without it, touch the display: it is split into
// 3 x 3 buttons (package picopad). In the browser, the arrow keys and Z
// (A), X (B), A (X), S (Y).
//
//	make flash-s028 PKG=./examples/23_train   # ESP32-2432S028R
//	make wasm PKG=./examples/23_train && make serve
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/games/train"
	"github.com/sat0ken/tinygo-cyd/platform"
)

func main() {
	time.Sleep(500 * time.Millisecond)
	d, t, err := platform.Init()
	if err != nil {
		println("platform.Init:", err.Error())
		platform.Halt()
	}
	g := train.New(d, t, uint32(time.Now().UnixNano()))
	g.Pad = platform.InitPad() // the mini joystick module, if it is connected
	println("23_train: started")
	for {
		g.Run()
		println("23_train: Y, again")
	}
}
