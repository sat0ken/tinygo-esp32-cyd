package othello

import (
	"math/bits"
	"testing"

	"github.com/sat0ken/tinygo-cyd/board"
	"github.com/sat0ken/tinygo-cyd/internal/goldentest"
	"github.com/sat0ken/tinygo-cyd/memlcd"
)

const dt = float32(0.05)

func newTestGame() (*Game, *memlcd.Display, *memlcd.Touch) {
	d := memlcd.New(board.Width, board.Height)
	touch := &memlcd.Touch{}
	g := NewGame(d, touch, 7)
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
	_, d, _ := newTestGame()
	goldentest.Check(t, "othello_menu", d.Image())
}

func TestStartScreen(t *testing.T) {
	g, d, touch := newTestGame()
	tapRect(g, touch, levelButton(1)) // NORMAL, player is black
	settle(t, g)
	if g.phase != humanTurn || stones(g) != 4 {
		t.Fatalf("phase %d stones %d", g.phase, stones(g))
	}
	goldentest.Check(t, "othello_start", d.Image())
}

func TestMoveAndCPUReply(t *testing.T) {
	g, d, touch := newTestGame()
	tapRect(g, touch, levelButton(1))
	settle(t, g)

	// An illegal square (a1) is ignored.
	tapRect(g, touch, squareRect(0))
	if g.phase != humanTurn || stones(g) != 4 {
		t.Fatal("illegal move was played")
	}

	// d3 is legal for black.
	tapRect(g, touch, squareRect(sq("d3")))
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
	r := squareRect(g.last)
	if got := d.Pixel(r.x+r.w/2, r.y+r.h/2); got != rgb565(lastColor) {
		t.Fatalf("last move marker %04x", got)
	}

	// UNDO goes back to the start position.
	tapRect(g, touch, undoButton())
	if g.pos != Start() || !g.blackMove || g.phase != humanTurn {
		t.Fatal("undo did not restore the start")
	}
}

func TestPlayAsWhite(t *testing.T) {
	g, _, touch := newTestGame()
	tapRect(g, touch, colorButton(1)) // white
	tapRect(g, touch, levelButton(0))
	settle(t, g)
	// The CPU (black) has moved first.
	if stones(g) != 5 || g.blackMove {
		t.Fatalf("stones %d blackMove %v", stones(g), g.blackMove)
	}
}

func TestWholeGame(t *testing.T) {
	g, _, touch := newTestGame()
	tapRect(g, touch, levelButton(0)) // EASY
	settle(t, g)
	for moves := 0; g.phase != gameOver; moves++ {
		if moves > 70 {
			t.Fatal("game does not end")
		}
		// Play the first legal move (passes are handled by the game).
		m := g.pos.Moves()
		s := bits.TrailingZeros64(m)
		tapRect(g, touch, squareRect(s))
		settle(t, g)
	}
	b, w := g.counts()
	t.Logf("final %d-%d: %s", b, w, g.message)
	if b+w > 64 || g.message == "" {
		t.Fatalf("bad end: %d-%d %q", b, w, g.message)
	}
	tapRect(g, touch, newButton())
	if g.phase != menu {
		t.Fatal("NEW did not go back to the menu")
	}
}

func TestHumanPasses(t *testing.T) {
	g, _, touch := newTestGame()
	tapRect(g, touch, levelButton(1))
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
