package othello

import (
	"math/bits"
	"testing"
	"time"

	"github.com/sat0ken/tinygo-cyd/board"
	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/internal/goldentest"
	"github.com/sat0ken/tinygo-cyd/memlcd"
	"github.com/sat0ken/tinygo-cyd/pad"
)

const dt = float32(0.05)

// sizes are the screens the game has a layout for; golden is the prefix of
// the golden images.
var sizes = []struct {
	w, h   int16
	golden string
}{
	{board.Width, board.Height, "othello_"}, // ESP32-4827S043, ESP32-3248S035C
	{320, 240, "othello_small_"},            // ESP32-2432S028R
}

func newTestGame() (*Game, *memlcd.Display, *memlcd.Touch) {
	return newTestGameSize(board.Width, board.Height)
}

func newTestGameSize(w, h int16) (*Game, *memlcd.Display, *memlcd.Touch) {
	d := memlcd.New(w, h)
	touch := &memlcd.Touch{}
	g := NewGame(d, touch, 7)
	g.Now = func() time.Time { return time.Time{} } // "0ms" on the screen
	g.DrawMenu()
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

// settle runs updates until it is the player's turn or the game is over.
func settle(t *testing.T, g *Game) {
	t.Helper()
	for i := 0; i < 2000; i++ {
		if g.phase == humanTurn || g.phase == gameOver {
			return
		}
		g.Update(dt)
	}
	t.Fatalf("stuck in phase %d", g.phase)
}

func stones(g *Game) int { return bits.OnesCount64(g.pos.Me | g.pos.Opp) }

func TestMenuScreen(t *testing.T) {
	for _, s := range sizes {
		g, d, _ := newTestGameSize(s.w, s.h)
		if s.w == 320 {
			g.OnExit = func() {} // the "< GAMES" button must clear the title
			g.DrawMenu()
		}
		goldentest.Check(t, s.golden+"menu", d.Image())
	}
}

func TestStartScreen(t *testing.T) {
	for _, s := range sizes {
		g, d, touch := newTestGameSize(s.w, s.h)
		tapRect(g, touch, g.l.levelButton(1)) // NORMAL, player is black
		settle(t, g)
		if g.phase != humanTurn || stones(g) != 4 {
			t.Fatalf("phase %d stones %d", g.phase, stones(g))
		}
		goldentest.Check(t, s.golden+"start", d.Image())
	}
}

// TestLayout checks that everything is on the screen and that nothing that
// can be tapped overlaps.
func TestLayout(t *testing.T) {
	for _, s := range sizes {
		l := layoutFor(s.w, s.h)
		screen := rect{0, 0, s.w, s.h}
		panel := rect{l.panelX, 0, l.panelW, l.undo.y - 8}
		taps := []rect{l.boardRect(), l.undo, l.new, panel}
		all := append([]rect{backButton(), l.colorButton(0), l.colorButton(1)}, taps...)
		for k := range levels {
			all = append(all, l.levelButton(k))
		}
		for _, r := range all {
			if !inside(r, screen) {
				t.Errorf("%dx%d: %v is off the screen", s.w, s.h, r)
			}
		}
		for i, a := range taps {
			for _, b := range taps[i+1:] {
				if overlap(a, b) {
					t.Errorf("%dx%d: %v overlaps %v", s.w, s.h, a, b)
				}
			}
		}
	}
}

func inside(r, outer rect) bool {
	return r.x >= outer.x && r.y >= outer.y && r.x+r.w <= outer.x+outer.w && r.y+r.h <= outer.y+outer.h
}

func overlap(a, b rect) bool {
	return a.x < b.x+b.w && b.x < a.x+a.w && a.y < b.y+b.h && b.y < a.y+a.h
}

func TestMoveAndCPUReply(t *testing.T) {
	for _, s := range sizes {
		testMoveAndCPUReply(t, s.w, s.h)
	}
}

func testMoveAndCPUReply(t *testing.T, w, h int16) {
	g, d, touch := newTestGameSize(w, h)
	tapRect(g, touch, g.l.levelButton(1))
	settle(t, g)

	// An illegal square (a1) is ignored.
	tapRect(g, touch, g.l.squareRect(0))
	if g.phase != humanTurn || stones(g) != 4 {
		t.Fatal("illegal move was played")
	}

	// d3 is legal for black.
	tapRect(g, touch, g.l.squareRect(sq("d3")))
	settle(t, g)
	if stones(g) != 6 || !g.blackMove || g.phase != humanTurn {
		t.Fatalf("after one move each: stones %d blackMove %v phase %d", stones(g), g.blackMove, g.phase)
	}
	// The screen shows the position: every square's look was drawn.
	for s := 0; s < 64; s++ {
		if g.drawn[s] != g.look(s) {
			t.Fatalf("square %d not redrawn", s)
		}
	}
	// The CPU's last move has the red dot drawn at its centre.
	r := g.l.squareRect(g.last)
	if got := d.Pixel(r.x+r.w/2, r.y+r.h/2); got != rgb565(lastColor) {
		t.Fatalf("last move marker %04x", got)
	}

	// UNDO goes back to the start position.
	tapRect(g, touch, g.l.undo)
	if g.pos != Start() || !g.blackMove || g.phase != humanTurn {
		t.Fatal("undo did not restore the start")
	}
}

func TestPlayAsWhite(t *testing.T) {
	g, _, touch := newTestGame()
	tapRect(g, touch, g.l.colorButton(1)) // white
	tapRect(g, touch, g.l.levelButton(0))
	settle(t, g)
	// The CPU (black) has moved first.
	if stones(g) != 5 || g.blackMove {
		t.Fatalf("stones %d blackMove %v", stones(g), g.blackMove)
	}
}

func TestWholeGame(t *testing.T) {
	for _, s := range sizes {
		testWholeGame(t, s.w, s.h, s.golden)
	}
}

func testWholeGame(t *testing.T, sw, sh int16, golden string) {
	g, d, touch := newTestGameSize(sw, sh)
	tapRect(g, touch, g.l.levelButton(0)) // EASY
	settle(t, g)
	for moves := 0; g.phase != gameOver; moves++ {
		if moves > 70 {
			t.Fatal("game does not end")
		}
		// Play the first legal move (passes are handled by the game).
		m := g.pos.Moves()
		s := bits.TrailingZeros64(m)
		tapRect(g, touch, g.l.squareRect(s))
		settle(t, g)
	}
	b, w := g.counts()
	t.Logf("final %d-%d: %s", b, w, g.message)
	if b+w > 64 || g.message == "" {
		t.Fatalf("bad end: %d-%d %q", b, w, g.message)
	}
	if golden == "othello_small_" {
		goldentest.Check(t, golden+"end", d.Image())
	}
	tapRect(g, touch, g.l.new)
	if g.phase != menu {
		t.Fatal("NEW did not go back to the menu")
	}
}

func TestHumanPasses(t *testing.T) {
	g, _, touch := newTestGame()
	tapRect(g, touch, g.l.levelButton(1))
	settle(t, g)
	// Black (the player) to move but without a legal move; white can move.
	// Black b2; white on the diagonal c3..h8 up to the edge, so black cannot
	// flank it, but white can play a1 and flank b2.
	g.pos = Position{Me: 1 << sq("b2")}
	for _, s := range []string{"c3", "d4", "e5", "f6", "g7", "h8"} {
		g.pos.Opp |= 1 << sq(s)
	}
	g.blackMove = true
	if g.pos.Moves() != 0 || g.pos.Pass().Moves() == 0 {
		t.Fatal("test position is not a pass for black")
	}
	g.nextTurn()
	if g.phase != passing || g.message != "YOU PASS" {
		t.Fatalf("phase %d message %q", g.phase, g.message)
	}
	// After the delay the CPU (white) moves.
	for i := 0; i < 100 && g.phase == passing; i++ {
		g.Update(dt)
	}
	if g.phase != cpuTurn && g.phase != animating {
		t.Fatalf("phase %d after the pass", g.phase)
	}
	settle(t, g)
	if g.pos.Opp&(1<<sq("a1")) == 0 && g.pos.Me&(1<<sq("a1")) == 0 {
		t.Fatal("the CPU did not play a1")
	}
	_ = touch
}

func rgb565(c interface{ RGBA() (r, g, b, a uint32) }) uint16 {
	r, gg, b, _ := c.RGBA()
	return uint16(r>>8&0xF8)<<8 | uint16(gg>>8&0xFC)<<3 | uint16(b>>8)>>3
}

// press presses b on the controller for one frame and lets go.
func press(g *Game, f *pad.Fake, b hal.Buttons) {
	f.B = b
	g.Update(dt)
	f.B = 0
	g.Update(dt)
}

// TestPad plays with the controller only: WHITE and EASY on the menu, a
// move with the cursor, UNDO with X, and NEW.
func TestPad(t *testing.T) {
	for _, s := range sizes {
		g, d, _ := newTestGameSize(s.w, s.h)
		f := &pad.Fake{}
		g.Pad = f
		g.Update(dt)
		press(g, f, hal.PadUp) // from NORMAL to the colours
		if g.menuFocus >= 2 {
			t.Fatalf("%dx%d: up from NORMAL: focus %d", s.w, s.h, g.menuFocus)
		}
		if g.menuFocus == 0 {
			press(g, f, hal.PadRight)
		}
		press(g, f, hal.PadA)
		if g.humanBlack {
			t.Fatalf("%dx%d: WHITE not chosen", s.w, s.h)
		}
		press(g, f, hal.PadDown)
		for n := 0; g.menuFocus != 2; n++ {
			if n > 3 {
				t.Fatalf("%dx%d: EASY not reached", s.w, s.h)
			}
			press(g, f, hal.PadLeft)
		}
		press(g, f, hal.PadA)
		settle(t, g)
		if g.phase != humanTurn || stones(g) != 5 {
			t.Fatalf("%dx%d: phase %d stones %d", s.w, s.h, g.phase, stones(g))
		}
		r := g.l.squareRect(g.focus)
		if d.Pixel(r.x, r.y) != rgb565(focusColor) {
			t.Fatalf("%dx%d: no cursor on square %d", s.w, s.h, g.focus)
		}

		// Walk the cursor to the first legal move and play it.
		to := bits.TrailingZeros64(g.pos.Moves())
		for n := 0; g.focus != to; n++ {
			if n > 16 {
				t.Fatalf("%dx%d: cursor at %d, not at %d", s.w, s.h, g.focus, to)
			}
			switch {
			case g.focus/8 > to/8:
				press(g, f, hal.PadUp)
			case g.focus/8 < to/8:
				press(g, f, hal.PadDown)
			case g.focus%8 > to%8:
				press(g, f, hal.PadLeft)
			default:
				press(g, f, hal.PadRight)
			}
		}
		before := g.pos
		press(g, f, hal.PadA)
		settle(t, g)
		if stones(g) != 7 || g.phase != humanTurn {
			t.Fatalf("%dx%d: after a move each: stones %d phase %d", s.w, s.h, stones(g), g.phase)
		}
		press(g, f, hal.PadX) // UNDO
		if g.pos != before {
			t.Fatalf("%dx%d: X did not undo the move", s.w, s.h)
		}

		// Right from the last column reaches UNDO and NEW; NEW goes back
		// to the menu.
		for n := 0; g.focus < focusUndo; n++ {
			if n > 8 {
				t.Fatalf("%dx%d: right does not reach the buttons", s.w, s.h)
			}
			press(g, f, hal.PadRight)
		}
		if g.focus == focusUndo {
			press(g, f, hal.PadRight)
		}
		press(g, f, hal.PadA)
		if g.phase != menu {
			t.Fatalf("%dx%d: NEW did not go back to the menu", s.w, s.h)
		}
	}
}
