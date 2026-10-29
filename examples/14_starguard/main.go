// 14_starguard is the first prototype of a space shooting game (package
// games/starguard): rocks and robot ships come at the ship; aim and fire
// before they hit the shield. Every second the log shows the frames per
// second and the time of a frame.
//
// With the mini joystick module: the stick moves the crosshair, A fires.
// Without it: touch the view to aim there and fire.
//
//	make flash-s028 PKG=./examples/14_starguard   # ESP32-2432S028R
//	make wasm PKG=./examples/14_starguard && make serve
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/games/starguard"
	"github.com/sat0ken/tinygo-cyd/platform"
)

func main() {
	time.Sleep(500 * time.Millisecond)
	d, t, err := platform.Init()
	if err != nil {
		println("platform.Init:", err.Error())
		platform.Halt()
	}
	g := starguard.NewGame(d, t, uint32(time.Now().UnixNano()))
	g.Pad = platform.InitPad() // the mini joystick module, if it is connected
	g.DrawAll()
	println("14_starguard: started")

	last := time.Now()
	var shown starguard.Stats
	for {
		now := time.Now()
		dt := float32(now.Sub(last).Seconds())
		last = now
		if dt > 0.1 {
			dt = 0.1
		}
		g.Update(dt)
		if s := g.Stats(); s != shown {
			shown = s
			println("fps", s.FPS, "frame", s.Frame.Microseconds(), "us send", s.Send.Microseconds(),
				"us enemies", s.Enemies, "score", s.Score)
		}
	}
}
