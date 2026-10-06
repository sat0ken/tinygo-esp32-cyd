package cards

import "testing"

func TestCards(t *testing.T) {
	for s := Spades; s <= Clubs; s++ {
		for r := 1; r <= 13; r++ {
			c := New(s, r)
			if c.Suit() != s || c.Rank() != r {
				t.Fatalf("%v: suit %d rank %d", c, c.Suit(), c.Rank())
			}
			if c.Red() != (s == Hearts || s == Diamonds) {
				t.Fatalf("%v: red %v", c, c.Red())
			}
		}
	}
	if New(Hearts, 10).String() != "10H" || New(Spades, 1).String() != "AS" {
		t.Fatal("names")
	}
}

func TestShuffled(t *testing.T) {
	a := Shuffled(NewRand(5))
	b := Shuffled(NewRand(5))
	c := Shuffled(NewRand(6))
	if a != b || a == c {
		t.Fatal("not repeatable per seed")
	}
	var seen [52]bool
	for _, x := range a {
		if seen[x] {
			t.Fatalf("%v twice", x)
		}
		seen[x] = true
	}
}
