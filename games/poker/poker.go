package poker

import "github.com/sat0ken/tinygo-cyd/games/cards"

// Jacks or Better rules. Pure logic, no drawing.

// Hand is the value of five cards, from Nothing up to RoyalFlush.
type Hand uint8

const (
	Nothing Hand = iota
	JacksOrBetter
	TwoPair
	ThreeOfAKind
	Straight
	Flush
	FullHouse
	FourOfAKind
	StraightFlush
	RoyalFlush
	numHands
)

var handNames = [numHands]string{
	"", "JACKS OR BETTER", "TWO PAIR", "THREE OF A KIND", "STRAIGHT",
	"FLUSH", "FULL HOUSE", "FOUR OF A KIND", "STRAIGHT FLUSH", "ROYAL FLUSH",
}

func (h Hand) String() string { return handNames[h] }

// pays is the "9/6" pay table: credits won per credit bet.
var pays = [numHands]int{0, 1, 2, 3, 4, 6, 9, 25, 50, 250}

// MaxBet is the largest bet. A royal flush pays 800 per credit (4000)
// instead of 250 only at this bet, as on real machines.
const MaxBet = 5

// Payout returns the credits won by hand h with bet credits bet (the bet
// itself included, as machines show it).
func Payout(h Hand, bet int) int {
	if h == RoyalFlush && bet == MaxBet {
		return 4000
	}
	return pays[h] * bet
}

// Evaluate returns the value of five cards.
func Evaluate(h [5]cards.Card) Hand {
	var count [14]int // by rank, 1..13
	flush := true
	for i, c := range h {
		count[c.Rank()]++
		if i > 0 && c.Suit() != h[0].Suit() {
			flush = false
		}
	}

	// Group sizes: how many ranks appear 2, 3 or 4 times, and the pair rank.
	pairs, threes, fours, pairRank := 0, 0, 0, 0
	for r := 1; r <= 13; r++ {
		switch count[r] {
		case 2:
			pairs++
			pairRank = r
		case 3:
			threes++
		case 4:
			fours++
		}
	}

	// A straight is five different ranks in a row; the ace is low (A2345)
	// or high (10JQKA).
	straight, aceHigh := false, false
	if pairs+threes+fours == 0 {
		lo := 1
		for count[lo] == 0 {
			lo++
		}
		switch {
		case lo+4 <= 13 && count[lo+1]+count[lo+2]+count[lo+3]+count[lo+4] == 4:
			straight = true
		case count[1]+count[10]+count[11]+count[12]+count[13] == 5:
			straight, aceHigh = true, true
		}
	}

	switch {
	case straight && flush && aceHigh:
		return RoyalFlush
	case straight && flush:
		return StraightFlush
	case fours == 1:
		return FourOfAKind
	case threes == 1 && pairs == 1:
		return FullHouse
	case flush:
		return Flush
	case straight:
		return Straight
	case threes == 1:
		return ThreeOfAKind
	case pairs == 2:
		return TwoPair
	case pairs == 1 && (pairRank == 1 || pairRank >= 11):
		return JacksOrBetter
	}
	return Nothing
}

// Round is one hand: the shuffled deck, the five cards shown and which
// of them are held.
type Round struct {
	deck [52]cards.Card
	next int // deck[next] is the next card to deal
	Hand [5]cards.Card
	Held [5]bool
}

// NewRound shuffles and deals five cards.
func NewRound(r *cards.Rand) Round {
	var o Round
	o.deck = cards.Shuffled(r)
	copy(o.Hand[:], o.deck[:5])
	o.next = 5
	return o
}

// Draw replaces the cards that are not held with new ones from the deck
// and returns the value of the final hand.
func (o *Round) Draw() Hand {
	for i := range o.Hand {
		if !o.Held[i] {
			o.Hand[i] = o.deck[o.next]
			o.next++
		}
	}
	return Evaluate(o.Hand)
}
