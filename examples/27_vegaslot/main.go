// 27_vegaslot runs the Vegaslot slot machine of PicoLibSDK (package
// games/vegaslot), ported from the PicoPad. A starts a game, X autostarts,
// B changes the bet, LEFT plays the even/odd toss, UP fills 10 credits,
// DOWN moves the bank to the credit, Y ends Run; after a win LEFT/RIGHT
// move it to the bank/credit and A risks it.
//
// The mini joystick module (platform.InitPad) has the PicoPad's buttons:
// the stick is the d-pad. Without it, touch the display: it is split into
// 3 x 3 buttons (package picopad). In the browser, the arrow keys and Z
// (A), X (B), A (X), S (Y).
//
//	make flash-s028 PKG=./examples/27_vegaslot   # ESP32-2432S028R
//	make wasm PKG=./examples/27_vegaslot && make serve
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/games/vegaslot"
	"github.com/sat0ken/tinygo-cyd/platform"
)

func main() {
	time.Sleep(500 * time.Millisecond)
	d, t, err := platform.Init()
	if err != nil {
		println("platform.Init:", err.Error())
		platform.Halt()
	}
	g := vegaslot.New(d, t, uint32(time.Now().UnixNano()))
	g.Pad = platform.InitPad() // the mini joystick module, if it is connected
	println("27_vegaslot: started")
	for {
		g.Run()
		println("27_vegaslot: Y, again")
	}
}
