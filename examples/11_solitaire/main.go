// 11_solitaire runs klondike solitaire (package games/solitaire) alone.
// cmd/games has it together with the other games.
//
// The game:
//
//   - Tap a card to move it (with the cards on it) where it fits: a single
//     card goes home to its foundation if it can, otherwise to a column.
//   - Tap the stock to turn cards over; when it is empty, tapping it turns
//     the waste over again.
//   - NEW deals again, UNDO takes back moves, DRAW 1 / DRAW 3 switches how
//     many cards are turned over (and deals again).
//   - When every card is face up and the stock is empty, the rest goes home
//     by itself.
//
// It uses package platform, so it runs on the board and in the browser:
//
//	make flash-noerase PKG=./examples/11_solitaire && make monitor
//	make wasm PKG=./examples/11_solitaire && make serve
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/games/solitaire"
	"github.com/sat0ken/tinygo-cyd/platform"
)

func main() {
	time.Sleep(500 * time.Millisecond)
	d, t, err := platform.Init()
	if err != nil {
		println("platform.Init:", err.Error())
		platform.Halt()
	}
	n := uint32(0)
	g := solitaire.NewGame(d, t, func() uint32 {
		n++
		return uint32(time.Now().UnixNano()) ^ n*0x9E3779B9
	})
	g.Pad = platform.InitPad() // the mini joystick module, if it is connected
	g.Start()
	println("11_solitaire: started")

	last := time.Now()
	for {
		now := time.Now()
		dt := float32(now.Sub(last).Seconds())
		last = now
		if dt > 0.1 {
			dt = 0.1
		}
		g.Update(dt)
		time.Sleep(15 * time.Millisecond)
	}
}
