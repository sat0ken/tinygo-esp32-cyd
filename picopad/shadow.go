package picopad

// The shadows of PicoLibSDK (lib_draw.c): DrawRectShadow and
// DrawBlit1Shadow darken the pixels already in the frame buffer to
// shad/16 of their colour. The frame buffer holds palette indices, so the
// darker colour gets an index of its own (the nearest one when the
// palette is full); shadowTab remembers it for every index and level, so
// that a shadow costs a table lookup a pixel.

// shadowTab maps the palette indices to the ones of the darker colours of
// one shadow level.
type shadowTab struct {
	gen  uint32    // the palette generation of the entries
	done [8]uint32 // the entries computed, a bit each
	tab  [256]uint8
}

// Shadow returns col darkened to shad/16 (shad 0..15), computed as
// PicoLibSDK does on RGB565: the sum of col shifted right by 4, 3, 2 and 1
// for the bits 0, 1, 2 and 3 of shad, each channel on its own.
func Shadow(col Color, shad uint8) Color {
	c := uint16(col)
	var c2 uint16
	if shad&1 != 0 {
		c2 = (c >> 4) &^ 0xF79E // keep bits 0, 5, 6, 11
	}
	if shad&2 != 0 {
		c2 += (c >> 3) &^ 0xE71C // keep bits 0-1, 5-7, 11-12
	}
	if shad&4 != 0 {
		c2 += (c >> 2) &^ 0xC618 // keep bits 0-2, 5-8, 11-13
	}
	if shad&8 != 0 {
		c2 += (c >> 1) &^ 0x8410 // keep bits 0-3, 5-9, 11-14
	}
	return Color(c2)
}

// shadow returns the index of the colour of index e darkened to shad/16.
func (c *Console) shadow(e, shad uint8) uint8 {
	shad &= 15
	s := c.shadows[shad]
	if s == nil {
		s = &shadowTab{gen: c.gen}
		c.shadows[shad] = s
	}
	if s.gen != c.gen {
		s.gen = c.gen
		s.done = [8]uint32{}
	}
	if bit := uint32(1) << (e & 31); s.done[e>>5]&bit == 0 {
		s.tab[e] = c.index(Shadow(Color(c.pal[e]), shad))
		s.done[e>>5] |= bit
	}
	return s.tab[e]
}

// DrawRectShadow darkens a rectangle to shad/16.
func (c *Console) DrawRectShadow(x, y, w, h int, shad uint8) {
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
	c.DispDirtyRect(x, y, w, h)
	for ; h > 0; h-- {
		d := c.pix[y*Width+x : y*Width+x+w]
		for i, e := range d {
			d[i] = c.shadow(e, shad)
		}
		y++
	}
}

// DrawBlit1Shadow darkens to shad/16 the pixels where the 1-bit image src
// (ws pixels wide, the high bit first) has a 1.
func (c *Console) DrawBlit1Shadow(src string, xs, ys, xd, yd, w, h, ws int, shad uint8) {
	xs, ys, xd, yd, w, h, ok := clipImg(xs, ys, xd, yd, w, h, ws)
	if !ok {
		return
	}
	c.DispDirtyRect(xd, yd, w, h)
	pitch := (ws + 7) / 8
	for ; h > 0; h-- {
		d := c.pix[yd*Width+xd : yd*Width+xd+w]
		row := src[ys*pitch:]
		for i := range d {
			if x := xs + i; row[x/8]<<(x&7)&0x80 != 0 {
				d[i] = c.shadow(d[i], shad)
			}
		}
		ys++
		yd++
	}
}
