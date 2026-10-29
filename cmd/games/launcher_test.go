package main

import (
	"image"
	"testing"

	"github.com/sat0ken/tinygo-cyd/board"
	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/internal/goldentest"
	"github.com/sat0ken/tinygo-cyd/memlcd"
	"github.com/sat0ken/tinygo-cyd/pad"
	"tinygo.org/x/tinyfont"
)

const dt = float32(1.0 / 60)

// sizes are the screens the launcher has a layout for.
var sizes = []struct {
	w, h   int16
	golden string
}{
	{board.Width, board.Height, "games_"}, // ESP32-4827S043, ESP32-3248S035C
	{320, 240, "games_small_"},            // ESP32-2432S028R
}

func newTestLauncher(w, h int16) (*Launcher, *memlcd.Display, *memlcd.Touch) {
	d := memlcd.New(w, h)
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
	for _, s := range sizes {
		_, d, _ := newTestLauncher(s.w, s.h)
		goldentest.Check(t, s.golden+"menu", d.Image())
	}
}

func TestStartEachGameAndComeBack(t *testing.T) {
	// Where the "< GAMES" button is on each game's first screen.
	backs := map[string][2]int16{
		"INVADERS":  {board.Width - 40, 10},
		"SUDOKU":    {50, 22},
		"OTHELLO":   {50, 22},
		"SOLITAIRE": {37, 11},
		"POKER":     {33, 253},
	}
	// Tap each tile, where possible where the game's first screen has a
	// button or a card (invaders: anywhere starts the game; othello: WHITE;
	// solitaire: a column; poker: MAX BET), so a tap leaking into the game
	// would be noticed. (No sudoku button is under its tile.)
	taps := map[string][2]int16{
		"INVADERS":  {60, 80},
		"SUDOKU":    {240, 80},
		"OTHELLO":   {350, 120},
		"SOLITAIRE": {60, 150},
		"POKER":     {200, 238},
	}
	testStartEachGameAndComeBack(t, board.Width, board.Height, backs, taps)
}

func TestStartEachGameAndComeBackSmall(t *testing.T) {
	backs := map[string][2]int16{
		"INVADERS":  {320 - 40, 9},
		"SUDOKU":    {50, 22},
		"OTHELLO":   {50, 22},
		"SOLITAIRE": {27, 11},
		"POKER":     {27, 217},
	}
	taps := map[string][2]int16{
		"INVADERS":  {60, 80},
		"SUDOKU":    {160, 80},
		"OTHELLO":   {250, 115},
		"SOLITAIRE": {25, 150},
		"POKER":     {138, 215},
	}
	testStartEachGameAndComeBack(t, 320, 240, backs, taps)
}

func testStartEachGameAndComeBack(t *testing.T, w, h int16, backs, taps map[string][2]int16) {
	for k, e := range entries {
		l, d, touch := newTestLauncher(w, h)
		r := l.l.tileRect(k)
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
		if diff := goldentest.Compare(launcherImage(w, h), d.Image()); diff != "" {
			t.Fatalf("%s: launcher not redrawn: %s", e.name, diff)
		}
		tap(l, touch, p[0], p[1])
		if l.Playing() != e.name {
			t.Fatalf("%s: cannot start again", e.name)
		}
	}
}

// launcherImage is a freshly drawn launcher screen.
func launcherImage(w, h int16) *image.RGBA {
	_, d, _ := newTestLauncher(w, h)
	return d.Image()
}

func TestTilesFit(t *testing.T) {
	for _, s := range sizes {
		l := layoutFor(s.w, s.h)
		for k, e := range entries {
			r := l.tileRect(k)
			for _, x := range []struct {
				font tinyfont.Fonter
				text string
			}{{nameFont, e.name}, {aboutFont, e.about}} {
				if _, w := tinyfont.LineWidth(x.font, x.text); int16(w) > r.w-4 {
					t.Errorf("%dx%d: %s: %q is %dpx, tile is %dpx", s.w, s.h, e.name, x.text, w, r.w)
				}
			}
		}
		// Five tiles, but the grid has room for six.
		if last := l.tileRect(5); last.x+last.w > s.w || last.y+last.h > l.footBase-8 {
			t.Errorf("%dx%d: tiles end at %d,%d", s.w, s.h, last.x+last.w, last.y+last.h)
		}
	}
}

// press presses b on the controller for a few frames and lets go.
func press(l *Launcher, f *pad.Fake, b hal.Buttons) {
	f.B = b
	for i := 0; i < 3; i++ {
		l.Update(dt)
	}
	f.B = 0
	for i := 0; i < 3; i++ {
		l.Update(dt)
	}
}

// TestPad starts every game with the controller and comes back with B (or
// with A on "< GAMES" in the games that have it in their bar). A held down
// from the launcher must not press anything in the game.
func TestPad(t *testing.T) {
	for _, s := range sizes {
		l, d, _ := newTestLauncher(s.w, s.h)
		f := &pad.Fake{}
		l.Pad = f
		l.Update(dt)
		press(l, f, hal.PadLeft) // nothing to the left: only shows the focus
		for k, e := range entries {
			// Walk to tile k: the first row, then the second.
			for n := 0; l.focus != k; n++ {
				if n > 6 {
					t.Fatalf("%dx%d: tile %d not reached (focus %d)", s.w, s.h, k, l.focus)
				}
				switch {
				case k/3 > l.focus/3:
					press(l, f, hal.PadDown)
				case k/3 < l.focus/3:
					press(l, f, hal.PadUp)
				case k%3 > l.focus%3:
					press(l, f, hal.PadRight)
				default:
					press(l, f, hal.PadLeft)
				}
			}
			r := l.l.tileRect(k)
			if d.Pixel(r.x, r.y) != rgb(focusColor) {
				t.Fatalf("%dx%d: no frame on tile %d", s.w, s.h, k)
			}
			press(l, f, hal.PadA)
			if l.Playing() != e.name {
				t.Fatalf("%dx%d: tile %d started %q", s.w, s.h, k, l.Playing())
			}
			switch e.name {
			case "SOLITAIRE", "POKER":
				// "< GAMES" is in the bar: walk there and press A.
				for n := 0; l.Playing() != "" && n < 12; n++ {
					if n%6 == 5 {
						press(l, f, hal.PadA)
						continue
					}
					if e.name == "SOLITAIRE" {
						press(l, f, []hal.Buttons{hal.PadUp, hal.PadUp, hal.PadLeft, hal.PadLeft, hal.PadLeft}[n%6])
					} else {
						press(l, f, hal.PadLeft)
					}
				}
			default:
				press(l, f, hal.PadB)
			}
			if l.Playing() != "" {
				t.Fatalf("%dx%d: %s: not back on the launcher", s.w, s.h, e.name)
			}
		}
	}
}

func rgb(c interface{ RGBA() (r, g, b, a uint32) }) uint16 {
	r, gg, b, _ := c.RGBA()
	return uint16(r>>8&0xF8)<<8 | uint16(gg>>8&0xFC)<<3 | uint16(b>>8)>>3
}
