package fifteen

import (
	"image"
	"math/rand"
	"testing"
	"time"

	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/internal/goldentest"
	"github.com/sat0ken/tinygo-cyd/internal/picotest"
	"github.com/sat0ken/tinygo-cyd/memlcd"
	"github.com/sat0ken/tinygo-cyd/picopad"
)

const ms = time.Millisecond

func newTest(t *testing.T, w, h int16) (*Game, *memlcd.Display, *picotest.Clock, *picotest.Pad) {
	d := memlcd.New(w, h)
	g := New(d, nil, 12345)
	clock := picotest.NewClock(t, 10*time.Minute)
	g.Clock = clock
	p := &picotest.Pad{Clock: clock}
	g.Pad = p
	return g, d, clock, p
}

// screen returns the PicoPad's screen in the middle of d.
func screen(d *memlcd.Display) *image.RGBA {
	x, y := (int(d.W)-picopad.Width)/2, (int(d.H)-picopad.Height)/2
	img := d.Image()
	out := image.NewRGBA(image.Rect(0, 0, picopad.Width, picopad.Height))
	for yy := 0; yy < picopad.Height; yy++ {
		copy(out.Pix[yy*out.Stride:], img.Pix[(yy+y)*img.Stride+x*4:(yy+y)*img.Stride+(x+picopad.Width)*4])
	}
	return out
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

func TestStart(t *testing.T) {
	for _, size := range []struct{ w, h int16 }{{320, 240}, {480, 272}} {
		g, d, clock, p := newTest(t, size.w, size.h)
		clock.At(1*time.Second, func() {
			// shuffled
			goldentest.Check(t, "fifteen_start", screen(d))
			if inOrder(g) || g.check() {
				t.Errorf("%dx%d: not shuffled", size.w, size.h)
			}
		})
		p.Tap(hal.PadY, 1200*ms)
		g.Run()
		if clock.T > 1500*ms {
			t.Errorf("%dx%d: Run returned at %v", size.w, size.h, clock.T)
		}
	}
}

// TestShiftKeepsCounts checks the tiles out of place and the distance that
// shift keeps up to date against calcPos, which counts them again.
func TestShiftKeepsCounts(t *testing.T) {
	g := New(memlcd.New(320, 240), nil, 1)
	r := rand.New(rand.NewSource(1))
	for round := 0; round < 20; round++ {
		for i := range g.board {
			g.board[i] = uint8(i)
			g.solved[i] = r.Intn(2) == 0
		}
		g.calcPos()
		for n := 0; n < 200; n++ {
			var moves []int
			if g.holeX() > 0 {
				moves = append(moves, -1)
			}
			if g.holeX() < tilesX-1 {
				moves = append(moves, 1)
			}
			if g.holeY() > 0 {
				moves = append(moves, -tilesX)
			}
			if g.holeY() < tilesY-1 {
				moves = append(moves, tilesX)
			}
			g.shift(moves[r.Intn(len(moves))])
			err, dist, pos := g.posErr, g.dist, g.pos
			g.calcPos()
			if g.posErr != err || g.dist != dist || g.pos != pos {
				t.Fatalf("round %d, move %d: shift kept %d out of place, distance %d; calcPos: %d, %d",
					round, n, err, dist, g.posErr, g.dist)
			}
		}
	}
}

func TestMoves(t *testing.T) {
	g, _, clock, p := newTest(t, 320, 240)
	var want Game
	// each button moves the tile next to the hole into it (if there is
	// one that side), as in the C source
	steps := []struct {
		b  hal.Buttons
		sh int
		ok func(*Game) bool
	}{
		{hal.PadRight, -1, func(g *Game) bool { return g.holeX() > 0 }},
		{hal.PadLeft, 1, func(g *Game) bool { return g.holeX() < tilesX-1 }},
		{hal.PadUp, tilesX, func(g *Game) bool { return g.holeY() < tilesY-1 }},
		{hal.PadDown, -tilesX, func(g *Game) bool { return g.holeY() > 0 }},
		{hal.PadDown, -tilesX, func(g *Game) bool { return g.holeY() > 0 }},
		{hal.PadDown, -tilesX, func(g *Game) bool { return g.holeY() > 0 }},
		{hal.PadDown, -tilesX, func(g *Game) bool { return g.holeY() > 0 }}, // at the top by now: nothing
	}
	at := 1 * time.Second
	for _, s := range steps {
		s := s
		clock.At(at, func() {
			want = *g
			if s.ok(&want) {
				want.shift(s.sh)
			}
		})
		p.Tap(s.b, at+50*ms)
		clock.At(at+200*ms, func() {
			if g.board != want.board || g.hole() != want.hole() {
				t.Errorf("button %v: board %v, want %v", s.b, g.board, want.board)
			}
		})
		at += 300 * ms
	}
	p.Tap(hal.PadY, at)
	g.Run()
}

func TestSolver(t *testing.T) {
	g, d, clock, p := newTest(t, 320, 240)
	p.Tap(hal.PadX, 1*time.Second)
	var solvedAt time.Duration
	var watch func()
	watch = func() {
		if !inOrder(g) {
			clock.At(clock.T+100*ms, watch)
			return
		}
		solvedAt = clock.T
		// after the blinking, the sorted board waits for a button
		clock.At(solvedAt+2*time.Second, func() {
			goldentest.Check(t, "fifteen_solved", screen(d))
		})
		p.Tap(hal.PadA, solvedAt+2500*ms)
		clock.At(solvedAt+4*time.Second, func() {
			if inOrder(g) {
				t.Error("no new game after the button")
			}
		})
		p.Tap(hal.PadY, solvedAt+4500*ms)
	}
	clock.At(1100*ms, watch)
	g.Run()
	if solvedAt == 0 {
		t.Fatal("not solved")
	}
	t.Logf("solved after %v", solvedAt-time.Second)
}
