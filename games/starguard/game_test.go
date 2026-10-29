package starguard

import (
	"testing"

	"github.com/sat0ken/tinygo-cyd/board"
	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/internal/goldentest"
	"github.com/sat0ken/tinygo-cyd/memlcd"
	"github.com/sat0ken/tinygo-cyd/pad"
)

const dt = float32(1.0 / 30)

// sizes are the screens the game has a layout for; golden is the prefix of
// the golden images.
var sizes = []struct {
	w, h   int16
	golden string
}{
	{board.Width, board.Height, "starguard_"}, // ESP32-4827S043, ESP32-3248S035C
	{320, 240, "starguard_small_"},            // ESP32-2432S028R
}

func newTestGame(w, h int16) (*Game, *memlcd.Display, *pad.Fake, *memlcd.Touch) {
	d := memlcd.New(w, h)
	touch := &memlcd.Touch{}
	g := NewGame(d, touch, 7)
	f := &pad.Fake{}
	g.Pad = f
	g.DrawMenu()
	g.Update(dt)
	return g, d, f, touch
}

// press presses b for one frame and lets go.
func press(g *Game, f *pad.Fake, b hal.Buttons) {
	f.B = b
	g.Update(dt)
	f.B = 0
	g.Update(dt)
}

// play starts a game at the level in focus and waits for "STAGE 1".
func play(t *testing.T, g *Game, f *pad.Fake) {
	t.Helper()
	press(g, f, hal.PadA)
	if g.screen != intro {
		t.Fatalf("screen %d after A on the title", g.screen)
	}
	wait(g, introTime)
	if g.screen != playing {
		t.Fatalf("screen %d after the intro", g.screen)
	}
}

func wait(g *Game, sec float32) {
	for i := 0; i < int(sec*30)+2; i++ {
		g.Update(dt)
	}
}

// only leaves enemy e alone in the field, with no new ones coming.
func only(g *Game, e enemy) {
	g.enemies = [maxEnemies]enemy{e}
	g.spawnT = -100
}

// aimAt puts the crosshair on enemy i.
func aimAt(g *Game, i int) {
	e := &g.enemies[i]
	g.aimX, g.aimY, _ = g.project(e.x, e.y, e.z, 0)
}

func TestHit(t *testing.T) {
	for _, s := range sizes {
		g, d, f, _ := newTestGame(s.w, s.h)
		play(t, g, f)
		only(g, enemy{alive: true, kind: rock, z: 400, z0: 400, vz: 1})
		g.Update(dt)
		cx, cy := g.w/2, g.l.viewH/2
		if d.Pixel(cx+5, cy+5) == 0 {
			t.Fatalf("%dx%d: nothing drawn where the rock is", s.w, s.h)
		}
		if d.Pixel(cx, cy-6) != rgbOf(g.lockColor) {
			t.Fatalf("%dx%d: the crosshair is not red over the rock", s.w, s.h)
		}
		f.B = hal.PadA
		g.Update(dt)
		if e := &g.enemies[0]; e.alive || e.boom <= 0 || g.score != 10 || g.killed != 1 {
			t.Fatalf("%dx%d: alive %v boom %v score %d killed %d", s.w, s.h, e.alive, e.boom, g.score, g.killed)
		}
	}
}

func TestAim(t *testing.T) {
	g, _, f, touch := newTestGame(320, 240)
	play(t, g, f)
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

func TestCapsule(t *testing.T) {
	g, _, f, _ := newTestGame(320, 240)
	play(t, g, f)
	g.shield = 50
	only(g, enemy{alive: true, kind: capsule, z: 300, z0: 300, vz: 1})
	aimAt(g, 0)
	press(g, f, hal.PadA)
	if g.shield != 50+capsuleBoost || g.killed != 0 {
		t.Fatalf("shield %d killed %d after shooting a capsule", g.shield, g.killed)
	}
	// A capsule that gets through does no harm.
	only(g, enemy{alive: true, kind: capsule, z: nearZ + 1, z0: 300, vz: 100})
	wait(g, 0.2)
	if g.shield != 50+capsuleBoost {
		t.Fatalf("shield %d after a capsule got through", g.shield)
	}
}

// TestStages destroys the enemies of each stage, then the mothership.
func TestStages(t *testing.T) {
	for _, s := range sizes {
		g, d, f, _ := newTestGame(s.w, s.h)
		play(t, g, f)
		for n := 0; n < bossStage; n++ {
			if g.stage != n || g.screen != playing {
				t.Fatalf("%dx%d: stage %d screen %d, want stage %d", s.w, s.h, g.stage, g.screen, n)
			}
			g.killed = stages[n].count - 1
			only(g, enemy{alive: true, kind: robot, z: 500, z0: 500, vz: 1})
			aimAt(g, 0)
			press(g, f, hal.PadA)
			if g.screen != cleared {
				t.Fatalf("%dx%d: stage %d not cleared", s.w, s.h, n)
			}
			wait(g, clearedTime+introTime)
		}
		if g.stage != bossStage || !g.boss.active || g.screen != playing {
			t.Fatalf("%dx%d: no mothership", s.w, s.h)
		}
		// Let it come in and launch a drone or two.
		wait(g, 5.5)
		if g.boss.z > bossStop+1 {
			t.Fatalf("%dx%d: the mothership stopped at %v", s.w, s.h, g.boss.z)
		}
		drones := 0
		for i := range g.enemies {
			if g.enemies[i].alive && g.enemies[i].kind == drone {
				drones++
			}
		}
		if drones == 0 {
			t.Fatalf("%dx%d: no drones", s.w, s.h)
		}
		if s.golden == "starguard_small_" {
			goldentest.Check(t, s.golden+"boss", d.Image())
		}
		g.shield = shield0 // the drones are not the point here
		f.B = hal.PadA
		for i := 0; i < 60*30 && g.boss.hp > 0; i++ {
			b := &g.boss
			g.aimX, g.aimY, _ = g.project(b.x, b.y, b.z, 0)
			g.shield = shield0
			g.Update(dt)
		}
		f.B = 0
		if g.boss.hp > 0 {
			t.Fatalf("%dx%d: the mothership has %d hit points left", s.w, s.h, g.boss.hp)
		}
		wait(g, bossDying+0.2)
		if g.screen != results || !g.complete {
			t.Fatalf("%dx%d: screen %d complete %v after the mothership", s.w, s.h, g.screen, g.complete)
		}
		g.score = 4321
		g.drawGauges()
		g.drawResults()
		goldentest.Check(t, s.golden+"complete", d.Image())
		press(g, f, hal.PadRight) // MENU
		press(g, f, hal.PadA)
		if g.screen != menuScreen {
			t.Fatalf("%dx%d: not back on the title", s.w, s.h)
		}
	}
}

// TestShieldAndRestart lets everything through: the game is over, AGAIN
// starts again.
func TestShieldAndRestart(t *testing.T) {
	g, _, f, _ := newTestGame(320, 240)
	play(t, g, f)
	for i := 0; i < 180*30 && g.screen == playing; i++ {
		g.Update(dt)
	}
	if g.screen != results || g.complete || g.shield != 0 {
		t.Fatalf("screen %d complete %v shield %d after three minutes", g.screen, g.complete, g.shield)
	}
	press(g, f, hal.PadA) // AGAIN
	if g.screen != intro || g.shield != shield0 || g.score != 0 {
		t.Fatal("AGAIN did not start again")
	}
}

func TestQuit(t *testing.T) {
	g, _, f, touch := newTestGame(320, 240)
	play(t, g, f)
	press(g, f, hal.PadY)
	if g.screen != menuScreen {
		t.Fatal("Y did not give up")
	}
	play(t, g, f)
	r := g.l.radar
	touch.X, touch.Y, touch.Pressed = r.x+5, r.y+5, true
	g.Update(dt)
	touch.Pressed = false
	if g.screen != menuScreen {
		t.Fatal("tapping the radar did not give up")
	}
	left := false
	g.OnExit = func() { left = true }
	g.DrawMenu()
	press(g, f, hal.PadB)
	if !left {
		t.Fatal("B on the title did not call OnExit")
	}
}

func TestScreens(t *testing.T) {
	for _, s := range sizes {
		g, d, f, _ := newTestGame(s.w, s.h)
		g.OnExit = func() {}
		g.Pad = nil // the touch hint
		g.DrawMenu()
		goldentest.Check(t, s.golden+"menu", d.Image())
		g.Pad = f
		play(t, g, f)
		wait(g, 6)
		// Fire at the nearest enemy, for the lasers and an explosion.
		best := -1
		for i := range g.enemies {
			if e := &g.enemies[i]; e.alive && (best < 0 || e.z < g.enemies[best].z) {
				best = i
			}
		}
		if best < 0 {
			t.Fatalf("%dx%d: no enemies after six seconds", s.w, s.h)
		}
		aimAt(g, best)
		f.B = hal.PadA
		g.Update(dt)
		g.Update(dt)
		goldentest.Check(t, s.golden+"play", d.Image())
	}
}

func rgbOf(c uint16) uint16 { return bigEndian(c) }

// TestNoAllocs checks that a frame of play allocates nothing (while the
// score and the stage stay the same). It counts with go test: under
// tinygo test (0.44) AllocsPerRun always returns 0, so for TinyGo check
// with -print-allocs=starguard as well.
func TestNoAllocs(t *testing.T) {
	g, _, f, _ := newTestGame(320, 240)
	play(t, g, f)
	f.B = hal.PadRight // move the crosshair, without firing
	if n := testing.AllocsPerRun(60, func() { g.Update(dt) }); n != 0 {
		t.Errorf("%.1f allocations per frame", n)
	}
}
