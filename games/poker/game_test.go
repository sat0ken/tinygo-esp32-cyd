package poker

import (
	"image/color"
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
	{board.Width, board.Height, "poker_"}, // ESP32-4827S043, ESP32-3248S035C
	{320, 240, "poker_small_"},            // ESP32-2432S028R
}

func newTestGame() (*Game, *memlcd.Display, *memlcd.Touch) {
	return newTestGameSize(board.Width, board.Height)
}

func newTestGameSize(w, h int16) (*Game, *memlcd.Display, *memlcd.Touch) {
	d := memlcd.New(w, h)
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
	for _, s := range sizes {
		g, d, _ := newTestGameSize(s.w, s.h)
		if s.w == 320 {
			g.OnExit = func() {}
			g.Start()
		}
		goldentest.Check(t, s.golden+"start", d.Image())
	}
}

func TestDealHoldDrawWin(t *testing.T) {
	for _, s := range sizes {
		testDealHoldDrawWin(t, s.w, s.h, s.golden)
	}
}

func testDealHoldDrawWin(t *testing.T, w, h int16, golden string) {
	g, d, touch := newTestGameSize(w, h)
	tapRect(g, touch, g.l.bet.r)
	tapRect(g, touch, g.l.bet.r)
	if g.bet != 3 {
		t.Fatalf("bet %d", g.bet)
	}
	tapRect(g, touch, g.l.deal.r)
	if g.credits != startCr-3 || g.phase != dealing {
		t.Fatalf("credits %d phase %d", g.credits, g.phase)
	}
	stack(g, [5]cards.Card{
		cards.New(cards.Hearts, 9), cards.New(cards.Spades, 9), cards.New(cards.Clubs, 2),
		cards.New(cards.Diamonds, 13), cards.New(cards.Diamonds, 9),
	}, cards.New(cards.Clubs, 9), cards.New(cards.Hearts, 5))
	tapRect(g, touch, g.l.cardRect(0)) // ignored while the cards turn over
	settle(g)
	if g.phase != holding || g.round.Held != [5]bool{} {
		t.Fatalf("phase %d held %v", g.phase, g.round.Held)
	}
	for _, i := range []int{0, 1, 4} {
		tapRect(g, touch, g.l.cardRect(i))
	}
	tapRect(g, touch, g.l.cardRect(3)) // hold and let go again
	tapRect(g, touch, g.l.cardRect(3))
	if g.round.Held != [5]bool{true, true, false, false, true} {
		t.Fatalf("held %v", g.round.Held)
	}
	goldentest.Check(t, golden+"hold", d.Image())

	tapRect(g, touch, g.l.deal.r)
	settle(g)
	if g.phase != idle || g.result != FourOfAKind || g.won != 75 || g.credits != startCr-3+75 {
		t.Fatalf("phase %d result %v won %d credits %d", g.phase, g.result, g.won, g.credits)
	}
	goldentest.Check(t, golden+"win", d.Image())
}

func TestMaxBetDeals(t *testing.T) {
	g, _, touch := newTestGame()
	tapRect(g, touch, g.l.max.r)
	if g.bet != MaxBet || g.credits != startCr-MaxBet || g.phase != dealing {
		t.Fatalf("bet %d credits %d phase %d", g.bet, g.credits, g.phase)
	}
	tapRect(g, touch, g.l.bet.r) // not while playing
	settle(g)
	tapRect(g, touch, g.l.bet.r)
	if g.bet != MaxBet {
		t.Fatal("bet changed during a hand")
	}
}

func TestOutOfCredits(t *testing.T) {
	g, _, touch := newTestGame()
	g.credits, g.bet = 2, 5
	tapRect(g, touch, g.l.deal.r)
	if g.bet != 2 || g.credits != 0 {
		t.Fatalf("bet %d credits %d", g.bet, g.credits)
	}
	settle(g)
	tapRect(g, touch, g.l.deal.r)
	settle(g)
	g.credits = 0 // make sure, whatever the hand paid
	tapRect(g, touch, g.l.deal.r)
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
		tapRect(g, touch, g.l.deal.r)
		if g.refill {
			bets -= startCr // count the refill as a negative bet
			continue
		}
		bets += before - g.credits
		settle(g)
		for i := 0; i < 5; i++ {
			if r.Intn(2) == 0 {
				tapRect(g, touch, g.l.cardRect(i))
			}
		}
		held := g.round.Held
		kept := g.round.Hand
		tapRect(g, touch, g.l.deal.r)
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
	tapRect(g, touch, g.l.back.r)
	if !left {
		t.Fatal("OnExit not called")
	}
}

func TestTextsFit(t *testing.T) {
	for _, s := range sizes {
		l := layoutFor(s.w, s.h)
		for _, b := range []struct {
			b    button
			name string
		}{
			{l.back, "< GAMES"}, {l.bet, "BET ONE"}, {l.max, "MAX BET"}, {l.deal, "DEAL"}, {l.deal, "DRAW"},
			{button{l.info, l.creditFont, 0}, "CREDIT 9999"},
		} {
			if _, w := tinyfont.LineWidth(b.b.font, b.name); int16(w) > b.b.r.w-6 {
				t.Errorf("%dx%d: %q is %dpx, button is %dpx", s.w, s.h, b.name, w, b.b.r.w)
			}
		}
		for _, m := range []string{"OUT OF CREDITS - TAP DEAL FOR 100 MORE", "STRAIGHT FLUSH   WIN 4000"} {
			font, _ := l.msgFont(m)
			if _, w := tinyfont.LineWidth(font, m); int16(w) > l.labels.w-8 {
				t.Errorf("%dx%d: %q is %dpx", s.w, s.h, m, w)
			}
		}
		if _, w := tinyfont.LineWidth(&proggy.TinySZ8pt7b, StraightFlush.String()); l.nameX+int16(w) > l.betColX {
			t.Errorf("%dx%d: hand names reach the bet columns", s.w, s.h)
		}
		if _, w := tinyfont.LineWidth(&proggy.TinySZ8pt7b, "4000"); int16(w)+l.numRight > l.betColW-4 {
			t.Errorf("%dx%d: the royal flush pay does not fit its column", s.w, s.h)
		}
		if last := l.cardRect(4); last.x+last.w > s.w || last.y+last.h > l.barY {
			t.Errorf("%dx%d: cards end at %d,%d", s.w, s.h, last.x+last.w, last.y+last.h)
		}
		if col := l.betColX + MaxBet*l.betColW; col > l.pay.x+l.pay.w {
			t.Errorf("%dx%d: the pay table ends at %d", s.w, s.h, col)
		}
		bar := []rect{l.back.r, l.bet.r, l.max.r, l.info, l.deal.r}
		for i, a := range bar {
			if a.x < 0 || a.x+a.w > s.w || a.y+a.h > s.h {
				t.Errorf("%dx%d: %v is off the screen", s.w, s.h, a)
			}
			for _, b := range bar[i+1:] {
				if a.x < b.x+b.w && b.x < a.x+a.w {
					t.Errorf("%dx%d: %v overlaps %v", s.w, s.h, a, b)
				}
			}
		}
	}
}

// press presses b on the controller for one frame and lets go.
func press(g *Game, f *pad.Fake, b hal.Buttons) {
	f.B = b
	g.Update(dt)
	f.B = 0
	g.Update(dt)
}

// TestPad plays a hand with the controller only: DEAL, hold the first two
// cards, DRAW with X, then "< GAMES".
func TestPad(t *testing.T) {
	for _, s := range sizes {
		g, d, _ := newTestGameSize(s.w, s.h)
		left := false
		g.OnExit = func() { left = true }
		g.Start()
		f := &pad.Fake{}
		g.Pad = f
		g.Update(dt)
		press(g, f, hal.PadA) // DEAL has the focus
		if g.phase != dealing || g.credits != startCr-1 || g.focus != 0 {
			t.Fatalf("%dx%d: phase %d credits %d focus %d", s.w, s.h, g.phase, g.credits, g.focus)
		}
		settle(g)
		press(g, f, hal.PadA)
		press(g, f, hal.PadRight)
		press(g, f, hal.PadA)
		if g.round.Held != [5]bool{true, true} {
			t.Fatalf("%dx%d: held %v", s.w, s.h, g.round.Held)
		}
		r := g.l.cardRect(1)
		if d.Pixel(r.x, r.y) != rgb565(focusColor) {
			t.Fatalf("%dx%d: no focus on the second card", s.w, s.h)
		}
		kept := g.round.Hand
		press(g, f, hal.PadX) // DRAW
		settle(g)
		if g.phase != idle || g.round.Hand[0] != kept[0] || g.round.Hand[1] != kept[1] || g.focus != focusDeal {
			t.Fatalf("%dx%d: phase %d focus %d after the draw", s.w, s.h, g.phase, g.focus)
		}
		for n := 0; g.focus != focusBack; n++ {
			if n > 4 {
				t.Fatalf("%dx%d: < GAMES not reached (focus %d)", s.w, s.h, g.focus)
			}
			press(g, f, hal.PadLeft)
		}
		press(g, f, hal.PadA)
		if !left {
			t.Fatalf("%dx%d: < GAMES did not call OnExit", s.w, s.h)
		}
	}
}

func rgb565(c color.RGBA) uint16 {
	return uint16(c.R&0xF8)<<8 | uint16(c.G&0xFC)<<3 | uint16(c.B)>>3
}
