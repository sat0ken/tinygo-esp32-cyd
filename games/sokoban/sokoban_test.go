package sokoban

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
	clock.Step = 100 * time.Microsecond
	g.Clock = clock
	p := &picotest.Pad{Clock: clock}
	g.Pad = p
	return g, d, clock, p
}

// TestSolutions plays the solutions of all the levels as the AUTOMODE of
// the C source does: every level must be solved.
func TestSolutions(t *testing.T) {
	g, _, _, _ := newTest(t, 1)
	g.Clock = picotest.NewClock(t, 1000*time.Hour)
	g.fastMode = true
	n := 0
	for a := range authors {
		g.selectAuthor(a)
		for c := 0; c < g.collNum; c++ {
			g.selectColl(c)
			for l := 0; l < g.levNum; l++ {
				g.level = l
				g.playSolve()
				if g.markNum != 0 {
					t.Errorf("%s, %s, level %d: %d marks left", g.autName, g.collName, l+1, g.markNum)
				}
				n++
			}
		}
	}
	if n != 2020 {
		t.Errorf("%d levels", n)
	}
}

func TestScreens(t *testing.T) {
	g, d, clock, p := newTest(t, 1)
	clock.At(150*ms, func() { goldentest.Check(t, "sokoban_intro", d.Image()) })
	p.Tap(hal.PadB, time.Second) // a key ends the demo
	clock.At(2*time.Second, func() {
		goldentest.Check(t, "sokoban_authors", d.Image())
		if g.levelTotal != 2020 || g.collTotal != 37 {
			t.Errorf("totals: %d collections, %d levels", g.collTotal, g.levelTotal)
		}
	})
	p.Tap(hal.PadDown, 2200*ms)
	p.Tap(hal.PadDown, 2400*ms) // David W. Skinner
	p.Tap(hal.PadA, 2600*ms)
	clock.At(3*time.Second, func() {
		goldentest.Check(t, "sokoban_collections", d.Image())
		if g.autName != "David W. Skinner" || g.collNum != 16 {
			t.Errorf("author %q, %d collections", g.autName, g.collNum)
		}
	})
	at := 3200 * ms
	for k := 0; k < 4; k++ { // up from the first: Microban IV, III, II, Microban
		p.Tap(hal.PadUp, at)
		at += 200 * ms
	}
	p.Tap(hal.PadA, at)
	clock.At(at+400*ms, func() {
		goldentest.Check(t, "sokoban_levels", d.Image())
		if g.collName != "Microban" || g.levNum != 155 {
			t.Errorf("collection %q, %d levels", g.collName, g.levNum)
		}
	})
	p.Tap(hal.PadA, at+600*ms)
	clock.At(at+time.Second, func() {
		goldentest.Check(t, "sokoban_game", d.Image())
		if g.tileSize != 32 || g.markNum != 1 || g.posX != 2 || g.posY != 3 {
			t.Errorf("level 1: tiles %d, %d marks, player at %d,%d", g.tileSize, g.markNum, g.posX, g.posY)
		}
	})
	at += 1200 * ms
	for _, b := range []hal.Buttons{hal.PadY, hal.PadY, hal.PadY, hal.PadY} { // the levels, collections, authors, the demo
		p.Tap(b, at)
		at += 300 * ms
	}
	p.Hold(hal.PadY, at+time.Second, 2*time.Second) // held: the demo reads the keys after its steps
	g.Run()
}

// play goes from the demo to level 1 of Microban (David W. Skinner) and
// returns the time after.
func play(p *picotest.Pad) time.Duration {
	p.Tap(hal.PadB, 500*ms)
	at := time.Second
	for _, b := range []hal.Buttons{hal.PadDown, hal.PadDown, hal.PadA, hal.PadUp, hal.PadUp, hal.PadUp, hal.PadUp, hal.PadA, hal.PadA} {
		p.Tap(b, at)
		at += 200 * ms
	}
	return at + 300*ms
}

// quit goes back from a level to the demo and ends Run.
func quit(p *picotest.Pad, at time.Duration) {
	for k := 0; k < 4; k++ {
		p.Tap(hal.PadY, at)
		at += 300 * ms
	}
	p.Hold(hal.PadY, at+time.Second, 2*time.Second)
}

// TestPlay solves Microban 1 with the buttons (a tap a step in the slow
// mode), then level 2 starts.
func TestPlay(t *testing.T) {
	g, _, clock, p := newTest(t, 1)
	at := play(p)
	_, sol := level(skinnerCol13, 0)
	dirs := map[byte]hal.Buttons{'u': hal.PadUp, 'd': hal.PadDown, 'l': hal.PadLeft, 'r': hal.PadRight}
	for i := 0; i < len(sol); i++ {
		p.Tap(dirs[sol[i]|0x20], at)
		if i == 5 {
			clock.At(at+350*ms, func() {
				// d l u r r r: the box on the mark at 1,3 pushed up to 1,2
				if g.board[1+2*g.mapW] != tileCrate || g.board[1+3*g.mapW] != tileMark || g.markNum != 2 || g.posX != 4 || g.posY != 3 {
					t.Errorf("after 6 steps: player at %d,%d, %d marks", g.posX, g.posY, g.markNum)
				}
			})
		}
		at += 400 * ms
	}
	clock.At(at+3*time.Second, func() {
		if g.level != 1 || g.markNum == 0 || g.collName != "Microban" {
			t.Errorf("after level 1: level %d, %d marks", g.level+1, g.markNum)
		}
	})
	quit(p, at+3500*ms)
	g.Run()
}

// TestKeys: B starts the level again, A+RIGHT goes to the next level, X
// shows the solution.
func TestKeys(t *testing.T) {
	g, _, clock, p := newTest(t, 1)
	at := play(p)
	p.Tap(hal.PadDown, at)
	clock.At(at+500*ms, func() {
		if g.posY != 4 {
			t.Errorf("DOWN: player at %d,%d", g.posX, g.posY)
		}
	})
	p.Tap(hal.PadB, at+600*ms)
	clock.At(at+900*ms, func() {
		if g.posX != 2 || g.posY != 3 {
			t.Errorf("B: player at %d,%d", g.posX, g.posY)
		}
	})
	p.Hold(hal.PadA, at+time.Second, 400*ms)
	p.Tap(hal.PadRight, at+1200*ms)
	clock.At(at+2500*ms, func() {
		if g.level != 1 {
			t.Errorf("A+RIGHT: level %d", g.level+1)
		}
	})
	p.Tap(hal.PadX, at+3*time.Second) // the solution of level 2
	solved := false
	for s := 4; s < 60; s++ {
		clock.At(at+time.Duration(s)*time.Second, func() {
			if g.markNum == 0 {
				solved = true
			}
		})
	}
	clock.At(at+60*time.Second, func() {
		if !solved || g.level != 1 || g.markNum == 0 {
			t.Errorf("X: solved %v, level %d, %d marks", solved, g.level+1, g.markNum)
		}
	})
	quit(p, at+61*time.Second)
	g.Run()
}

// TestOneGame: with OneGame, Y from a level back to the authors and once
// more returns from Run (without the demo).
func TestOneGame(t *testing.T) {
	g, _, clock, p := newTest(t, 1)
	g.OneGame = true
	at := play(p)
	clock.At(at, func() {
		if g.collName != "Microban" || g.level != 0 {
			t.Errorf("not in Microban 1: %q %d", g.collName, g.level)
		}
	})
	for k := 0; k < 4; k++ {
		p.Tap(hal.PadY, at+100*ms)
		at += 300 * ms
	}
	g.Run()
	if clock.T > at+200*ms {
		t.Errorf("Run returned at %v, the last Y at %v", clock.T, at-200*ms)
	}
}
