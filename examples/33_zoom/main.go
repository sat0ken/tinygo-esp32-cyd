// 33_zoom runs ZOOM, renamed Gate Runner, of PicoLibSDK (package
// games/zoom), ported from the PicoPad. A ship flies low over a planet
// through the gates of pylons and shoots the saucers. A starts on the
// title; LEFT/RIGHT steer, UP/DOWN change the speed, A shoots, Y ends the
// game; Y on the title ends Run.
//
// The mini joystick module (platform.InitPad) has the PicoPad's buttons:
// the stick is the d-pad. Without it, touch the display: it is split into
// 3 x 3 buttons (package picopad). In the browser, the arrow keys and Z
// (A), S (Y).
//
//	make flash-s028 PKG=./examples/33_zoom   # ESP32-2432S028R
//	make wasm PKG=./examples/33_zoom && make serve
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/games/zoom"
	"github.com/sat0ken/tinygo-cyd/platform"
)

func main() {
	time.Sleep(500 * time.Millisecond)
	d, t, err := platform.Init()
	if err != nil {
		println("platform.Init:", err.Error())
		platform.Halt()
	}
	g := zoom.New(d, t, uint32(time.Now().UnixNano()))
	g.Pad = platform.InitPad() // the mini joystick module, if it is connected
	println("33_zoom: started")
	for {
		g.Run()
		println("33_zoom: Y, again")
	}
}
