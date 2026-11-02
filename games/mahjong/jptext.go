package mahjong

import (
	"image/color"

	"github.com/sat0ken/tinygo-cyd/hal"
)

// Japanese text in two sizes (font_data.go, from tools/jpfontgen): scale 1 is
// 8 x 8 (Misaki Gothic, with the digits of the tiles), scale 2 is 16 x 16
// (Shinonome), and larger even scales are 16 x 16 made bigger.

const (
	smallH = 7 // the height of a small character (the cell has an empty row below)
	bigH   = 16
)

// jpHeight is the height of a text at a scale.
func jpHeight(scale int16) int16 {
	if scale == 1 {
		return smallH
	}
	return bigH * (scale / 2)
}

// runeIndex is where a character is in the fonts, -1 if it is not.
func runeIndex(r rune) int {
	lo, hi := 0, len(fontRunes)
	for lo < hi {
		m := (lo + hi) / 2
		switch {
		case rune(fontRunes[m]) == r:
			return m
		case rune(fontRunes[m]) < r:
			lo = m + 1
		default:
			hi = m
		}
	}
	return -1
}

// signs is the signs of the small text, 5 wide.
var signs = [...]struct {
	r    rune
	rows [smallH]uint8
}{
	{'+', [smallH]uint8{0b00000, 0b00100, 0b00100, 0b11111, 0b00100, 0b00100, 0b00000}},
	{'-', [smallH]uint8{0b00000, 0b00000, 0b00000, 0b11111, 0b00000, 0b00000, 0b00000}},
	{'?', [smallH]uint8{0b01110, 0b10001, 0b00001, 0b00010, 0b00100, 0b00000, 0b00100}},
}

var digit0 = [7]string{".###.", "#...#", "#...#", "#...#", "#...#", "#...#", ".###."}

// runeWidth is how far a rune moves the text at a scale.
func runeWidth(r rune, scale int16) int16 {
	if scale == 1 {
		switch {
		case r == ' ':
			return 4
		case r < 0x80:
			return 6
		}
		return 8
	}
	if r < 0x80 {
		return 8 * (scale / 2)
	}
	return 16 * (scale / 2)
}

// jpWidth is the width of a text at a scale.
func jpWidth(s string, scale int16) int16 {
	w := int16(0)
	for _, r := range s {
		w += runeWidth(r, scale)
	}
	return w
}

// drawRow draws one row of a glyph: bit n-1 of v is the leftmost pixel.
func drawRow(d hal.Display, v uint16, n, x, y, px int16, c color.RGBA) {
	for i := int16(0); i < n; {
		if v&(1<<(n-1-i)) == 0 {
			i++
			continue
		}
		start := i
		for i < n && v&(1<<(n-1-i)) != 0 {
			i++
		}
		d.FillRectangle(x+start*px, y, (i-start)*px, px, c)
	}
}

// jpText draws a text with its top left at (x, y) and returns the x after it.
func jpText(d hal.Display, x, y int16, s string, scale int16, c color.RGBA) int16 {
	for _, r := range s {
		if scale == 1 {
			drawSmall(d, r, x, y, c)
		} else {
			drawBig(d, r, x, y, scale/2, c)
		}
		x += runeWidth(r, scale)
	}
	return x
}

func drawSmall(d hal.Display, r rune, x, y int16, c color.RGBA) {
	switch {
	case r >= '1' && r <= '9':
		glyph(d, digits5[r-'1'][:], x, y, 1, c)
	case r == '0':
		glyph(d, digit0[:], x, y, 1, c)
	case r < 0x80:
		for i := range signs {
			if signs[i].r == r {
				for j, v := range signs[i].rows {
					drawRow(d, uint16(v), 5, x, y+int16(j), 1, c)
				}
			}
		}
	default:
		if i := runeIndex(r); i >= 0 {
			for j := 0; j < 8; j++ {
				drawRow(d, uint16(fontSmall[i*8+j]), 8, x, y+int16(j), 1, c)
			}
		}
	}
}

func drawBig(d hal.Display, r rune, x, y, px int16, c color.RGBA) {
	if r < 0x80 {
		for i := 0; i < len(bigASCII); i++ {
			if rune(bigASCII[i]) == r {
				for j := 0; j < bigH; j++ {
					drawRow(d, uint16(fontBigASCII[i*bigH+j]), 8, x, y+int16(j)*px, px, c)
				}
			}
		}
		return
	}
	if i := runeIndex(r); i >= 0 {
		for j := 0; j < bigH; j++ {
			v := uint16(fontBig[i*2*bigH+2*j])<<8 | uint16(fontBig[i*2*bigH+2*j+1])
			drawRow(d, v, 16, x, y+int16(j)*px, px, c)
		}
	}
}
