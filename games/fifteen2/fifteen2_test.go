package fifteen2

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
	clock := picotest.NewClock(t, 10*time.Minute)
	g.Clock = clock
	p := &picotest.Pad{Clock: clock}
	g.Pad = p
	return g, d, clock, p
}

// inOrder reports whether the tiles are sorted (the hole last).
func inOrder(g *Game) bool {
	for i, b := range g.board {
		if int(b) != i {
			return false
		}
	}
	return true
}

func TestSelect(t *testing.T) {
	g, d, clock, p := newTest(t)
	clock.At(200*ms, func() { goldentest.Check(t, "fifteen2_select", d.Image()) })
	p.Tap(hal.PadDown, 400*ms)
	p.Tap(hal.PadDown, 600*ms) // Kittens in Box
	p.Tap(hal.PadA, 800*ms)
	clock.At(3*time.Second, func() {
		if g.puzzleInx != 2 || g.img != kittensInBoxImg || inOrder(g) {
			t.Errorf("game: picture %d, in order %v", g.puzzleInx, inOrder(g))
		}
		goldentest.Check(t, "fifteen2_game", d.Image())
	})
	p.Tap(hal.PadA, 3200*ms) // the numbers
	clock.At(3500*ms, func() {
		if !g.dispTileNum {
			t.Error("A: no numbers")
		}
		goldentest.Check(t, "fifteen2_numbers", d.Image())
	})
	p.Tap(hal.PadY, 4*time.Second) // back to the pictures
	p.Tap(hal.PadRight, 4300*ms)   // the last one
	p.Tap(hal.PadY, 4600*ms)
	clock.At(4500*ms, func() {
		if g.puzzleInx != len(puzzleList)-1 {
			t.Errorf("RIGHT: picture %d", g.puzzleInx)
		}
	})
	g.Run()
}

// TestSolver lets the solver play: then the whole picture, a key goes back
// to the pictures.
func TestSolver(t *testing.T) {
	g, d, clock, p := newTest(t)
	p.Tap(hal.PadA, 200*ms) // Digits
	p.Tap(hal.PadX, 3*time.Second)
	var solvedAt time.Duration
	var watch func()
	watch = func() {
		if !inOrder(g) {
			clock.At(clock.T+100*ms, watch)
			return
		}
		solvedAt = clock.T
		clock.At(solvedAt+2*time.Second, func() {
			if !g.gameEnd {
				t.Error("solved: not the end of the game")
			}
			goldentest.Check(t, "fifteen2_solved", d.Image())
		})
		p.Tap(hal.PadB, solvedAt+2500*ms)
		p.Tap(hal.PadY, solvedAt+3500*ms)
	}
	clock.At(3100*ms, watch)
	g.Run()
	if solvedAt == 0 {
		t.Fatal("not solved")
	}
}
