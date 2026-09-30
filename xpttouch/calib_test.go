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

func TestFromCornersGood(t *testing.T) {
	// Synthetic panel: raw x = 300 + 7*px (left..right), raw y = 3800 - 12*py
	// (top has the larger value), with a little noise.
	rawAt := func(px, py int32) [2]int32 { return [2]int32{300 + 7*px, 3800 - 12*py} }
	raw := [4][2]int32{rawAt(20, 20), rawAt(459, 20), rawAt(459, 251), rawAt(20, 251)}
	raw[0][0] += 15
	raw[2][1] -= 20
	c, err := FromCorners(raw, 480, 272, 20)
	if err != nil {
		t.Fatal(err)
	}
	if c.SwapXY {
		t.Fatal("unexpected swap")
	}
	for _, p := range [][2]int32{{20, 20}, {240, 136}, {459, 251}, {0, 0}, {479, 271}} {
		r := rawAt(p[0], p[1])
		x, y := c.Map(r[0], r[1])
		if abs32(int32(x)-p[0]) > 3 || abs32(int32(y)-p[1]) > 3 {
			t.Errorf("(%d,%d) -> (%d,%d)", p[0], p[1], x, y)
		}
	}

	// Same panel with swapped axes.
	var swapped [4][2]int32
	for i, r := range raw {
		swapped[i] = [2]int32{r[1], r[0]}
	}
	c, err = FromCorners(swapped, 480, 272, 20)
	if err != nil || !c.SwapXY {
		t.Fatalf("swap: %+v %v", c, err)
	}
	r := rawAt(240, 136)
	if x, y := c.Map(r[1], r[0]); abs32(int32(x)-240) > 3 || abs32(int32(y)-136) > 3 {
		t.Errorf("swap: (240,136) -> (%d,%d)", x, y)
	}
}

func TestFromCornersRejectsRandomTouches(t *testing.T) {
	// Readings logged on the board while touching random places instead of
	// the targets (2026-09-30).
	bad := [][4][2]int32{
		{{1969, 1764}, {2054, 1657}, {2431, 1284}, {263, 3711}},
		{{3872, 439}, {2916, 1822}, {1326, 914}, {1343, 2881}},
		{{3749, 2140}, {2451, 1447}, {2358, 2711}, {1483, 1617}},
	}
	for i, raw := range bad {
		if c, err := FromCorners(raw, 480, 272, 20); err == nil {
			t.Errorf("%d: accepted %+v", i, c)
		}
	}
}
