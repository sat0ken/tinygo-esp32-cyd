package solitaire

import (
	"testing"

	"github.com/sat0ken/tinygo-cyd/board"
	"github.com/sat0ken/tinygo-cyd/games/cards"
	"github.com/sat0ken/tinygo-cyd/internal/goldentest"
	"github.com/sat0ken/tinygo-cyd/memlcd"
)

const dt = float32(1.0 / 60)

func newTestGame() (*Game, *memlcd.Display, *memlcd.Touch) {
	d := memlcd.New(board.Width, board.Height)
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
	tapAt(g, touch, colX(col)+cardW/2, ys[i]+3)
}

func TestInitialScreen(t *testing.T) {
	_, d, _ := newTestGame()
	goldentest.Check(t, "solitaire_start", d.Image())
}

func TestStockAndUndo(t *testing.T) {
	g, _, touch := newTestGame()
	top := g.s.Stock[len(g.s.Stock)-1]
	tapRect(g, touch, stockRect())
	if len(g.s.Waste) != 1 || g.s.Waste[0] != top || g.s.Moves != 1 {
		t.Fatalf("waste %v moves %d", g.s.Waste, g.s.Moves)
	}
	tapRect(g, touch, undoButton())
	if len(g.s.Waste) != 0 || len(g.s.Stock) != 24 || g.s.Moves != 0 {
		t.Fatal("undo did not put the card back")
	}
}

func TestDrawThree(t *testing.T) {
	g, _, touch := newTestGame()
	tapRect(g, touch, drawButton())
	if g.s.Draw != 3 || g.s.Moves != 0 {
		t.Fatal("not a new draw-3 deal")
	}
	tapRect(g, touch, stockRect())
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
	tapRect(g, touch, wasteRect())
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
	if d.Pixel(foundationRect(0).x+cardW/2, foundationRect(0).y+cardH/2) == rgb(tableColor) {
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
	g, _, _ := newTestGame()
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
	if last := ys[len(ys)-1] + cardH; last > bottom {
		t.Fatalf("column ends at %d, screen is %d", last, bottom)
	}
	for i := 7; i < len(ys); i++ {
		if ys[i]-ys[i-1] < minUp {
			t.Fatalf("face-up cards %d px apart", ys[i]-ys[i-1])
		}
	}
}

func TestBackButton(t *testing.T) {
	g, _, touch := newTestGame()
	left := false
	g.OnExit = func() { left = true }
	g.Start()
	tapRect(g, touch, backButton())
	if !left {
		t.Fatal("OnExit not called")
	}
}

func rgb(c interface{ RGBA() (r, g, b, a uint32) }) uint16 {
	r, gg, b, _ := c.RGBA()
	return uint16(r>>8&0xF8)<<8 | uint16(gg>>8&0xFC)<<3 | uint16(b>>8)>>3
}
