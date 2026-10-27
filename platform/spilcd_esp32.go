//go:build esp32_3248s035c || esp32_2432s028r

package platform

import (
	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/spilcd"
)

// flushTouch sends the pixels the panel still holds (see spilcd) before
// reading the touch panel. Every game loop reads the touch panel once a
// frame, so whatever was drawn in the last frame is on the screen even if
// the code does not call Display.
type flushTouch struct {
	t hal.Touch
	d *spilcd.Device
}

func (f flushTouch) ReadTouch() (x, y int16, pressed bool) {
	f.d.Display()
	return f.t.ReadTouch()
}
