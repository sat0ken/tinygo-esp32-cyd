package kart

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
	{board.Width, board.Height, "kart_"}, // ESP32-4827S043, ESP32-3248S035C
	{320, 240, "kart_small_"},            // ESP32-2432S028R
}

func newTestGame(w, h int16) (*Game, *memlcd.Display, *pad.Fake) {
	d := memlcd.New(w, h)
	g := NewGame(d, &memlcd.Touch{})
	f := &pad.Fake{}
	g.Pad = f
	g.DrawMenu()
	g.Update(dt)
	return g, d, f
}

// press presses b for one frame and lets go.
func press(g *Game, f *pad.Fake, b hal.Buttons) {
	f.B = b
	g.Update(dt)
	f.B = 0
	g.Update(dt)
}

// startRace starts a race at the level in focus and waits for the
// countdown.
func startRace(t *testing.T, g *Game, f *pad.Fake) {
	t.Helper()
	press(g, f, hal.PadA)
	if g.screen != countdown {
		t.Fatalf("screen %d after A on the menu", g.screen)
	}
	for i := 0; i < 4*30 && g.screen == countdown; i++ {
		g.Update(dt)
	}
	if g.screen != racing {
		t.Fatalf("screen %d after the countdown", g.screen)
	}
}

func TestTrack(t *testing.T) {
	tr := newTrack()
	for i, p := range tr.path {
		if k := tr.at(p[0]*cellSize, p[1]*cellSize); k == grass || k == curb {
			t.Fatalf("point %d of the middle line (%v) is %d, not road", i, p, k)
		}
	}
	if k := tr.at(tr.path[0][0]*cellSize, tr.path[0][1]*cellSize); k != startLine {
		t.Fatalf("the first point is %d, not the start line", k)
	}
	for _, f := range boostAt {
		p := tr.path[int(f*float32(len(tr.path)))]
		if k := tr.at(p[0]*cellSize, p[1]*cellSize); k != boost {
			t.Fatalf("no boost pad at %v of the course", f)
		}
	}
	if tr.at(4, 4) != grass || tr.at(-10, 5000) != grass {
		t.Fatal("the corner is not grass")
	}
}

// TestCountdown: nobody moves before GO, not even with the gas held.
func TestCountdown(t *testing.T) {
	g, _, f := newTestGame(320, 240)
	press(g, f, hal.PadA)
	x, y := g.player().x, g.player().y
	f.B = hal.PadA
	for i := 0; i < 2*30; i++ {
		g.Update(dt)
	}
	if g.screen != countdown || g.player().x != x || g.player().y != y || g.karts[1].s != -2 {
		t.Fatal("something moved during the countdown")
	}
}

func TestDrive(t *testing.T) {
	for _, s := range sizes {
		g, _, f := newTestGame(s.w, s.h)
		startRace(t, g, f)
		p := g.player()
		f.B = hal.PadA
		for i := 0; i < 30; i++ {
			g.Update(dt)
		}
		if p.speed < 40 {
			t.Fatalf("%dx%d: after a second of gas: speed %v", s.w, s.h, p.speed)
		}
		f.B = hal.PadA | hal.PadLeft
		for i := 0; i < 45; i++ {
			g.Update(dt)
		}
		f.B = hal.PadA
		for i := 0; i < 60; i++ {
			g.Update(dt)
		}
		if g.track.at(p.x, p.y) == grass && p.speed > grassSpeed+1 {
			t.Fatalf("%dx%d: on the grass at speed %v", s.w, s.h, p.speed)
		}
		f.B = hal.PadB
		for i := 0; i < 45; i++ {
			g.Update(dt)
		}
		if p.speed != 0 {
			t.Fatalf("%dx%d: the brakes did not stop the kart: %v", s.w, s.h, p.speed)
		}
	}
}

// TestCPUs: alone, the CPU karts go round and stay on the road.
func TestCPUs(t *testing.T) {
	g, _, f := newTestGame(320, 240)
	startRace(t, g, f)
	n := float32(len(g.track.path))
	for i := 0; i < 60*30; i++ {
		g.Update(dt)
		for k := 1; k < len(g.karts); k++ {
			if c := &g.karts[k]; g.track.at(c.x, c.y) == grass {
				t.Fatalf("CPU %d on the grass at %v", k, c.s)
			}
		}
	}
	for k := 1; k < len(g.karts); k++ {
		if s := g.karts[k].s; s < n {
			t.Errorf("CPU %d: only %v of %v points in a minute", k, s, n)
		}
	}
	if g.placeNow() != 4 {
		t.Errorf("standing still, the player is %d", g.placeNow())
	}
}

// TestLaps moves the player round the course point by point: the laps
// count, going backwards over the line does not, and after three laps the
// results show.
func TestLaps(t *testing.T) {
	for _, s := range sizes {
		g, d, f := newTestGame(s.w, s.h)
		startRace(t, g, f)
		p := g.player()
		n := len(g.track.path)
		at := func(i int) {
			p.x, p.y = g.track.point(float32(i), 0)
			g.Update(dt)
		}
		// Back and forth over the line: no lap.
		for i := -8; i < 5; i++ {
			at(i)
		}
		for i := 5; i > -8; i-- {
			at(i)
		}
		if p.lap != -1 {
			t.Fatalf("%dx%d: lap %d after backing over the line", s.w, s.h, p.lap)
		}
		for i := -8; g.screen == racing && i < 3*n+20; i += 2 {
			at(i)
			if i == n/2 && p.lap != 0 {
				t.Fatalf("%dx%d: lap %d half way round the first lap", s.w, s.h, p.lap)
			}
		}
		if g.screen != finished || p.lap != laps {
			t.Fatalf("%dx%d: screen %d lap %d after three laps", s.w, s.h, g.screen, p.lap)
		}
		if g.bestLap <= 0 || g.place != 1 {
			t.Fatalf("%dx%d: best lap %v place %d", s.w, s.h, g.bestLap, g.place)
		}
		g.raceTime, g.bestLap = 95.4, 28.7 // repeatable
		g.shown = [4]string{}
		g.drawHUD()
		g.drawResults()
		goldentest.Check(t, s.golden+"results", d.Image())

		press(g, f, hal.PadRight) // MENU
		press(g, f, hal.PadA)
		if g.screen != menuScreen {
			t.Fatalf("%dx%d: not back on the menu", s.w, s.h)
		}
	}
}

func TestBoost(t *testing.T) {
	g, _, f := newTestGame(320, 240)
	startRace(t, g, f)
	p := g.player()
	i := int(boostAt[0] * float32(len(g.track.path)))
	p.x, p.y = g.track.point(float32(i-6), 0)
	dx, dy := g.track.dir(float32(i - 6))
	p.angle = atan2(dy, dx)
	p.speed = maxSpeed
	f.B = hal.PadA
	top := float32(0)
	for k := 0; k < 30; k++ {
		g.Update(dt)
		if p.speed > top {
			top = p.speed
		}
	}
	if top <= maxSpeed+10 {
		t.Fatalf("top speed %v over the boost pad", top)
	}
}

func TestBump(t *testing.T) {
	g, _, f := newTestGame(320, 240)
	startRace(t, g, f)
	p, c := g.player(), &g.karts[1]
	p.x, p.y = c.x+2, c.y+1
	g.bump()
	dx, dy := p.x-c.x, p.y-c.y
	if d := sqrt(dx*dx + dy*dy); d < bumpDist-0.01 {
		t.Fatalf("karts %v apart after the bump", d)
	}
}

func TestScreens(t *testing.T) {
	for _, s := range sizes {
		g, d, f := newTestGame(s.w, s.h)
		g.OnExit = func() {}
		g.Pad = nil // the touch hint
		g.DrawMenu()
		goldentest.Check(t, s.golden+"menu", d.Image())
		g.Pad = f
		press(g, f, hal.PadRight) // shows the focus, HARD
		press(g, f, hal.PadLeft)  // NORMAL
		startRace(t, g, f)
		f.B = hal.PadA
		for i := 0; i < 45; i++ {
			g.Update(dt)
		}
		goldentest.Check(t, s.golden+"race", d.Image())
	}
}

func TestMenuBack(t *testing.T) {
	g, _, f := newTestGame(320, 240)
	left := false
	g.OnExit = func() { left = true }
	g.DrawMenu()
	press(g, f, hal.PadB)
	if !left {
		t.Fatal("B on the menu did not call OnExit")
	}
}
