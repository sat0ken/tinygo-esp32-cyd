package poker

import (
	"testing"

	"github.com/sat0ken/tinygo-cyd/games/cards"
)

// hand builds five cards from strings like "AS", "10H", "QD".
func hand(t *testing.T, s ...string) [5]cards.Card {
	var h [5]cards.Card
	for i, x := range s {
		found := false
		for c := cards.Card(0); c < 52; c++ {
			if c.String() == x {
				h[i], found = c, true
			}
		}
		if !found {
			t.Fatalf("no card %q", x)
		}
	}
	return h
}

func TestEvaluate(t *testing.T) {
	for _, c := range []struct {
		cards []string
		want  Hand
	}{
		{[]string{"10S", "JS", "QS", "KS", "AS"}, RoyalFlush},
		{[]string{"AH", "KH", "QH", "JH", "10H"}, RoyalFlush},
		{[]string{"9D", "10D", "JD", "QD", "KD"}, StraightFlush},
		{[]string{"AC", "2C", "3C", "4C", "5C"}, StraightFlush}, // ace low
		{[]string{"7S", "7H", "7D", "7C", "2S"}, FourOfAKind},
		{[]string{"3S", "3H", "3D", "9C", "9S"}, FullHouse},
		{[]string{"2H", "7H", "9H", "JH", "KH"}, Flush},
		{[]string{"5S", "6H", "7D", "8C", "9S"}, Straight},
		{[]string{"10S", "JH", "QD", "KC", "AS"}, Straight}, // ace high
		{[]string{"AS", "2H", "3D", "4C", "5S"}, Straight},  // ace low
		{[]string{"QS", "KH", "AD", "2C", "3S"}, Nothing},   // no wrapping
		{[]string{"4S", "4H", "4D", "KC", "2S"}, ThreeOfAKind},
		{[]string{"4S", "4H", "KD", "KC", "2S"}, TwoPair},
		{[]string{"JS", "JH", "3D", "7C", "2S"}, JacksOrBetter},
		{[]string{"AS", "AH", "3D", "7C", "2S"}, JacksOrBetter},
		{[]string{"10S", "10H", "3D", "7C", "2S"}, Nothing}, // pair too low
		{[]string{"2S", "5H", "9D", "JC", "KS"}, Nothing},
	} {
		if got := Evaluate(hand(t, c.cards...)); got != c.want {
			t.Errorf("%v: %v, want %v", c.cards, got, c.want)
		}
	}
}

// TestEvaluateAllHands checks the evaluator against the well-known number
// of each kind of hand among all 2,598,960 five-card hands.
func TestEvaluateAllHands(t *testing.T) {
	if testing.Short() {
		t.Skip("all hands")
	}
	var n [numHands]int
	var h [5]cards.Card
	for a := 0; a < 52; a++ {
		for b := a + 1; b < 52; b++ {
			for c := b + 1; c < 52; c++ {
				for d := c + 1; d < 52; d++ {
					for e := d + 1; e < 52; e++ {
						h = [5]cards.Card{cards.Card(a), cards.Card(b), cards.Card(c), cards.Card(d), cards.Card(e)}
						n[Evaluate(h)]++
					}
				}
			}
		}
	}
	want := [numHands]int{
		RoyalFlush:    4,
		StraightFlush: 36,
		FourOfAKind:   624,
		FullHouse:     3744,
		Flush:         5108,
		Straight:      10200,
		ThreeOfAKind:  54912,
		TwoPair:       123552,
		JacksOrBetter: 1098240 * 4 / 13, // pairs of J, Q, K or A
	}
	want[Nothing] = 2598960
	for k := JacksOrBetter; k < numHands; k++ {
		want[Nothing] -= want[k]
	}
	if n != want {
		t.Errorf("counts %v, want %v", n, want)
	}
}

func TestPayout(t *testing.T) {
	if Payout(RoyalFlush, 4) != 1000 || Payout(RoyalFlush, 5) != 4000 {
		t.Error("royal flush")
	}
	if Payout(FullHouse, 3) != 27 || Payout(JacksOrBetter, 1) != 1 || Payout(Nothing, 5) != 0 {
		t.Error("pay table")
	}
}

func TestDrawKeepsHeldCards(t *testing.T) {
	o := NewRound(cards.NewRand(7))
	first := o.Hand
	o.Held = [5]bool{true, false, true, false, false}
	o.Draw()
	seen := map[cards.Card]bool{}
	for i, c := range o.Hand {
		if o.Held[i] != (c == first[i]) {
			t.Errorf("card %d: held %v, %v -> %v", i, o.Held[i], first[i], c)
		}
		seen[c] = true
	}
	for _, c := range first {
		seen[c] = true
	}
	if len(seen) != 8 { // 5 dealt + 3 new, all different
		t.Errorf("%d different cards", len(seen))
	}
}
