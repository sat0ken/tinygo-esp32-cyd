// Package memlcd is an in-memory hal.Display for tests on the host.
//
// It has the same frame buffer format as rgblcd and wasmlcd, and can write
// the buffer as a PNG so screens can be compared with golden images.
package memlcd

import (
	"image"
	"image/png"
	"io"
	"os"

	"github.com/sat0ken/tinygo-cyd/framebuf"
)

// Display is an in-memory RGB565 display.
type Display struct {
	*framebuf.Buffer
	Displays int // number of Display() calls
}

// New allocates a w x h display, initially black.
func New(w, h int16) *Display {
	return &Display{Buffer: framebuf.New(make([]uint16, int(w)*int(h)), w, h)}
}

// Display counts the call; the buffer is always "on screen".
func (d *Display) Display() error {
	d.Displays++
	return nil
}

// Image converts the frame buffer to an RGBA image, expanding RGB565 the
// same way wasmlcd does for the browser.
func (d *Display) Image() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, int(d.W), int(d.H)))
	for i, v := range d.Pix {
		c := framebuf.ToRGBA(v)
		o := i * 4
		img.Pix[o+0] = c.R
		img.Pix[o+1] = c.G
		img.Pix[o+2] = c.B
		img.Pix[o+3] = 0xFF
	}
	return img
}

// WritePNG writes the screen as a PNG.
func (d *Display) WritePNG(w io.Writer) error {
	return png.Encode(w, d.Image())
}

// SavePNG writes the screen to a PNG file.
func (d *Display) SavePNG(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := d.WritePNG(f); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// Touch is a hal.Touch whose state is set by the test.
type Touch struct {
	X, Y    int16
	Pressed bool
}

// ReadTouch implements hal.Touch.
func (t *Touch) ReadTouch() (x, y int16, pressed bool) {
	return t.X, t.Y, t.Pressed
}
