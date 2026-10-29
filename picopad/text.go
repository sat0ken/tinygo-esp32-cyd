package picopad

// The text functions of PicoLibSDK (_lib/src/lib_draw.c): characters of
// the current font, transparent or with their background, at normal,
// double height (H), double width (W) or double size (2).

// Font is a PicoLibSDK font: 256 characters, a byte per row (the left
// pixel in the high bit); row i of character ch is data[i*256+ch].
type Font struct {
	data string
	w, h int
}

// The fonts of PicoLibSDK the games use (_font/*.c).
var (
	FontBold8x8  = Font{fontBold8x8, 8, 8}
	FontBold8x14 = Font{fontBold8x14, 8, 14}
	FontBold8x16 = Font{fontBold8x16, 8, 16} // the default
	FontCond6x8  = Font{fontCond6x8, 6, 8}
	FontTiny5x8  = Font{fontTiny5x8, 5, 8}
)

// SetFont selects the font (pDrawFont, DrawFontWidth and DrawFontHeight
// of PicoLibSDK).
func (c *Console) SetFont(f Font) { c.font = f }

// SelFont8x8 selects FontBold8x8.
func (c *Console) SelFont8x8() { c.font = FontBold8x8 }

// SelFont8x14 selects FontBold8x14.
func (c *Console) SelFont8x14() { c.font = FontBold8x14 }

// SelFont8x16 selects FontBold8x16.
func (c *Console) SelFont8x16() { c.font = FontBold8x16 }

// SelFont6x8 selects FontCond6x8.
func (c *Console) SelFont6x8() { c.font = FontCond6x8 }

// SelFont5x8 selects FontTiny5x8.
func (c *Console) SelFont5x8() { c.font = FontTiny5x8 }

// DrawFontWidth returns the width of a character of the font.
func (c *Console) DrawFontWidth() int { return c.font.w }

// DrawFontHeight returns the height of a character of the font.
func (c *Console) DrawFontHeight() int { return c.font.h }

// drawChar draws character ch of the font sx times as wide and sy times
// as high, with the indices fg and bg (opaque: also the background).
func (c *Console) drawChar(ch byte, x, y, sx, sy int, fg, bg uint8, opaque bool) {
	f := &c.font
	w, h := f.w*sx, f.h*sy
	if x+w <= 0 || x >= Width || y+h <= 0 || y >= Height {
		return
	}
	c.DispDirtyRect(x, y, w, h) // clips
	for i := 0; i < h; i++ {
		py := y + i
		if py < 0 || py >= Height {
			continue
		}
		bits := f.data[i/sy*256+int(ch)]
		for j := 0; j < w; j++ {
			px := x + j
			on := bits&(0x80>>(j/sx)) != 0
			if px < 0 || px >= Width || !on && !opaque {
				continue
			}
			if on {
				c.pix[py*Width+px] = fg
			} else {
				c.pix[py*Width+px] = bg
			}
		}
	}
}

// drawText draws a text, each character sx times as wide and sy times as
// high.
func (c *Console) drawText(text string, x, y, sx, sy int, fg, bg uint8, opaque bool) {
	for i := 0; i < len(text); i++ {
		c.drawChar(text[i], x, y, sx, sy, fg, bg, opaque)
		x += c.font.w * sx
	}
}

// DrawChar draws a character (transparent background).
func (c *Console) DrawChar(ch byte, x, y int, col Color) {
	c.drawChar(ch, x, y, 1, 1, c.index(col), 0, false)
}

// DrawCharH draws a character, double height.
func (c *Console) DrawCharH(ch byte, x, y int, col Color) {
	c.drawChar(ch, x, y, 1, 2, c.index(col), 0, false)
}

// DrawCharW draws a character, double width.
func (c *Console) DrawCharW(ch byte, x, y int, col Color) {
	c.drawChar(ch, x, y, 2, 1, c.index(col), 0, false)
}

// DrawChar2 draws a character, double size.
func (c *Console) DrawChar2(ch byte, x, y int, col Color) {
	c.drawChar(ch, x, y, 2, 2, c.index(col), 0, false)
}

// DrawCharBg draws a character with its background.
func (c *Console) DrawCharBg(ch byte, x, y int, col, bgcol Color) {
	fg := c.index(col)
	c.drawChar(ch, x, y, 1, 1, fg, c.index(bgcol), true)
}

// DrawText draws a text (transparent background).
func (c *Console) DrawText(text string, x, y int, col Color) {
	c.drawText(text, x, y, 1, 1, c.index(col), 0, false)
}

// DrawTextH draws a text, double height.
func (c *Console) DrawTextH(text string, x, y int, col Color) {
	c.drawText(text, x, y, 1, 2, c.index(col), 0, false)
}

// DrawTextW draws a text, double width.
func (c *Console) DrawTextW(text string, x, y int, col Color) {
	c.drawText(text, x, y, 2, 1, c.index(col), 0, false)
}

// DrawText2 draws a text, double size.
func (c *Console) DrawText2(text string, x, y int, col Color) {
	c.drawText(text, x, y, 2, 2, c.index(col), 0, false)
}

// DrawTextBg draws a text with its background.
func (c *Console) DrawTextBg(text string, x, y int, col, bgcol Color) {
	fg := c.index(col)
	c.drawText(text, x, y, 1, 1, fg, c.index(bgcol), true)
}

// DrawTextBgH draws a text with its background, double height.
func (c *Console) DrawTextBgH(text string, x, y int, col, bgcol Color) {
	fg := c.index(col)
	c.drawText(text, x, y, 1, 2, fg, c.index(bgcol), true)
}

// DrawTextBgW draws a text with its background, double width.
func (c *Console) DrawTextBgW(text string, x, y int, col, bgcol Color) {
	fg := c.index(col)
	c.drawText(text, x, y, 2, 1, fg, c.index(bgcol), true)
}

// DrawTextBg2 draws a text with its background, double size.
func (c *Console) DrawTextBg2(text string, x, y int, col, bgcol Color) {
	fg := c.index(col)
	c.drawText(text, x, y, 2, 2, fg, c.index(bgcol), true)
}
