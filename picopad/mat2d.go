package picopad

import "math"

// The 2D transformation matrix of PicoLibSDK (_lib/src/lib_mat2d.c) and
// the drawing functions that use it (lib_draw.c): DrawImg4PalMat draws a
// 4-bit image scaled, rotated or sheared, DrawTile4PalMap draws a map of
// tiles in perspective (a floor seen from above, as in GHOSTRAC). They
// compute as the SDK does: the matrix in float32, the pixels in 32-bit
// fixed point with 12 fractional bits.

const fract = 12 // fractional bits of the fixed point numbers (FRACT)

// Mat2D is a 2D transformation matrix (sMat2D): it maps a destination
// pixel (x, y, from the middle of the destination) to the source image.
type Mat2D struct {
	M11, M12, M13 float32
	M21, M22, M23 float32
}

// The modes of DrawImg4PalMat (DRAWIMG_*).
const (
	DrawImgWrap     = 0 // wrap the image
	DrawImgNoBorder = 1 // no border (transparent)
	DrawImgClamp    = 2 // clamp: the last pixel is the border
	DrawImgColor    = 3 // the border is a colour
	DrawImgTransp   = 4 // a transparent key colour
)

// PrepDrawImg prepares the matrix (Mat2D_PrepDrawImg) for an image ws x hs
// with the reference point x0, y0, drawn wd x hd (negative: flipped),
// sheared, rotated by rot radians and shifted by tx, ty. A size of 0 gives
// the zero matrix (nothing is drawn).
func (m *Mat2D) PrepDrawImg(ws, hs, x0, y0, wd, hd int, shearx, sheary, rot, tx, ty float32) {
	if wd == 0 || hd == 0 {
		*m = Mat2D{}
		return
	}
	*m = Mat2D{M11: 1, M22: 1} // unit
	m.M13 += tx
	m.M23 += ty
	m.rot(rot)
	m.shearX(shearx)
	m.shearY(sheary)
	m.scaleX(1.0 / float32(wd))
	m.scaleY(1.0 / float32(hd))
	m.scaleX(float32(ws))
	m.scaleY(float32(hs))
	m.M13 += float32(x0)
	m.M23 += float32(y0)
}

func (m *Mat2D) rot(a float32) {
	sina := float32(math.Sin(float64(a)))
	cosa := float32(math.Cos(float64(a)))
	t1, t2 := m.M11, m.M21
	m.M11 = t1*cosa - t2*sina
	m.M21 = t1*sina + t2*cosa
	t1, t2 = m.M12, m.M22
	m.M12 = t1*cosa - t2*sina
	m.M22 = t1*sina + t2*cosa
	t1, t2 = m.M13, m.M23
	m.M13 = t1*cosa - t2*sina
	m.M23 = t1*sina + t2*cosa
}

func (m *Mat2D) shearX(dx float32) {
	m.M11 += m.M21 * dx
	m.M12 += m.M22 * dx
	m.M13 += m.M23 * dx
}

func (m *Mat2D) shearY(dy float32) {
	m.M21 += m.M11 * dy
	m.M22 += m.M12 * dy
	m.M23 += m.M13 * dy
}

func (m *Mat2D) scaleX(sx float32) {
	m.M11 *= sx
	m.M12 *= sx
	m.M13 *= sx
}

func (m *Mat2D) scaleY(sy float32) {
	m.M21 *= sy
	m.M22 *= sy
	m.M23 *= sy
}

// toFract converts to fixed point, rounding down (TOFRACT, float2int).
func toFract(f float32) int32 {
	return int32(math.Floor(float64(f * (1 << fract))))
}

// fixed returns the matrix in fixed point.
func (m *Mat2D) fixed() (m11, m12, m13, m21, m22, m23 int32) {
	return toFract(m.M11), toFract(m.M12), toFract(m.M13), toFract(m.M21), toFract(m.M22), toFract(m.M23)
}

// clipMat clips the destination rectangle and returns the start of x0 and
// y0 (the coordinates from the middle) and h0, the height before the
// clipping.
func clipMat(x, y, w, h int, persp bool) (x2, y2, w2, h2 int, x0, y0, h0 int32, ok bool) {
	x0 = int32(-w / 2)
	if x < 0 {
		w += x
		x0 -= int32(x)
		x = 0
	}
	if x+w > Width {
		w = Width - x
	}
	if w <= 0 {
		return
	}
	h0 = int32(h)
	y0 = int32(-h / 2)
	if persp {
		y0 = int32(-h)
	}
	if y < 0 {
		h += y
		y0 -= int32(y)
		y = 0
	}
	if y+h > Height {
		h = Height - y
	}
	if h <= 0 {
		return
	}
	return x, y, w, h, x0, y0, h0, true
}

// DrawImg4PalMat draws the 4-bit image src (ws x hs) transformed by m into
// the rectangle x, y, w, h. mode says what is out of the image
// (DrawImg*); col is the border colour or the transparent key colour.
func (c *Console) DrawImg4PalMat(src string, pal Palette, ws, hs, x, y, w, h int, m *Mat2D, mode uint8, col Color) {
	x, y, w, h, x0, y0, _, ok := clipMat(x, y, w, h, false)
	if !ok {
		return
	}
	c.DispDirtyRect(x, y, w, h)
	m11, m12, m13, m21, m22, m23 := m.fixed()
	zero := m11|m12|m13|m21|m22|m23 == 0
	if zero && mode != DrawImgClamp && mode != DrawImgColor && mode != DrawImgWrap {
		return
	}
	tab := c.table(pal)
	var colInx uint8
	if mode == DrawImgColor {
		colInx = c.index(col)
	}
	pitch := int32(ws / 2)
	for ; h > 0; h-- {
		xy0m := x0*m11 + y0*m12 + m13
		yy0m := x0*m21 + y0*m22 + m23
		d := c.pix[y*Width+x : y*Width+x+w]
		for i := range d {
			x2, y2 := xy0m>>fract, yy0m>>fract
			xy0m += m11
			yy0m += m21
			in := uint32(x2) < uint32(ws) && uint32(y2) < uint32(hs)
			switch mode {
			case DrawImgWrap:
				x2 %= int32(ws)
				if x2 < 0 {
					x2 += int32(ws)
				}
				y2 %= int32(hs)
				if y2 < 0 {
					y2 += int32(hs)
				}
				d[i] = tab[nibbleAt(src, pitch, x2, y2)]
			case DrawImgNoBorder:
				if in {
					d[i] = tab[nibbleAt(src, pitch, x2, y2)]
				}
			case DrawImgClamp:
				if zero {
					d[i] = tab[src[0]>>4]
					continue
				}
				x2 = min(max(x2, 0), int32(ws-1))
				y2 = min(max(y2, 0), int32(hs-1))
				d[i] = tab[nibbleAt(src, pitch, x2, y2)]
			case DrawImgColor:
				if in && !zero {
					d[i] = tab[nibbleAt(src, pitch, x2, y2)]
				} else {
					d[i] = colInx
				}
			case DrawImgTransp:
				if in {
					if b := nibbleAt(src, pitch, x2, y2); pal.color(int(b)) != col {
						d[i] = tab[b]
					}
				}
			}
		}
		y0++
		y++
	}
}

// nibbleAt returns the pixel x, y of a 4-bit image.
func nibbleAt(src string, pitch, x, y int32) byte {
	b := src[x/2+y*pitch]
	if x&1 == 0 {
		return b >> 4
	}
	return b & 0x0f
}

// DrawTile4PalMap draws a map of square 4-bit tiles (src: a column of
// tiles 1<<tilebits wide; tilemap: the tile of each place, a byte each,
// 1<<mapwbits x 1<<maphbits) into the rectangle x, y, w, h transformed by
// m, in perspective when horizon is not 0 (the larger, the flatter).
func (c *Console) DrawTile4PalMap(src string, pal Palette, tilemap string, mapwbits, maphbits, tilebits, x, y, w, h int, m *Mat2D, horizon uint8) {
	x, y, w, h, x0, y0, h0, ok := clipMat(x, y, w, h, horizon != 0)
	if !ok {
		return
	}
	c.DispDirtyRect(x, y, w, h)
	m11, m12, m13, m21, m22, m23 := m.fixed()
	m11b, m12b, m21b, m22b := m11, m12, m21, m22
	tab := c.table(pal)

	tilesize := int32(1) << tilebits
	tilemask := tilesize - 1
	mapw := int32(1) << mapwbits
	maph := int32(1) << maphbits
	mapmaskx := mapw*tilesize - 1
	mapmasky := maph*tilesize - 1

	for ; h > 0; h-- {
		if horizon != 0 {
			dist := (1 << fract) * h0 / (h0 + y0 + int32(horizon))
			m11b = (m11 * dist) >> fract
			m12b = (m12 * dist) >> fract
			m21b = (m21 * dist) >> fract
			m22b = (m22 * dist) >> fract
		}
		xy0m := x0*m11b + y0*m12b + m13
		yy0m := x0*m21b + y0*m22b + m23
		d := c.pix[y*Width+x : y*Width+x+w]
		for i := range d {
			x2 := (xy0m >> fract) & mapmaskx
			xy0m += m11b
			y2 := (yy0m >> fract) & mapmasky
			yy0m += m21b

			tileinx := int32(tilemap[(x2>>tilebits)+(y2>>tilebits)*mapw])
			bb := src[tileinx<<(2*tilebits-1)+(x2&tilemask)/2+(y2&tilemask)<<(tilebits-1)]
			if x2&1 == 0 {
				bb >>= 4
			} else {
				bb &= 0x0f
			}
			d[i] = tab[bb]
		}
		y0++
		y++
	}
}
