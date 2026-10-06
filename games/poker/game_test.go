package poker

import (
	"testing"

	"github.com/sat0ken/tinygo-cyd/board"
	"github.com/sat0ken/tinygo-cyd/games/cards"
	"github.com/sat0ken/tinygo-cyd/internal/goldentest"
	"github.com/sat0ken/tinygo-cyd/memlcd"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/freesans"
	"tinygo.org/x/tinyfont/proggy"
)

const dt = float32(1.0 / 60)

func newTestGame() (*Game, *memlcd.Display, *memlcd.Touch) {
	d := memlcd.New(board.Width, board.Height)
	touch := &memlcd.Touch{}
	g := NewGame(d, touch, func() uint32 { return 12 })
	g.Start()
	return g, d, touch
}

func tapRect(g *Game, touch *memlcd.Touch, r rect) {
	touch.Pressed = false
	g.Update(dt)
	g.Update(0.2) // more than tapGap since the last tap
	touch.X, touch.Y, touch.Pressed = r.x+r.w/2, r.y+r.h/2, true
	g.Update(dt)
	touch.Pressed = false
	g.Update(dt)
}

// settle runs frames until the cards have turned over.
func settle(g *Game) {
	for i := 0; i < 100 && (g.phase == dealing || g.phase == drawing); i++ {
		g.Update(dt)
	}
}

// stack replaces the round just dealt (its cards are still face down)
// with one that has hand and then draws the cards in more.
func stack(g *Game, hand [5]cards.Card, more ...cards.Card) {
	if g.phase != dealing || g.faceUp != [5]bool{} {
		panic("stack: not right after a deal")
	}
	o := NewRound(cards.NewRand(1))
	// Reorder the deck, so it is still the 52 cards.
	for i, c := range append(hand[:], more...) {
		for j := range o.deck {
			if o.deck[j] == c {
				o.deck[i], o.deck[j] = o.deck[j], o.deck[i]
			}
		}
	}
	copy(o.Hand[:], o.deck[:5])
	g.round = o
}

func TestInitialScreen(t *testing.T) {
	_, d, _ := newTestGame()
	goldentest.Check(t, "poker_start", d.Image())
}

func TestDealHoldDrawWin(t *testing.T) {
	g, d, touch := newTestGame()
	tapRect(g, touch, betButton())
	tapRect(g, touch, betButton())
	if g.bet != 3 {
		t.Fatalf("bet %d", g.bet)
	}
	tapRect(g, touch, dealButton())
	if g.credits != startCr-3 || g.phase != dealing {
		t.Fatalf("credits %d phase %d", g.credits, g.phase)
	}
	stack(g, [5]cards.Card{
		cards.New(cards.Hearts, 9), cards.New(cards.Spades, 9), cards.New(cards.Clubs, 2),
		cards.New(cards.Diamonds, 13), cards.New(cards.Diamonds, 9),
	}, cards.New(cards.Clubs, 9), cards.New(cards.Hearts, 5))
	tapRect(g, touch, cardRect(0)) // ignored while the cards turn over
	settle(g)
	if g.phase != holding || g.round.Held != [5]bool{} {
		t.Fatalf("phase %d held %v", g.phase, g.round.Held)
	}
	for _, i := range []int{0, 1, 4} {
		tapRect(g, touch, cardRect(i))
	}
	tapRect(g, touch, cardRect(3)) // hold and let go again
	tapRect(g, touch, cardRect(3))
	if g.round.Held != [5]bool{true, true, false, false, true} {
		t.Fatalf("held %v", g.round.Held)
	}
	goldentest.Check(t, "poker_hold", d.Image())

	tapRect(g, touch, dealButton())
	settle(g)
	if g.phase != idle || g.result != FourOfAKind || g.won != 75 || g.credits != startCr-3+75 {
		t.Fatalf("phase %d result %v won %d credits %d", g.phase, g.result, g.won, g.credits)
	}
	goldentest.Check(t, "poker_win", d.Image())
}

func TestMaxBetDeals(t *testing.T) {
	g, _, touch := newTestGame()
	tapRect(g, touch, maxButton())
	if g.bet != MaxBet || g.credits != startCr-MaxBet || g.phase != dealing {
		t.Fatalf("bet %d credits %d phase %d", g.bet, g.credits, g.phase)
	}
	tapRect(g, touch, betButton()) // not while playing
	settle(g)
	tapRect(g, touch, betButton())
	if g.bet != MaxBet {
		t.Fatal("bet changed during a hand")
	}
}

func TestOutOfCredits(t *testing.T) {
	g, _, touch := newTestGame()
	g.credits, g.bet = 2, 5
	tapRect(g, touch, dealButton())
	if g.bet != 2 || g.credits != 0 {
		t.Fatalf("bet %d credits %d", g.bet, g.credits)
	}
	settle(g)
	tapRect(g, touch, dealButton())
	settle(g)
	g.credits = 0 // make sure, whatever the hand paid
	tapRect(g, touch, dealButton())
	if g.credits != startCr || !g.refill || g.phase != idle {
		t.Fatalf("credits %d refill %v phase %d", g.credits, g.refill, g.phase)
	}
}

// TestManyHands plays random hands and checks that the credits add up.
func TestManyHands(t *testing.T) {
	g, _, touch := newTestGame()
	r := cards.NewRand(99)
	bets, wins := 0, 0
	for n := 0; n < 300; n++ {
		g.bet = 1 + r.Intn(MaxBet)
		before := g.credits
		tapRect(g, touch, dealButton())
		if g.refill {
			bets -= startCr // count the refill as a negative bet
			continue
		}
		bets += before - g.credits
		settle(g)
		for i := 0; i < 5; i++ {
			if r.Intn(2) == 0 {
				tapRect(g, touch, cardRect(i))
			}
		}
		held := g.round.Held
		kept := g.round.Hand
		tapRect(g, touch, dealButton())
		settle(g)
		for i := range held {
			if held[i] && g.round.Hand[i] != kept[i] {
				t.Fatalf("hand %d: held card %d changed", n, i)
			}
		}
		if g.result != Evaluate(g.round.Hand) {
			t.Fatalf("hand %d: result %v of %v", n, g.result, g.round.Hand)
		}
		wins += g.won
		if g.credits != startCr-bets+wins {
			t.Fatalf("hand %d: credits %d, want %d", n, g.credits, startCr-bets+wins)
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

func TestTextsFit(t *testing.T) {
	for _, b := range []struct {
		r    rect
		font tinyfont.Fonter
		name string
	}{
		{backButton(), backFont, "< GAMES"}, {betButton(), btnFont, "BET ONE"}, {maxButton(), btnFont, "MAX BET"},
		{dealButton(), btnFont, "DEAL"}, {infoRect(), btnFont, "CREDIT 9999"},
	} {
		if _, w := tinyfont.LineWidth(b.font, b.name); int16(w) > b.r.w-6 {
			t.Errorf("%q is %dpx, button is %dpx", b.name, w, b.r.w)
		}
	}
	for _, s := range []string{"OUT OF CREDITS - TAP DEAL FOR 100 MORE", "STRAIGHT FLUSH   WIN 4000"} {
		if _, w := tinyfont.LineWidth(&freesans.Bold9pt7b, s); w > 470 {
			t.Errorf("%q is %dpx", s, w)
		}
	}
	if _, w := tinyfont.LineWidth(&proggy.TinySZ8pt7b, StraightFlush.String()); nameX+int16(w) > betColX {
		t.Errorf("hand names reach the bet columns")
	}
	if last := cardRect(4); last.x+last.w > board.Width || last.y+last.h > barY {
		t.Errorf("cards end at %d,%d", last.x+last.w, last.y+last.h)
	}
}
