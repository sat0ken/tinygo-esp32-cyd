// 10_othello is othello (reversi) against the CPU on the touch screen.
//
//   - In the menu, choose your colour (black moves first) and the CPU level
//     (EASY / NORMAL / HARD); tapping the level starts the game.
//   - Tap a square with a small dot to play there. The last move has a red
//     dot. UNDO takes back your last move (and the CPU's reply), NEW goes
//     back to the menu.
//   - The CPU searches with alpha-beta; HARD looks 6 moves ahead and reads
//     the last 12 moves to the end. Its depth, nodes and time are shown on
//     the right and in the log.
//
// It uses package platform, so it runs on the board and in the browser:
//
//	make flash-noerase PKG=./examples/10_othello && make monitor
//	make wasm PKG=./examples/10_othello && make serve
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/platform"
)

func main() {
	time.Sleep(500 * time.Millisecond)
	d, t, err := platform.Init()
	if err != nil {
		println("platform.Init:", err.Error())
		platform.Halt()
	}
	g := NewGame(d, t, uint32(time.Now().UnixNano()))
	g.DrawMenu()
	println("10_othello: started")

	last := time.Now()
	for {
		now := time.Now()
		dt := float32(now.Sub(last).Seconds())
		last = now
		if dt > 0.1 {
			dt = 0.1 // after the CPU has been thinking, do not skip the animation
		}
		g.Update(dt)
		time.Sleep(15 * time.Millisecond)
	}
}
