// Package framebuf implements drawing on an in-memory RGB565 frame buffer.
//
// The same buffer format ([]uint16, native endian RGB565, row-major, red in
// the top 5 bits) is used by every backend: rgblcd (real panel, scanned out by
// DMA), wasmlcd (browser canvas) and memlcd (tests). Only the way the buffer
// reaches the screen differs.
package framebuf

import (
	"errors"
	"image/color"

	"tinygo.org/x/drivers/pixel"
)

var ErrOutOfBounds = errors.New("framebuf: rectangle out of bounds")

// Buffer is a width x height RGB565 frame buffer.
type Buffer struct {
	Pix []uint16
	W   int16
	H   int16
}

// New wraps pix (len must be at least w*h) as a frame buffer.
func New(pix []uint16, w, h int16) *Buffer {
	if len(pix) < int(w)*int(h) {
		panic("framebuf: buffer too small")
	}
	return &Buffer{Pix: pix[:int(w)*int(h)], W: w, H: h}
}

// RGB565 converts c to a native endian RGB565 value.
func RGB565(c color.RGBA) uint16 {
	return uint16(c.R&0xF8)<<8 | uint16(c.G&0xFC)<<3 | uint16(c.B)>>3
}

// ToRGBA expands an RGB565 value to 8 bits per channel. The top bits are
// copied into the low bits so that 0x1F/0x3F become 0xFF.
func ToRGBA(v uint16) color.RGBA {
	r := uint8(v>>11) & 0x1F
	g := uint8(v>>5) & 0x3F
	b := uint8(v) & 0x1F
	return color.RGBA{
		R: r<<3 | r>>2,
		G: g<<2 | g>>4,
		B: b<<3 | b>>2,
		A: 0xFF,
	}
}

// Size returns the size of the buffer in pixels.
func (b *Buffer) Size() (x, y int16) {
	return b.W, b.H
}

// SetPixel writes one pixel. Coordinates outside the buffer are ignored.
func (b *Buffer) SetPixel(x, y int16, c color.RGBA) {
	if uint16(x) >= uint16(b.W) || uint16(y) >= uint16(b.H) {
		return
	}
	b.Pix[int(y)*int(b.W)+int(x)] = RGB565(c)
}

// Pixel returns the raw RGB565 value at (x, y), or 0 outside the buffer.
func (b *Buffer) Pixel(x, y int16) uint16 {
	if uint16(x) >= uint16(b.W) || uint16(y) >= uint16(b.H) {
		return 0
	}
	return b.Pix[int(y)*int(b.W)+int(x)]
}

// clip clips a rectangle to the buffer. ok is false if nothing is left.
func (b *Buffer) clip(x, y, w, h int16) (x0, y0, x1, y1 int, ok bool) {
	x0, y0 = int(x), int(y)
	x1, y1 = x0+int(w), y0+int(h)
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > int(b.W) {
		x1 = int(b.W)
	}
	if y1 > int(b.H) {
		y1 = int(b.H)
	}
	return x0, y0, x1, y1, x0 < x1 && y0 < y1
}

// FillRectangle fills a rectangle, clipped to the buffer.
func (b *Buffer) FillRectangle(x, y, w, h int16, c color.RGBA) error {
	x0, y0, x1, y1, ok := b.clip(x, y, w, h)
	if !ok {
		return nil
	}
	v := RGB565(c)
	stride := int(b.W)
	// Fill the first row, then copy it: copy() is much faster than a loop
	// on both the ESP32-S3 (memcpy from ROM) and in wasm.
	first := b.Pix[y0*stride+x0 : y0*stride+x1]
	for i := range first {
		first[i] = v
	}
	for yy := y0 + 1; yy < y1; yy++ {
		copy(b.Pix[yy*stride+x0:yy*stride+x1], first)
	}
	return nil
}

// FillScreen fills the whole buffer with c.
func (b *Buffer) FillScreen(c color.RGBA) {
	b.FillRectangle(0, 0, b.W, b.H, c)
}

// DrawFastHLine draws a horizontal line from x0 to x1 (inclusive).
func (b *Buffer) DrawFastHLine(x0, x1, y int16, c color.RGBA) {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	b.FillRectangle(x0, y, x1-x0+1, 1, c)
}

// DrawFastVLine draws a vertical line from y0 to y1 (inclusive).
func (b *Buffer) DrawFastVLine(x, y0, y1 int16, c color.RGBA) {
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	b.FillRectangle(x, y0, 1, y1-y0+1, c)
}

// DrawRGBBitmap copies native endian RGB565 pixels (w*h values) to (x, y).
// Unlike FillRectangle, the bitmap must fit inside the buffer.
func (b *Buffer) DrawRGBBitmap(x, y int16, data []uint16, w, h int16) error {
	if x < 0 || y < 0 || w <= 0 || h <= 0 || int(x)+int(w) > int(b.W) || int(y)+int(h) > int(b.H) {
		return ErrOutOfBounds
	}
	if len(data) < int(w)*int(h) {
		return errors.New("framebuf: bitmap data too short")
	}
	stride := int(b.W)
	for row := 0; row < int(h); row++ {
		dst := (int(y)+row)*stride + int(x)
		copy(b.Pix[dst:dst+int(w)], data[row*int(w):(row+1)*int(w)])
	}
	return nil
}

// DrawRGBBitmap8 copies big endian RGB565 bytes (the format used by the
// ST7789 driver and pixel.RGB565BE) to (x, y).
func (b *Buffer) DrawRGBBitmap8(x, y int16, data []uint8, w, h int16) error {
	if x < 0 || y < 0 || w <= 0 || h <= 0 || int(x)+int(w) > int(b.W) || int(y)+int(h) > int(b.H) {
		return ErrOutOfBounds
	}
	if len(data) < int(w)*int(h)*2 {
		return errors.New("framebuf: bitmap data too short")
	}
	stride := int(b.W)
	i := 0
	for row := 0; row < int(h); row++ {
		line := b.Pix[(int(y)+row)*stride+int(x):]
		for col := 0; col < int(w); col++ {
			line[col] = uint16(data[i])<<8 | uint16(data[i+1])
			i += 2
		}
	}
	return nil
}

// DrawBitmap copies a pixel.Image in RGB565BE format to (x, y), the same
// signature as st7789.Device.DrawBitmap.
func (b *Buffer) DrawBitmap(x, y int16, bitmap pixel.Image[pixel.RGB565BE]) error {
	w, h := bitmap.Size()
	return b.DrawRGBBitmap8(x, y, bitmap.RawBuffer(), int16(w), int16(h))
}
