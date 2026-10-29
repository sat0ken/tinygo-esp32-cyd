// 24_picoinvaders runs the Space Invaders of PicoLibSDK (package
// games/picoinvaders), ported from the PicoPad. (Example 08_invaders is
// another one, written for this repository.) LEFT and RIGHT move the ship,
// A shoots, Y ends Run; after GAME OVER a key starts a new game.
//
// The mini joystick module (platform.InitPad) has the PicoPad's buttons:
// the stick is the d-pad. Without it, touch the display: it is split into
// 3 x 3 buttons (package picopad). In the browser, the arrow keys and Z
// (A), S (Y).
//
//	make flash-s028 PKG=./examples/24_picoinvaders   # ESP32-2432S028R
//	make wasm PKG=./examples/24_picoinvaders && make serve
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/games/picoinvaders"
	"github.com/sat0ken/tinygo-cyd/platform"
)

func main() {
	time.Sleep(500 * time.Millisecond)
	d, t, err := platform.Init()
	if err != nil {
		println("platform.Init:", err.Error())
		platform.Halt()
	}
	g := picoinvaders.New(d, t, uint32(time.Now().UnixNano()))
	g.Pad = platform.InitPad() // the mini joystick module, if it is connected
	println("24_picoinvaders: started")
	for {
		g.Run()
		println("24_picoinvaders: Y, again")
	}
}
