package xpttouch

import "testing"

func TestMap(t *testing.T) {
	c := Calibration{Width: 480, Height: 272, RawXLeft: 200, RawXRight: 3900, RawYTop: 3900, RawYBottom: 200}
	tests := []struct {
		rx, ry int32
		x, y   int16
	}{
		{200, 3900, 0, 0},
		{3900, 200, 479, 271},
		{2050, 2050, 239, 135},
		{0, 5000, 0, 0},     // clamped
		{5000, 0, 479, 271}, // clamped
	}
	for _, tc := range tests {
		x, y := c.Map(tc.rx, tc.ry)
		if x != tc.x || y != tc.y {
			t.Errorf("Map(%d,%d) = %d,%d want %d,%d", tc.rx, tc.ry, x, y, tc.x, tc.y)
		}
	}
	c.SwapXY = true
	// Swapped: raw X is the vertical axis. (200, 3900) -> raw x 3900, raw y 200.
	if x, y := c.Map(200, 3900); x != 479 || y != 271 {
		t.Errorf("swap: %d,%d", x, y)
	}
}
