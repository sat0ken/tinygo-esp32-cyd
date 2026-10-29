//go:build js && wasm

package platform

import (
	"github.com/sat0ken/tinygo-cyd/board"
	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/wasmlcd"
)

var display *wasmlcd.Display

// Init attaches to the <canvas id="lcd"> of web/index.html. The same
// display also implements hal.Touch from pointer events on the canvas.
func Init() (hal.Display, hal.Touch, error) {
	d, err := wasmlcd.New("lcd", board.Width, board.Height)
	if err != nil {
		return nil, nil, err
	}
	display = d
	return d, d, nil
}

// InitPad returns the keyboard as a game controller (call Init first):
// the arrow keys are the stick, Z is A, X is B, A is X, S is Y and Enter
// is OK.
func InitPad() hal.Pad {
	if display == nil {
		return nil
	}
	return display
}

// SetBacklight dims the canvas with CSS so the app behaves the same.
func SetBacklight(percent int) {
	wasmlcd.SetBrightness("lcd", percent)
}
