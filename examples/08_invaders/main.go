// 08_invaders is a space invaders game for the touch screen.
//
//   - Slide your finger (or the mouse in the browser) to move the cannon.
//   - While you touch the screen the cannon fires (one shot at a time).
//   - Tap to start, and to play again after GAME OVER.
//
// 55 invaders move one per frame like the original, so they speed up as
// they are destroyed. Bunkers crumble, a UFO flies over now and then, each
// wave starts lower. Sprites are redrawn together with the area they left
// in one transfer, and on the board drawing starts right after VSYNC.
// It uses package platform, so it runs on the board and in the browser:
//
//	make flash-noerase PKG=./examples/08_invaders && make monitor
//	make wasm PKG=./examples/08_invaders && make serve
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/platform"
)

// vsyncWaiter is implemented by rgblcd.Device.
type vsyncWaiter interface {
	WaitVSync(timeout time.Duration) bool
}

const frameTime = time.Second / 60

func main() {
	time.Sleep(500 * time.Millisecond)
	d, t, err := platform.Init()
	if err != nil {
		println("platform.Init:", err.Error())
		platform.Halt()
	}
	g := NewGame(d, t, uint32(time.Now().UnixNano()))
	g.DrawAll()
	println("08_invaders: started")

	vs, hasVSync := d.(vsyncWaiter)
	last := time.Now()
	frames, fpsStart := 0, last
	for {
		now := time.Now()
		dt := float32(now.Sub(last).Seconds())
		last = now
		if dt > 0.05 {
			dt = 0.05
		}
		if hasVSync {
			vs.WaitVSync(50 * time.Millisecond)
		}
		g.Update(dt)

		frames++
		if el := now.Sub(fpsStart); el >= 5*time.Second {
			println("fps", frames*1000/int(el.Milliseconds()), "score", g.score, "lives", g.lives, "wave", g.wave, "invaders", g.alive)
			frames, fpsStart = 0, now
		}
		if !hasVSync {
			if rest := frameTime - time.Since(now); rest > 0 {
				time.Sleep(rest)
			}
		}
	}
}
