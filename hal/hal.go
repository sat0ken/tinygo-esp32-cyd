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
