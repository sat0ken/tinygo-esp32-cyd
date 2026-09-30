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

// FromCorners computes a calibration from raw readings at four targets
// placed inset pixels from the corners, in the order top-left, top-right,
// bottom-right, bottom-left. It detects swapped axes and extrapolates the
// readings to the screen edges.
//
// It returns an error when the readings cannot come from touching the four
// targets: the span between left and right (top and bottom) is too small,
// the two readings on one edge disagree too much, or the axis direction is
// different on the two edges.
func FromCorners(raw [4][2]int32, width, height, inset int16) (Calibration, error) {
	// If raw X changes more between top-left and bottom-left than between
	// top-left and top-right, raw X measures the vertical axis.
	swap := abs32(raw[3][0]-raw[0][0]) > abs32(raw[1][0]-raw[0][0])
	if swap {
		for i := range raw {
			raw[i][0], raw[i][1] = raw[i][1], raw[i][0]
		}
	}
	// Horizontal: top edge 0->1, bottom edge 3->2. Vertical: left 0->3, right 1->2.
	if err := checkAxis(raw[1][0]-raw[0][0], raw[2][0]-raw[3][0], raw[3][0]-raw[0][0], raw[2][0]-raw[1][0], minSpanX, "x"); err != nil {
		return Calibration{}, err
	}
	if err := checkAxis(raw[3][1]-raw[0][1], raw[2][1]-raw[1][1], raw[1][1]-raw[0][1], raw[2][1]-raw[3][1], minSpanY, "y"); err != nil {
		return Calibration{}, err
	}

	// Average the two readings on each edge, then extrapolate from the
	// target position (inset) to the screen edge (0 and size-1).
	left := (raw[0][0] + raw[3][0]) / 2
	right := (raw[1][0] + raw[2][0]) / 2
	top := (raw[0][1] + raw[1][1]) / 2
	bottom := (raw[2][1] + raw[3][1]) / 2
	in := int32(inset)
	left, right = extrapolate(left, right, in, int32(width)-1-in, int32(width)-1)
	top, bottom = extrapolate(top, bottom, in, int32(height)-1-in, int32(height)-1)
	return Calibration{
		Width: width, Height: height,
		RawXLeft: left, RawXRight: right, RawYTop: top, RawYBottom: bottom,
		SwapXY: swap,
	}, nil
}

// Minimum raw spans between the targets. The XPT2046 is 12-bit (0..4095)
// and the targets are 439 x 231 pixels apart on the 480x272 panel, so a
// real calibration spans roughly 3000 x 1600; these limits only reject
// readings that are clearly not from the targets.
const (
	minSpanX = 1000
	minSpanY = 500
)

// ErrCalibration is returned by FromCorners for unusable readings.
type ErrCalibration struct{ Reason string }

func (e ErrCalibration) Error() string { return "xpttouch: bad calibration: " + e.Reason }

// checkAxis checks one axis: span1/span2 are the differences along the axis
// on the two edges, off1/off2 the differences across it (should be ~0).
func checkAxis(span1, span2, off1, off2, minSpan int32, name string) error {
	if abs32(span1) < minSpan || abs32(span2) < minSpan {
		return ErrCalibration{name + " span too small"}
	}
	if (span1 < 0) != (span2 < 0) {
		return ErrCalibration{name + " direction differs between edges"}
	}
	span := (abs32(span1) + abs32(span2)) / 2
	if abs32(off1) > span/4 || abs32(off2) > span/4 {
		return ErrCalibration{name + " readings on the same edge disagree"}
	}
	return nil
}

// extrapolate maps raw readings a (at pixel pa) and b (at pixel pb) to the
// raw values at pixel 0 and pixel max.
func extrapolate(a, b, pa, pb, max int32) (int32, int32) {
	perPixel := float32(b-a) / float32(pb-pa)
	lo := a - int32(perPixel*float32(pa))
	hi := b + int32(perPixel*float32(max-pb))
	return lo, hi
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}
