package starguard

import (
	"testing"

	"github.com/sat0ken/tinygo-cyd/board"
	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/memlcd"
	"github.com/sat0ken/tinygo-cyd/pad"
)

const dt = float32(1.0 / 30)

var sizes = [][2]int16{{board.Width, board.Height}, {320, 240}}

func newTestGame(w, h int16) (*Game, *memlcd.Display, *pad.Fake, *memlcd.Touch) {
	d := memlcd.New(w, h)
	touch := &memlcd.Touch{}
	g := NewGame(d, touch, 7)
	f := &pad.Fake{}
	g.Pad = f
	g.DrawAll()
	g.Update(dt)
	return g, d, f, touch
}

// TestHit puts a rock in the middle of the view and fires at it.
func TestHit(t *testing.T) {
	for _, s := range sizes {
		g, d, f, _ := newTestGame(s[0], s[1])
		g.enemies = [maxEnemies]enemy{{alive: true, kind: rock, z: 400, vz: 1}}
		g.spawnT = -100 // no new ones
		g.Update(dt)
		// The rock is drawn in the middle of the view, the crosshair red.
		cx, cy := g.w/2, g.l.viewH/2
		if d.Pixel(cx+5, cy+5) == 0 {
			t.Fatalf("%v: nothing drawn where the rock is", s)
		}
		if d.Pixel(cx, cy-6) != rgbOf(g.lockColor) {
			t.Fatalf("%v: the crosshair is not red over the rock", s)
		}
		f.B = hal.PadA
		g.Update(dt)
		if e := &g.enemies[0]; e.alive || e.boom <= 0 || g.score != 10 {
			t.Fatalf("%v: alive %v boom %v score %d", s, e.alive, e.boom, g.score)
		}
	}
}

// TestAim moves the crosshair with the stick and with a touch.
func TestAim(t *testing.T) {
	g, _, f, touch := newTestGame(320, 240)
	x0 := g.aimX
	f.B = hal.PadRight
	for i := 0; i < 15; i++ {
		g.Update(dt)
	}
	if moved := g.aimX - x0; moved < g.l.aimSpeed/2-5 || moved > g.l.aimSpeed/2+5 {
		t.Fatalf("the crosshair moved %v in half a second", moved)
	}
	f.B = 0
	touch.X, touch.Y, touch.Pressed = 40, 30, true
	g.Update(dt)
	if g.aimX != 40 || g.aimY != 30 || g.laser <= 0 {
		t.Fatal("a touch does not aim and fire")
	}
}

// TestShieldAndRestart lets everything through: the shield runs out, the
// game is over, A starts again.
func TestShieldAndRestart(t *testing.T) {
	g, _, f, _ := newTestGame(320, 240)
	for i := 0; i < 120*30 && !g.over; i++ {
		g.Update(dt)
	}
	if !g.over || g.shield != 0 {
		t.Fatalf("over %v shield %d after two minutes", g.over, g.shield)
	}
	f.B = hal.PadA
	g.Update(dt)
	if g.over || g.shield != shield0 || g.score != 0 {
		t.Fatal("A did not start again")
	}
}

func rgbOf(c uint16) uint16 { return bigEndian(c) }
