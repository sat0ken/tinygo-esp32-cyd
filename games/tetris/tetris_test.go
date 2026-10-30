package tetris

import (
	"image"
	"testing"
	"time"

	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/internal/goldentest"
	"github.com/sat0ken/tinygo-cyd/internal/picotest"
	"github.com/sat0ken/tinygo-cyd/memlcd"
	"github.com/sat0ken/tinygo-cyd/picopad"
)

func newTest(t *testing.T, w, h int16) (*Game, *memlcd.Display, *picotest.Clock, *picotest.Pad) {
	d := memlcd.New(w, h)
	g := New(d, nil, 12345)
	clock := picotest.NewClock(t, 10*time.Minute)
	g.Clock = clock
	p := &picotest.Pad{Clock: clock}
	g.Pad = p
	return g, d, clock, p
}

const sec = time.Second

// screen returns the PicoPad's screen in the middle of d.
func screen(d *memlcd.Display) *image.RGBA {
	x, y := (int(d.W)-picopad.Width)/2, (int(d.H)-picopad.Height)/2
	img := d.Image()
	out := image.NewRGBA(image.Rect(0, 0, picopad.Width, picopad.Height))
	for yy := 0; yy < picopad.Height; yy++ {
		copy(out.Pix[yy*out.Stride:], img.Pix[(yy+y)*img.Stride+x*4:(yy+y)*img.Stride+(x+picopad.Width)*4])
	}
	return out
}

// checkField checks that the game field on d shows the tiles of the board
// (paused: but for the row of the message).
func checkField(t *testing.T, g *Game, d *memlcd.Display, paused bool) {
	t.Helper()
	ox, oy := (int(d.W)-picopad.Width)/2, (int(d.H)-picopad.Height)/2
	for y := 1; y < mapH-1; y++ {
		if paused && y == picopad.Height/2/tileSize {
			continue
		}
		for x := 8; x < 18; x++ {
			tile := int(g.board[x+y*mapW])
			for ty := 0; ty < tileSize; ty++ {
				for tx := 0; tx < tileSize; tx++ {
					e := int(tilesImg[ty*tilesImgW+tile*tileSize+tx])
					want := uint16(tilesImgPal[2*e]) | uint16(tilesImgPal[2*e+1])<<8
					px, py := ox+x*tileSize+tx, oy+y*tileSize+ty
					if got := d.Pix[py*int(d.W)+px]; got != want {
						t.Fatalf("tile %d, %d (%d) pixel %d, %d: %#04x, want %#04x", x, y, tile, tx, ty, got, want)
					}
				}
			}
		}
	}
}

// quit holds Y from t: it ends the game, and the demo after it the next
// time the demo reads the buttons (which it does not while it deletes
// rows), so that Run returns.
func quit(p *picotest.Pad, t time.Duration) { p.Hold(hal.PadY, t, 4*sec) }

func TestStart(t *testing.T) {
	for _, size := range []struct{ w, h int16 }{{320, 240}, {480, 272}} {
		g, d, clock, p := newTest(t, size.w, size.h)
		clock.At(1*sec, func() {
			// paused, with the first brick at the top
			goldentest.Check(t, "tetris_start", screen(d))
			checkField(t, g, d, true)
			if size.w > 320 {
				for _, xy := range [][2]int{{0, 0}, {79, 100}, {400, 271}, {200, 15}} {
					if v := d.Pix[xy[1]*int(d.W)+xy[0]]; v != 0 {
						t.Errorf("%dx%d: pixel %v outside the screen is %#04x, want black", size.w, size.h, xy, v)
					}
				}
			}
		})
		p.Tap(hal.PadA, 1500*time.Millisecond) // play
		quit(p, 2*sec)
		g.Run()
		if clock.T > 6*sec {
			t.Errorf("%dx%d: Run returned at %v", size.w, size.h, clock.T)
		}
	}
}

func TestMoveTurnDrop(t *testing.T) {
	g, d, clock, p := newTest(t, 320, 240)
	p.Tap(hal.PadA, 500*time.Millisecond) // play
	var brick uint8
	clock.At(700*time.Millisecond, func() {
		brick = g.brick
		if g.brickX != brickX0 {
			t.Errorf("at the start: x %d, want %d", g.brickX, brickX0)
		}
	})
	p.Tap(hal.PadLeft, 800*time.Millisecond)
	clock.At(900*time.Millisecond, func() {
		if g.brickX != brickX0-1 {
			t.Errorf("after LEFT: x %d, want %d", g.brickX, brickX0-1)
		}
	})
	p.Hold(hal.PadRight, 1000*time.Millisecond, 1000*time.Millisecond) // repeats
	clock.At(2100*time.Millisecond, func() {
		if g.brickX <= brickX0+1 {
			t.Errorf("after holding RIGHT: x %d, want more than %d", g.brickX, brickX0+1)
		}
		checkField(t, g, d, false)
	})
	p.Tap(hal.PadUp, 2200*time.Millisecond)
	clock.At(2300*time.Millisecond, func() {
		if brick != 1 && g.brick != brickRot[brick] { // 1 is the square
			t.Errorf("after UP: brick %d, want %d", g.brick, brickRot[brick])
		}
	})
	p.Tap(hal.PadDown, 2400*time.Millisecond)
	clock.At(3000*time.Millisecond, func() {
		// landed on the bottom: a stone in the last row, the next brick at
		// the top
		landed := false
		for x := 8; x < 18; x++ {
			landed = landed || g.board[x+(mapH-2)*mapW] != tileEmpty
		}
		if !landed || g.brickY > 1 || g.stat[brick] != 1 {
			t.Errorf("after DOWN: landed %v, y %d, statistics %v", landed, g.brickY, g.stat)
		}
		if g.score == 0 {
			t.Error("after DOWN: no score")
		}
		checkField(t, g, d, false)
	})
	quit(p, 3500*time.Millisecond)
	g.Run()
}

func TestDemo(t *testing.T) {
	g, d, clock, p := newTest(t, 320, 240)
	// No button: the pause times out after 10 seconds and the demo plays.
	clock.At(120*sec, func() {
		if !g.demo {
			t.Fatal("no demo after 120 seconds")
		}
		if g.score == 0 || g.lines == 0 {
			t.Errorf("demo: score %d, lines %d", g.score, g.lines)
		}
		checkField(t, g, d, false)
	})
	quit(p, 121*sec)
	g.Run()
	if g.scoreBest != 0 {
		t.Errorf("the demo set the best score: %d", g.scoreBest)
	}
}

func TestDelRows(t *testing.T) {
	g, d, _, _ := newTest(t, 320, 240)
	g.newGame()
	// rows 17 and 18 full, a stone above them
	for x := 8; x < 18; x++ {
		g.putTile(x, 17, 3)
		g.putTile(x, 18, 4)
	}
	g.putTile(9, 16, 5)
	g.nextLevel = 2
	g.delAllRows()
	if g.lines != 2 || g.level != 2 || g.nextLevel != nextLevelTab[1] {
		t.Errorf("lines %d, level %d, next level %d", g.lines, g.level, g.nextLevel)
	}
	for x := 8; x < 18; x++ {
		want := uint8(tileEmpty)
		if x == 9 {
			want = 5
		}
		if got := g.board[x+18*mapW]; got != want {
			t.Errorf("bottom row, x %d: tile %d, want %d", x, got, want)
		}
		if got := g.board[x+17*mapW]; got != tileEmpty {
			t.Errorf("row 17, x %d: tile %d, want empty", x, got)
		}
	}
	checkField(t, g, d, false) // the LEVEL UP message is gone
}

func TestGameOver(t *testing.T) {
	g, _, clock, p := newTest(t, 320, 240)
	g.newGame()
	// the field full but for one place in each row
	for y := 1; y < mapH-1; y++ {
		for x := 8; x < 18; x++ {
			if x != 8+y%10 {
				g.board[x+y*mapW] = 2
			}
		}
	}
	g.score, g.lines = 123, 4
	p.Tap(hal.PadB, clock.T+3*sec) // after the blinking
	if g.checkEnd() {
		t.Fatal("checkEnd: the game goes on")
	}
	if g.scoreBest != 123 || g.linesBest != 4 {
		t.Errorf("best score %d, lines %d", g.scoreBest, g.linesBest)
	}
	if clock.T < 3*sec {
		t.Errorf("checkEnd returned at %v, before the key", clock.T)
	}
}

// TestOneGame: with OneGame, Run returns after the game over (and a key)
// and on Y in a game, without the demo.
func TestOneGame(t *testing.T) {
	g, _, clock, p := newTest(t, 320, 240)
	g.OneGame = true
	p.Tap(hal.PadA, sec) // the end of the pause
	clock.At(2*sec, func() {
		// the field full but for one place in each row: the next brick
		// does not fit
		for y := 1; y < mapH-1; y++ {
			for x := 8; x < 18; x++ {
				if x != 8+y%10 {
					g.board[x+y*mapW] = 2
				}
			}
		}
	})
	p.Tap(hal.PadB, 8*sec) // after GAME OVER
	g.Run()
	if g.demo || clock.T < 8*sec || clock.T > 9*sec {
		t.Errorf("Run returned at %v (demo %v)", clock.T, g.demo)
	}

	g, _, clock, p = newTest(t, 320, 240)
	g.OneGame = true
	p.Tap(hal.PadA, sec)
	p.Tap(hal.PadY, 3*sec)
	g.Run()
	if g.demo || clock.T > 4*sec {
		t.Errorf("Y: Run returned at %v (demo %v)", clock.T, g.demo)
	}
}
