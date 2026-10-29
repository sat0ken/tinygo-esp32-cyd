// Command games is a launcher for the games: choose INVADERS, SUDOKU,
// OTHELLO, SOLITAIRE or POKER on the first screen; the "< GAMES" button of
// a game comes back here.
//
// The games are the packages games/invaders, games/sudoku, games/othello,
// games/solitaire and games/poker; examples/08 to 12 run each of them
// alone.
//
// With the mini joystick module connected (see platform.InitPad), the
// stick moves a frame between the tiles and the buttons of the games, A
// presses what is framed, B goes back; in the browser the arrow keys and
// Z (A), X (B), A (X), S (Y) do the same.
//
//	make flash-noerase PKG=./cmd/games && make monitor   # ESP32-4827S043
//	make flash-s035 PKG=./cmd/games                      # ESP32-3248S035C
//	make flash-s028 PKG=./cmd/games                      # ESP32-2432S028R (320x240 layouts)
//	make wasm PKG=./cmd/games && make serve
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/platform"
)

const frameTime = time.Second / 60

func main() {
	println("games: main")
	time.Sleep(500 * time.Millisecond)
	println("games: init")
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
	l.Pad = platform.InitPad() // the mini joystick module, if it is connected
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
