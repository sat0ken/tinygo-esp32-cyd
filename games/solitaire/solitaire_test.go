package solitaire

import (
	"testing"

	"github.com/sat0ken/tinygo-cyd/games/cards"
)

// check verifies the invariants of a table: each card exactly once,
// foundations built up by suit, face-down counts in range and face-up runs
// alternating in colour and descending.
func check(t *testing.T, s *State) {
	t.Helper()
	for c, n := range s.count() {
		if n != 1 {
			t.Fatalf("%v is on the table %d times", cards.Card(c), n)
		}
	}
	for f, p := range s.Foundation {
		for i, c := range p {
			if c.Rank() != i+1 || c.Suit() != p[0].Suit() {
				t.Fatalf("foundation %d: %v at %d", f, c, i)
			}
		}
	}
	for col, c := range s.Tableau {
		if c.Down < 0 || (len(c.Cards) > 0 && c.Down >= len(c.Cards)) || (len(c.Cards) == 0 && c.Down != 0) {
			t.Fatalf("column %d: %d cards, %d down", col, len(c.Cards), c.Down)
		}
		for i := c.Down + 1; i < len(c.Cards); i++ {
			a, b := c.Cards[i-1], c.Cards[i]
			if a.Red() == b.Red() || b.Rank() != a.Rank()-1 {
				t.Fatalf("column %d: %v on %v", col, b, a)
			}
		}
	}
}

func TestDeal(t *testing.T) {
	s := Deal(9, 1)
	check(t, &s)
	for col, c := range s.Tableau {
		if len(c.Cards) != col+1 || c.Down != col {
			t.Fatalf("column %d: %d cards, %d down", col, len(c.Cards), c.Down)
		}
	}
	if len(s.Stock) != 24 || len(s.Waste) != 0 {
		t.Fatalf("stock %d waste %d", len(s.Stock), len(s.Waste))
	}
}

func TestRules(t *testing.T) {
	var s State
	if !s.CanFound(cards.New(cards.Hearts, 1), 0) || s.CanFound(cards.New(cards.Hearts, 2), 0) {
		t.Fatal("only an ace starts a foundation")
	}
	s.Foundation[0] = []cards.Card{cards.New(cards.Hearts, 1)}
	if !s.CanFound(cards.New(cards.Hearts, 2), 0) || s.CanFound(cards.New(cards.Spades, 2), 0) {
		t.Fatal("foundations go up by suit")
	}
	if !s.CanStack(cards.New(cards.Spades, 13), 0) || s.CanStack(cards.New(cards.Spades, 12), 0) {
		t.Fatal("only a king goes on an empty column")
	}
	s.Tableau[0].Cards = []cards.Card{cards.New(cards.Hearts, 9)}
	if !s.CanStack(cards.New(cards.Clubs, 8), 0) || s.CanStack(cards.New(cards.Diamonds, 8), 0) ||
		s.CanStack(cards.New(cards.Clubs, 7), 0) {
		t.Fatal("columns go down in alternating colours")
	}
}

func TestDrawAndRecycle(t *testing.T) {
	for _, draw := range []int{1, 3} {
		s := Deal(4, draw)
		orig := append([]cards.Card(nil), s.Stock...)
		for len(s.Stock) > 0 {
			before := len(s.Stock)
			s.DrawStock()
			want := draw
			if before < draw {
				want = before
			}
			if before-len(s.Stock) != want {
				t.Fatalf("draw %d: took %d", draw, before-len(s.Stock))
			}
		}
		if len(s.Waste) != 24 {
			t.Fatalf("waste %d", len(s.Waste))
		}
		s.DrawStock() // turn the waste over
		if len(s.Waste) != 0 || len(s.Stock) != 24 {
			t.Fatal("waste not turned over")
		}
		for i := range orig {
			if s.Stock[i] != orig[i] {
				t.Fatalf("draw %d: stock order changed at %d", draw, i)
			}
		}
		check(t, &s)
	}
}

func TestMoveTurnsOverAndUndoEncoding(t *testing.T) {
	var s State
	s.Draw = 1
	s.Tableau[0] = Column{Cards: []cards.Card{cards.New(cards.Spades, 5), cards.New(cards.Hearts, 9)}, Down: 1}
	s.Tableau[1] = Column{Cards: []cards.Card{cards.New(cards.Clubs, 10)}}
	b := s.encode()
	p := Pos{Kind: TableauPile, Pile: 0, Index: 1}
	d, ok := s.AutoDest(p)
	if !ok || d != (Dest{TableauPile, 1}) {
		t.Fatalf("dest %+v %v", d, ok)
	}
	s.Move(p, d)
	if s.Tableau[0].Down != 0 || len(s.Tableau[1].Cards) != 2 || s.Moves != 1 {
		t.Fatalf("after move: %+v", s.Tableau)
	}
	var r State
	r.decode(b)
	if r.Tableau[0].Down != 1 || len(r.Tableau[0].Cards) != 2 || r.Moves != 0 || len(r.Tableau[1].Cards) != 1 {
		t.Fatalf("decoded: %+v", r.Tableau)
	}
}

func TestAutoDestPrefersFoundationAndAvoidsUselessKingMoves(t *testing.T) {
	var s State
	s.Waste = []cards.Card{cards.New(cards.Diamonds, 1)}
	if d, ok := s.AutoDest(Pos{Kind: WastePile}); !ok || d.Kind != FoundationPile {
		t.Fatal("an ace goes to a foundation")
	}
	s.Tableau[2] = Column{Cards: []cards.Card{cards.New(cards.Spades, 13)}}
	if _, ok := s.AutoDest(Pos{Kind: TableauPile, Pile: 2}); ok {
		t.Fatal("a king at the bottom moved to another empty column")
	}
	s.Tableau[3] = Column{Cards: []cards.Card{cards.New(cards.Hearts, 4), cards.New(cards.Clubs, 13)}, Down: 1}
	if d, ok := s.AutoDest(Pos{Kind: TableauPile, Pile: 3, Index: 1}); !ok || d != (Dest{TableauPile, 0}) {
		t.Fatalf("a king on a face-down card goes to the empty column: %+v %v", d, ok)
	}
}

// TestRandomPlay plays many random moves on many deals and checks the
// invariants after every one.
func TestRandomPlay(t *testing.T) {
	for seed := uint32(1); seed <= 30; seed++ {
		s := Deal(seed, 1+2*int(seed%2))
		r := cards.NewRand(seed * 7)
		for step := 0; step < 400 && !s.Won(); step++ {
			moved := false
			// Try a random position; fall back to drawing.
			for try := 0; try < 20 && !moved; try++ {
				var p Pos
				switch k := r.Intn(3); k {
				case 0:
					p = Pos{Kind: WastePile}
				case 1:
					p = Pos{Kind: FoundationPile, Pile: r.Intn(4)}
				default:
					col := r.Intn(7)
					n := len(s.Tableau[col].Cards)
					if n == 0 {
						continue
					}
					p = Pos{Kind: TableauPile, Pile: col, Index: s.Tableau[col].Down + r.Intn(n-s.Tableau[col].Down)}
				}
				if d, ok := s.AutoDest(p); ok {
					s.Move(p, d)
					moved = true
				}
			}
			if !moved {
				s.DrawStock()
			}
			check(t, &s)
		}
	}
}

func TestFinishSolvedGame(t *testing.T) {
	// Every card face up in four columns, kings at the bottom, in order.
	var s State
	// Runs K..A alternating colour: column 0 = KS QH JS 10H ..., etc.
	pairs := [4][2]cards.Suit{{cards.Spades, cards.Hearts}, {cards.Hearts, cards.Spades},
		{cards.Clubs, cards.Diamonds}, {cards.Diamonds, cards.Clubs}}
	for col, p := range pairs {
		for rank := 13; rank >= 1; rank-- {
			s.Tableau[col].Cards = append(s.Tableau[col].Cards, cards.New(p[(13-rank)%2], rank))
		}
	}
	check(t, &s)
	if !s.Solved() || s.Won() {
		t.Fatal("not solved")
	}
	for i := 0; i < 52; i++ {
		p, d, ok := s.NextHome()
		if !ok {
			t.Fatalf("stuck after %d cards", i)
		}
		s.Move(p, d)
		check(t, &s)
	}
	if !s.Won() {
		t.Fatal("not won")
	}
}
