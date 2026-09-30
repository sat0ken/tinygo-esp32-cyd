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
				x := v.x + invW/2
				d := x - (g.cannonX + cannonSprite.pw()/2)
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
			// First set bit of the sprite.
			for i, on := range sp.bits {
				if on {
					x := sr.x + int16(i%int(sp.w))*scale
					y := sr.y + int16(i/int(sp.w))*scale
					if got, want := d.Pixel(x, y), rgb565(kindColors[v.kind]); got != want {
						t.Fatalf("invader %d,%d at (%d,%d): pixel %04x want %04x", r, c, x, y, got, want)
					}
					break
				}
			}
		}
	}
}

func TestInitialScreen(t *testing.T) {
	_, d, _ := newTestGame()
	goldentest.Check(t, "invaders_initial", d.Image())
}

func TestPlay(t *testing.T) {
	g, d, touch := newTestGame()
	tap(g, touch, 240)
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
				if v.alive && (v.x < 0 || v.x+invW > g.w) {
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
	g, _, touch := newTestGame()
	tap(g, touch, 240)
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
		g.bombs[0] = shot{active: true, x: float32(g.cannonX + 12), y: float32(g.cannonY() - 10)}
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
	if g.inv[0][0].y != formY0+dropY {
		t.Fatalf("wave 2 starts at y %d, want %d", g.inv[0][0].y, formY0+dropY)
	}
	checkScreen(t, g, d)
}
