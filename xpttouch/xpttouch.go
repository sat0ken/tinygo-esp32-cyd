//go:build esp32s3

package xpttouch

import (
	"machine"

	"tinygo.org/x/drivers/xpt2046"
)

// Touch is an XPT2046 touch panel returning screen coordinates.
type Touch struct {
	dev xpt2046.Device
	Cal Calibration
}

// New configures the XPT2046 on the given pins (bit-banged SPI, as done by
// the tinygo drivers xpt2046 package).
func New(sck, cs, mosi, miso, irq machine.Pin, cal Calibration) *Touch {
	t := &Touch{dev: xpt2046.New(sck, cs, mosi, miso, irq), Cal: cal}
	t.dev.Configure(&xpt2046.Config{Precision: 10})
	return t
}

// ReadRaw returns the averaged raw 12-bit readings. ok is false when the
// panel is not pressed.
//
// The xpt2046 driver scales X to 16 bits (x << 4) and inverts Y
// ((4096 - y) << 4); this undoes the scaling but keeps the inversion.
func (t *Touch) ReadRaw() (x, y int32, ok bool) {
	if !t.dev.Touched() {
		return 0, 0, false
	}
	p := t.dev.ReadTouchPoint()
	if p.X == 0 && p.Y == 0 {
		return 0, 0, false
	}
	return int32(p.X >> 4), int32(p.Y >> 4), true
}

// ReadTouch implements hal.Touch.
func (t *Touch) ReadTouch() (x, y int16, pressed bool) {
	rx, ry, ok := t.ReadRaw()
	if !ok {
		return 0, 0, false
	}
	x, y = t.Cal.Map(rx, ry)
	return x, y, true
}
