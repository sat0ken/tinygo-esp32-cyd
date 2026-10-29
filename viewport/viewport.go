// Package viewport shows a smaller screen inside a display: drawing is
// moved by an offset and clipped to the window, and touch positions are
// moved back. The ESP32-3248S035C (480x320) runs the 480x272 games in a
// window in the middle of its screen this way.
package viewport

import (
	"image/color"

	"github.com/sat0ken/tinygo-cyd/framebuf"
	"github.com/sat0ken/tinygo-cyd/hal"
)

type bitmapDrawer interface {
	DrawRGBBitmap8(x, y int16, data []uint8, w, h int16) error
}

// asyncDrawer is spilcd.Device.
type asyncDrawer interface {
	DrawRGBBitmap8Async(x, y int16, data []uint8, w, h int16) error
	Wait()
}

// Display is a w x h window at (x, y) of another display.
type Display struct {
	d          hal.Display
	x, y, w, h int16
}

// New returns the window of d at (x, y) with size w x h.
func New(d hal.Display, x, y, w, h int16) *Display {
	return &Display{d: d, x: x, y: y, w: w, h: h}
}

func (v *Display) Size() (int16, int16) { return v.w, v.h }

func (v *Display) SetPixel(x, y int16, c color.RGBA) {
	if x < 0 || y < 0 || x >= v.w || y >= v.h {
		return
	}
	v.d.SetPixel(v.x+x, v.y+y, c)
}

func (v *Display) Display() error { return v.d.Display() }

// clip clips a rectangle to the window (in window coordinates).
func (v *Display) clip(x, y, w, h int16) (int16, int16, int16, int16, bool) {
	if x < 0 {
		w += x
		x = 0
	}
	if y < 0 {
		h += y
		y = 0
	}
	if x+w > v.w {
		w = v.w - x
	}
	if y+h > v.h {
		h = v.h - y
	}
	return x, y, w, h, w > 0 && h > 0
}

func (v *Display) FillRectangle(x, y, w, h int16, c color.RGBA) error {
	x, y, w, h, ok := v.clip(x, y, w, h)
	if !ok {
		return nil
	}
	return v.d.FillRectangle(v.x+x, v.y+y, w, h, c)
}

// DrawRGBBitmap8 draws big endian RGB565 bytes, clipped to the window. If
// the display cannot draw bitmaps, the pixels are set one by one.
func (v *Display) DrawRGBBitmap8(x, y int16, data []uint8, w, h int16) error {
	bd, fast := v.d.(bitmapDrawer)
	if fast && x >= 0 && y >= 0 && x+w <= v.w && y+h <= v.h {
		return bd.DrawRGBBitmap8(v.x+x, v.y+y, data, w, h)
	}
	cx, cy, cw, ch, ok := v.clip(x, y, w, h)
	if !ok {
		return nil
	}
	for row := cy; row < cy+ch; row++ {
		line := data[(int(row-y)*int(w)+int(cx-x))*2:]
		if fast {
			if err := bd.DrawRGBBitmap8(v.x+cx, v.y+row, line, cw, 1); err != nil {
				return err
			}
			continue
		}
		for col := int16(0); col < cw; col++ {
			c := framebuf.ToRGBA(uint16(line[2*col])<<8 | uint16(line[2*col+1]))
			v.d.SetPixel(v.x+cx+col, v.y+row, c)
		}
	}
	return nil
}

// Touch moves the positions of another touch panel into the window.
// Touches outside the window are moved to its nearest edge, so a finger
// slightly off a button at the edge still hits it.
type Touch struct {
	t          hal.Touch
	x, y, w, h int16
}

// NewTouch returns t for the window at (x, y) with size w x h.
func NewTouch(t hal.Touch, x, y, w, h int16) *Touch {
	return &Touch{t: t, x: x, y: y, w: w, h: h}
}

func (v *Touch) ReadTouch() (x, y int16, pressed bool) {
	x, y, pressed = v.t.ReadTouch()
	return clamp(x-v.x, v.w), clamp(y-v.y, v.h), pressed
}

func clamp(v, size int16) int16 {
	if v < 0 {
		return 0
	}
	if v >= size {
		return size - 1
	}
	return v
}

// DrawRGBBitmap8Async draws in the background if the display can (see
// spilcd.Device.DrawRGBBitmap8Async) and the bitmap is inside the window.
func (v *Display) DrawRGBBitmap8Async(x, y int16, data []uint8, w, h int16) error {
	ad, ok := v.d.(asyncDrawer)
	if ok && x >= 0 && y >= 0 && x+w <= v.w && y+h <= v.h {
		return ad.DrawRGBBitmap8Async(v.x+x, v.y+y, data, w, h)
	}
	return v.DrawRGBBitmap8(x, y, data, w, h)
}

// Wait waits for a DrawRGBBitmap8Async block to be sent.
func (v *Display) Wait() {
	if ad, ok := v.d.(asyncDrawer); ok {
		ad.Wait()
	}
}
