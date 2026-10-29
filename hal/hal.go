// Package hal defines the interfaces the application depends on.
//
// Implementations: rgblcd (ESP32-S3 RGB panel), wasmlcd (browser canvas) and
// memlcd (in-memory, for tests). This package has no build tags and must not
// import the machine package.
package hal

import (
	"image/color"

	"tinygo.org/x/drivers"
)

// Display is a drivers.Displayer with a fast rectangle fill.
type Display interface {
	drivers.Displayer
	FillRectangle(x, y, w, h int16, c color.RGBA) error
}

// Touch returns the touch position already converted to screen coordinates.
type Touch interface {
	ReadTouch() (x, y int16, pressed bool)
}

// NoTouch is a Touch that is never pressed.
type NoTouch struct{}

func (NoTouch) ReadTouch() (x, y int16, pressed bool) { return 0, 0, false }

// Buttons is the state of a game controller: the directions the stick (or
// a d-pad) is pushed to and the buttons that are down.
type Buttons uint16

const (
	PadUp Buttons = 1 << iota
	PadDown
	PadLeft
	PadRight
	PadA  // press what has the focus (like a tap)
	PadB  // back
	PadX  // a shortcut, depending on the game
	PadY  // a shortcut, depending on the game
	PadOK // pressing the stick

	PadDirections = PadUp | PadDown | PadLeft | PadRight
)

// Pad is a game controller. A game that has none gets a nil Pad.
type Pad interface {
	ReadPad() Buttons
}
