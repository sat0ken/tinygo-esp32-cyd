// Package pad is what the games need from a game controller (hal.Pad):
// the buttons pressed in this frame, the directions repeating while the
// stick is held, moving a focus between the things on the screen, and the
// mini joystick module (package minijoy) as a hal.Pad.
//
// The games show their focus only after the controller has been used, so
// with touch alone they look as before.
package pad

import (
	"image/color"

	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/minijoy"
)

// Directions repeat while they are held: first after repeatDelay, then
// every repeatEvery seconds.
const (
	repeatDelay = 0.35
	repeatEvery = 0.1
)

// Input reads a hal.Pad once per frame.
type Input struct {
	held, pressed hal.Buttons
	dirT          float32 // how long the directions in held have been held
	primed        bool

	// Used is set when the controller is used for the first time.
	Used bool
}

// Update reads p (nil: no controller) after dt seconds. The first update
// only notes what is held, so that a button still down from the previous
// screen (A on the launcher) does not count as pressed.
func (in *Input) Update(p hal.Pad, dt float32) {
	var b hal.Buttons
	if p != nil {
		b = p.ReadPad()
	}
	if !in.primed {
		in.held, in.pressed, in.dirT, in.primed = b, 0, 0, true
		return
	}
	in.pressed = b &^ in.held
	dirs := b & hal.PadDirections
	if dirs == 0 || dirs != in.held&hal.PadDirections {
		in.dirT = 0
	} else {
		before := repeats(in.dirT)
		in.dirT += dt
		if repeats(in.dirT) > before {
			in.pressed |= dirs
		}
	}
	in.held = b
	if b != 0 {
		in.Used = true
	}
}

// repeats is how many times a direction held for t seconds has repeated.
func repeats(t float32) int {
	if t < repeatDelay {
		return 0
	}
	return 1 + int((t-repeatDelay)/repeatEvery)
}

// Reset makes the next Update note what is held again (after a screen
// that read the controller itself).
func (in *Input) Reset() { in.primed = false }

// Pressed reports whether one of b was pressed in this frame (or a
// direction repeated).
func (in *Input) Pressed(b hal.Buttons) bool { return in.pressed&b != 0 }

// Held reports whether one of b is down.
func (in *Input) Held(b hal.Buttons) bool { return in.held&b != 0 }

// Direction returns the direction pressed in this frame (one of PadUp,
// PadDown, PadLeft, PadRight), or 0. When the stick is pushed diagonally,
// up and down come first.
func (in *Input) Direction() hal.Buttons {
	for _, d := range [...]hal.Buttons{hal.PadUp, hal.PadDown, hal.PadLeft, hal.PadRight} {
		if in.pressed&d != 0 {
			return d
		}
	}
	return 0
}

// Box is something on the screen that can have the focus.
type Box struct{ X, Y, W, H int16 }

// Move returns the box the focus goes to from boxes[cur] in direction dir:
// the nearest box whose middle is beyond the edge of boxes[cur] in that
// direction. Boxes in line with it (overlapping it the other way) come
// first: the nearest other box only if there is none. It returns cur if
// there is no box that way.
func Move(boxes []Box, cur int, dir hal.Buttons) int {
	if cur < 0 || cur >= len(boxes) {
		return cur
	}
	c := boxes[cur]
	best, bestScore := cur, int32(-1)     // any box
	inline, inlineScore := cur, int32(-1) // a box in line
	for i, b := range boxes {
		if i == cur {
			continue
		}
		// along: distance between the middles in the direction (x2, to
		// stay in integers), beyond: how far the middle of b is past the
		// edge of c; across: the gap between the boxes the other way (0 if
		// they overlap) and the distance between their middles.
		var along, beyond, gap, off int32
		switch dir {
		case hal.PadLeft, hal.PadRight:
			along = mid(b.X, b.W) - mid(c.X, c.W)
			beyond = mid(b.X, b.W) - 2*int32(c.X+c.W)
			if dir == hal.PadLeft {
				along, beyond = -along, 2*int32(c.X)-mid(b.X, b.W)
			}
			gap = span(c.Y, c.H, b.Y, b.H)
			off = abs(mid(b.Y, b.H) - mid(c.Y, c.H))
		case hal.PadUp, hal.PadDown:
			along = mid(b.Y, b.H) - mid(c.Y, c.H)
			beyond = mid(b.Y, b.H) - 2*int32(c.Y+c.H)
			if dir == hal.PadUp {
				along, beyond = -along, 2*int32(c.Y)-mid(b.Y, b.H)
			}
			gap = span(c.X, c.W, b.X, b.W)
			off = abs(mid(b.X, b.W) - mid(c.X, c.W))
		default:
			return cur
		}
		if beyond <= 0 {
			continue
		}
		score := along + 8*gap + off/2
		if bestScore < 0 || score < bestScore {
			best, bestScore = i, score
		}
		if gap == 0 && (inlineScore < 0 || score < inlineScore) {
			inline, inlineScore = i, score
		}
	}
	if inlineScore >= 0 {
		return inline
	}
	return best
}

// mid is twice the middle of [a, a+w).
func mid(a, w int16) int32 { return 2*int32(a) + int32(w) }

// span is the gap between [a, a+aw) and [b, b+bw), 0 if they overlap.
func span(a, aw, b, bw int16) int32 {
	switch {
	case b >= a+aw:
		return int32(b - a - aw)
	case a >= b+bw:
		return int32(a - b - bw)
	}
	return 0
}

func abs(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

// Frame draws a frame t pixels thick inside the rectangle: the focus.
func Frame(d hal.Display, x, y, w, h, t int16, c color.RGBA) {
	d.FillRectangle(x, y, w, t, c)
	d.FillRectangle(x, y+h-t, w, t, c)
	d.FillRectangle(x, y+t, t, h-2*t, c)
	d.FillRectangle(x+w-t, y+t, t, h-2*t, c)
}

// FromJoystick converts a reading of the mini joystick module: the stick
// counts as pushed beyond 48 of 128 from the middle (X grows to the right,
// Y upwards), a button as down while minijoy says so.
func FromJoystick(s minijoy.State) hal.Buttons {
	const dead = 48
	var b hal.Buttons
	switch {
	case s.X > 128+dead:
		b |= hal.PadRight
	case s.X < 128-dead:
		b |= hal.PadLeft
	}
	switch {
	case s.Y > 128+dead:
		b |= hal.PadUp
	case s.Y < 128-dead:
		b |= hal.PadDown
	}
	for k, bit := range [...]hal.Buttons{minijoy.A: hal.PadA, minijoy.B: hal.PadB,
		minijoy.X: hal.PadX, minijoy.Y: hal.PadY, minijoy.OK: hal.PadOK} {
		if s.Buttons[k].Down() {
			b |= bit
		}
	}
	return b
}

// Joystick is the mini joystick module as a hal.Pad.
type Joystick struct {
	dev *minijoy.Device
	s   minijoy.State
}

// NewJoystick returns the module on bus as a hal.Pad.
func NewJoystick(bus minijoy.Bus) *Joystick { return &Joystick{dev: minijoy.New(bus)} }

// ReadPad reads the module (two block reads, about 1.2ms at 100kHz). If it
// does not answer, nothing is pressed.
func (j *Joystick) ReadPad() hal.Buttons {
	if err := j.dev.ReadBlock(&j.s); err != nil {
		return 0
	}
	return FromJoystick(j.s)
}

// Fake is a hal.Pad for tests: B is what is held.
type Fake struct{ B hal.Buttons }

func (f *Fake) ReadPad() hal.Buttons { return f.B }
