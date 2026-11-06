// 36_futsal runs FUTSAL (package games/futsal), an original five-a-side
// football game on one screen seen from above: SKY HAWKS against RED
// FOXES. On the title, UP/DOWN choose 1P vs CPU or CPU vs CPU, LEFT/RIGHT
// the length of a half (1 to 3 minutes), A kicks off; Y ends Run.
//
// The arrows run the player under the yellow mark. With the ball A passes
// (to the teammate the arrows point at) and B shoots (hold it for a harder
// shot; UP or DOWN as it is let go aim high or low). Without the ball B
// tackles and A changes to the player nearest the ball. Y in the game asks
// to quit.
//
// The mini joystick module (platform.InitPad) has the PicoPad's buttons:
// the stick is the d-pad. Without it, touch the display: it is split into
// 3 x 3 buttons (package picopad). In the browser, the arrow keys and Z
// (A), X (B), S (Y).
//
//	make flash-s028 PKG=./examples/36_futsal   # ESP32-2432S028R
//	make wasm PKG=./examples/36_futsal && make serve
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/games/futsal"
	"github.com/sat0ken/tinygo-cyd/platform"
)

func main() {
	time.Sleep(500 * time.Millisecond)
	d, t, err := platform.InitFull() // on the S035, the screen 4/3 bigger
	if err != nil {
		println("platform.InitFull:", err.Error())
		platform.Halt()
	}
	g := futsal.New(d, t, uint32(time.Now().UnixNano()))
	g.Pad = platform.InitPad() // the mini joystick module, if it is connected
	println("36_futsal: started")
	for {
		g.Run()
		println("36_futsal: Y, again")
	}
}
