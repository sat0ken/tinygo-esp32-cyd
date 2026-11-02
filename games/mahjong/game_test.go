package mahjong

import (
	"testing"

	"github.com/sat0ken/tinygo-cyd/board"
	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/internal/goldentest"
	"github.com/sat0ken/tinygo-cyd/memlcd"
	"github.com/sat0ken/tinygo-cyd/pad"
)

const dt = float32(0.05)

var sizes = []struct {
	w, h   int16
	golden string
}{
	{board.Width, board.Height, "mahjong_"}, // ESP32-4827S043, ESP32-3248S035C
	{320, 240, "mahjong_small_"},            // ESP32-2432S028R
}

func newTestGame(w, h int16) (*Game, *memlcd.Display, *memlcd.Touch) {
	d := memlcd.New(w, h)
	touch := &memlcd.Touch{}
	g := NewGame(d, touch, 7)
	g.DrawMenu()
	return g, d, touch
}

func tapAt(g *Game, touch *memlcd.Touch, x, y int16) {
	touch.Pressed = false
	g.Update(dt)
	g.Update(0.2)
	touch.X, touch.Y, touch.Pressed = x, y, true
	g.Update(dt)
	touch.Pressed = false
	g.Update(dt)
}

func tapRect(g *Game, touch *memlcd.Touch, r rect) { tapAt(g, touch, r.x+r.w/2, r.y+r.h/2) }

// waitForPlayer runs until the player has something to do.
func waitForPlayer(t *testing.T, g *Game) {
	t.Helper()
	for i := 0; i < 5000; i++ {
		if g.scr != scrTable {
			return
		}
		if g.tb.Phase == phaseTurn && g.tb.Turn == human && !g.tb.P[human].Riichi {
			return
		}
		if g.tb.Waiting(human) {
			return
		}
		g.Update(dt)
	}
	t.Fatal("the player's turn never comes")
}

func TestMenuScreen(t *testing.T) {
	for _, s := range sizes {
		g, d, _ := newTestGame(s.w, s.h)
		g.OnExit = func() {}
		g.DrawMenu()
		goldentest.Check(t, s.golden+"menu", d.Image())
	}
}

func TestStartScreen(t *testing.T) {
	for _, s := range sizes {
		g, d, touch := newTestGame(s.w, s.h)
		tapRect(g, touch, g.l.short)
		if g.scr != scrTable {
			t.Fatalf("screen %d", g.scr)
		}
		waitForPlayer(t, g)
		goldentest.Check(t, s.golden+"start", d.Image())
	}
}

// playHand plays until the end of the hand with the player discarding the
// tile just drawn, or winning, or passing.
func playHand(t *testing.T, g *Game, touch *memlcd.Touch) {
	t.Helper()
	for i := 0; i < 8000 && g.scr == scrTable; i++ {
		if g.tb.Waiting(human) {
			tapRect(g, touch, g.buttonRect(g.nButtons-1)) // PASS
			continue
		}
		if tb := g.tb; tb.Phase == phaseTurn && tb.Turn == human && !tb.P[human].Riichi {
			s := g.nSlots - 1
			for !g.slotAllowed(s) {
				s--
			}
			r := g.slotRect(s)
			tapRect(g, touch, r)
			tapRect(g, touch, r)
			continue
		}
		g.Update(dt)
	}
}

func TestPlayWholeGame(t *testing.T) {
	for _, s := range sizes {
		g, _, touch := newTestGame(s.w, s.h)
		tapRect(g, touch, g.l.short)
		for i := 0; i < 200 && g.scr != scrMenu; i++ {
			switch g.scr {
			case scrTable:
				playHand(t, g, touch)
			default:
				for j := 0; j < 10; j++ {
					g.Update(dt)
				}
				tapAt(g, touch, 10, 10)
			}
		}
		if g.scr != scrMenu {
			t.Fatalf("%dx%d: the game does not end (screen %d)", s.w, s.h, g.scr)
		}
	}
}

// resultGame sets up a known end of a hand for the screens.
func resultGame(g *Game, kind int) {
	g.start(false)
	tb := g.tb
	switch kind {
	case endWin:
		p := &tb.P[1]
		p.Hand = parseHand("234m678m123p567s22m")
		p.Melds = [4]Meld{}
		p.NMeld = 0
		p.Riichi, p.ippatsu = true, true
		tb.Dealer = 0
		tb.Discarder = human
		tb.Discard0 = pin
		tb.Result = Result{Kind: endWin}
		p.Hand[pin]--
		tb.addWin(1, human, pin)
		tb.finishWins(1)
	case endDraw:
		for s := range tb.P {
			tb.P[s].Hand = parseHand("123m456p789s1122z")
			tb.P[s].Hand[east+1]--
		}
		tb.P[2].Hand = parseHand("147m258p369s1234z")
		tb.exhausted()
	}
	g.page = 0
	g.showResult()
}

func TestResultScreens(t *testing.T) {
	for _, s := range sizes {
		g, d, _ := newTestGame(s.w, s.h)
		resultGame(g, endWin)
		goldentest.Check(t, s.golden+"win", d.Image())
		g, d, _ = newTestGame(s.w, s.h)
		resultGame(g, endDraw)
		goldentest.Check(t, s.golden+"draw", d.Image())
	}
}

// TestLayout checks that everything is on the screen and that what can be
// tapped does not overlap.
func TestLayout(t *testing.T) {
	for _, s := range sizes {
		l := layoutFor(s.w, s.h)
		g := &Game{l: l, nSlots: 14, nHand: 13, nButtons: 5}
		screen := rect{0, 0, s.w, s.h}
		var all []rect
		for i := 0; i < 14; i++ {
			all = append(all, g.slotRect(i))
		}
		for i := 0; i < 5; i++ {
			all = append(all, g.buttonRect(i))
		}
		all = append(all, l.quit)
		for _, r := range all {
			if !inside(r, screen) {
				t.Errorf("%dx%d: %v is off the screen", s.w, s.h, r)
			}
		}
		for i, a := range all {
			for _, b := range all[i+1:] {
				if overlap(a, b) {
					t.Errorf("%dx%d: %v overlaps %v", s.w, s.h, a, b)
				}
			}
		}
		for seat := 0; seat < 4; seat++ {
			// The riichi tile on its side makes one row longer.
			r := rect{l.river[seat][0], l.river[seat][1], 6*l.rw + l.rh - l.rw, 4 * l.rh}
			if !inside(r, rect{0, 0, s.w, l.handY - l.lift}) {
				t.Errorf("%dx%d: river %d %v", s.w, s.h, seat, r)
			}
			if overlap(r, l.center) {
				t.Errorf("%dx%d: river %d overlaps the middle", s.w, s.h, seat)
			}
			if m := (rect{l.melds[seat][0], l.melds[seat][1], 4 * l.rw, 2 * l.rh}); seat != human {
				for o := 0; o < 4; o++ {
					if q := (rect{l.river[o][0], l.river[o][1], 6*l.rw + l.rh - l.rw, 4 * l.rh}); overlap(m, q) {
						t.Errorf("%dx%d: the melds of %d overlap river %d", s.w, s.h, seat, o)
					}
				}
			}
			if overlap(r, l.quit) {
				t.Errorf("%dx%d: river %d overlaps QUIT", s.w, s.h, seat)
			}
			for other := seat + 1; other < 4; other++ {
				o := rect{l.river[other][0], l.river[other][1], 6*l.rw + l.rh - l.rw, 4 * l.rh}
				if overlap(r, o) {
					t.Errorf("%dx%d: rivers %d and %d overlap", s.w, s.h, seat, other)
				}
			}
			for _, b := range []rect{g.buttonRect(0), g.buttonRect(4)} {
				if overlap(r, b) {
					t.Errorf("%dx%d: river %d overlaps a button", s.w, s.h, seat)
				}
			}
		}
	}
}

func inside(r, o rect) bool {
	return r.x >= o.x && r.y >= o.y && r.x+r.w <= o.x+o.w && r.y+r.h <= o.y+o.h
}

func overlap(a, b rect) bool {
	return a.x < b.x+b.w && b.x < a.x+a.w && a.y < b.y+b.h && b.y < a.y+a.h
}

// TestPadPlay discards and passes with the controller alone.
func TestPadPlay(t *testing.T) {
	for _, s := range sizes {
		g, _, _ := newTestGame(s.w, s.h)
		f := &pad.Fake{}
		g.Pad = f
		press := func(b hal.Buttons) {
			f.B = b
			g.Update(dt)
			f.B = 0
			g.Update(dt)
		}
		g.Update(dt)
		press(hal.PadA) // the first button: starts
		if g.scr != scrTable {
			t.Fatalf("%dx%d: not started", s.w, s.h)
		}
		discards := 0
		for i := 0; i < 3000 && discards < 5; i++ {
			switch {
			case g.scr != scrTable:
				press(hal.PadA)
			case g.tb.Waiting(human):
				press(hal.PadB)
			case g.tb.Phase == phaseTurn && g.tb.Turn == human && !g.tb.P[human].Riichi:
				if g.slotAllowed(g.focus) {
					press(hal.PadA)
					discards++
				} else {
					press(hal.PadLeft)
				}
			default:
				g.Update(dt)
			}
		}
		if discards < 5 {
			t.Fatalf("%dx%d: %d discards", s.w, s.h, discards)
		}
	}
}

// TestTwoTaps checks that the first tap only lifts the tile.
func TestTwoTaps(t *testing.T) {
	g, _, touch := newTestGame(board.Width, board.Height)
	tapRect(g, touch, g.l.short)
	for g.tb.Phase != phaseTurn || g.tb.Turn != human || g.tb.Waiting(human) {
		if g.tb.Waiting(human) {
			tapRect(g, touch, g.buttonRect(g.nButtons-1))
			continue
		}
		g.Update(dt)
	}
	n := g.tb.P[human].NRiv
	r := g.slotRect(0)
	tapRect(g, touch, r)
	if g.lifted != 0 || g.tb.P[human].NRiv != n {
		t.Fatalf("lifted %d, %d discards", g.lifted, g.tb.P[human].NRiv)
	}
	tapRect(g, touch, r)
	if g.tb.P[human].NRiv != n+1 {
		t.Fatal("no discard on the second tap")
	}
}

// playerTurn sets up the player's turn with a hand and the tile just drawn.
func playerTurn(g *Game, hand string, drawn Tile) {
	tb := g.tb
	p := &tb.P[human]
	p.Hand = parseHand(hand)
	p.Waits = waitsOf(&p.Hand, 0)
	p.Hand[drawn]++
	p.Drawn = drawn
	p.NRiv = 0
	tb.pure = false
	tb.Turn = human
	tb.Phase = phaseTurn
	g.changed()
}

// buttonOf returns the rectangle of the button doing act.
func buttonOf(t *testing.T, g *Game, act uint8) rect {
	t.Helper()
	for i := 0; i < g.nButtons; i++ {
		if g.buttons[i].act == act {
			return g.buttonRect(i)
		}
	}
	t.Fatalf("no button %d in %+v", act, g.buttons[:g.nButtons])
	return rect{}
}

func started(s struct {
	w, h   int16
	golden string
}) (*Game, *memlcd.Display, *memlcd.Touch) {
	g, d, touch := newTestGame(s.w, s.h)
	tapRect(g, touch, g.l.short)
	return g, d, touch
}

func TestPlayerTsumo(t *testing.T) {
	for _, s := range sizes {
		g, _, touch := started(s)
		playerTurn(g, "234m678m23p567s22m", pin)
		tapRect(g, touch, buttonOf(t, g, actTsumo))
		if g.tb.Phase != phaseOver || g.tb.Result.Wins[0].Seat != human {
			t.Fatalf("phase %d %+v", g.tb.Phase, g.tb.Result)
		}
		for i := 0; i < 40; i++ {
			g.Update(dt)
		}
		if g.scr != scrResult {
			t.Fatalf("screen %d", g.scr)
		}
	}
}

func TestPlayerRiichi(t *testing.T) {
	for _, s := range sizes {
		g, d, touch := started(s)
		playerTurn(g, "234m678m23p567s22m", north)
		goldentest.Check(t, s.golden+"riichi", d.Image())
		score := g.tb.P[human].Score
		tapRect(g, touch, buttonOf(t, g, actRiichi))
		// Only the tile that keeps the hand ready can go.
		for i := 0; i < g.nSlots; i++ {
			if g.slotAllowed(i) != (g.slots[i] == north) {
				t.Fatalf("slot %d (tile %d) allowed %v", i, g.slots[i], g.slotAllowed(i))
			}
		}
		r := g.slotRect(g.nSlots - 1)
		tapRect(g, touch, r)
		tapRect(g, touch, r)
		p := &g.tb.P[human]
		if !p.Riichi || p.NRiv != 1 || !p.River[0].Riichi {
			t.Fatalf("riichi %v, %d discards", p.Riichi, p.NRiv)
		}
		for i := 0; i < 200 && g.tb.Waiting(human); i++ {
			tapRect(g, touch, buttonOf(t, g, actPass))
		}
		for i := 0; i < 40 && g.tb.Phase == phaseClaims; i++ {
			g.Update(dt)
		}
		if g.tb.Phase != phaseOver && p.Score != score-1000 {
			t.Fatalf("score %d, was %d", p.Score, score)
		}
	}
}

// otherDiscards has the seat before the player discard k.
func otherDiscards(g *Game, hand string, k Tile) {
	tb := g.tb
	p := &tb.P[human]
	p.Hand = parseHand(hand)
	p.Waits = waitsOf(&p.Hand, 0)
	p.Drawn = noTile
	p.NRiv = 0
	tb.Turn = 3
	tb.P[3].Hand[k]++
	tb.P[3].Drawn = noTile
	tb.DiscardTile(k, false)
	g.changed()
}

func TestPlayerRon(t *testing.T) {
	for _, s := range sizes {
		g, _, touch := started(s)
		otherDiscards(g, "234m678m23p567s22m", pin)
		tapRect(g, touch, buttonOf(t, g, actRon))
		for i := 0; i < 40 && g.tb.Phase != phaseOver; i++ {
			g.Update(dt)
		}
		if g.tb.Phase != phaseOver || g.tb.Result.Wins[0].Seat != human || g.tb.Result.Wins[0].From != 3 {
			t.Fatalf("phase %d %+v", g.tb.Phase, g.tb.Result)
		}
	}
}

func TestPlayerPon(t *testing.T) {
	for _, s := range sizes {
		g, _, touch := started(s)
		otherDiscards(g, "11m234m678m123p56s", man)
		tapRect(g, touch, buttonOf(t, g, actPon))
		for i := 0; i < 40 && g.tb.Waiting(3); i++ {
			g.Update(dt)
		}
		for i := 0; i < 40 && g.tb.Phase == phaseClaims; i++ {
			g.Update(dt)
		}
		p := &g.tb.P[human]
		if p.NMeld != 1 || p.Melds[0].Kind != ponMeld || g.tb.Turn != human || g.tb.Phase != phaseTurn {
			t.Fatalf("%d melds, turn %d, phase %d", p.NMeld, g.tb.Turn, g.tb.Phase)
		}
		if g.slotAllowed(0) && g.slots[0] == man {
			t.Fatal("may discard the tile just called")
		}
		r := g.slotRect(g.nSlots - 1)
		tapRect(g, touch, r)
		tapRect(g, touch, r)
		if p.NRiv != 1 {
			t.Fatalf("%d discards after the pon", p.NRiv)
		}
	}
}

func TestPlayerChiChoice(t *testing.T) {
	for _, s := range sizes {
		g, d, touch := started(s)
		otherDiscards(g, "12m45m678m123p56s9s", man+2)
		tapRect(g, touch, buttonOf(t, g, actChi))
		if g.mode != modeChi || g.nButtons != 4 {
			t.Fatalf("mode %d, %d buttons", g.mode, g.nButtons)
		}
		goldentest.Check(t, s.golden+"chi", d.Image())
		tapRect(g, touch, g.buttonRect(2))
		for i := 0; i < 40 && g.tb.Phase == phaseClaims; i++ {
			g.Update(dt)
		}
		p := &g.tb.P[human]
		if p.NMeld != 1 || p.Melds[0].Kind != chiMeld {
			t.Fatalf("%d melds", p.NMeld)
		}
		// 3 called with 45: neither 3 nor 6 may go.
		if !p.forbid.has(man+2) || !p.forbid.has(man+5) {
			t.Fatalf("forbid %x, meld %+v", uint64(p.forbid), p.Melds[0])
		}
	}
}

// TestQuit needs two presses.
func TestQuit(t *testing.T) {
	g, _, touch := newTestGame(board.Width, board.Height)
	tapRect(g, touch, g.l.short)
	tapRect(g, touch, g.l.quit)
	if g.scr != scrTable {
		t.Fatal("left at the first press")
	}
	tapRect(g, touch, g.l.quit)
	if g.scr != scrMenu {
		t.Fatal("still playing after the second press")
	}
}

// TestPadCannotQuitByAccident presses A all the time while the CPUs play.
func TestPadCannotQuitByAccident(t *testing.T) {
	g, _, _ := newTestGame(board.Width, board.Height)
	f := &pad.Fake{}
	g.Pad = f
	press := func(b hal.Buttons) {
		f.B = b
		g.Update(dt)
		f.B = 0
		g.Update(dt)
	}
	g.Update(dt)
	press(hal.PadA)
	for i := 0; i < 500 && g.scr == scrTable; i++ {
		press(hal.PadA)
		if g.armed > 0 {
			t.Fatal("QUIT? armed by pressing A")
		}
	}
	if g.scr == scrMenu {
		t.Fatal("back on the menu")
	}
}

// TestAddedKanRedraws checks that an added kan shows on the screen.
func TestAddedKanRedraws(t *testing.T) {
	g, d, _ := newTestGame(board.Width, board.Height)
	g.start(false)
	tb := g.tb
	p := &tb.P[human]
	p.Hand = parseHand("234m678m123p5s")
	p.Melds[0] = Meld{Kind: ponMeld, Tile: red, Called: red, From: 1}
	p.NMeld = 1
	p.Waits = waitsOf(&p.Hand, 1)
	tb.Turn = human
	p.Hand[red]++
	p.Drawn = red
	tb.Phase = phaseTurn
	g.changed()
	before := d.Image()
	for s := range tb.P {
		tb.P[s].Waits = 0 // nobody wins with it
	}
	tb.Kan(red)
	g.changed()
	if p.Melds[0].Kind != addedKan {
		t.Fatalf("meld %+v, phase %d", p.Melds[0], tb.Phase)
	}
	if goldentest.Compare(before, d.Image()) == "" {
		t.Fatal("the screen did not change")
	}
}

// fullRiver fills a seat's discards, with the riichi tile at position at.
func fullRiver(tb *Table, seat, at int) {
	p := &tb.P[seat]
	p.NRiv = 24
	for i := 0; i < 24; i++ {
		p.River[i] = River{Tile: Tile((i*7 + seat) % numKinds)}
	}
	p.River[at].Riichi = true
	p.Riichi = true
}

// TestRiichiTable shows the worst case: all four in riichi, with long
// rivers, the riichi tile at the start of a row so that all of it moves.
func TestRiichiTable(t *testing.T) {
	for _, s := range sizes {
		g, d, _ := newTestGame(s.w, s.h)
		g.start(false)
		tb := g.tb
		for seat := range tb.P {
			fullRiver(tb, seat, 6*seat)
			tb.Sticks++
			tb.P[seat].Score -= 1000
		}
		tb.Discarder = 3
		tb.Turn = human
		tb.Phase = phaseDraw
		g.changed()
		goldentest.Check(t, s.golden+"sticks", d.Image())
	}
}

// TestRiichiTileTaken checks that the next discard lies on its side.
func TestRiichiTileTaken(t *testing.T) {
	g, _, _ := newTestGame(board.Width, board.Height)
	g.start(false)
	tb := g.tb
	p := &tb.P[1]
	p.NRiv = 2
	p.River[0] = River{Tile: east, Riichi: true, Called: true}
	p.River[1] = River{Tile: south}
	g.changed()
	if v := g.drawnRiver[1][0]; v.tile != south || v.flags&flagRiichi == 0 {
		t.Fatalf("%+v", v)
	}
}

// TestStickOnlyWhenPaid checks that a declared riichi shows its stick after
// nobody won with the tile.
func TestStickOnlyWhenPaid(t *testing.T) {
	g, d, _ := newTestGame(board.Width, board.Height)
	g.start(false)
	tb := g.tb
	playerTurn(g, "234m678m23p567s22m", north)
	r := g.stickRect(human)
	tb.DiscardTile(north, true)
	for s := range tb.P {
		if tb.Waiting(s) {
			tb.AnswerClaim(s, Claim{})
		}
	}
	g.changed()
	if got := d.Pixel(r.x, r.y); got != rgb565(textColor) {
		t.Fatalf("no stick: %x", got)
	}
	if tb.Sticks != 1 {
		t.Fatalf("%d sticks", tb.Sticks)
	}
}

func rgb565(c interface{ RGBA() (r, g, b, a uint32) }) uint16 {
	r, gg, b, _ := c.RGBA()
	return uint16(r>>8&0xF8)<<8 | uint16(gg>>8&0xFC)<<3 | uint16(b>>8)>>3
}
