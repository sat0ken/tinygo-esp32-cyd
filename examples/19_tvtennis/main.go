// 19_tvtennis runs the TV Tennis of PicoLibSDK (package games/tvtennis),
// ported from the PicoPad. The first screen chooses: LEFT or RIGHT plays
// one paddle against the computer, DOWN is two players, UP a demo, A one
// player with both paddles tied. UP/DOWN move the left paddle, X/A the
// right one, LEFT, RIGHT or B serves, Y goes back; Y on the first screen
// ends the game and it starts again (on the PicoPad: back to the boot
// loader).
//
// The mini joystick module (platform.InitPad) has the PicoPad's buttons:
// the stick is the d-pad. Without it, touch the display: it is split into
// 3 x 3 buttons (package picopad). In the browser, the arrow keys and Z
// (A), X (B), A (X), S (Y).
//
//	make flash-s028 PKG=./examples/19_tvtennis   # ESP32-2432S028R
//	make wasm PKG=./examples/19_tvtennis && make serve
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/games/tvtennis"
	"github.com/sat0ken/tinygo-cyd/platform"
)

func main() {
	time.Sleep(500 * time.Millisecond)
	d, t, err := platform.Init()
	if err != nil {
		println("platform.Init:", err.Error())
		platform.Halt()
	}
	g := tvtennis.New(d, t, uint32(time.Now().UnixNano()))
	g.Pad = platform.InitPad() // the mini joystick module, if it is connected
	println("19_tvtennis: started")
	for {
		g.Run()
		println("19_tvtennis: Y, again")
	}
}
