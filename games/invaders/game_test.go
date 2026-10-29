package invaders

import (
	"testing"

	"github.com/sat0ken/tinygo-cyd/board"
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
	{board.Width, board.Height, "invaders_"}, // ESP32-4827S043, ESP32-3248S035C
	{320, 240, "invaders_small_"},            // ESP32-2432S028R
}

func newTestGame() (*Game, *memlcd.Display, *memlcd.Touch) {
	return newTestGameSize(board.Width, board.Height)
}

func newTestGameSize(w, h int16) (*Game, *memlcd.Display, *memlcd.Touch) {
	d := memlcd.New(w, h)
	touch := &memlcd.Touch{}
	g := NewGame(d, touch, 12345)
	g.DrawAll()
	return g, d, touch
}

func tap(g *Game, touch *memlcd.Touch, x int16) {
	touch.Pressed = false
	g.Update(dt)
	touch.X, touch.Y, touch.Pressed = x, 200, true
	g.Update(dt)
	touch.Pressed = false
	g.Update(dt)
}

// target returns the x of the lowest living invader nearest to the cannon.
func target(g *Game) int16 {
	best, bestD := g.cannonX, int16(1000)
	for r := rows - 1; r >= 0; r-- {
		for c := 0; c < cols; c++ {
			if v := &g.inv[r][c]; v.alive {
				x := v.x + g.l.invW/2
				d := x - (g.cannonX + g.l.pw(&cannonSprite)/2)
				if d < 0 {
					d = -d
				}
				if d < bestD {
					best, bestD = x, d
				}
			}
		}
	}
	return best
}

// checkScreen verifies that every living invader is drawn where the game
// thinks it is, with its colour.
func checkScreen(t *testing.T, g *Game, d *memlcd.Display) {
	t.Helper()
	for r := range g.inv {
		for c := range g.inv[r] {
			v := &g.inv[r][c]
			if !v.alive {
				continue
			}
			sp := g.invSprite(v)
			sr := g.spriteRect(v)
			// The first pixel of a set bit.
		find:
			for y := sr.y; y < sr.y+sr.h; y++ {
				for x := sr.x; x < sr.x+sr.w; x++ {
					if sp.on(g.l.bit(x-sr.x, sp.w), g.l.bit(y-sr.y, sp.h)) {
						if got, want := d.Pixel(x, y), rgb565(kindColors[v.kind]); got != want {
							t.Fatalf("invader %d,%d at (%d,%d): pixel %04x want %04x", r, c, x, y, got, want)
						}
						break find
					}
				}
			}
		}
	}
}

func TestInitialScreen(t *testing.T) {
	for _, s := range sizes {
		g, d, _ := newTestGameSize(s.w, s.h)
		if s.w == 320 {
			g.OnExit = func() {}
			g.DrawAll()
		}
		goldentest.Check(t, s.golden+"initial", d.Image())
	}
}

// TestBit checks the pixel to bit mapping of both scales: every bit gets
// at least one pixel, in order, and the pattern is symmetric.
func TestBit(t *testing.T) {
	for _, s := range sizes {
		l := layoutFor(s.w, s.h)
		for _, w := range []int16{7, 8, 12, 13, 16} {
			pw := l.size(w)
			seen := make([]bool, w)
			for p := int16(0); p < pw; p++ {
				b := l.bit(p, w)
				seen[b] = true
				if p > 0 && b < l.bit(p-1, w) {
					t.Errorf("%dx%d: w %d: bit %d after %d", s.w, s.h, w, b, l.bit(p-1, w))
				}
				if b+l.bit(pw-1-p, w) != w-1 {
					t.Errorf("%dx%d: w %d: pixel %d is bit %d, not symmetric", s.w, s.h, w, p, b)
				}
			}
			for b, ok := range seen {
				if !ok {
					t.Errorf("%dx%d: w %d: bit %d has no pixel", s.w, s.h, w, b)
				}
			}
			if l.bit(-1, w) != -1 || l.bit(pw, w) != -1 {
				t.Errorf("%dx%d: w %d: pixels outside the sprite", s.w, s.h, w)
			}
		}
	}
}

// TestLayout checks that the formation, the bunkers, the cannon, the HUD
// and the message fit on the screen without running into each other.
func TestLayout(t *testing.T) {
	for _, s := range sizes {
		g, _, _ := newTestGameSize(s.w, s.h)
		l := &g.l
		if l.invW != l.pw(&invaderSprites[2][0]) || l.invH != l.ph(&invaderSprites[2][0]) {
			t.Errorf("%dx%d: invW/invH %d,%d", s.w, s.h, l.invW, l.invH)
		}
		right := l.formX0 + (cols-1)*l.cellW + l.invW
		if l.formX0 < l.edge || right > s.w-l.edge || right-l.formX0 > s.w-2*l.edge-16*l.stepX {
			t.Errorf("%dx%d: formation %d..%d leaves too little room to march", s.w, s.h, l.formX0, right)
		}
		// Wave 1 starts above the message ("TAP TO START"), the lowest
		// start (wave 5 and later) above the bunkers; the message is above
		// the bunkers, the bunkers above the cannon.
		bottom := l.formY0 + (rows-1)*l.cellH + l.invH
		lowest := bottom + 4*l.dropY
		bunkerBottom := l.bunkerY + int16(len(bunkerShape))*l.bunkerCell
		if l.ufoY < l.hudH || l.ufoY+l.ph(&ufoSprite) > l.formY0 || bottom > l.msg.y || lowest > l.bunkerY ||
			l.msg.y+l.msg.h > l.bunkerY || bunkerBottom > g.cannonY()-l.shotH {
			t.Errorf("%dx%d: hud %d ufo %d formation %d..%d..%d msg %d bunkers %d..%d cannon %d", s.w, s.h,
				l.hudH, l.ufoY, l.formY0, bottom, lowest, l.msg.y, l.bunkerY, bunkerBottom, g.cannonY())
		}
		if g.bunkerX(0) < 0 || g.bunkerX(bunkers-1)+int16(len(bunkerShape[0]))*l.bunkerCell > s.w {
			t.Errorf("%dx%d: bunkers off the screen", s.w, s.h)
		}
		g.score, g.hi, g.lives, g.wave = 99999, 99999, 3, 99
		hud := "SCORE " + fiveDigits(g.score) + l.hudSep + "HI " + fiveDigits(g.hi) + l.hudSep + "LIVES 3" + l.hudSep + "WAVE 99"
		if _, w := tinyfont.LineWidth(&proggy.TinySZ8pt7b, hud); 8+int16(w) > g.backRect().x-4 {
			t.Errorf("%dx%d: the HUD (%d pixels) runs into the back button", s.w, s.h, w)
		}
		if _, w := tinyfont.LineWidth(&proggy.TinySZ8pt7b, "TAP TO START - SLIDE TO MOVE, HOLD TO FIRE"); int16(w) > l.msg.w {
			t.Errorf("%dx%d: the start message is wider than its line", s.w, s.h)
		}
	}
}

func TestPlay(t *testing.T) {
	for _, s := range sizes {
		testPlay(t, s.w, s.h)
	}
}

func testPlay(t *testing.T, w, h int16) {
	g, d, touch := newTestGameSize(w, h)
	tap(g, touch, w/2)
	if g.state != playing {
		t.Fatalf("state %d after tap", g.state)
	}
	start := g.alive
	for i := 0; i < 60*40 && g.state != gameOver; i++ {
		touch.X, touch.Pressed = target(g), true
		g.Update(dt)
		for r := range g.inv {
			for c := range g.inv[r] {
				v := &g.inv[r][c]
				if v.alive && (v.x < 0 || v.x+g.l.invW > g.w) {
					t.Fatalf("invader %d,%d left the screen: x=%d", r, c, v.x)
				}
			}
		}
		if g.state == playing {
			checkScreen(t, g, d)
		}
	}
	t.Logf("score %d, invaders %d -> %d, lives %d, wave %d, state %d", g.score, start, g.alive, g.lives, g.wave, g.state)
	if g.score == 0 || g.alive >= start && g.wave == 1 {
		t.Fatal("no invader destroyed")
	}
}

func TestFormationMarchesAndDrops(t *testing.T) {
	for _, s := range sizes {
		testFormationMarchesAndDrops(t, s.w, s.h)
	}
}

func testFormationMarchesAndDrops(t *testing.T, w, h int16) {
	g, _, touch := newTestGameSize(w, h)
	tap(g, touch, w/2)
	y0 := g.inv[0][0].y
	// Keep the cannon away from bombs is not needed: count on time only.
	for i := 0; i < 60*30 && g.state == playing; i++ {
		touch.Pressed = false
		g.Update(dt)
	}
	if g.inv[0][0].y <= y0 && g.state == playing {
		t.Fatalf("formation never dropped (y %d)", g.inv[0][0].y)
	}
}

func TestCannonHitAndGameOver(t *testing.T) {
	g, _, touch := newTestGame()
	tap(g, touch, 240)
	for life := lives0; life > 0; life-- {
		// Drop a bomb right above the cannon.
		g.bombs[0] = shot{active: true, x: float32(g.cannonX + g.l.pw(&cannonSprite)/2), y: float32(g.cannonY() - 10)}
		for i := 0; i < 60 && g.state == playing; i++ {
			touch.Pressed = false
			g.Update(dt)
		}
		if g.state != dying || g.lives != life-1 {
			t.Fatalf("after hit: state %d lives %d", g.state, g.lives)
		}
		for i := 0; i < 90; i++ {
			g.Update(dt)
		}
	}
	if g.state != gameOver {
		t.Fatalf("state %d, want game over", g.state)
	}
	tap(g, touch, 240)
	if g.state != ready || g.lives != lives0 || g.score != 0 || g.alive != rows*cols {
		t.Fatalf("restart: state %d lives %d score %d alive %d", g.state, g.lives, g.score, g.alive)
	}
}

func TestInvasionEndsTheGame(t *testing.T) {
	g, _, touch := newTestGame()
	tap(g, touch, 240)
	for r := range g.inv {
		for c := range g.inv[r] {
			g.inv[r][c].y += 170
		}
	}
	g.bombs = [maxBombs]shot{}
	for i := 0; i < 120 && g.state == playing; i++ {
		touch.Pressed = false
		g.Update(dt)
	}
	if g.state != gameOver {
		t.Fatalf("state %d, want game over", g.state)
	}
}

func TestWaveClear(t *testing.T) {
	g, d, touch := newTestGame()
	tap(g, touch, 240)
	// Leave one invader. A lone invader moves every frame (it is very fast,
	// as in the original), so shoot it directly: shooting is covered by
	// TestPlay.
	for r := range g.inv {
		for c := range g.inv[r] {
			g.inv[r][c].alive = r == rows-1 && c == 5
		}
	}
	g.alive = 1
	g.DrawAll()
	g.kill(&g.inv[rows-1][5])
	if g.state != cleared {
		t.Fatalf("state %d, want cleared", g.state)
	}
	for i := 0; i < 120; i++ {
		touch.Pressed = false
		g.Update(dt)
	}
	if g.wave != 2 || g.alive != rows*cols || g.state != playing {
		t.Fatalf("wave %d alive %d state %d", g.wave, g.alive, g.state)
	}
	if want := g.l.formY0 + g.l.dropY; g.inv[0][0].y != want {
		t.Fatalf("wave 2 starts at y %d, want %d", g.inv[0][0].y, want)
	}
	checkScreen(t, g, d)
}

// TestPad plays with the controller only: A starts, the stick moves the
// cannon, A held fires, B on the game over screen is "< GAMES".
func TestPad(t *testing.T) {
	for _, s := range sizes {
		g, _, _ := newTestGameSize(s.w, s.h)
		left := false
		g.OnExit = func() { left = true }
		g.DrawAll()
		f := &pad.Fake{}
		g.Pad = f
		g.Update(dt)
		f.B = hal.PadRight
		g.Update(dt)
		if g.msgShown != "PRESS A TO START - STICK TO MOVE, A TO FIRE" {
			t.Fatalf("%dx%d: message %q", s.w, s.h, g.msgShown)
		}
		f.B = hal.PadA
		g.Update(dt)
		if g.state != playing {
			t.Fatalf("%dx%d: A did not start the game", s.w, s.h)
		}
		g.Update(dt) // still held: fires
		if !g.bullet.active {
			t.Fatalf("%dx%d: A did not fire", s.w, s.h)
		}
		x0 := g.cannonX
		f.B = hal.PadRight
		for i := 0; i < 30; i++ {
			g.Update(dt)
		}
		if moved := float32(g.cannonX - x0); moved < g.l.cannonSpeed/2-2 || moved > g.l.cannonSpeed/2+2 {
			t.Fatalf("%dx%d: the cannon moved %v in half a second", s.w, s.h, moved)
		}
		f.B = hal.PadLeft
		for i := 0; i < 600; i++ {
			g.Update(dt)
		}
		if g.cannonX != 0 && g.state == playing {
			t.Fatalf("%dx%d: the cannon stopped at %d, not at the edge", s.w, s.h, g.cannonX)
		}
		g.lives = 0
		g.setState(gameOver)
		f.B = 0
		g.Update(dt)
		f.B = hal.PadB
		g.Update(dt)
		if !left {
			t.Fatalf("%dx%d: B on game over did not call OnExit", s.w, s.h)
		}
	}
}
