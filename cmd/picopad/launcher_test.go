package main

import (
	"testing"
	"time"

	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/internal/goldentest"
	"github.com/sat0ken/tinygo-cyd/internal/picotest"
	"github.com/sat0ken/tinygo-cyd/memlcd"
)

const ms = time.Millisecond

// touch is a touch panel pressed at x, y from from for 100 ms.
type touch struct {
	clock *picotest.Clock
	taps  []tap
}

type tap struct {
	from time.Duration
	x, y int16
}

func (t *touch) ReadTouch() (int16, int16, bool) {
	for _, p := range t.taps {
		if t.clock.T >= p.from && t.clock.T < p.from+100*ms {
			return p.x, p.y, true
		}
	}
	return 0, 0, false
}

func newTest(t *testing.T, w, h int16) (*Launcher, *memlcd.Display, *picotest.Clock, *picotest.Pad, *touch) {
	d := memlcd.New(w, h)
	clock := picotest.NewClock(t, 30*time.Minute)
	clock.Step = 20 * time.Microsecond
	tp := &touch{clock: clock}
	l := NewLauncher(d, tp, func() uint32 { return 7 })
	l.Clock = clock
	p := &picotest.Pad{Clock: clock}
	l.Pad = p
	return l, d, clock, p, tp
}

// TestMenu: the menu, the arrows and A.
func TestMenu(t *testing.T) {
	l, d, clock, p, _ := newTest(t, 320, 240)
	if len(l.games) != 21 {
		t.Fatalf("%d games", len(l.games))
	}
	clock.At(100*ms, func() { goldentest.Check(t, "picopad_menu", d.Image()) })
	p.Tap(hal.PadUp, 200*ms) // from Tetris round to the last
	clock.At(400*ms, func() {
		if l.sel != 20 {
			t.Errorf("UP: %d", l.sel)
		}
	})
	p.Tap(hal.PadLeft, 500*ms) // 20 - 11
	p.Tap(hal.PadDown, 700*ms) // 10, the last of the first column
	p.Tap(hal.PadUp, 800*ms)
	clock.At(900*ms, func() {
		if l.sel != 9 {
			t.Errorf("LEFT, DOWN, UP: %d", l.sel)
		}
		goldentest.Check(t, "picopad_menu_invaders", d.Image())
	})
	p.Tap(hal.PadRight, time.Second) // 20, the last
	p.Tap(hal.PadA, 1200*ms)
	if i := l.menu(); i != 20 || l.games[i].name != "Futsal" {
		t.Errorf("A: %d", i)
	}
}

// TestTouch: a tap chooses a game, a tap on it plays it (the menu in the
// middle of a larger display).
func TestTouch(t *testing.T) {
	l, _, _, _, tp := newTest(t, 480, 272)
	ox, oy := int16(80), int16(16)
	tp.taps = []tap{
		{50 * ms, 100, 100},                           // still from the game: not a tap
		{300 * ms, ox + 200, oy + listY + 2*rowH + 5}, // the third of the second column
		{600 * ms, ox + 200, oy + listY + 2*rowH + 5},
	}
	if i := l.menu(); i != 13 || l.games[i].name != "Raptor" {
		t.Errorf("tap: %d", i)
	}
}

// TestScaled: on 480x320 (the ESP32-3248S035C) the menu is 4/3 bigger,
// and a tap finds the game under it.
func TestScaled(t *testing.T) {
	l, d, clock, _, tp := newTest(t, 480, 320)
	clock.At(100*ms, func() { goldentest.Check(t, "picopad_menu_scaled", d.Image()) })
	// Where pixel x, y of the menu is drawn.
	at := func(x, y int) (int16, int16) { return int16(26 + (4*x+2)/3), int16((4*y + 2) / 3) }
	x, y := at(200, listY+2*rowH+5) // the third of the second column
	tp.taps = []tap{{300 * ms, x, y}, {600 * ms, x, y}}
	if i := l.menu(); i != 13 || l.games[i].name != "Raptor" {
		t.Errorf("tap: %d", i)
	}
}

// TestPlay plays Tetris, which comes back with Y: the menu is there
// again, the same.
func TestPlay(t *testing.T) {
	l, d, clock, p, _ := newTest(t, 320, 240)
	p.Tap(hal.PadA, 200*ms) // Tetris
	i := l.menu()
	p.Tap(hal.PadA, clock.T+500*ms) // the end of the pause
	clock.At(clock.T+time.Second, func() { goldentest.Check(t, "picopad_tetris", d.Image()) })
	p.Tap(hal.PadY, clock.T+1500*ms)
	at := clock.T
	l.play(i)
	if clock.T-at > 2*time.Second {
		t.Errorf("Tetris came back after %v", clock.T-at)
	}
	clock.At(clock.T+300*ms, func() { goldentest.Check(t, "picopad_menu", d.Image()) })
	p.Tap(hal.PadA, clock.T+500*ms)
	if i := l.menu(); i != 0 {
		t.Errorf("the menu again: %d", i)
	}
}

// TestAllGames starts every game and brings it back with Y held for a
// moment every second (Pictor: Y, then X in its menu): each comes back,
// and the next one starts with the frame buffer of the last.
func TestAllGames(t *testing.T) {
	l, _, clock, _, _ := newTest(t, 320, 240)
	for i, e := range l.games {
		p := &picotest.Pad{Clock: clock}
		l.Pad = p
		at := clock.T
		for k := 1; k <= 30; k++ {
			p.Hold(hal.PadY, at+time.Duration(k)*time.Second, 400*ms)
			if e.name == "Pictor" {
				p.Tap(hal.PadX, at+time.Duration(k)*time.Second+600*ms)
			}
		}
		l.play(i)
		if d := clock.T - at; d < 500*ms || d > 25*time.Second {
			t.Errorf("%s came back after %v", e.name, d)
		}
	}
}
