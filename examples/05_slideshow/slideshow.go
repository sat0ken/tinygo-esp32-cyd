package main

import (
	"errors"
	"image/color"
	"io"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"
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
	Cut       Transition = iota // draw at once
	WipeDown                    // top to bottom, 8 lines at a time
	WipeRight                   // left to right, 40 columns at a time
	Blinds                      // 8 bands opening line by line
	numTransitions
)

func (t Transition) String() string {
	return [...]string{"cut", "wipe-down", "wipe-right", "blinds"}[t]
}

const (
	wipeLines   = 8
	wipeColumns = 40
	blindBands  = 8
)

// Show is a slide show of raw RGB565 (big endian) images of exactly the
// screen size, read from an fs.FS (normally go:embed, i.e. flash).
type Show struct {
	d     hal.Display
	bd    bitmapDrawer
	t     hal.Touch
	w, h  int16
	files []io.ReaderAt
	names []string

	cur      int
	next     Transition
	paused   bool
	Interval time.Duration // time each slide is shown
	Duration time.Duration // duration of a transition

	shown     time.Time // when the current slide was completed
	caption   bool
	captionAt time.Time
	capW      int16
	wasTouch  bool

	buf   []byte              // one chunk: max(wipe lines, wipe columns)
	sleep func(time.Duration) // replaced in tests
	now   func() time.Time    // replaced in tests
	log   func(msg string)    // serial / console log
}

var ErrNoSlides = errors.New("slideshow: no .rgb565 files")

// NewShow opens every *.rgb565 file in dir of fsys, in name order.
func NewShow(d hal.Display, t hal.Touch, fsys fs.FS, dir string) (*Show, error) {
	bd, ok := d.(bitmapDrawer)
	if !ok {
		return nil, errors.New("slideshow: display has no DrawRGBBitmap8")
	}
	w, h := d.Size()
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, err
	}
	s := &Show{
		d: d, bd: bd, t: t, w: w, h: h,
		Interval: 5 * time.Second,
		Duration: 400 * time.Millisecond,
		next:     WipeDown,
		sleep:    time.Sleep,
		now:      time.Now,
		log:      func(msg string) { println(msg) },
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	size := int64(w) * int64(h) * 2
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".rgb565") {
			continue
		}
		f, err := fsys.Open(path.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		st, err := f.Stat()
		if err != nil {
			return nil, err
		}
		if st.Size() != size {
			return nil, errors.New("slideshow: " + e.Name() + " is not " + strconv.Itoa(int(w)) + "x" + strconv.Itoa(int(h)) + " RGB565")
		}
		ra, ok := f.(io.ReaderAt)
		if !ok {
			return nil, errors.New("slideshow: file does not implement io.ReaderAt")
		}
		s.files = append(s.files, ra)
		s.names = append(s.names, strings.TrimSuffix(e.Name(), ".rgb565"))
	}
	if len(s.files) == 0 {
		return nil, ErrNoSlides
	}
	n := int(w) * wipeLines * 2
	if m := wipeColumns * int(h) * 2; m > n {
		n = m
	}
	s.buf = make([]byte, n)
	return s, nil
}

// Len returns the number of slides.
func (s *Show) Len() int { return len(s.files) }

// Current returns the index of the slide on screen.
func (s *Show) Current() int { return s.cur }

// Paused reports whether automatic advance is paused.
func (s *Show) Paused() bool { return s.paused }

// rows draws lines y0..y0+n-1 of slide i at full width.
func (s *Show) rows(i int, y0, n int16) error {
	stride := int64(s.w) * 2
	b := s.buf[:int(n)*int(stride)]
	if _, err := s.files[i].ReadAt(b, int64(y0)*stride); err != nil {
		return err
	}
	return s.bd.DrawRGBBitmap8(0, y0, b, s.w, n)
}

// rect draws the rectangle (x, y, w, h) of slide i. w*h*2 must fit s.buf.
func (s *Show) rect(i int, x, y, w, h int16) error {
	stride := int64(s.w) * 2
	line := int(w) * 2
	for r := 0; r < int(h); r++ {
		off := int64(int(y)+r)*stride + int64(x)*2
		if _, err := s.files[i].ReadAt(s.buf[r*line:(r+1)*line], off); err != nil {
			return err
		}
	}
	return s.bd.DrawRGBBitmap8(x, y, s.buf[:line*int(h)], w, h)
}

// Draw shows slide i with transition tr. Each step of the transition is
// followed by a sleep so that the whole transition takes s.Duration.
func (s *Show) Draw(i int, tr Transition) error {
	var err error
	switch tr {
	case Cut:
		for y := int16(0); y < s.h && err == nil; y += wipeLines {
			err = s.rows(i, y, min16(wipeLines, s.h-y))
		}
	case WipeDown:
		steps := (s.h + wipeLines - 1) / wipeLines
		for y := int16(0); y < s.h && err == nil; y += wipeLines {
			err = s.rows(i, y, min16(wipeLines, s.h-y))
			s.step(steps)
		}
	case WipeRight:
		steps := (s.w + wipeColumns - 1) / wipeColumns
		for x := int16(0); x < s.w && err == nil; x += wipeColumns {
			err = s.rect(i, x, 0, min16(wipeColumns, s.w-x), s.h)
			s.step(steps)
		}
	case Blinds:
		band := (s.h + blindBands - 1) / blindBands
		for k := int16(0); k < band && err == nil; k++ {
			for b := int16(0); b < blindBands && err == nil; b++ {
				if y := b*band + k; y < s.h {
					err = s.rows(i, y, 1)
				}
			}
			s.step(band)
		}
	}
	if err != nil {
		return err
	}
	s.cur = i
	s.caption = false
	s.d.Display()
	return nil
}

func (s *Show) step(steps int16) {
	s.d.Display()
	s.sleep(s.Duration / time.Duration(steps))
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
	captionTime = 2 * time.Second
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
		s.log("caption: " + err.Error())
	}
	text := strconv.Itoa(s.cur+1) + "/" + strconv.Itoa(len(s.files)) + " " + s.names[s.cur]
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
	s.captionAt = s.now()
}

// HideCaption restores the slide under the caption.
func (s *Show) HideCaption() error {
	if !s.caption {
		return nil
	}
	s.caption = false
	if err := s.rect(s.cur, capX, s.capY(), s.capW, capH); err != nil {
		return err
	}
	s.d.Display()
	return nil
}

// Next shows the following slide with the next transition in turn.
func (s *Show) Next(delta int) error {
	i := (s.cur + delta + len(s.files)) % len(s.files)
	tr := s.next
	s.next = (s.next + 1) % numTransitions
	start := s.now()
	if err := s.Draw(i, tr); err != nil {
		return err
	}
	s.log("slide " + strconv.Itoa(i+1) + "/" + strconv.Itoa(len(s.files)) + " " + s.names[i] +
		" (" + tr.String() + ", " + strconv.Itoa(int(s.now().Sub(start).Milliseconds())) + " ms)")
	s.ShowCaption()
	s.shown = s.now()
	return nil
}

// Start draws the first slide.
func (s *Show) Start() error {
	if err := s.Draw(0, Cut); err != nil {
		return err
	}
	s.log("slideshow: " + strconv.Itoa(len(s.files)) + " slides")
	s.ShowCaption()
	s.shown = s.now()
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
				s.log("paused")
			} else {
				s.log("resumed")
				s.shown = s.now()
			}
			s.ShowCaption()
			return nil
		}
	}
	if s.caption && !s.paused && s.now().Sub(s.captionAt) >= captionTime {
		if err := s.HideCaption(); err != nil {
			return err
		}
	}
	if !s.paused && s.now().Sub(s.shown) >= s.Interval {
		return s.Next(1)
	}
	return nil
}
