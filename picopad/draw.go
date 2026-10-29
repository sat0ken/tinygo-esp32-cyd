package picopad

// The drawing functions of PicoLibSDK (_lib/src/lib_draw.c), with the same
// clipping. They draw into the frame buffer and add what they drew to the
// part of the screen DispUpdate sends.

// DrawPoint draws one pixel.
func (c *Console) DrawPoint(x, y int, col Color) {
	if uint(x) >= Width || uint(y) >= Height {
		return
	}
	c.pix[y*Width+x] = c.index(col)
	c.DispDirtyRect(x, y, 1, 1)
}

// DrawGetPoint returns the colour of one pixel of the frame buffer (black
// out of the screen). Games compare it with the colours they drew, which
// it returns unchanged while the frame buffer has at most 256 colours.
func (c *Console) DrawGetPoint(x, y int) Color {
	if uint(x) >= Width || uint(y) >= Height {
		return ColBlack
	}
	return Color(c.pal[c.pix[y*Width+x]])
}

// DrawRect fills a rectangle (a negative w or h extends to the left or
// up).
func (c *Console) DrawRect(x, y, w, h int, col Color) {
	if w < 0 {
		x += w
		w = -w
	}
	if h < 0 {
		y += h
		h = -h
	}
	if x < 0 {
		w += x
		x = 0
	}
	if x+w > Width {
		w = Width - x
	}
	if w <= 0 {
		return
	}
	if y < 0 {
		h += y
		y = 0
	}
	if y+h > Height {
		h = Height - y
	}
	if h <= 0 {
		return
	}
	if w == Width && h == Height {
		c.freePalette() // nothing of the old screen stays
	}
	c.DispDirtyRect(x, y, w, h)
	v := c.index(col)
	for ; h > 0; h-- {
		row := c.pix[y*Width+x : y*Width+x+w]
		for i := range row {
			row[i] = v
		}
		y++
	}
}

// DrawFrameW draws the edge of a rectangle, width pixels wide.
func (c *Console) DrawFrameW(x, y, w, h int, col Color, width int) {
	if w < 0 {
		x += w
		w = -w
	}
	if h < 0 {
		y += h
		h = -h
	}
	if w == 0 || h == 0 {
		return
	}
	c.DrawRect(x, y, w-width, width, col)
	c.DrawRect(x+w-width, y, width, h-width, col)
	c.DrawRect(x+width, y+h-width, w-width, width, col)
	c.DrawRect(x, y+width, width, h-width, col)
}

// DrawFrame draws the edge of a rectangle, one pixel wide.
func (c *Console) DrawFrame(x, y, w, h int, col Color) { c.DrawFrameW(x, y, w, h, col, 1) }

// DrawClearCol fills the screen with col.
func (c *Console) DrawClearCol(col Color) { c.DrawRect(0, 0, Width, Height, col) }

// DrawClear fills the screen with black.
func (c *Console) DrawClear() { c.DrawClearCol(ColBlack) }

// clipImg clips an image of source width ws drawn at xd, yd: it returns
// the new coordinates and size, and ok = false if nothing is left. The
// source is not clipped at its height (as in PicoLibSDK).
func clipImg(xs, ys, xd, yd, w, h, ws int) (int, int, int, int, int, int, bool) {
	if xs < 0 {
		w += xs
		xd -= xs
		xs = 0
	}
	if xd < 0 {
		w += xd
		xs -= xd
		xd = 0
	}
	if xd+w > Width {
		w = Width - xd
	}
	if xs+w > ws {
		w = ws - xs
	}
	if w <= 0 {
		return 0, 0, 0, 0, 0, 0, false
	}
	if ys < 0 {
		h += ys
		yd -= ys
		ys = 0
	}
	if yd < 0 {
		h += yd
		ys -= yd
		yd = 0
	}
	if yd+h > Height {
		h = Height - yd
	}
	if h <= 0 {
		return 0, 0, 0, 0, 0, 0, false
	}
	return xs, ys, xd, yd, w, h, true
}

// DrawImgPal draws the part xs, ys, w x h of an 8-bit paletted image src
// (ws pixels wide) at xd, yd.
func (c *Console) DrawImgPal(src string, pal Palette, xs, ys, xd, yd, w, h, ws int) {
	xs, ys, xd, yd, w, h, ok := clipImg(xs, ys, xd, yd, w, h, ws)
	if !ok {
		return
	}
	c.DispDirtyRect(xd, yd, w, h)
	tab := c.table(pal)
	for ; h > 0; h-- {
		d := c.pix[yd*Width+xd : yd*Width+xd+w]
		s := src[ys*ws+xs : ys*ws+xs+w]
		for i := range d {
			d[i] = tab[s[i]]
		}
		ys++
		yd++
	}
}

// DrawImg4Pal is DrawImgPal for a 4-bit paletted image (two pixels a
// byte, the left one in the high bits).
func (c *Console) DrawImg4Pal(src string, pal Palette, xs, ys, xd, yd, w, h, ws int) {
	xs, ys, xd, yd, w, h, ok := clipImg(xs, ys, xd, yd, w, h, ws)
	if !ok {
		return
	}
	c.DispDirtyRect(xd, yd, w, h)
	tab := c.table(pal)
	pitch := (ws + 1) / 2
	for ; h > 0; h-- {
		d := c.pix[yd*Width+xd : yd*Width+xd+w]
		for i := range d {
			d[i] = tab[nibble(src, ys*pitch, xs+i)]
		}
		ys++
		yd++
	}
}

// nibble returns pixel x of the 4-bit row that starts at src[row].
func nibble(src string, row, x int) byte {
	b := src[row+x/2]
	if x&1 == 0 {
		return b >> 4
	}
	return b & 0x0F
}

// DrawBlitPal is DrawImgPal without the pixels whose colour is col
// (transparent).
func (c *Console) DrawBlitPal(src string, pal Palette, xs, ys, xd, yd, w, h, ws int, col Color) {
	xs, ys, xd, yd, w, h, ok := clipImg(xs, ys, xd, yd, w, h, ws)
	if !ok {
		return
	}
	c.DispDirtyRect(xd, yd, w, h)
	tab := c.table(pal)
	for ; h > 0; h-- {
		d := c.pix[yd*Width+xd : yd*Width+xd+w]
		s := src[ys*ws+xs : ys*ws+xs+w]
		for i := range d {
			if e := s[i]; pal.color(int(e)) != col {
				d[i] = tab[e]
			}
		}
		ys++
		yd++
	}
}

// DrawBlit4Pal is DrawImg4Pal without the pixels whose colour is col
// (transparent).
func (c *Console) DrawBlit4Pal(src string, pal Palette, xs, ys, xd, yd, w, h, ws int, col Color) {
	xs, ys, xd, yd, w, h, ok := clipImg(xs, ys, xd, yd, w, h, ws)
	if !ok {
		return
	}
	c.DispDirtyRect(xd, yd, w, h)
	tab := c.table(pal)
	pitch := (ws + 1) / 2
	for ; h > 0; h-- {
		d := c.pix[yd*Width+xd : yd*Width+xd+w]
		for i := range d {
			if e := nibble(src, ys*pitch, xs+i); pal.color(int(e)) != col {
				d[i] = tab[e]
			}
		}
		ys++
		yd++
	}
}

// rleImage decodes the RLE images of PicoLibSDK. A code c of bits bits
// (8 or 4) of half or more (128 or 8) repeats the next pixel c-half+3
// times; a smaller code is followed by c+1 pixels. The stream goes on
// from one row to the next.
type rleImage struct {
	src  string
	bits int // 8 or 4
	pos  int // the next byte, or the next nibble
	rle  int // pixels left of a repeated pixel
	raw  int // pixels left of a raw run
	pix  byte
}

// next returns the next code or pixel of the stream.
func (r *rleImage) next() byte {
	if r.bits == 8 {
		b := r.src[r.pos]
		r.pos++
		return b
	}
	b := nibble(r.src, 0, r.pos)
	r.pos++
	return b
}

// pixel returns the next pixel (a palette entry).
func (r *rleImage) pixel() byte {
	if r.rle == 0 && r.raw == 0 {
		half := byte(1) << (r.bits - 1)
		if b := r.next(); b >= half {
			r.rle = int(b-half) + 3
			r.pix = r.next()
		} else {
			r.raw = int(b) + 1
		}
	}
	if r.rle != 0 {
		r.rle--
		return r.pix
	}
	r.raw--
	return r.next()
}

// drawRle draws a w x h RLE image at xd, yd. As in PicoLibSDK the rows
// above the screen are decoded and skipped and the image is cut at the
// bottom; the pixels left or right of the screen (which PicoLibSDK would
// write out of its buffer) are skipped too.
func (c *Console) drawRle(r *rleImage, pal Palette, xd, yd, w, h int) {
	c.DispDirtyRect(xd, yd, w, h)
	tab := c.table(pal)
	if yd+h > Height {
		h = Height - yd
	}
	for ; h > 0; h-- {
		for i := 0; i < w; i++ {
			p := tab[r.pixel()]
			if x := xd + i; yd >= 0 && x >= 0 && x < Width {
				c.pix[yd*Width+x] = p
			}
		}
		yd++
	}
}

// DrawImgRle draws an 8-bit paletted image compressed with RLE.
func (c *Console) DrawImgRle(src string, pal Palette, xd, yd, w, h int) {
	c.drawRle(&rleImage{src: src, bits: 8}, pal, xd, yd, w, h)
}

// DrawImg4Rle draws a 4-bit paletted image compressed with RLE.
func (c *Console) DrawImg4Rle(src string, pal Palette, xd, yd, w, h int) {
	c.drawRle(&rleImage{src: src, bits: 4}, pal, xd, yd, w, h)
}
