// 13_kart is the first prototype of a kart racing game (package
// games/kart): drive round the course, see how fast the ground can be
// drawn. Every second the log shows the frames per second, the time of a
// frame and how much of it is spent sending to the panel.
//
// With the mini joystick module: the stick steers, A accelerates, B
// brakes. Without it: touch to accelerate, the touch's side steers.
//
//	make flash-s028 PKG=./examples/13_kart   # ESP32-2432S028R
//	make wasm PKG=./examples/13_kart && make serve
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/games/kart"
	"github.com/sat0ken/tinygo-cyd/platform"
)

func main() {
	time.Sleep(500 * time.Millisecond)
	d, t, err := platform.Init()
	if err != nil {
		println("platform.Init:", err.Error())
		platform.Halt()
	}
	g := kart.NewGame(d, t)
	g.Pad = platform.InitPad() // the mini joystick module, if it is connected
	g.DrawAll()
	println("13_kart: started")

	last := time.Now()
	var shown kart.Stats
	for {
		now := time.Now()
		dt := float32(now.Sub(last).Seconds())
		last = now
		if dt > 0.1 {
			dt = 0.1
		}
		g.Update(dt) // as fast as it goes: the frame time is what we measure
		if s := g.Stats(); s != shown {
			shown = s
			println("fps", s.FPS, "frame", s.Frame.Microseconds(), "us send", s.Send.Microseconds(),
				"us speed", s.Speed, "at", s.X, s.Y, "road", s.OnRoad)
		}
	}
}
