// Package romfont is a tinyfont.Fonter whose glyphs are kept in strings,
// so that TinyGo puts them in flash (rodata) instead of RAM.
//
// A tinyfont.Font is a variable holding slices: TinyGo cannot know that
// nothing writes to it, so the glyph table and every bitmap end up in
// .data, i.e. in RAM. The five fonts of cmd/games took 28KB of RAM, a lot
// on the ESP32 (192KB for everything); with romfont they stay in flash.
//
// The subpackages freesans and proggy have the fonts this repository uses,
// with the same names as in tinyfont, generated from tinyfont by
// tools/fontgen. Drawing is pixel for pixel the same as tinyfont's.
package romfont

import (
	"image/color"

	"tinygo.org/x/drivers"
	"tinygo.org/x/tinyfont"
)

// Font is a font in two strings.
type Font struct {
	YAdvance uint8
	// Index has 6 bytes per glyph, sorted by rune: the rune (3 bytes, big
	// endian) and the offset of the glyph in Data (3 bytes, big endian).
	Index string
	// Data has per glyph: width, height, x advance, x offset, y offset
	// (signed), then the bitmap (1 bit per pixel, row after row, MSB first).
	Data string

	glyph Glyph // returned by GetGlyph (like tinyfont, one at a time)
}

// Glyph is the glyph last returned by Font.GetGlyph.
type Glyph struct {
	data     string
	off, end int // bitmap in data
	info     tinyfont.GlyphInfo
}

const (
	indexSize  = 6
	headerSize = 5
)

func (f *Font) GetYAdvance() uint8 { return f.YAdvance }

func (f *Font) entry(i int) (r rune, off int) {
	p := f.Index[i*indexSize:]
	r = rune(p[0])<<16 | rune(p[1])<<8 | rune(p[2])
	off = int(p[3])<<16 | int(p[4])<<8 | int(p[5])
	return
}

// GetGlyph returns the glyph of r. Like tinyfont.Font, a rune that is not
// in the font gives an empty glyph with the advance of the first glyph.
func (f *Font) GetGlyph(r rune) tinyfont.Glypher {
	n := len(f.Index) / indexSize
	s, e := 0, n-1
	for s <= e {
		m := (s + e) / 2
		if mr, _ := f.entry(m); mr < r {
			s = m + 1
		} else {
			e = m - 1
		}
	}
	g := &f.glyph
	g.data = f.Data
	if s == n {
		s = -1
	} else if sr, _ := f.entry(s); sr != r {
		s = -1
	}
	if s < 0 {
		_, off := f.entry(0)
		g.info = tinyfont.GlyphInfo{
			Rune:     r,
			XAdvance: f.Data[off+2],
			XOffset:  int8(f.Data[off+3]),
			YOffset:  int8(f.Data[off+4]),
		}
		g.off, g.end = 0, 0
		return g
	}
	_, off := f.entry(s)
	end := len(f.Data)
	if s+1 < n {
		_, end = f.entry(s + 1)
	}
	g.info = tinyfont.GlyphInfo{
		Rune:     r,
		Width:    f.Data[off],
		Height:   f.Data[off+1],
		XAdvance: f.Data[off+2],
		XOffset:  int8(f.Data[off+3]),
		YOffset:  int8(f.Data[off+4]),
	}
	g.off, g.end = off+headerSize, end
	return g
}

func (g *Glyph) Info() tinyfont.GlyphInfo { return g.info }

// Draw sets the pixels of the glyph, in the same order as tinyfont.
func (g *Glyph) Draw(display drivers.Displayer, x, y int16, c color.RGBA) {
	i := g.off
	bitmap := byte(0)
	if i < g.end {
		bitmap = g.data[i]
	}
	bit := 0
	x += int16(g.info.XOffset)
	y += int16(g.info.YOffset)
	for j := int16(0); j < int16(g.info.Height); j++ {
		for k := int16(0); k < int16(g.info.Width); k++ {
			if bitmap&0x80 != 0 {
				display.SetPixel(x+k, y+j, c)
			}
			bitmap <<= 1
			bit++
			if bit > 7 {
				i++
				if i < g.end {
					bitmap = g.data[i]
				}
				bit = 0
			}
		}
	}
}

//go:generate go run ../tools/fontgen ..
