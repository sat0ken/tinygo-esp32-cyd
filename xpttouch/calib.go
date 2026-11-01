// Package xpttouch adapts the tinygo drivers xpt2046 touch controller to
// hal.Touch, converting raw readings to screen coordinates.
//
// calib.go has no build tags so the conversion can be tested on the host.
package xpttouch

import "github.com/sat0ken/tinygo-cyd/touchcal"

// Calibration maps raw 12-bit readings to screen pixels (see touchcal).
type Calibration = touchcal.Calibration

// Minimum raw spans between the calibration targets. The XPT2046 is 12-bit
// (0..4095) and the targets are 439 x 231 pixels apart on the 480x272
// panel, so a real calibration spans roughly 3000 x 1600; these limits
// only reject readings that are clearly not from the targets.
const (
	minSpanX = 1000
	minSpanY = 500
)

// FromCorners is touchcal.FromCorners with the XPT2046 limits.
func FromCorners(raw [4][2]int32, width, height, inset int16) (Calibration, error) {
	return touchcal.FromCorners(raw, width, height, inset, minSpanX, minSpanY)
}

// pressThreshold is the lowest pressure value of a press (see pressed). On
// the ESP32-4827S043 it is at most 18 when not pressed, 371 for the lightest
// press measured (in a corner) and about 1800 for a firm one in the middle.
const pressThreshold = 200

// pressed tells from the two pressure readings whether the panel is
// pressed: Z1 rises from 0 and Z2 falls from 4095 as it is pressed harder.
func pressed(z1, z2 int32) bool {
	return z1+4095-z2 >= pressThreshold
}
