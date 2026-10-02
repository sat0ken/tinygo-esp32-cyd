package main

import (
	"math/bits"
	"testing"
)

// perft counts the positions after `depth` moves (a pass counts as a move
// only when the game is not over), the standard check of move generation.
func perft(p Position, depth int) int {
	if depth == 0 {
		return 1
	}
	moves := p.Moves()
	if moves == 0 {
		if p.Pass().Moves() == 0 {
			return 1 // game over
		}
		return perft(p.Pass(), depth-1)
	}
	n := 0
	for b := moves; b != 0; b &= b - 1 {
		n += perft(p.Play(bits.TrailingZeros64(b)), depth-1)
	}
	return n
}

func TestPerft(t *testing.T) {
	// Known values for othello from the start position.
	want := []int{1, 4, 12, 56, 244, 1396, 8200, 55092}
	for d, w := range want {
		if got := perft(Start(), d); got != w {
			t.Fatalf("perft(%d) = %d, want %d", d, got, w)
		}
	}
}

func sq(name string) int { return int(name[1]-'1')*8 + int(name[0]-'a') }

func TestOpeningMovesAndFlips(t *testing.T) {
	p := Start()
	want := uint64(1)<<sq("d3") | 1<<sq("c4") | 1<<sq("f5") | 1<<sq("e6")
	if p.Moves() != want {
		t.Fatalf("moves %x, want %x", p.Moves(), want)
	}
	// Black plays d3 and turns d4.
	if f := p.Flips(sq("d3")); f != 1<<sq("d4") {
		t.Fatalf("flips %x", f)
	}
	q := p.Play(sq("d3"))
	// Now white is to move: white has e5 only; black has d3 d4 d5 e4.
	if me, opp := q.Count(); me != 1 || opp != 4 {
		t.Fatalf("count %d %d", me, opp)
	}
}

func TestFlipsInManyDirections(t *testing.T) {
	// Black at the ends of lines through the empty d4, white in between.
	var p Position
	for _, s := range []string{"b2", "d2", "f2", "b4", "f4", "b6", "d6", "f6"} {
		p.Me |= 1 << sq(s)
	}
	for _, s := range []string{"c3", "d3", "e3", "c4", "e4", "c5", "d5", "e5"} {
		p.Opp |= 1 << sq(s)
	}
	if f := p.Flips(sq("d4")); f != p.Opp {
		t.Fatalf("flips %x, want all eight %x", f, p.Opp)
	}
	// No wrap-around: a stone on h1 must not flank anything on the next row.
	var q Position
	q.Me = 1 << sq("h1")
	q.Opp = 1 << sq("a2")
	if q.Moves()&(1<<sq("b2")) != 0 {
		t.Fatal("move found by wrapping around the edge")
	}
}

func TestCPUTakesTheCorner(t *testing.T) {
	// White (to move) can take a1 by flanking b2 diagonally, or play
	// elsewhere. A one-move look-ahead and NORMAL must take the corner.
	// (HARD is not checked here: in this tiny artificial position it finds
	// a line that wipes out all opponent stones, which beats the corner.
	// Its strength is checked by TestCPUGamesEndLegally.)
	var p Position
	p.Me = 1<<sq("c3") | 1<<sq("e5")
	p.Opp = 1<<sq("b2") | 1<<sq("d4") | 1<<sq("e4")
	if p.Moves()&(1<<sq("a1")) == 0 {
		t.Fatal("a1 is not legal in the test position")
	}
	for _, lv := range []Level{{Name: "depth1", Depth: 1, MaxNodes: 1000}, levels[1]} {
		m, _, _ := BestMove(p, lv, nil)
		if m != sq("a1") {
			t.Errorf("%s played %d, want a1 (%d)", lv.Name, m, sq("a1"))
		}
	}
}

func TestCPUGamesEndLegally(t *testing.T) {
	r := uint32(1)
	noise := func(n int) int {
		r ^= r << 13
		r ^= r >> 17
		r ^= r << 5
		return int(r%uint32(2*n+1)) - n
	}
	// HARD (black) against EASY (white): every move must be legal and HARD
	// should win.
	p := Start()
	black := true
	maxNodes := 0
	for !p.Over() {
		if p.Moves() == 0 {
			p, black = p.Pass(), !black
			continue
		}
		lv := levels[0]
		if black {
			lv = levels[2]
		}
		m, _, nodes := BestMove(p, lv, noise)
		if nodes > maxNodes {
			maxNodes = nodes
		}
		if p.Moves()&(1<<m) == 0 {
			t.Fatalf("illegal move %d", m)
		}
		p, black = p.Play(m), !black
	}
	me, opp := p.Count()
	if !black {
		me, opp = opp, me // make `me` black
	}
	t.Logf("HARD (black) %d - EASY (white) %d, max nodes per move %d", me, opp, maxNodes)
	if me+opp > 64 || me <= opp {
		t.Fatalf("HARD did not win: %d-%d", me, opp)
	}
	if maxNodes > levels[2].MaxNodes {
		t.Fatalf("node budget exceeded: %d", maxNodes)
	}
}
