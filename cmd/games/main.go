// Command games is a launcher for the games: choose INVADERS, SUDOKU,
// OTHELLO or SOLITAIRE on the first screen; the "< GAMES" button of a game
// comes back here.
//
// The games are the packages games/invaders, games/sudoku, games/othello
// and games/solitaire; examples/08 to 11 run each of them alone.
//
//	make flash-noerase PKG=./cmd/games && make monitor
//	make wasm PKG=./cmd/games && make serve
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/platform"
)

const frameTime = time.Second / 60

func main() {
	time.Sleep(500 * time.Millisecond)
	d, t, err := platform.Init()
	if err != nil {
		println("platform.Init:", err.Error())
		platform.Halt()
	}
	n := uint32(0)
	l := NewLauncher(d, t, func() uint32 {
		n++
		return uint32(time.Now().UnixNano()) ^ n*0x9E3779B9
	})
	l.DrawMenu()
	println("games: started")

	vs, hasVSync := d.(vsyncWaiter)
	last := time.Now()
	for {
		now := time.Now()
		dt := float32(now.Sub(last).Seconds())
		last = now
		if dt > 0.05 {
			dt = 0.05 // after a long frame (e.g. the CPU thinking), do not jump
		}
		if hasVSync {
			vs.WaitVSync(50 * time.Millisecond) // draw right after VSYNC: no tearing
		}
		l.Update(dt)
		if !hasVSync {
			if rest := frameTime - time.Since(now); rest > 0 {
				time.Sleep(rest)
			}
		}
	}
}
