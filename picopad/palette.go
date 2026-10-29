package picopad

// Palettes for the games that draw many more than 256 colours (PICTOR:
// three background layers, the hero and the enemies of a scene, each an
// image of 256 colours, and the background drawn twice as large with the
// colours between the pixels blended). The game makes a palette for each
// scene (internal/palquant), SetPalette puts it in the frame buffer, and
// every colour drawn then gets its nearest entry.

// colorCache maps colours (4 bits of red, green and blue) to the indices
// of their nearest entries, for the colours that are made while drawing
// (blends), which have no palette table.
type colorCache struct {
	gen  uint32 // Console.gen of the entries
	done [4096 / 32]uint32
	inx  [4096]uint8
}

// SetPalette makes pal (up to 256 colours) the palette of the frame
// buffer: every colour drawn after it gets its nearest entry (a new entry
// while pal has fewer than 256). Call it just before the whole screen is
// drawn again: the pixels already drawn keep their indices, which now mean
// the colours of pal. (The PicoPad has a 16-bit frame buffer and no such
// function.)
func (c *Console) SetPalette(pal Palette) {
	c.freePalette()
	n := min(len(pal)/2, len(c.pal))
	for i := 0; i < n; i++ {
		c.pal[i] = uint16(pal.color(i))
	}
	c.ncol = n
}

// nearest returns the index of col, through a cache of 4096 colours (the
// colours of a cell of the cache share the index of the first one).
func (c *Console) nearest(col Color) uint8 {
	cc := c.colors
	if cc == nil {
		cc = &colorCache{gen: c.gen}
		c.colors = cc
	}
	if cc.gen != c.gen {
		cc.gen = c.gen
		cc.done = [4096 / 32]uint32{}
	}
	k := (col>>12&0xF)<<8 | (col>>7&0xF)<<4 | (col >> 1 & 0xF)
	if bit := uint32(1) << (k & 31); cc.done[k>>5]&bit == 0 {
		cc.inx[k] = c.index(col)
		cc.done[k>>5] |= bit
	}
	return cc.inx[k]
}

// MapRect changes the colour of each pixel of a rectangle to f(colour),
// computing f once for each palette index there.
func (c *Console) MapRect(x, y, w, h int, f func(Color) Color) {
	x1, x2 := max(x, 0), min(x+w, Width)
	y1, y2 := max(y, 0), min(y+h, Height)
	if x1 >= x2 || y1 >= y2 {
		return
	}
	c.DispDirtyRect(x1, y1, x2-x1, y2-y1)
	var m [256]uint8
	var done [256]bool
	for row := y1; row < y2; row++ {
		d := c.pix[row*Width+x1 : row*Width+x2]
		for i, e := range d {
			if !done[e] {
				m[e] = c.index(f(Color(c.pal[e])))
				done[e] = true
			}
			d[i] = m[e]
		}
	}
}

// blend2 is the colour between a and b (Col16Blend2 of PICTOR).
func blend2(a, b Color) Color {
	rb := ((uint32(a)&0xf81f + uint32(b)&0xf81f) >> 1) & 0xf81f
	g := ((uint32(a)&0x07e0 + uint32(b)&0x07e0) >> 1) & 0x07e0
	return Color(rb | g)
}

// blend4 is the colour between a, b, c and d (Col16Blend4 of PICTOR).
func blend4(a, b, c, d Color) Color {
	rb := ((uint32(a)&0xf81f + uint32(b)&0xf81f + uint32(c)&0xf81f + uint32(d)&0xf81f) >> 2) & 0xf81f
	g := ((uint32(a)&0x07e0 + uint32(b)&0x07e0 + uint32(c)&0x07e0 + uint32(d)&0x07e0) >> 2) & 0x07e0
	return Color(rb | g)
}

// DrawImgPalDbl draws an 8-bit image twice as large, the pixels between
// blended (DrawImgPalDbl of PICTOR's disp.cpp, not of the SDK). The
// coordinates and sizes of the source are those of the doubled image: xs
// (can be odd), ys (even), w, h and ws are twice the source's; a row
// wraps around (the pixel left of column 0 is the last one).
func (c *Console) DrawImgPalDbl(src string, pal Palette, xs, ys, xd, yd, w, h, ws int) {
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
		return
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
		return
	}
	c.DispDirtyRect(xd, yd, w, h)

	tab := c.table(pal)
	ws /= 2
	h /= 2
	d := xd + yd*Width
	s0 := xs/2 + (ys/2)*ws
	col := func(s int) Color { return pal.color(int(src[s])) }

	// a row as drawn: the odd columns are pixels, the even ones blends
	row := func(d, s int) {
		i := w
		var last Color
		if xs&1 != 0 {
			last = col(s)
			c.pix[d] = tab[src[s]]
			d++
			s++
			i--
		} else if xs > 0 {
			last = col(s - 1)
		} else {
			last = col(s + ws - 1)
		}
		for ; i > 1; i -= 2 {
			cur := col(s)
			c.pix[d] = c.nearest(blend2(last, cur))
			c.pix[d+1] = tab[src[s]]
			d += 2
			s++
			last = cur
		}
		if i > 0 {
			c.pix[d] = c.nearest(blend2(last, col(s)))
		}
	}

	for ; h > 0; h-- {
		row(d, s0)
		d += Width
		if h == 1 { // the last line is not blended with the next
			row(d, s0)
			break
		}

		// the line between: blended with the next source line
		s := s0
		dd := d
		i := w
		var last, last2 Color
		if xs&1 != 0 {
			last, last2 = col(s), col(s+ws)
			c.pix[dd] = c.nearest(blend2(last, last2))
			dd++
			s++
			i--
		} else if xs > 0 {
			last, last2 = col(s-1), col(s+ws-1)
		} else {
			last, last2 = col(s+ws-1), col(s+2*ws-1)
		}
		for ; i > 1; i -= 2 {
			cur, cur2 := col(s), col(s+ws)
			c.pix[dd] = c.nearest(blend4(last, cur, last2, cur2))
			c.pix[dd+1] = c.nearest(blend2(cur, cur2))
			dd += 2
			s++
			last, last2 = cur, cur2
		}
		if i > 0 {
			cur, cur2 := col(s), col(s+ws)
			c.pix[dd] = c.nearest(blend4(last, cur, last2, cur2))
		}
		d += Width
		s0 += ws
	}
}
