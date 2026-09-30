package main

import (
	"testing"

	"github.com/sat0ken/tinygo-cyd/board"
	"github.com/sat0ken/tinygo-cyd/internal/goldentest"
	"github.com/sat0ken/tinygo-cyd/memlcd"
)

const dt = float32(1.0 / 60)

func newTestGame() (*Game, *memlcd.Display, *memlcd.Touch) {
	d := memlcd.New(board.Width, board.Height)
	touch := &memlcd.Touch{}
	g := NewGame(d, touch)
	g.DrawAll()
	return g, d, touch
}

// tap releases the panel (a tap must be a new press), then taps at x.
func tap(g *Game, touch *memlcd.Touch, x int16) {
	touch.Pressed = false
	g.Step(dt)
	touch.X, touch.Y, touch.Pressed = x, 200, true
	g.Step(dt)
	touch.Pressed = false
	g.Step(dt)
}

// autoplay keeps the paddle under the ball, hitting it off-centre so the
// ball does not bounce straight up forever.
func autoplay(g *Game, touch *memlcd.Touch, frames int) {
	for i := 0; i < frames; i++ {
		touch.X = int16(g.ballX) + ballSize/2 + 15
		touch.Y, touch.Pressed = 200, true
		g.Step(dt)
		g.Draw()
	}
}

func TestInitialScreen(t *testing.T) {
	_, d, _ := newTestGame()
	goldentest.Check(t, "breakout_initial", d.Image())
}

func TestAutoplayScoresWithoutMissing(t *testing.T) {
	g, d, touch := newTestGame()
	tap(g, touch, 240)
	if g.state != playing {
		t.Fatal("not playing after tap")
	}
	autoplay(g, touch, 60*60) // one minute
	if g.lives != lives0 {
		t.Fatalf("lost a life while the paddle follows the ball (lives %d)", g.lives)
	}
	if g.score == 0 || g.left == brickRows*brickCols {
		t.Fatalf("no brick broken: score %d, left %d", g.score, g.left)
	}
	t.Logf("after 60s: score %d, bricks left %d, state %d", g.score, g.left, g.state)

	// The screen must match the game state: removed bricks are background,
	// remaining bricks keep their colour.
	for r := range g.bricks {
		for c := range g.bricks[r] {
			br := g.brickRect(r, c)
			got := d.Pixel(br.x+br.w/2, br.y+br.h/2)
			want := memlcdColor(bgColor)
			if g.bricks[r][c] {
				want = memlcdColor(rowColors[r])
			}
			if got != want {
				t.Fatalf("brick %d,%d: pixel %04x want %04x", r, c, got, want)
			}
		}
	}
}

func TestBallNeverInsideABrick(t *testing.T) {
	g, _, touch := newTestGame()
	g.level = 5 // faster ball
	tap(g, touch, 240)
	for i := 0; i < 60*30; i++ {
		touch.X = int16(g.ballX) + ballSize/2 + 15
		touch.Pressed = true
		g.Step(dt)
		for r := range g.bricks {
			for c := range g.bricks[r] {
				if g.bricks[r][c] && overlap(g.ballX, g.ballY, ballSize, ballSize, g.brickRect(r, c)) {
					t.Fatalf("frame %d: ball inside brick %d,%d", i, r, c)
				}
			}
		}
	}
}

func TestMissLosesLifeAndGameOver(t *testing.T) {
	g, _, touch := newTestGame()
	for life := lives0; life > 0; life-- {
		tap(g, touch, 240)
		// Keep the paddle in the far left corner, away from the ball.
		for i := 0; i < 60*20 && g.state == playing; i++ {
			touch.X, touch.Pressed = 0, true
			g.Step(dt)
			touch.Pressed = false
			g.Step(dt)
			if g.ballX < 100 && g.vy > 0 && g.ballY > 200 {
				// Avoid a lucky hit: move away from the ball.
				touch.X, touch.Pressed = 479, true
				g.Step(dt)
				touch.Pressed = false
			}
		}
		if g.lives != life-1 {
			t.Fatalf("lives %d, want %d", g.lives, life-1)
		}
	}
	if g.state != gameOver {
		t.Fatalf("state %d, want game over", g.state)
	}
	tap(g, touch, 240)
	if g.state != ready || g.lives != lives0 || g.score != 0 || g.left != brickRows*brickCols {
		t.Fatalf("restart: state %d lives %d score %d left %d", g.state, g.lives, g.score, g.left)
	}
}

func TestClearAndNextLevel(t *testing.T) {
	g, d, touch := newTestGame()
	// Leave one brick in the bottom row, right above the paddle's centre.
	for r := range g.bricks {
		for c := range g.bricks[r] {
			g.bricks[r][c] = r == brickRows-1 && c == 5
		}
	}
	g.left = 1
	g.redrawBricks()
	tap(g, touch, 240)
	autoplay(g, touch, 60*30)
	if g.state != cleared {
		t.Fatalf("state %d, want cleared (left %d)", g.state, g.left)
	}
	tap(g, touch, 240)
	if g.level != 2 || g.left != brickRows*brickCols || g.state != ready {
		t.Fatalf("next level: level %d left %d state %d", g.level, g.left, g.state)
	}
	if g.speed() <= baseSpeed {
		t.Fatal("level 2 is not faster")
	}
	g.Draw()
	br := g.brickRect(0, 0)
	if d.Pixel(br.x+1, br.y+1) != memlcdColor(rowColors[0]) {
		t.Fatal("bricks not redrawn for the new level")
	}
}

func memlcdColor(c interface{ RGBA() (r, g, b, a uint32) }) uint16 {
	r, g, b, _ := c.RGBA()
	return uint16(r>>8&0xF8)<<8 | uint16(g>>8&0xFC)<<3 | uint16(b>>8)>>3
}
