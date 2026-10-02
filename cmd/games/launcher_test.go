package main

import (
	"image"
	"testing"

	"github.com/sat0ken/tinygo-cyd/board"
	"github.com/sat0ken/tinygo-cyd/internal/goldentest"
	"github.com/sat0ken/tinygo-cyd/memlcd"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/proggy"
)

const dt = float32(1.0 / 60)

func newTestLauncher() (*Launcher, *memlcd.Display, *memlcd.Touch) {
	d := memlcd.New(board.Width, board.Height)
	touch := &memlcd.Touch{}
	seed := uint32(3)
	l := NewLauncher(d, touch, func() uint32 { seed++; return seed })
	l.DrawMenu()
	return l, d, touch
}

// tap presses at (x, y) for a few frames and releases, then lets a few
// frames pass (some games ignore taps that come too quickly).
func tap(l *Launcher, touch *memlcd.Touch, x, y int16) {
	touch.X, touch.Y, touch.Pressed = x, y, true
	for i := 0; i < 3; i++ {
		l.Update(dt)
	}
	touch.Pressed = false
	for i := 0; i < 20; i++ {
		l.Update(dt)
	}
}

func TestMenu(t *testing.T) {
	_, d, _ := newTestLauncher()
	goldentest.Check(t, "games_menu", d.Image())
}

func TestStartEachGameAndComeBack(t *testing.T) {
	// Where the "< GAMES" button is on each game's first screen.
	backs := map[string][2]int16{
		"INVADERS": {board.Width - 40, 10},
		"SUDOKU":   {50, 22},
		"OTHELLO":  {50, 22},
	}
	// Tap each tile where the game's first screen has a button (invaders:
	// anywhere starts the game; sudoku: MEDIUM; othello: HARD), so a tap
	// leaking into the game would be noticed.
	taps := map[string][2]int16{
		"INVADERS": {100, 150},
		"SUDOKU":   {240, 200},
		"OTHELLO":  {380, 210},
	}
	for k, e := range entries {
		l, d, touch := newTestLauncher()
		r := tileRect(k)
		p := taps[e.name]
		if !r.has(p[0], p[1]) {
			t.Fatalf("%s: tap point is not on its tile", e.name)
		}
		tap(l, touch, p[0], p[1])
		if got := l.Playing(); got != e.name {
			t.Fatalf("tile %d started %q, want %q", k, got, e.name)
		}
		// The tap on the tile must not have reached the game: it is still
		// on its first screen, so the "< GAMES" button is there and works.
		// (Otherwise sudoku/othello would have started a game and invaders
		// would be playing, and the button would be gone.)
		b := backs[e.name]
		tap(l, touch, b[0], b[1])
		if l.Playing() != "" {
			t.Fatalf("%s: still playing after tapping < GAMES", e.name)
		}
		// The launcher is back on screen and still works.
		if diff := goldentest.Compare(launcherImage(), d.Image()); diff != "" {
			t.Fatalf("%s: launcher not redrawn: %s", e.name, diff)
		}
		tap(l, touch, p[0], p[1])
		if l.Playing() != e.name {
			t.Fatalf("%s: cannot start again", e.name)
		}
	}
}

// launcherImage is a freshly drawn launcher screen.
func launcherImage() *image.RGBA {
	_, d, _ := newTestLauncher()
	return d.Image()
}

func TestDescriptionsFitTheirTiles(t *testing.T) {
	for k, e := range entries {
		_, w := tinyfont.LineWidth(&proggy.TinySZ8pt7b, e.about)
		if int16(w) > tileRect(k).w-8 {
			t.Errorf("%s: description %q is %dpx, tile is %dpx", e.name, e.about, w, tileRect(k).w)
		}
	}
}
