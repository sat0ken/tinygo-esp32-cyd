package main

import (
	"image/color"
	"io/fs"
	"testing"
	"time"

	"github.com/sat0ken/tinygo-cyd/board"
	"github.com/sat0ken/tinygo-cyd/internal/goldentest"
	"github.com/sat0ken/tinygo-cyd/memlcd"
)

// fakeClock replaces time.Now and time.Sleep so tests run instantly.
type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time          { return c.t }
func (c *fakeClock) sleep(d time.Duration)   { c.t = c.t.Add(d) }
func (c *fakeClock) advance(d time.Duration) { c.t = c.t.Add(d) }

var red = color.RGBA{255, 0, 0, 255}

func newTestShow(t *testing.T) (*Show, *memlcd.Display, *memlcd.Touch, *fakeClock) {
	t.Helper()
	d := memlcd.New(board.Width, board.Height)
	touch := &memlcd.Touch{}
	s, err := NewShow(d, touch, slidesFS, "slides")
	if err != nil {
		t.Fatal(err)
	}
	clk := &fakeClock{t: time.Unix(0, 0)}
	s.now, s.sleep = clk.now, clk.sleep
	s.log = func(string) {}
	return s, d, touch, clk
}

// wantSlide checks that the whole screen equals slide i's file.
func wantSlide(t *testing.T, d *memlcd.Display, i int) {
	t.Helper()
	entries, _ := fs.ReadDir(slidesFS, "slides")
	data, err := fs.ReadFile(slidesFS, "slides/"+entries[i].Name())
	if err != nil {
		t.Fatal(err)
	}
	for p, v := range d.Pix {
		if want := uint16(data[p*2])<<8 | uint16(data[p*2+1]); v != want {
			t.Fatalf("slide %d: pixel (%d,%d) = %04x, want %04x", i, p%board.Width, p/board.Width, v, want)
		}
	}
}

func TestEverySlideEveryTransition(t *testing.T) {
	s, d, _, _ := newTestShow(t)
	if s.Len() != 4 {
		t.Fatalf("%d slides", s.Len())
	}
	for i := 0; i < s.Len(); i++ {
		for tr := Transition(0); tr < numTransitions; tr++ {
			d.FillScreen(red) // make sure every pixel is overwritten
			if err := s.Draw(i, tr); err != nil {
				t.Fatal(err)
			}
			wantSlide(t, d, i)
		}
	}
}

func TestTransitionDuration(t *testing.T) {
	s, _, _, clk := newTestShow(t)
	for tr := WipeDown; tr < numTransitions; tr++ {
		start := clk.now()
		s.Draw(0, tr)
		if got := clk.now().Sub(start); got < s.Duration*9/10 || got > s.Duration {
			t.Errorf("%v took %v, want ~%v", tr, got, s.Duration)
		}
	}
}

func TestCaptionRestore(t *testing.T) {
	s, d, _, clk := newTestShow(t)
	s.Start()
	goldentest.Check(t, "slideshow_caption", d.Image())
	// The caption goes away after 2 seconds and the slide is intact again.
	clk.advance(captionTime)
	s.Step()
	wantSlide(t, d, 0)
}

func TestAutoAdvanceAndTouch(t *testing.T) {
	s, d, touch, clk := newTestShow(t)
	s.Start()
	tap := func(x int16) {
		touch.X, touch.Y, touch.Pressed = x, 136, true
		s.Step()
		s.Step() // holding does not repeat
		touch.Pressed = false
		s.Step()
	}

	clk.advance(s.Interval)
	s.Step()
	if s.Current() != 1 {
		t.Fatalf("auto advance: slide %d", s.Current())
	}

	tap(470) // right third: next
	if s.Current() != 2 {
		t.Fatalf("next: slide %d", s.Current())
	}
	tap(10) // left third: previous
	tap(10)
	tap(10) // wraps around to the last slide
	if s.Current() != 3 {
		t.Fatalf("previous: slide %d", s.Current())
	}

	tap(240) // middle: pause
	if !s.Paused() {
		t.Fatal("not paused")
	}
	clk.advance(3 * s.Interval)
	s.Step()
	if s.Current() != 3 {
		t.Fatalf("advanced while paused: slide %d", s.Current())
	}
	tap(240) // resume
	if s.Paused() {
		t.Fatal("still paused")
	}
	clk.advance(captionTime)
	s.Step()
	wantSlide(t, d, 3) // caption removed, nothing else changed
	clk.advance(s.Interval)
	s.Step()
	if s.Current() != 0 {
		t.Fatalf("resume: slide %d", s.Current())
	}
}
