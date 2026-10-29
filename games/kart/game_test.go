package kart

import (
	"testing"

	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/memlcd"
	"github.com/sat0ken/tinygo-cyd/pad"
)

const dt = float32(1.0 / 30)

func newTestGame(w, h int16) (*Game, *memlcd.Display, *pad.Fake) {
	d := memlcd.New(w, h)
	g := NewGame(d, &memlcd.Touch{})
	f := &pad.Fake{}
	g.Pad = f
	g.DrawAll()
	g.Update(dt)
	return g, d, f
}

func TestTrack(t *testing.T) {
	tr := newTrack()
	for i, p := range tr.path {
		if k := tr.at(p[0]*cellSize, p[1]*cellSize); k != road && k != startLine {
			t.Fatalf("point %d of the middle line (%v) is %d, not road", i, p, k)
		}
	}
	if k := tr.at(tr.path[0][0]*cellSize, tr.path[0][1]*cellSize); k != startLine {
		t.Fatalf("the first point is %d, not the start line", k)
	}
	if tr.at(4, 4) != grass || tr.at(-10, 5000) != grass {
		t.Fatal("the corner is not grass")
	}
}

func TestDrive(t *testing.T) {
	for _, s := range [][2]int16{{320, 240}, {480, 272}} {
		g, _, f := newTestGame(s[0], s[1])
		f.B = hal.PadA
		for i := 0; i < 30; i++ {
			g.Update(dt)
		}
		if g.speed < 40 || g.track.at(g.x, g.y) == grass {
			t.Fatalf("%v: after a second of gas: speed %v, on %d", s, g.speed, g.track.at(g.x, g.y))
		}
		// Full lock to the left for a while: off the road, slow.
		f.B = hal.PadA | hal.PadLeft
		for i := 0; i < 30; i++ {
			g.Update(dt)
		}
		f.B = hal.PadA
		for i := 0; i < 60; i++ {
			g.Update(dt)
		}
		if g.track.at(g.x, g.y) == grass && g.speed > grassSpeed+1 {
			t.Fatalf("%v: on the grass at speed %v", s, g.speed)
		}
		f.B = hal.PadB
		for i := 0; i < 30; i++ {
			g.Update(dt)
		}
		if g.speed != 0 {
			t.Fatalf("%v: the brakes did not stop the kart: %v", s, g.speed)
		}
	}
}
