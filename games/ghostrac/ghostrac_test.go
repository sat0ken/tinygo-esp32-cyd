package ghostrac

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
	g := New(d, nil)
	clock := picotest.NewClock(t, 30*time.Minute)
	clock.Step = 5 * time.Millisecond // the game draws in a loop: about 20 frames a second
	g.Clock = clock
	p := &picotest.Pad{Clock: clock}
	g.Pad = p
	return g, d, clock, p
}

// at puts the car on the middle of a tile.
func at(g *Game, x, y int) {
	g.carX = (float32(x) + 0.5) * tileSize
	g.carY = (float32(y) + 0.5) * tileSize
}

func TestRace(t *testing.T) {
	g, d, clock, p := newTest(t)
	clock.At(300*ms, func() {
		goldentest.Check(t, "ghostrac_start", d.Image())
		if g.checkNext != grass || g.speed != 0 {
			t.Errorf("start: next %d, speed %v", g.checkNext, g.speed)
		}
	})

	// UP: north along the straight, over the start line (row 14)
	p.Hold(hal.PadUp, time.Second, 4*time.Second)
	clock.At(4*time.Second, func() {
		if g.checkNext != checkA || g.speed < 50 || g.carDir != 0 || g.carY >= 15*tileSize {
			t.Errorf("driving: next %d, speed %v, direction %v, y %v", g.checkNext, g.speed, g.carDir, g.carY/tileSize)
		}
	})
	clock.At(5500*ms, func() {
		goldentest.Check(t, "ghostrac_drive", d.Image())
	})

	// the check points in the order, then the start: a lap
	clock.At(6*time.Second, func() { at(g, 23, 11) })
	clock.At(6500*ms, func() {
		at(g, 11, 14)
		if !g.checkOK[0] || g.checkNext != checkB {
			t.Errorf("A: %v, next %d", g.checkOK, g.checkNext)
		}
	})
	clock.At(7*time.Second, func() { at(g, 22, 20) })
	clock.At(7500*ms, func() { at(g, 8, 14) })
	clock.At(8*time.Second, func() {
		if g.lastTime < 4 || g.bestTime != g.lastTime || !g.newBest || g.recordBestNum < 3 || g.checkNext != checkA {
			t.Errorf("a lap: last %d, best %d, %d records, next %d", g.lastTime, g.bestTime, g.recordBestNum, g.checkNext)
		}
		// back behind the start: the ghost drives the lap again ahead
		at(g, 8, 15)
		g.carY += 40
		g.speed = 0
	})
	p.Hold(hal.PadUp, 8100*ms, 1500*ms)
	clock.At(9500*ms, func() {
		goldentest.Check(t, "ghostrac_ghost", d.Image())
	})
	p.Tap(hal.PadY, 10*time.Second)
	g.Run()
}
