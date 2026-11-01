package xpttouch

import "testing"

func TestPressed(t *testing.T) {
	for _, c := range []struct {
		z1, z2 int32
		want   bool
	}{
		{0, 4095, false},   // not pressed
		{0, 4077, false},   // the most noise seen
		{233, 3957, true},  // the lightest press seen
		{1300, 3600, true}, // a firm press
	} {
		if got := pressed(c.z1, c.z2); got != c.want {
			t.Errorf("pressed(%d, %d) = %v, want %v", c.z1, c.z2, got, c.want)
		}
	}
}
