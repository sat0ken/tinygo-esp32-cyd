package solitaire

import (
	"testing"

	"github.com/sat0ken/tinygo-cyd/board"
	"github.com/sat0ken/tinygo-cyd/games/cards"
	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/internal/goldentest"
	"github.com/sat0ken/tinygo-cyd/memlcd"
	"github.com/sat0ken/tinygo-cyd/pad"
	"github.com/sat0ken/tinygo-cyd/romfont/proggy"
	"tinygo.org/x/tinyfont"
)

const dt = float32(1.0 / 60)

// sizes are the screens the game has a layout for; golden is the prefix of
// the golden images.
var sizes = []struct {
	w, h   int16
	golden string
}{
	{board.Width, board.Height, "solitaire_"}, // ESP32-4827S043, ESP32-3248S035C
	{320, 240, "solitaire_small_"},            // ESP32-2432S028R
}

func newTestGame() (*Game, *memlcd.Display, *memlcd.Touch) {
	return newTestGameSize(board.Width, board.Height)
}

func newTestGameSize(w, h int16) (*Game, *memlcd.Display, *memlcd.Touch) {
	d := memlcd.New(w, h)
	touch := &memlcd.Touch{}
	seed := uint32(20)
	g := NewGame(d, touch, func() uint32 { seed++; return seed })
	g.Start()
	return g, d, touch
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

func tapRect(g *Game, touch *memlcd.Touch, r rect) { tapAt(g, touch, r.x+r.w/2, r.y+r.h/2) }

// tapCard taps the visible part of card i of column col.
func tapCard(g *Game, touch *memlcd.Touch, col, i int) {
	ys := g.offsets(col)
	tapAt(g, touch, g.l.colX(col)+g.l.cardW/2, ys[i]+3)
}

func TestInitialScreen(t *testing.T) {
	for _, s := range sizes {
		g, d, _ := newTestGameSize(s.w, s.h)
		if s.w == 320 {
			g.OnExit = func() {}
			g.Start()
		}
		goldentest.Check(t, s.golden+"start", d.Image())
	}
}

// TestLayout checks that everything is on the screen and that nothing that
// can be tapped overlaps.
func TestLayout(t *testing.T) {
	for _, s := range sizes {
		l := layoutFor(s.w, s.h)
		screen := rect{0, 0, s.w, s.h}
		bar := []rect{l.back, l.new, l.undo, l.draw, l.status}
		table := []rect{l.stockRect(), l.wasteRect(), l.foundationRect(0), l.foundationRect(3), l.columnRect(0), l.columnRect(6)}
		for _, r := range append(append([]rect{l.won}, bar...), table...) {
			if r.x < 0 || r.y < 0 || r.x+r.w > screen.w || r.y+r.h > screen.h {
				t.Errorf("%dx%d: %v is off the screen", s.w, s.h, r)
			}
		}
		for _, group := range [][]rect{bar, table} {
			for i, a := range group {
				for _, b := range group[i+1:] {
					if a.x < b.x+b.w && b.x < a.x+a.w && a.y < b.y+b.h && b.y < a.y+a.h {
						t.Errorf("%dx%d: %v overlaps %v", s.w, s.h, a, b)
					}
				}
			}
		}
		if _, w := tinyfont.LineWidth(&proggy.TinySZ8pt7b, "MOVES 999   59:59"); int16(w)+8 > l.status.w {
			t.Errorf("%dx%d: the status does not fit", s.w, s.h)
		}
	}
}

func TestStockAndUndo(t *testing.T) {
	g, _, touch := newTestGame()
	top := g.s.Stock[len(g.s.Stock)-1]
	tapRect(g, touch, g.l.stockRect())
	if len(g.s.Waste) != 1 || g.s.Waste[0] != top || g.s.Moves != 1 {
		t.Fatalf("waste %v moves %d", g.s.Waste, g.s.Moves)
	}
	tapRect(g, touch, g.l.undo)
	if len(g.s.Waste) != 0 || len(g.s.Stock) != 24 || g.s.Moves != 0 {
		t.Fatal("undo did not put the card back")
	}
}

func TestDrawThree(t *testing.T) {
	g, _, touch := newTestGame()
	tapRect(g, touch, g.l.draw)
	if g.s.Draw != 3 || g.s.Moves != 0 {
		t.Fatal("not a new draw-3 deal")
	}
	tapRect(g, touch, g.l.stockRect())
	if len(g.s.Waste) != 3 {
		t.Fatalf("waste %d", len(g.s.Waste))
	}
}

func TestTapMovesCard(t *testing.T) {
	g, d, touch := newTestGame()
	// Set up: an ace on the waste and a red 9 / black 8 pair in the columns.
	g.s.Waste = append(g.s.Waste, g.s.Stock[len(g.s.Stock)-1])
	g.s.Stock = g.s.Stock[:len(g.s.Stock)-1]
	ace := cards.New(cards.Hearts, 1)
	// Swap the ace into the waste top wherever it is.
	swapInto(&g.s, ace, &g.s.Waste[len(g.s.Waste)-1])
	g.dirty = true
	g.Update(dt)
	tapRect(g, touch, g.l.wasteRect())
	found := false
	for _, f := range g.s.Foundation {
		if len(f) == 1 && f[0] == ace {
			found = true
		}
	}
	if !found {
		t.Fatal("the ace did not go to a foundation")
	}
	// The foundation shows the ace (red ink in its corner area).
	if fr := g.l.foundationRect(0); d.Pixel(fr.x+fr.w/2, fr.y+fr.h/2) == rgb(tableColor) {
		t.Fatal("foundation not redrawn")
	}

	// Tapping a face-down card does nothing.
	before := g.s.Moves
	tapCard(g, touch, 6, 0)
	if g.s.Moves != before {
		t.Fatal("a face-down card moved")
	}
}

// swapInto exchanges card c, wherever it is on the table, with *slot.
func swapInto(s *State, c cards.Card, slot *cards.Card) {
	piles := [][]cards.Card{s.Stock, s.Waste}
	for _, f := range s.Foundation {
		piles = append(piles, f)
	}
	for _, col := range s.Tableau {
		piles = append(piles, col.Cards)
	}
	for _, p := range piles {
		for i := range p {
			if p[i] == c {
				p[i], *slot = *slot, p[i]
				return
			}
		}
	}
}

func TestFinishAndWin(t *testing.T) {
	g, _, touch := newTestGame()
	// A solved table: four alternating runs K..A, nothing else.
	var s State
	s.Draw = 1
	pairs := [4][2]cards.Suit{{cards.Spades, cards.Hearts}, {cards.Hearts, cards.Spades},
		{cards.Clubs, cards.Diamonds}, {cards.Diamonds, cards.Clubs}}
	for col, p := range pairs {
		for rank := 13; rank >= 1; rank-- {
			s.Tableau[col].Cards = append(s.Tableau[col].Cards, cards.New(p[(13-rank)%2], rank))
		}
	}
	g.s = s
	g.checkEnd()
	if g.phase != finishing {
		t.Fatalf("phase %d", g.phase)
	}
	for i := 0; i < 52*10 && g.phase != won; i++ {
		g.Update(dt)
	}
	if g.phase != won || !g.s.Won() {
		t.Fatal("not won")
	}
	tapAt(g, touch, 240, 150)
	if g.phase != playing || g.s.Won() {
		t.Fatal("no new deal after winning")
	}
}

func TestLongColumnFits(t *testing.T) {
	for _, s := range sizes {
		testLongColumnFits(t, s.w, s.h, s.golden)
	}
}

func testLongColumnFits(t *testing.T, w, h int16, golden string) {
	g, d, _ := newTestGameSize(w, h)
	// 6 face-down cards and a run of 13 on top: the longest possible column.
	c := &g.s.Tableau[6]
	c.Cards = c.Cards[:6]
	c.Down = 6
	for rank := 13; rank >= 1; rank-- {
		suit := cards.Spades
		if rank%2 == 0 {
			suit = cards.Hearts
		}
		c.Cards = append(c.Cards, cards.New(suit, rank))
	}
	ys := g.offsets(6)
	if last := ys[len(ys)-1] + g.l.cardH; last > g.l.bottom {
		t.Fatalf("%dx%d: column ends at %d, screen is %d", w, h, last, g.l.bottom)
	}
	for i := 7; i < len(ys); i++ {
		if ys[i]-ys[i-1] < minUp {
			t.Fatalf("%dx%d: face-up cards %d px apart", w, h, ys[i]-ys[i-1])
		}
	}
	if golden == "solitaire_small_" {
		g.dirty = true
		g.refresh()
		goldentest.Check(t, golden+"long", d.Image())
	}
}

func TestBackButton(t *testing.T) {
	g, _, touch := newTestGame()
	left := false
	g.OnExit = func() { left = true }
	g.Start()
	tapRect(g, touch, g.l.back)
	if !left {
		t.Fatal("OnExit not called")
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

// TestPad plays with the controller only: Y turns over the stock, the
// stick goes down to the columns and right to the fourth, A moves the ace
// on it home, X takes it back, and up from the stock reaches NEW.
func TestPad(t *testing.T) {
	for _, s := range sizes {
		g, d, _ := newTestGameSize(s.w, s.h)
		// Put an ace on top of the fourth column (swapped with a card of
		// the stock, so it is still one deck).
		col := &g.s.Tableau[3]
		top := len(col.Cards) - 1
		for i, c := range g.s.Stock {
			if c.Rank() == 1 {
				g.s.Stock[i], col.Cards[top] = col.Cards[top], c
				break
			}
		}
		ace := col.Cards[top]
		g.dirty = true
		f := &pad.Fake{}
		g.Pad = f
		g.Update(dt)

		press(g, f, hal.PadY)
		if len(g.s.Waste) != 1 {
			t.Fatalf("%dx%d: Y did not turn over the stock", s.w, s.h)
		}
		press(g, f, hal.PadDown)
		if g.focus != (target{kind: columnTarget, pile: 0, index: 0}) {
			t.Fatalf("%dx%d: down from the stock: %+v", s.w, s.h, g.focus)
		}
		r := g.focusRect(g.focus)
		if d.Pixel(r.x, r.y) != rgb(focusColor) {
			t.Fatalf("%dx%d: no frame on the first column", s.w, s.h)
		}
		for i := 0; i < 3; i++ {
			press(g, f, hal.PadRight)
		}
		if g.focus.kind != columnTarget || g.focus.pile != 3 || g.focus.index != top {
			t.Fatalf("%dx%d: focus %+v, want the top of the fourth column", s.w, s.h, g.focus)
		}
		press(g, f, hal.PadA)
		home := 0
		for _, p := range g.s.Foundation {
			home += len(p)
		}
		if home != 1 || len(col.Cards) != top {
			t.Fatalf("%dx%d: the ace did not go home", s.w, s.h)
		}
		if g.focus.index != top-1 {
			t.Fatalf("%dx%d: the focus did not stay on the column: %+v", s.w, s.h, g.focus)
		}
		press(g, f, hal.PadX)
		if len(col.Cards) != top+1 || col.Cards[top] != ace {
			t.Fatalf("%dx%d: X did not take the ace back", s.w, s.h)
		}

		for n := 0; g.focus.kind != buttonTarget; n++ {
			if n > 4 {
				t.Fatalf("%dx%d: up does not reach the bar", s.w, s.h)
			}
			press(g, f, hal.PadUp)
		}
		for g.focus.pile != 1 {
			press(g, f, hal.PadLeft)
		}
		press(g, f, hal.PadA) // NEW
		if g.s.Moves != 0 || len(g.history) != 0 {
			t.Fatalf("%dx%d: NEW did not deal again", s.w, s.h)
		}
	}
}
