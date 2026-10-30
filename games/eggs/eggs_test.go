package eggs

import (
	"testing"
	"time"

	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/internal/goldentest"
	"github.com/sat0ken/tinygo-cyd/internal/picotest"
	"github.com/sat0ken/tinygo-cyd/memlcd"
)

const ms = time.Millisecond

func newTest(t *testing.T) (*Game, *memlcd.Display, *picotest.Clock, *picotest.Pad) {
	d := memlcd.New(320, 240)
	g := New(d, nil, 12345)
	clock := picotest.NewClock(t, 30*time.Minute)
	g.Clock = clock
	p := &picotest.Pad{Clock: clock}
	g.Pad = p
	return g, d, clock, p
}

// count returns the eggs and the hens on the board.
func count(g *Game) (eggs, hens int) {
	for _, b := range g.board {
		switch (b-tileEmpty)%(tileOnMin-tileOffMin) + tileEmpty {
		case tileEgg, tileEggWin:
			eggs++
		case tileHen, tileHenWin:
			hens++
		}
	}
	return
}

func TestScreens(t *testing.T) {
	g, d, clock, p := newTest(t)
	clock.At(100*ms, func() { goldentest.Check(t, "eggs_open", d.Image()) })
	p.Tap(hal.PadDown, 200*ms) // 2 players
	clock.At(400*ms, func() {
		goldentest.Check(t, "eggs_game", d.Image())
		if g.pos != 12 || g.free != 60 {
			t.Errorf("start: %d possible places, %d free", g.pos, g.free)
		}
	})
	p.Tap(hal.PadY, 500*ms)
	p.Tap(hal.PadY, 700*ms)
	g.Run()
}

func TestTurn(t *testing.T) {
	g := New(memlcd.New(320, 240), nil, 1)
	g.Clock = picotest.NewClock(t, time.Minute)
	g.openGame(false, false)
	g.dispPos()
	// a line turns when it ends with a chicken: with one more at (5, 3),
	// player 1 puts a chicken at (2, 3) and turns (3, 3) and (4, 3)
	g.player = 0
	g.setBoard(5, 3, tileChicken)
	g.free--
	g.curX, g.curY = 2, 3
	g.putStone()
	// the two chickens between become eggs (player 1: chicken -> egg)
	if g.getBoard(3, 3) != tileEgg || g.getBoard(4, 3) != tileEgg || g.stones[0] != 2 {
		t.Errorf("player 1: %d %d, eggs %d", g.getBoard(3, 3), g.getBoard(4, 3), g.stones[0])
	}
	// player 2 from (6, 3) back to (2, 3): the eggs become chickens
	g.player = 1
	g.curX, g.curY = 6, 3
	g.putStone()
	if g.getBoard(3, 3) != tileChicken || g.getBoard(4, 3) != tileChicken || g.stones[0] != 0 {
		t.Errorf("player 2: %d %d, eggs %d", g.getBoard(3, 3), g.getBoard(4, 3), g.stones[0])
	}
	eggs, hens := count(g)
	if eggs != g.stones[0] || hens != g.stones[1] {
		t.Errorf("board %d eggs %d hens, score %v", eggs, hens, g.stones)
	}
}

func TestDemo(t *testing.T) {
	g, _, clock, p := newTest(t)
	p.Tap(hal.PadUp, 100*ms) // the computer plays both
	var watch func()
	ended := time.Duration(0)
	watch = func() {
		if g.player != noPlayer {
			eggs, hens := count(g)
			if eggs != g.stones[0] || hens != g.stones[1] {
				t.Fatalf("board %d eggs %d hens, score %v", eggs, hens, g.stones)
			}
		}
		if g.pos == 0 || g.free == 0 {
			ended = clock.T
			return
		}
		clock.At(clock.T+time.Second, watch)
	}
	clock.At(time.Second, watch)
	var stop func()
	stop = func() {
		if ended == 0 {
			clock.At(clock.T+time.Second, stop)
			return
		}
		p.Tap(hal.PadB, clock.T+3*time.Second) // ends the blinking
		p.Tap(hal.PadY, clock.T+4*time.Second)
	}
	clock.At(time.Second, stop)
	g.Run()
	if ended == 0 {
		t.Fatal("the demo did not end")
	}
	if g.wins[0]+g.wins[1] > 1 {
		t.Errorf("wins %v", g.wins)
	}
	t.Logf("the demo ended after %v: %v stones, wins %v", ended, g.stones, g.wins)
}

// TestOneGame: with OneGame, Run returns after the end of a game (the
// demo, without a key) and on Y in a game.
func TestOneGame(t *testing.T) {
	g, _, clock, p := newTest(t)
	g.OneGame = true
	p.Tap(hal.PadUp, 100*ms) // the computer plays both
	g.Run()
	if g.pos != 0 && g.free != 0 || g.wins[0]+g.wins[1] > 1 {
		t.Errorf("Run returned at %v before the end of the game", clock.T)
	}

	g, _, clock, p = newTest(t)
	g.OneGame = true
	p.Tap(hal.PadDown, 100*ms) // 2 players
	p.Tap(hal.PadY, 500*ms)
	g.Run()
	if clock.T > time.Second {
		t.Errorf("Y: Run returned at %v", clock.T)
	}
}
