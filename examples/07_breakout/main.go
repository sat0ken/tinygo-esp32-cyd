// 07_breakout is a breakout game for the touch screen.
//
//   - Slide your finger (or the mouse in the browser) to move the paddle.
//   - Tap to launch the ball, and to restart after GAME OVER / CLEAR.
//   - 50 bricks, 3 lives; each level is 10% faster.
//
// Only what moved is redrawn each frame (ball, paddle, removed bricks,
// score), and on the board drawing starts right after VSYNC, so there is
// no flicker. It uses package platform, so it runs on the board and in the
// browser:
//
//	make flash-noerase PKG=./examples/07_breakout && make monitor
//	make wasm PKG=./examples/07_breakout && make serve
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
	g := NewGame(d, t)
	g.DrawAll()
	println("07_breakout: started")

	vs, hasVSync := d.(vsyncWaiter)
	last := time.Now()
	frames, fpsStart := 0, last
	for {
		now := time.Now()
		dt := float32(now.Sub(last).Seconds())
		last = now
		if dt > 0.05 {
			dt = 0.05 // after a pause (e.g. slow touch read), do not jump
		}
		g.Step(dt)
		if hasVSync {
			vs.WaitVSync(50 * time.Millisecond)
		}
		g.Draw()

		frames++
		if el := now.Sub(fpsStart); el >= 5*time.Second {
			println("fps", frames*1000/int(el.Milliseconds()), "score", g.score, "lives", g.lives, "level", g.level)
			frames, fpsStart = 0, now
		}
		if !hasVSync {
			// The browser has no VSYNC wait: pace to about 60 frames per second.
			if rest := frameTime - time.Since(now); rest > 0 {
				time.Sleep(rest)
			}
		}
	}
}
