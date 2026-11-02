// 34_mahjong runs riichi mahjong against three CPUs (package games/mahjong)
// alone. cmd/games has it together with the other games.
//
// The game:
//
//   - In the menu, choose tonpusen (east only) or hanchan (east and south);
//     tapping it starts the game.
//   - Tap a tile to lift it, tap it again to discard it. The buttons (tsumo,
//     riichi, kan, ron, pon, chi, pass, in katakana) appear when they can be
//     used.
//   - With the mini joystick module the stick moves between the tiles and the
//     buttons, A discards or presses, B passes.
//
// It uses package platform, so it runs on the board and in the browser:
//
//	make flash-noerase PKG=./examples/34_mahjong && make monitor
//	make wasm PKG=./examples/34_mahjong && make serve
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/games/mahjong"
	"github.com/sat0ken/tinygo-cyd/platform"
)

func main() {
	time.Sleep(500 * time.Millisecond)
	d, t, err := platform.Init()
	if err != nil {
		println("platform.Init:", err.Error())
		platform.Halt()
	}
	g := mahjong.NewGame(d, t, uint32(time.Now().UnixNano()))
	g.Pad = platform.InitPad() // the mini joystick module, if it is connected
	g.DrawMenu()
	println("34_mahjong: started")

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
