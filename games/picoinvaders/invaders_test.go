package picoinvaders

import (
	"testing"
	"time"

	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/internal/goldentest"
	"github.com/sat0ken/tinygo-cyd/internal/picotest"
	"github.com/sat0ken/tinygo-cyd/memlcd"
)

const ms = time.Millisecond

func newTest(t *testing.T, seed uint32) (*Game, *memlcd.Display, *picotest.Clock, *picotest.Pad) {
	d := memlcd.New(320, 240)
	g := New(d, nil, seed)
	clock := picotest.NewClock(t, 30*time.Minute)
	clock.Step = 100 * time.Microsecond // the game waits for its steps in a loop
	g.Clock = clock
	p := &picotest.Pad{Clock: clock}
	g.Pad = p
	return g, d, clock, p
}

func TestStart(t *testing.T) {
	g, d, clock, p := newTest(t, 1)
	clock.At(time.Second, func() {
		goldentest.Check(t, "picoinvaders_start", d.Image())
		if g.lives != 3 || g.alienNum != alienNum || g.shipX != 152 || g.alienX != 72 || g.alienY != 40 {
			t.Errorf("start: %d lives, %d aliens, ship at %d, aliens at %d,%d", g.lives, g.alienNum, g.shipX, g.alienX, g.alienY)
		}
	})
	p.Tap(hal.PadY, 2500*ms)
	g.Run()
}

func TestShootAndMove(t *testing.T) {
	g, _, clock, p := newTest(t, 1)
	p.Hold(hal.PadA, 2100*ms, 100*ms) // the shoot goes up between the houses
	clock.At(3*time.Second, func() {
		// the lowest alien of column 5 (10 points), the block moved left
		if g.score != 10 || g.maxScore != 10 || g.alienNum != alienNum-1 || g.alienLive[4*alienNumX+5] {
			t.Errorf("shoot: score %d (max %d), %d aliens", g.score, g.maxScore, g.alienNum)
		}
	})
	p.Hold(hal.PadRight, 3100*ms, time.Second) // 4 pixels a step of 50ms
	clock.At(4200*ms, func() {
		if g.shipX != 152+4*21 { // the release is seen 50ms later
			t.Errorf("ship at %d after a second of RIGHT", g.shipX)
		}
	})
	p.Tap(hal.PadY, 4500*ms)
	g.Run()
}

func TestHouse(t *testing.T) {
	g, _, _, _ := newTest(t, 1)
	g.newGame(0)
	if g.colHouse != 0xFC00 { // orange of the sprites
		t.Fatalf("house color %#04x", g.colHouse)
	}
	x, y := houseX+8, houseY+4
	before := g.checkHouse(x, y)
	g.shootOn, g.shootX, g.shootY = true, x, y
	g.shootHitHouse()
	if after := g.checkHouse(x, y); g.shootOn || before != 32 || after >= before {
		t.Errorf("shoot into the house: on %v, house pixels %d -> %d", g.shootOn, before, after)
	}
	// between the houses there is nothing to hit
	if n := g.checkHouse(houseX+houseW+4, y); n != 0 {
		t.Errorf("between the houses: %d", n)
	}
}

func TestLevelUp(t *testing.T) {
	g, _, clock, p := newTest(t, 1)
	clock.At(2500*ms, func() { // all the aliens shot
		for i := range g.alienLive {
			g.alienLive[i] = false
		}
		g.alienNum = 0
	})
	clock.At(4*time.Second, func() {
		if g.level != 1 || g.alienNum != alienNum || g.alienY != alienTotY+8 || g.lives != 3 {
			t.Errorf("level up: level %d, %d aliens at y %d, %d lives", g.level, g.alienNum, g.alienY, g.lives)
		}
	})
	p.Tap(hal.PadY, 4500*ms)
	g.Run()
}

// overPad presses B after GAME OVER.
type overPad struct {
	picotest.Pad
	g    *Game
	over time.Duration // when the game was over
}

func (p *overPad) ReadPad() hal.Buttons {
	b := p.Pad.ReadPad()
	if p.g.lives < 0 {
		if p.over == 0 {
			p.over = p.Clock.T
		} else if p.Clock.T > p.over+time.Second {
			b |= hal.PadB
		}
	}
	return b
}

// TestGameOver lets the aliens win: a new game after GAME OVER and a key.
func TestGameOver(t *testing.T) {
	g, _, clock, _ := newTest(t, 1)
	p := &overPad{Pad: picotest.Pad{Clock: clock}, g: g}
	g.Pad = p
	var seen bool
	for s := 10; s < 20*60; s += 10 {
		clock.At(time.Duration(s)*time.Second, func() {
			if p.over != 0 && !seen && g.lives == 3 && g.level == 0 {
				seen = true
				t.Logf("game over at %v", p.over)
				p.Tap(hal.PadY, clock.T+time.Second)
			}
		})
	}
	g.Run()
	if !seen {
		t.Error("no new game after GAME OVER")
	}
}

// TestOneGame: with OneGame, Run returns after GAME OVER and a key.
func TestOneGame(t *testing.T) {
	g, _, clock, _ := newTest(t, 1)
	g.OneGame = true
	p := &overPad{Pad: picotest.Pad{Clock: clock}, g: g}
	g.Pad = p
	g.Run()
	if p.over == 0 || clock.T > p.over+2*time.Second {
		t.Errorf("Run returned at %v, game over at %v", clock.T, p.over)
	}
}
