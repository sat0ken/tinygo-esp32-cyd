// Package pixelart draws small paletted images (pixel art) on the host and
// writes them as the images of PicoLibSDK that package picopad draws: a
// byte a pixel and a palette of RGB565 colours. The games use it to
// replace pictures of the C sources that cannot be distributed (artgen
// programs next to them draw the new ones with these functions).
package pixelart

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"os"

	"github.com/sat0ken/tinygo-cyd/picopad"
)

// Image is a paletted image being drawn. Entry 0 of the palette is the
// transparent colour of the sprite (Trans), which empty pixels have.
type Image struct {
	W, H int
	Pix  []uint8
	Pal  []picopad.Color
}

// New returns an empty w x h image with the transparent colour trans.
func New(w, h int, trans picopad.Color) *Image {
	return &Image{W: w, H: h, Pix: make([]uint8, w*h), Pal: []picopad.Color{trans}}
}

// C returns the index of the colour r, g, b (added to the palette).
func (im *Image) C(r, g, b uint8) uint8 {
	return im.Col(picopad.RGB(r, g, b))
}

// Col returns the index of col (added to the palette).
func (im *Image) Col(col picopad.Color) uint8 {
	for i, p := range im.Pal[1:] {
		if p == col {
			return uint8(i + 1)
		}
	}
	if len(im.Pal) == 256 {
		panic("pixelart: more than 255 colours")
	}
	im.Pal = append(im.Pal, col)
	return uint8(len(im.Pal) - 1)
}

// Shade returns the index of colour c made lighter (f > 1) or darker.
func (im *Image) Shade(c uint8, f float64) uint8 {
	p := im.Pal[c]
	r := float64(p>>11) * 255 / 31 * f
	g := float64(p>>5&0x3F) * 255 / 63 * f
	b := float64(p&0x1F) * 255 / 31 * f
	cl := func(v float64) uint8 { return uint8(math.Max(0, math.Min(255, math.Round(v)))) }
	return im.C(cl(r), cl(g), cl(b))
}

// At returns the pixel at x, y (0 outside).
func (im *Image) At(x, y int) uint8 {
	if x < 0 || y < 0 || x >= im.W || y >= im.H {
		return 0
	}
	return im.Pix[y*im.W+x]
}

// Set sets a pixel (nothing outside).
func (im *Image) Set(x, y int, c uint8) {
	if x >= 0 && y >= 0 && x < im.W && y < im.H {
		im.Pix[y*im.W+x] = c
	}
}

// Rect fills a rectangle.
func (im *Image) Rect(x, y, w, h int, c uint8) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			im.Set(xx, yy, c)
		}
	}
}

// Line draws a line.
func (im *Image) Line(x0, y0, x1, y1 int, c uint8) {
	dx, dy := abs(x1-x0), -abs(y1-y0)
	sx, sy := sign(x1-x0), sign(y1-y0)
	e := dx + dy
	for {
		im.Set(x0, y0, c)
		if x0 == x1 && y0 == y1 {
			return
		}
		if e2 := 2 * e; e2 >= dy {
			e += dy
			x0 += sx
		} else {
			e += dx
			y0 += sy
		}
	}
}

// Ellipse fills an ellipse (the centres of the pixels inside).
func (im *Image) Ellipse(cx, cy, rx, ry float64, c uint8) {
	for y := 0; y < im.H; y++ {
		for x := 0; x < im.W; x++ {
			dx, dy := (float64(x)+0.5-cx)/rx, (float64(y)+0.5-cy)/ry
			if dx*dx+dy*dy <= 1 {
				im.Set(x, y, c)
			}
		}
	}
}

// Ball fills a circle shaded as a ball lit from the upper left, from dark
// (ramp[0]) to light.
func (im *Image) Ball(cx, cy, r float64, ramp []uint8) {
	for y := 0; y < im.H; y++ {
		for x := 0; x < im.W; x++ {
			dx, dy := (float64(x)+0.5-cx)/r, (float64(y)+0.5-cy)/r
			d := dx*dx + dy*dy
			if d > 1 {
				continue
			}
			z := math.Sqrt(1 - d)
			l := (-0.5*dx - 0.6*dy + 0.62*z) // light from the upper left, in front
			k := int((l + 0.25) / 1.25 * float64(len(ramp)))
			im.Set(x, y, ramp[min(max(k, 0), len(ramp)-1)])
		}
	}
}

// Poly fills the polygon of the points x0, y0, x1, y1... (the centres of
// the pixels inside).
func (im *Image) Poly(c uint8, pts ...float64) {
	n := len(pts) / 2
	for y := 0; y < im.H; y++ {
		fy := float64(y) + 0.5
		for x := 0; x < im.W; x++ {
			fx := float64(x) + 0.5
			in := false
			for i, j := 0, n-1; i < n; j, i = i, i+1 {
				xi, yi, xj, yj := pts[2*i], pts[2*i+1], pts[2*j], pts[2*j+1]
				if (yi > fy) != (yj > fy) && fx < (xj-xi)*(fy-yi)/(yj-yi)+xi {
					in = !in
				}
			}
			if in {
				im.Set(x, y, c)
			}
		}
	}
}

// Outline draws c around the drawn pixels (the empty pixels next to them,
// up, down, left or right).
func (im *Image) Outline(c uint8) {
	var add []int
	for y := 0; y < im.H; y++ {
		for x := 0; x < im.W; x++ {
			if im.At(x, y) == 0 && (im.At(x-1, y) != 0 && im.At(x-1, y) != c || im.At(x+1, y) != 0 && im.At(x+1, y) != c ||
				im.At(x, y-1) != 0 && im.At(x, y-1) != c || im.At(x, y+1) != 0 && im.At(x, y+1) != c) {
				add = append(add, y*im.W+x)
			}
		}
	}
	for _, i := range add {
		im.Pix[i] = c
	}
}

// Stamp draws rows of characters: each character other than a space or a
// '.' is the colour key[ch].
func (im *Image) Stamp(x, y int, key map[byte]uint8, rows ...string) {
	for j, r := range rows {
		for i := 0; i < len(r); i++ {
			if c, ok := key[r[i]]; ok && r[i] != ' ' && r[i] != '.' {
				im.Set(x+i, y+j, c)
			}
		}
	}
}

// FlipX mirrors the image left to right.
func (im *Image) FlipX() {
	for y := 0; y < im.H; y++ {
		row := im.Pix[y*im.W : (y+1)*im.W]
		for i, j := 0, len(row)-1; i < j; i, j = i+1, j-1 {
			row[i], row[j] = row[j], row[i]
		}
	}
}

// Join puts frames of the same size side by side (a sprite sheet) with
// one palette.
func Join(frames ...*Image) *Image {
	w, h := frames[0].W, frames[0].H
	out := New(w*len(frames), h, frames[0].Pal[0])
	for k, f := range frames {
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if c := f.At(x, y); c != 0 {
					out.Set(k*w+x, y, out.Col(f.Pal[c]))
				}
			}
		}
	}
	return out
}

// Data returns the pixels, a byte each (8-bit paletted pixel graphics).
func (im *Image) Data() []byte { return append([]byte(nil), im.Pix...) }

// PalData returns the palette, RGB565 little endian.
func (im *Image) PalData() []byte {
	b := make([]byte, 0, 2*len(im.Pal))
	for _, c := range im.Pal {
		b = append(b, byte(c), byte(c>>8))
	}
	return b
}

// WriteGo writes the image as the Go constants nameImgPal and nameImg,
// with the comments of tools/picoimg.
func (im *Image) WriteGo(w io.Writer, name, about string) {
	fmt.Fprintf(w, "\n// %s\n//\n// format: 8-bit paletted pixel graphics\n// image width: %d pixels\n// image height: %d lines\n// image pitch: %d bytes\n",
		about, im.W, im.H, im.W)
	fmt.Fprintf(w, "const %sImgPal = %s\n\n", name, Literal(im.PalData()))
	fmt.Fprintf(w, "const %sImg = %s\n", name, Literal(im.Data()))
}

// Literal writes b as a Go string literal in lines of 32 bytes.
func Literal(b []byte) string {
	if len(b) == 0 {
		return `""`
	}
	var s bytes.Buffer
	for i := 0; i < len(b); i += 32 {
		if i > 0 {
			s.WriteString(" +\n\t")
		}
		s.WriteByte('"')
		for _, c := range b[i:min(i+32, len(b))] {
			fmt.Fprintf(&s, "\\x%02x", c)
		}
		s.WriteByte('"')
	}
	return s.String()
}

// RGBA returns the image in colours (the transparent colour too), scaled
// by scale.
func (im *Image) RGBA(scale int) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, im.W*scale, im.H*scale))
	for y := 0; y < im.H*scale; y++ {
		for x := 0; x < im.W*scale; x++ {
			c := im.Pal[im.Pix[(y/scale)*im.W+x/scale]]
			out.Set(x, y, color.RGBA{uint8(c >> 11 << 3), uint8(c >> 5 << 2), uint8(c << 3), 255})
		}
	}
	return out
}

// WritePNG writes the image as a PNG (a preview).
func (im *Image) WritePNG(path string, scale int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, im.RGBA(scale)); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func sign(v int) int {
	switch {
	case v < 0:
		return -1
	case v > 0:
		return 1
	}
	return 0
}
