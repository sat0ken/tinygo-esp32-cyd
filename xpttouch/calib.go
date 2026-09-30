// Package xpttouch adapts the tinygo drivers xpt2046 touch controller to
// hal.Touch, converting raw readings to screen coordinates.
//
// calib.go has no build tags so the conversion can be tested on the host.
package xpttouch

// Calibration maps raw 12-bit readings to screen pixels. The raw value at
// the left edge may be larger than at the right edge (and top/bottom), which
// flips the axis.
type Calibration struct {
	Width, Height       int16
	RawXLeft, RawXRight int32
	RawYTop, RawYBottom int32
	SwapXY              bool // raw X measures the vertical axis
}

// Map converts a raw reading to screen coordinates, clamped to the screen.
func (c *Calibration) Map(rawX, rawY int32) (x, y int16) {
	if c.SwapXY {
		rawX, rawY = rawY, rawX
	}
	return scale(rawX, c.RawXLeft, c.RawXRight, c.Width), scale(rawY, c.RawYTop, c.RawYBottom, c.Height)
}

func scale(raw, lo, hi int32, size int16) int16 {
	if hi == lo {
		return 0
	}
	v := (raw - lo) * int32(size-1) / (hi - lo)
	if v < 0 {
		v = 0
	}
	if v > int32(size-1) {
		v = int32(size - 1)
	}
	return int16(v)
}
