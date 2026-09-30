// Package slideshow shows full-screen RGB565 images one after another with
// transitions, a caption and touch control. Where the images come from is a
// Source: a slidepack in flash or embedded (examples/05_slideshow), or files
// on a microSD card (examples/06_sdslideshow).
package slideshow

import (
	"errors"
	"image/color"
	"io"
	"strconv"
	"time"

	"github.com/sat0ken/tinygo-cyd/hal"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/proggy"
)

// bitmapDrawer is implemented by every backend (rgblcd, wasmlcd, memlcd via
// framebuf.Buffer): it copies big endian RGB565 bytes to the screen.
type bitmapDrawer interface {
	DrawRGBBitmap8(x, y int16, data []uint8, w, h int16) error
}

// Transition is the effect used to show the next slide.
type Transition int

const (
	Cut            Transition = iota // draw at once
	WipeDown                         // top to bottom, 8 lines at a time
	WipeRight                        // left to right, 40 columns at a time
	Blinds                           // 8 bands opening line by line
	NumTransitions                   // number of transitions
)

func (t Transition) String() string {
	return [...]string{"cut", "wipe-down", "wipe-right", "blinds"}[t]
}

const (
	wipeLines   = 8
	wipeColumns = 40
	blindBands  = 8
)

// Show is a slide show of the full-screen RGB565 images of a slidepack.
type Show struct {
	d    hal.Display
	bd   bitmapDrawer
	t    hal.Touch
	w, h int16
	src  Source
	curR io.ReaderAt // pixels of the slide on screen

	cur         int
	next        int          // index into Transitions
	Transitions []Transition // used in turn; default: all

	paused   bool
	Interval time.Duration // time each slide is shown
	Duration time.Duration // duration of a transition

	shown     time.Time // when the current slide was completed
	caption   bool
	captionAt time.Time
	capW      int16
	wasTouch  bool

	buf   []byte              // one chunk: max(wipe lines, wipe columns)
	Sleep func(time.Duration) // replaceable for tests
	Now   func() time.Time    // replaceable for tests
	Log   func(msg string)    // serial / console log
}

// NewShow shows the slides of src, which must match the screen size.
func NewShow(d hal.Display, t hal.Touch, src Source) (*Show, error) {
	bd, ok := d.(bitmapDrawer)
	if !ok {
		return nil, errors.New("slideshow: display has no DrawRGBBitmap8")
	}
	if src.Len() == 0 {
		return nil, ErrNoSlides
	}
	w, h := d.Size()
	s := &Show{
		d: d, bd: bd, t: t, w: w, h: h, src: src,
		Transitions: []Transition{WipeDown, WipeRight, Blinds, Cut},
		Interval:    5 * time.Second,
		Duration:    400 * time.Millisecond,
		Sleep:       time.Sleep,
		Now:         time.Now,
		Log:         func(msg string) { println(msg) },
	}
	n := int(w) * wipeLines * 2
	if m := wipeColumns * int(h) * 2; m > n {
		n = m
	}
	s.buf = make([]byte, n)
	return s, nil
}

// ErrNoSlides is returned by NewShow for an empty Source.
var ErrNoSlides = errors.New("slideshow: no slides")

// Len returns the number of slides.
func (s *Show) Len() int { return s.src.Len() }

// Current returns the index of the slide on screen.
func (s *Show) Current() int { return s.cur }

// Paused reports whether automatic advance is paused.
func (s *Show) Paused() bool { return s.paused }

// rows draws lines y0..y0+n-1 of the slide in r at full width.
func (s *Show) rows(r io.ReaderAt, y0, n int16) error {
	stride := int64(s.w) * 2
	b := s.buf[:int(n)*int(stride)]
	if _, err := r.ReadAt(b, int64(y0)*stride); err != nil {
		return err
	}
	return s.bd.DrawRGBBitmap8(0, y0, b, s.w, n)
}

// rect draws the rectangle (x, y, w, h) of the slide in r. w*h*2 must fit s.buf.
func (s *Show) rect(r io.ReaderAt, x, y, w, h int16) error {
	stride := int64(s.w) * 2
	line := int(w) * 2
	for row := 0; row < int(h); row++ {
		off := int64(int(y)+row)*stride + int64(x)*2
		if _, err := r.ReadAt(s.buf[row*line:(row+1)*line], off); err != nil {
			return err
		}
	}
	return s.bd.DrawRGBBitmap8(x, y, s.buf[:line*int(h)], w, h)
}

// Draw shows slide i with transition tr. Each step of the transition is
// followed by a sleep so that the whole transition takes s.Duration.
func (s *Show) Draw(i int, tr Transition) error {
	r, err := s.src.Open(i)
	if err != nil {
		return err
	}
	switch tr {
	case Cut:
		for y := int16(0); y < s.h && err == nil; y += wipeLines {
			err = s.rows(r, y, min16(wipeLines, s.h-y))
		}
	case WipeDown:
		steps := (s.h + wipeLines - 1) / wipeLines
		for y := int16(0); y < s.h && err == nil; y += wipeLines {
			err = s.rows(r, y, min16(wipeLines, s.h-y))
			s.step(steps)
		}
	case WipeRight:
		steps := (s.w + wipeColumns - 1) / wipeColumns
		for x := int16(0); x < s.w && err == nil; x += wipeColumns {
			err = s.rect(r, x, 0, min16(wipeColumns, s.w-x), s.h)
			s.step(steps)
		}
	case Blinds:
		band := (s.h + blindBands - 1) / blindBands
		for k := int16(0); k < band && err == nil; k++ {
			for b := int16(0); b < blindBands && err == nil; b++ {
				if y := b*band + k; y < s.h {
					err = s.rows(r, y, 1)
				}
			}
			s.step(band)
		}
	}
	if err != nil {
		return err
	}
	s.cur = i
	s.curR = r
	s.caption = false
	s.d.Display()
	return nil
}

func (s *Show) step(steps int16) {
	s.d.Display()
	s.Sleep(s.Duration / time.Duration(steps))
}

func min16(a, b int16) int16 {
	if a < b {
		return a
	}
	return b
}

// Caption box at the bottom left.
const (
	capX, capH  = 4, 18
	CaptionTime = 2 * time.Second
)

func (s *Show) capY() int16 { return s.h - 4 - capH }

var (
	capBG = color.RGBA{0, 0, 0, 255}
	capFG = color.RGBA{255, 255, 255, 255}
)

// ShowCaption draws "n/total name" (and "PAUSE") over the slide. It is
// removed after 2 seconds by redrawing that part of the slide.
func (s *Show) ShowCaption() {
	// A previous caption may be wider (e.g. with "PAUSE"): restore it first.
	if err := s.HideCaption(); err != nil {
		s.Log("caption: " + err.Error())
	}
	text := strconv.Itoa(s.cur+1) + "/" + strconv.Itoa(s.src.Len()) + " " + s.src.Name(s.cur)
	if s.paused {
		text += "  PAUSE"
	}
	// Size the box to the text; (w-8)*capH*2 bytes always fits in s.buf.
	_, tw := tinyfont.LineWidth(&proggy.TinySZ8pt7b, text)
	s.capW = min16(int16(tw)+12, s.w-2*capX)
	s.d.FillRectangle(capX, s.capY(), s.capW, capH, capBG)
	tinyfont.WriteLine(s.d, &proggy.TinySZ8pt7b, capX+6, s.capY()+13, text, capFG)
	s.d.Display()
	s.caption = true
	s.captionAt = s.Now()
}

// HideCaption restores the slide under the caption.
func (s *Show) HideCaption() error {
	if !s.caption {
		return nil
	}
	s.caption = false
	if s.curR == nil {
		return nil
	}
	if err := s.rect(s.curR, capX, s.capY(), s.capW, capH); err != nil {
		return err
	}
	s.d.Display()
	return nil
}

// Next shows the following slide with the next transition in turn.
func (s *Show) Next(delta int) error {
	n := s.src.Len()
	i := ((s.cur+delta)%n + n) % n
	tr := Cut
	if len(s.Transitions) > 0 {
		tr = s.Transitions[s.next%len(s.Transitions)]
		s.next++
	}
	start := s.Now()
	if err := s.Draw(i, tr); err != nil {
		return err
	}
	s.Log("slide " + strconv.Itoa(i+1) + "/" + strconv.Itoa(n) + " " + s.src.Name(i) +
		" (" + tr.String() + ", " + strconv.Itoa(int(s.Now().Sub(start).Milliseconds())) + " ms)")
	s.ShowCaption()
	s.shown = s.Now()
	return nil
}

// Start draws the first slide.
func (s *Show) Start() error {
	if err := s.Draw(0, Cut); err != nil {
		return err
	}
	s.Log("slideshow: " + strconv.Itoa(s.src.Len()) + " slides")
	s.ShowCaption()
	s.shown = s.Now()
	return nil
}

// Step handles touch and the timers. Call it every few milliseconds.
//
// Touch (on release of a new press): left third = previous slide, right
// third = next slide, middle = pause / resume.
func (s *Show) Step() error {
	x, _, pressed := s.t.ReadTouch()
	newPress := pressed && !s.wasTouch
	s.wasTouch = pressed
	if newPress {
		switch {
		case x < s.w/3:
			return s.Next(-1)
		case x >= s.w-s.w/3:
			return s.Next(1)
		default:
			s.paused = !s.paused
			if s.paused {
				s.Log("paused")
			} else {
				s.Log("resumed")
				s.shown = s.Now()
			}
			s.ShowCaption()
			return nil
		}
	}
	if s.caption && !s.paused && s.Now().Sub(s.captionAt) >= CaptionTime {
		if err := s.HideCaption(); err != nil {
			return err
		}
	}
	if !s.paused && s.Now().Sub(s.shown) >= s.Interval {
		return s.Next(1)
	}
	return nil
}
