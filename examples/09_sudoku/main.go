// 09_sudoku is a sudoku game for the touch screen.
//
//   - Choose EASY / MEDIUM / HARD (38 / 30 / 25 clues).
//   - Tap a cell, then a digit on the pad. ERASE clears the cell, HINT
//     fills the selected cell (or a random one), NEW goes back to the menu.
//   - The selected cell's row, column and box, and cells with the same
//     digit, are highlighted. Wrong digits are red and counted as misses.
//   - Every puzzle has exactly one solution.
//
// Inspired by Sparkadium/Sudoku-for-CYD (an Arduino sketch for the
// ESP32-2432S028R); this is an independent implementation, no code was
// taken from it. It uses package platform, so it runs on the board and in
// the browser:
//
//	make flash-noerase PKG=./examples/09_sudoku && make monitor
//	make wasm PKG=./examples/09_sudoku && make serve
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
	// The seed comes from the time of the tap that starts a game, which
	// differs every time even right after boot.
	n := uint32(0)
	g := NewGame(d, t, func() uint32 {
		n++
		return uint32(time.Now().UnixNano()) ^ n*0x9E3779B9
	})
	g.DrawMenu()
	println("09_sudoku: started")

	last := time.Now()
	for {
		now := time.Now()
		dt := float32(now.Sub(last).Seconds())
		last = now
		if dt > 0.1 {
			dt = 0.1 // after generating a puzzle, do not count it as play time
		}
		g.Update(dt)
		time.Sleep(20 * time.Millisecond)
	}
}
