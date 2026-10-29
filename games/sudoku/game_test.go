package sudoku

import (
	"testing"

	"github.com/sat0ken/tinygo-cyd/board"
	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/internal/goldentest"
	"github.com/sat0ken/tinygo-cyd/memlcd"
	"github.com/sat0ken/tinygo-cyd/pad"
)

const dt = float32(0.05)

// sizes are the screens the game has a layout for; golden is the prefix of
// the golden images.
var sizes = []struct {
	w, h   int16
	golden string
}{
	{board.Width, board.Height, "sudoku_"}, // ESP32-4827S043, ESP32-3248S035C
	{320, 240, "sudoku_small_"},            // ESP32-2432S028R
}

func newTestGame() (*Game, *memlcd.Display, *memlcd.Touch) {
	return newTestGameSize(board.Width, board.Height)
}

func newTestGameSize(w, h int16) (*Game, *memlcd.Display, *memlcd.Touch) {
	d := memlcd.New(w, h)
	touch := &memlcd.Touch{}
	seed := uint32(41)
	g := NewGame(d, touch, func() uint32 { seed++; return seed })
	g.DrawMenu()
	return g, d, touch
}

func tapRect(g *Game, touch *memlcd.Touch, r rect) {
	tapAt(g, touch, r.x+r.w/2, r.y+r.h/2)
}

func tapAt(g *Game, touch *memlcd.Touch, x, y int16) {
	touch.Pressed = false
	g.Update(dt)
	g.Update(0.2) // more than tapGap since the last tap
	touch.X, touch.Y, touch.Pressed = x, y, true
	g.Update(dt)
	touch.Pressed = false
	g.Update(dt)
}

// emptyCell returns the first cell that is not a clue.
func emptyCell(g *Game) int {
	for i, v := range g.puzzle {
		if v == 0 {
			return i
		}
	}
	return -1
}

func TestMenuScreen(t *testing.T) {
	for _, s := range sizes {
		_, d, _ := newTestGameSize(s.w, s.h)
		goldentest.Check(t, s.golden+"menu", d.Image())
	}
}

func TestPlayScreen(t *testing.T) {
	for _, s := range sizes {
		g, d, touch := newTestGameSize(s.w, s.h)
		tapRect(g, touch, g.l.menuButton(1)) // MEDIUM
		if g.screen != playScreen {
			t.Fatal("not playing")
		}
		tapRect(g, touch, g.l.cellRect(40)) // centre cell
		g.elapsed = 0                       // make the clock repeatable
		g.refresh()
		goldentest.Check(t, s.golden+"play", d.Image())
	}
}

// TestLayout checks that everything is on the screen and that nothing that
// can be tapped overlaps.
func TestLayout(t *testing.T) {
	for _, s := range sizes {
		l := layoutFor(s.w, s.h)
		screen := rect{0, 0, s.w, s.h}
		taps := []rect{l.gridRect(), l.buttons[0], l.buttons[1], l.buttons[2]}
		for k := 0; k < 9; k++ {
			taps = append(taps, l.padRect(k))
		}
		all := append([]rect{l.head, l.tail, l.solved, backButton()}, taps...)
		for k := range difficulties {
			all = append(all, l.menuButton(k))
		}
		for _, r := range all {
			if !inside(r, screen) {
				t.Errorf("%dx%d: %v is off the screen", s.w, s.h, r)
			}
		}
		for i, a := range append(taps, l.head, l.tail) {
			for _, b := range append(taps, l.head, l.tail)[i+1:] {
				if overlap(a, b) {
					t.Errorf("%dx%d: %v overlaps %v", s.w, s.h, a, b)
				}
			}
		}
	}
}

func inside(r, outer rect) bool {
	return r.x >= outer.x && r.y >= outer.y && r.x+r.w <= outer.x+outer.w && r.y+r.h <= outer.y+outer.h
}

func overlap(a, b rect) bool {
	return a.x < b.x+b.w && b.x < a.x+a.w && a.y < b.y+b.h && b.y < a.y+a.h
}

func TestEnterEraseHint(t *testing.T) {
	for _, s := range sizes {
		testEnterEraseHint(t, s.w, s.h)
	}
}

func testEnterEraseHint(t *testing.T, w, h int16) {
	g, d, touch := newTestGameSize(w, h)
	tapRect(g, touch, g.l.menuButton(0)) // EASY
	i := emptyCell(g)
	tapRect(g, touch, g.l.cellRect(i))
	if g.sel != i {
		t.Fatalf("selected %d, want %d", g.sel, i)
	}

	// A wrong digit: red, counted as a miss.
	wrong := g.solution[i]%9 + 1
	tapRect(g, touch, g.l.padRect(int(wrong-1)))
	if g.board[i] != wrong || g.mistakes != 1 {
		t.Fatalf("board %d mistakes %d", g.board[i], g.mistakes)
	}
	if g.drawn[i].fg != errorColor {
		t.Fatal("wrong digit not drawn in red")
	}

	// Erase.
	tapRect(g, touch, g.l.buttons[0])
	if g.board[i] != 0 {
		t.Fatal("not erased")
	}

	// The right digit.
	tapRect(g, touch, g.l.padRect(int(g.solution[i]-1)))
	if g.board[i] != g.solution[i] || g.drawn[i].fg != playerColor {
		t.Fatal("right digit not entered")
	}

	// A clue cannot be changed.
	clue := 0
	for g.puzzle[clue] == 0 {
		clue++
	}
	tapRect(g, touch, g.l.cellRect(clue))
	tapRect(g, touch, g.l.padRect(int(g.solution[clue]%9)))
	if g.board[clue] != g.puzzle[clue] {
		t.Fatal("a clue was changed")
	}

	// Hint fills the selected empty cell.
	j := -1
	for k, v := range g.board {
		if v == 0 {
			j = k
			break
		}
	}
	tapRect(g, touch, g.l.cellRect(j))
	tapRect(g, touch, g.l.buttons[1])
	if g.board[j] != g.solution[j] || !g.hinted[j] || g.drawn[j].fg != hintColor {
		t.Fatal("hint did not fill the selected cell")
	}

	// The cell drawn on the panel matches: the selected cell has selColor.
	r := g.l.cellRect(j)
	if d.Pixel(r.x+1, r.y+1) != rgb(selColor) {
		t.Fatalf("selected cell background %04x", d.Pixel(r.x+1, r.y+1))
	}
}

func TestSolveAndBackToMenu(t *testing.T) {
	for _, s := range sizes {
		g, d, touch := newTestGameSize(s.w, s.h)
		tapRect(g, touch, g.l.menuButton(2)) // HARD
		for i, v := range g.puzzle {
			if v != 0 {
				continue
			}
			tapRect(g, touch, g.l.cellRect(i))
			tapRect(g, touch, g.l.padRect(int(g.solution[i]-1)))
		}
		if g.screen != solvedScreen || g.mistakes != 0 {
			t.Fatalf("screen %d mistakes %d", g.screen, g.mistakes)
		}
		if s.w == 320 {
			g.elapsed = 0
			g.drawSolved()
			goldentest.Check(t, s.golden+"solved", d.Image())
		}
		tapAt(g, touch, 100, 100)
		if g.screen != menuScreen {
			t.Fatal("not back to the menu")
		}
	}
}

func TestTapGapIgnoresBounce(t *testing.T) {
	g, _, touch := newTestGame()
	tapRect(g, touch, g.l.menuButton(0))
	i := emptyCell(g)
	r := g.l.cellRect(i)
	// Two presses 50ms apart: the second is a bounce and must not deselect.
	touch.X, touch.Y, touch.Pressed = r.x+5, r.y+5, true
	g.Update(dt)
	touch.Pressed = false
	g.Update(dt)
	touch.Pressed = true
	g.Update(dt)
	if g.sel != i {
		t.Fatalf("bounce toggled the selection: sel %d", g.sel)
	}
}

func rgb(c interface{ RGBA() (r, g, b, a uint32) }) uint16 {
	r, gg, b, _ := c.RGBA()
	return uint16(r>>8&0xF8)<<8 | uint16(gg>>8&0xFC)<<3 | uint16(b>>8)>>3
}

// press presses b on the controller for one frame and lets go.
func press(g *Game, f *pad.Fake, b hal.Buttons) {
	f.B = b
	g.Update(dt)
	f.B = 0
	g.Update(dt)
}

// TestPad plays with the controller only: choose MEDIUM on the menu, move
// to an empty cell, enter the right digit through the number pad, erase
// it with X, get a hint with Y, and go back to the menu with NEW.
func TestPad(t *testing.T) {
	for _, s := range sizes {
		g, d, _ := newTestGameSize(s.w, s.h)
		f := &pad.Fake{}
		g.Pad = f
		g.Update(dt)
		press(g, f, hal.PadRight)
		r := g.l.menuButton(1)
		if g.focus != 1 || d.Pixel(r.x, r.y) != rgb(focusColor) {
			t.Fatalf("%dx%d: focus %d, MEDIUM has no frame", s.w, s.h, g.focus)
		}
		press(g, f, hal.PadA)
		if g.screen != playScreen || g.diff != 1 {
			t.Fatalf("%dx%d: screen %d diff %d", s.w, s.h, g.screen, g.diff)
		}

		press(g, f, hal.PadDown) // the first move selects the middle cell
		if g.sel != 40 {
			t.Fatalf("%dx%d: selected %d", s.w, s.h, g.sel)
		}
		i := emptyCell(g)
		for g.sel/9 > i/9 {
			press(g, f, hal.PadUp)
		}
		for g.sel/9 < i/9 {
			press(g, f, hal.PadDown)
		}
		for g.sel%9 > i%9 {
			press(g, f, hal.PadLeft)
		}
		for g.sel%9 < i%9 {
			press(g, f, hal.PadRight)
		}
		if g.sel != i {
			t.Fatalf("%dx%d: selected %d, want %d", s.w, s.h, g.sel, i)
		}

		press(g, f, hal.PadA) // to the number pad, on 5
		if g.focus != focusKeys+4 || d.Pixel(g.l.padRect(4).x, g.l.padRect(4).y) != rgb(focusColor) {
			t.Fatalf("%dx%d: focus %d after A on a cell", s.w, s.h, g.focus)
		}
		want := int(g.solution[i]) - 1
		for k := 4; k != want; {
			switch {
			case want/3 < k/3:
				press(g, f, hal.PadUp)
			case want/3 > k/3:
				press(g, f, hal.PadDown)
			case want%3 < k%3:
				press(g, f, hal.PadLeft)
			default:
				press(g, f, hal.PadRight)
			}
			if g.focus < focusKeys || g.focus >= focusButtons {
				t.Fatalf("%dx%d: the focus left the number pad: %d", s.w, s.h, g.focus)
			}
			k = g.focus - focusKeys
		}
		press(g, f, hal.PadA)
		if g.board[i] != g.solution[i] || g.mistakes != 0 || g.focus != i {
			t.Fatalf("%dx%d: board %d mistakes %d focus %d", s.w, s.h, g.board[i], g.mistakes, g.focus)
		}

		press(g, f, hal.PadX)
		if g.board[i] != 0 {
			t.Fatalf("%dx%d: X did not erase", s.w, s.h)
		}
		press(g, f, hal.PadY)
		if g.board[i] != g.solution[i] || !g.hinted[i] {
			t.Fatalf("%dx%d: Y did not give a hint", s.w, s.h)
		}

		// Right from the last column reaches the number pad, then the
		// buttons below it; NEW goes back to the menu.
		for n := 0; g.focus < focusKeys; n++ {
			if n > 9 {
				t.Fatalf("%dx%d: right does not reach the number pad", s.w, s.h)
			}
			press(g, f, hal.PadRight)
		}
		for n := 0; g.focus < focusButtons; n++ {
			if n > 4 {
				t.Fatalf("%dx%d: down does not reach the buttons", s.w, s.h)
			}
			press(g, f, hal.PadDown)
		}
		for n := 0; g.focus != focusButtons+2; n++ {
			if n > 4 {
				t.Fatalf("%dx%d: NEW not reached (focus %d)", s.w, s.h, g.focus)
			}
			if g.l.buttons[2].y > g.boxes[g.focus].Y { // 320x240: NEW is below
				press(g, f, hal.PadDown)
			} else {
				press(g, f, hal.PadRight)
			}
		}
		press(g, f, hal.PadA)
		if g.screen != menuScreen {
			t.Fatalf("%dx%d: NEW did not go back to the menu", s.w, s.h)
		}
	}
}

func TestPadBackToGames(t *testing.T) {
	g, _, _ := newTestGame()
	left := false
	g.OnExit = func() { left = true }
	f := &pad.Fake{}
	g.Pad = f
	g.DrawMenu()
	g.Update(dt)
	press(g, f, hal.PadB)
	if !left {
		t.Fatal("B on the menu did not call OnExit")
	}
}
