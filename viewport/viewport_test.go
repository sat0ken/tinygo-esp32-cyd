package viewport

import (
	"image/color"
	"math/rand"
	"testing"

	"github.com/sat0ken/tinygo-cyd/framebuf"
	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/memlcd"
	"github.com/sat0ken/tinygo-cyd/romfont/freesans"
	"tinygo.org/x/tinyfont"
)

// noBitmap hides DrawRGBBitmap8 of a display.
type noBitmap struct{ d hal.Display }

func (n noBitmap) Size() (int16, int16)              { return n.d.Size() }
func (n noBitmap) SetPixel(x, y int16, c color.RGBA) { n.d.SetPixel(x, y, c) }
func (n noBitmap) Display() error                    { return n.d.Display() }
func (n noBitmap) FillRectangle(x, y, w, h int16, c color.RGBA) error {
	return n.d.FillRectangle(x, y, w, h, c)
}

// TestSameAsSmallScreen draws on a 100x60 window at (10, 20) of a 130x100
// display and on a 100x60 display, and compares.
func TestSameAsSmallScreen(t *testing.T) {
	for _, fast := range []bool{true, false} {
		big := memlcd.New(130, 100)
		var base hal.Display = big
		if !fast {
			base = noBitmap{big}
		}
		v := New(base, 10, 20, 100, 60)
		small := memlcd.New(100, 60)
		r := rand.New(rand.NewSource(2))
		for i := 0; i < 300; i++ {
			x, y := int16(r.Intn(140)-20), int16(r.Intn(100)-20)
			w, h := int16(r.Intn(40)+1), int16(r.Intn(30)+1)
			c := color.RGBA{uint8(r.Intn(256)), uint8(r.Intn(256)), uint8(r.Intn(256)), 255}
			switch r.Intn(4) {
			case 0:
				v.FillRectangle(x, y, w, h, c)
				small.FillRectangle(x, y, w, h, c)
			case 1:
				v.SetPixel(x, y, c)
				small.SetPixel(x, y, c)
			case 2:
				data := make([]byte, int(w)*int(h)*2)
				r.Read(data)
				v.DrawRGBBitmap8(x, y, data, w, h)
				for k := 0; k < int(w)*int(h); k++ {
					px := framebuf.ToRGBA(uint16(data[2*k])<<8 | uint16(data[2*k+1]))
					small.SetPixel(x+int16(k%int(w)), y+int16(k/int(w)), px)
				}
			case 3:
				tinyfont.WriteLine(v, &freesans.Bold9pt7b, x, y, "Ag", c)
				tinyfont.WriteLine(small, &freesans.Bold9pt7b, x, y, "Ag", c)
			}
		}
		for y := int16(0); y < 100; y++ {
			for x := int16(0); x < 130; x++ {
				want := uint16(0)
				if x >= 10 && x < 110 && y >= 20 && y < 80 {
					want = small.Pixel(x-10, y-20)
				}
				if got := big.Pixel(x, y); got != want {
					t.Fatalf("fast %v: pixel %d,%d = %04x, want %04x", fast, x, y, got, want)
				}
			}
		}
	}
}

func TestTouchIsMovedAndClamped(t *testing.T) {
	m := &memlcd.Touch{}
	v := NewTouch(m, 0, 24, 480, 272)
	for _, c := range []struct{ x, y, wx, wy int16 }{
		{100, 124, 100, 100},
		{5, 10, 5, 0},        // in the bar above: top edge
		{479, 319, 479, 271}, // in the bar below: bottom edge
	} {
		m.X, m.Y, m.Pressed = c.x, c.y, true
		if x, y, p := v.ReadTouch(); x != c.wx || y != c.wy || !p {
			t.Errorf("%d,%d -> %d,%d %v", c.x, c.y, x, y, p)
		}
	}
}
