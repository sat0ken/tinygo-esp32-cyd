package solitaire

import "github.com/sat0ken/tinygo-cyd/games/cards"

// Klondike rules. Pure logic, no drawing.

// Column is a tableau pile: Cards[0] is at the bottom, the first Down
// cards are face down.
type Column struct {
	Cards []cards.Card
	Down  int
}

// State is the whole table.
type State struct {
	Stock      []cards.Card // face down; the last one is drawn first
	Waste      []cards.Card // face up; the last one is playable
	Foundation [4][]cards.Card
	Tableau    [7]Column
	Draw       int // cards drawn from the stock at a time: 1 or 3
	Moves      int
}

// Deal lays out a new game from a shuffled deck.
func Deal(seed uint32, draw int) State {
	deck := cards.Shuffled(cards.NewRand(seed))
	var s State
	s.Draw = draw
	k := 0
	for col := 0; col < 7; col++ {
		s.Tableau[col].Cards = append([]cards.Card(nil), deck[k:k+col+1]...)
		s.Tableau[col].Down = col
		k += col + 1
	}
	s.Stock = append([]cards.Card(nil), deck[k:]...)
	return s
}

// Where a card is.
type PileKind uint8

const (
	StockPile PileKind = iota
	WastePile
	FoundationPile
	TableauPile
	NoPile
)

// Pos is a card position: the pile and, for the tableau, the column and
// the index of the card in it.
type Pos struct {
	Kind  PileKind
	Pile  int // foundation 0..3 or column 0..6
	Index int // index in the column (tableau only)
}

// top returns the last card of a pile and whether there is one.
func top(p []cards.Card) (cards.Card, bool) {
	if len(p) == 0 {
		return 0, false
	}
	return p[len(p)-1], true
}

// CanFound reports whether c can go on foundation f.
func (s *State) CanFound(c cards.Card, f int) bool {
	t, ok := top(s.Foundation[f])
	if !ok {
		return c.Rank() == 1
	}
	return t.Suit() == c.Suit() && c.Rank() == t.Rank()+1
}

// CanStack reports whether c (with any cards on it) can go on column col:
// alternating colours, one rank lower, or a king on an empty column.
func (s *State) CanStack(c cards.Card, col int) bool {
	t, ok := top(s.Tableau[col].Cards)
	if !ok {
		return c.Rank() == 13
	}
	return t.Red() != c.Red() && c.Rank() == t.Rank()-1
}

// Movable returns the cards that would move from p (one card, or a run of
// face-up cards in the tableau), or nil if p cannot be moved.
func (s *State) Movable(p Pos) []cards.Card {
	switch p.Kind {
	case WastePile:
		if len(s.Waste) > 0 {
			return s.Waste[len(s.Waste)-1:]
		}
	case FoundationPile:
		if f := s.Foundation[p.Pile]; len(f) > 0 {
			return f[len(f)-1:]
		}
	case TableauPile:
		c := &s.Tableau[p.Pile]
		if p.Index >= c.Down && p.Index < len(c.Cards) {
			return c.Cards[p.Index:]
		}
	}
	return nil
}

// Dest is where a card or run can go.
type Dest struct {
	Kind PileKind // FoundationPile or TableauPile
	Pile int
}

// AutoDest picks where the cards at p go when tapped: a single card goes
// to its foundation if it can; otherwise the first column that takes it,
// preferring a non-empty column (a king moves to an empty column only if
// it is not already at the bottom of one). ok is false if nothing fits.
func (s *State) AutoDest(p Pos) (d Dest, ok bool) {
	run := s.Movable(p)
	if run == nil {
		return Dest{}, false
	}
	first := run[0]
	if len(run) == 1 && p.Kind != FoundationPile {
		for f := 0; f < 4; f++ {
			if s.CanFound(first, f) {
				return Dest{FoundationPile, f}, true
			}
		}
	}
	empty := -1
	for col := 0; col < 7; col++ {
		if p.Kind == TableauPile && col == p.Pile {
			continue
		}
		if len(s.Tableau[col].Cards) == 0 {
			if empty < 0 {
				empty = col
			}
			continue
		}
		if s.CanStack(first, col) {
			return Dest{TableauPile, col}, true
		}
	}
	if empty >= 0 && first.Rank() == 13 {
		// Moving a king that already sits at the bottom of a column (with
		// nothing face down under it) to another empty column is pointless.
		if p.Kind == TableauPile && p.Index == 0 {
			return Dest{}, false
		}
		return Dest{TableauPile, empty}, true
	}
	return Dest{}, false
}

// Move moves the cards at p to d (which must be legal) and turns over the
// card left on top of a column.
func (s *State) Move(p Pos, d Dest) {
	run := append([]cards.Card(nil), s.Movable(p)...)
	switch p.Kind {
	case WastePile:
		s.Waste = s.Waste[:len(s.Waste)-1]
	case FoundationPile:
		f := &s.Foundation[p.Pile]
		*f = (*f)[:len(*f)-1]
	case TableauPile:
		c := &s.Tableau[p.Pile]
		c.Cards = c.Cards[:p.Index]
		if c.Down > 0 && c.Down >= len(c.Cards) {
			c.Down = len(c.Cards) - 1 // turn over the new top card
		}
	}
	switch d.Kind {
	case FoundationPile:
		s.Foundation[d.Pile] = append(s.Foundation[d.Pile], run...)
	case TableauPile:
		s.Tableau[d.Pile].Cards = append(s.Tableau[d.Pile].Cards, run...)
	}
	s.Moves++
}

// DrawStock turns over Draw cards from the stock onto the waste, or, if
// the stock is empty, turns the waste over to make the stock again.
// It returns false if both are empty.
func (s *State) DrawStock() bool {
	if len(s.Stock) == 0 {
		if len(s.Waste) == 0 {
			return false
		}
		for i := len(s.Waste) - 1; i >= 0; i-- {
			s.Stock = append(s.Stock, s.Waste[i])
		}
		s.Waste = s.Waste[:0]
		s.Moves++
		return true
	}
	for i := 0; i < s.Draw && len(s.Stock) > 0; i++ {
		c := s.Stock[len(s.Stock)-1]
		s.Stock = s.Stock[:len(s.Stock)-1]
		s.Waste = append(s.Waste, c)
	}
	s.Moves++
	return true
}

// Won reports whether all cards are on the foundations.
func (s *State) Won() bool {
	for _, f := range s.Foundation {
		if len(f) != 13 {
			return false
		}
	}
	return true
}

// Solved reports whether the game can be finished without decisions: the
// stock and waste are empty and every tableau card is face up.
func (s *State) Solved() bool {
	if len(s.Stock) > 0 || len(s.Waste) > 0 {
		return false
	}
	for _, c := range s.Tableau {
		if c.Down > 0 {
			return false
		}
	}
	return true
}

// NextHome returns a card that can go to a foundation now (the lowest
// one), for finishing a solved game automatically.
func (s *State) NextHome() (Pos, Dest, bool) {
	best, bestRank := Pos{Kind: NoPile}, 14
	var bestDest Dest
	for col, c := range s.Tableau {
		t, ok := top(c.Cards)
		if !ok {
			continue
		}
		for f := 0; f < 4; f++ {
			if s.CanFound(t, f) && t.Rank() < bestRank {
				best = Pos{TableauPile, col, len(c.Cards) - 1}
				bestDest, bestRank = Dest{FoundationPile, f}, t.Rank()
			}
		}
	}
	return best, bestDest, best.Kind != NoPile
}

// ---- snapshots for UNDO ----

// encode packs the state into a few dozen bytes (much less memory than a
// deep copy of all the slices).
func (s *State) encode() []byte {
	b := make([]byte, 0, 52+20)
	put := func(p []cards.Card) {
		b = append(b, byte(len(p)))
		for _, c := range p {
			b = append(b, byte(c))
		}
	}
	put(s.Stock)
	put(s.Waste)
	for _, f := range s.Foundation {
		put(f)
	}
	for _, c := range s.Tableau {
		b = append(b, byte(c.Down))
		put(c.Cards)
	}
	b = append(b, byte(s.Moves), byte(s.Moves>>8))
	return b
}

func (s *State) decode(b []byte) {
	get := func() []cards.Card {
		n := int(b[0])
		p := make([]cards.Card, n)
		for i := range p {
			p[i] = cards.Card(b[1+i])
		}
		b = b[1+n:]
		return p
	}
	s.Stock = get()
	s.Waste = get()
	for f := range s.Foundation {
		s.Foundation[f] = get()
	}
	for col := range s.Tableau {
		s.Tableau[col].Down = int(b[0])
		b = b[1:]
		s.Tableau[col].Cards = get()
	}
	s.Moves = int(b[0]) | int(b[1])<<8
}

// count returns how many times each card is on the table (for tests: it
// must be exactly once each).
func (s *State) count() [52]int {
	var n [52]int
	add := func(p []cards.Card) {
		for _, c := range p {
			n[c]++
		}
	}
	add(s.Stock)
	add(s.Waste)
	for _, f := range s.Foundation {
		add(f)
	}
	for _, c := range s.Tableau {
		add(c.Cards)
	}
	return n
}
