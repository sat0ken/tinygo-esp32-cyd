// 35_baseball runs BASEBALL (package games/baseball), an original baseball
// game in the picopad style: BLUE STARS against RED COMETS, nine players
// with their abilities each. On the title, UP/DOWN choose 1P vs CPU or CPU
// vs CPU, LEFT/RIGHT the innings, A starts; Y ends Run.
//
// Batting: the arrows move the meet cursor, A swings, B switches between a
// meet swing and a power swing (a smaller cursor, harder hit). Pitching:
// an arrow chooses the breaking ball (A the straight ball), A to aim, the
// arrows aim, A pitches. Y in the game asks to quit.
//
// The mini joystick module (platform.InitPad) has the PicoPad's buttons:
// the stick is the d-pad. Without it, touch the display: it is split into
// 3 x 3 buttons (package picopad). In the browser, the arrow keys and Z
// (A), X (B), S (Y).
//
//	make flash-s028 PKG=./examples/35_baseball   # ESP32-2432S028R
//	make wasm PKG=./examples/35_baseball && make serve
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/games/baseball"
	"github.com/sat0ken/tinygo-cyd/platform"
)

func main() {
	time.Sleep(500 * time.Millisecond)
	d, t, err := platform.InitFull() // on the S035, the screen 4/3 bigger
	if err != nil {
		println("platform.InitFull:", err.Error())
		platform.Halt()
	}
	g := baseball.New(d, t, uint32(time.Now().UnixNano()))
	g.Pad = platform.InitPad() // the mini joystick module, if it is connected
	println("35_baseball: started")
	for {
		g.Run()
		println("35_baseball: Y, again")
	}
}
