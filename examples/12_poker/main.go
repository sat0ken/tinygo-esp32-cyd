// 12_poker runs "Jacks or Better" video poker (package games/poker) alone.
// cmd/games has it together with the other games.
//
// The game:
//
//   - BET ONE raises the bet (1 to 5 credits), MAX BET bets 5 and deals,
//     DEAL deals five cards.
//   - Tap the cards to keep (HELD), then DRAW replaces the others.
//   - The hand is paid by the table at the top: a pair of jacks or better
//     gets the bet back, a royal flush with 5 credits bet pays 4000. The
//     column of the bet is red; the row of the hand lights up.
//   - With no credits left, DEAL gives 100 new ones.
//
// It uses package platform, so it runs on the board and in the browser:
//
//	make flash-noerase PKG=./examples/12_poker && make monitor
//	make wasm PKG=./examples/12_poker && make serve
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/games/poker"
	"github.com/sat0ken/tinygo-cyd/platform"
)

func main() {
	time.Sleep(500 * time.Millisecond)
	d, t, err := platform.Init()
	if err != nil {
		println("platform.Init:", err.Error())
		platform.Halt()
	}
	g := poker.NewGame(d, t, func() uint32 { return uint32(time.Now().UnixNano()) })
	g.Pad = platform.InitPad() // the mini joystick module, if it is connected
	g.Start()
	println("12_poker: started")

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
