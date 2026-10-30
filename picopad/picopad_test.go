package picopad

import (
	"math"
	"testing"
	"time"

	"github.com/sat0ken/tinygo-cyd/framebuf"
	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/internal/picotest"
	"github.com/sat0ken/tinygo-cyd/memlcd"
)

func newTest(t *testing.T, w, h int16) (*Console, *memlcd.Display, *picotest.Clock, *picotest.Pad) {
	d := memlcd.New(w, h)
	c := New(d, nil, 1)
	clock := picotest.NewClock(t, time.Hour)
	c.Clock = clock
	p := &picotest.Pad{Clock: clock}
	c.Pad = p
	return c, d, clock, p
}

// at returns the colour of pixel x, y of the console's screen on d.
func at(c *Console, d *memlcd.Display, x, y int) Color {
	return Color(d.Pix[(y+int(c.oy))*int(d.W)+x+int(c.ox)])
}

// palette returns a Palette of the colours.
func palette(cols ...Color) Palette {
	b := make([]byte, 0, 2*len(cols))
	for _, c := range cols {
		b = append(b, byte(c), byte(c>>8))
	}
	return Palette(b)
}

func TestRGB(t *testing.T) {
	for _, tc := range []struct {
		r, g, b uint8
		want    Color
	}{
		{0, 255, 0, ColGreen}, {255, 255, 0, ColYellow}, {128, 128, 128, ColGray}, {0, 0, 128, 0x0010},
	} {
		if got := RGB(tc.r, tc.g, tc.b); got != tc.want {
			t.Errorf("RGB(%d, %d, %d) = %#04x, want %#04x", tc.r, tc.g, tc.b, got, tc.want)
		}
	}
}

func TestDrawImgPal(t *testing.T) {
	c, d, _, _ := newTest(t, Width, Height)
	pal := palette(ColRed, ColGreen, ColBlue)
	// 4 x 2 pixels
	img := "\x00\x01\x02\x00" + "\x02\x01\x00\x01"
	c.DrawImgPal(img, pal, 0, 0, 10, 20, 4, 2, 4)
	c.DrawImgPal(img, pal, 1, 0, -1, -1, 3, 2, 4)     // clipped: the second row of x 2..3 at 0, 0
	c.DrawImgPal(img, pal, 0, 0, Width-2, 0, 4, 1, 4) // clipped at the right
	c.DispUpdate()
	want := map[[2]int]Color{
		{10, 20}: ColRed, {11, 20}: ColGreen, {12, 20}: ColBlue, {13, 20}: ColRed,
		{10, 21}: ColBlue, {11, 21}: ColGreen, {12, 21}: ColRed, {13, 21}: ColGreen,
		{0, 0}: ColRed, {1, 0}: ColGreen, {2, 0}: ColBlack,
		{Width - 2, 0}: ColRed, {Width - 1, 0}: ColGreen,
	}
	for p, col := range want {
		if got := at(c, d, p[0], p[1]); got != col {
			t.Errorf("pixel %v = %#04x, want %#04x", p, got, col)
		}
	}
}

func TestDrawGetPoint(t *testing.T) {
	c, _, _, _ := newTest(t, Width, Height)
	orange := RGB(255, 128, 0)
	c.DrawRect(10, 10, 2, 2, orange)
	c.DrawPoint(11, 11, ColWhite)
	for _, tc := range []struct {
		x, y int
		want Color
	}{
		{10, 10, orange}, {11, 10, orange}, {11, 11, ColWhite}, {12, 10, ColBlack}, {-1, 10, ColBlack}, {Width, 0, ColBlack},
	} {
		if got := c.DrawGetPoint(tc.x, tc.y); got != tc.want {
			t.Errorf("DrawGetPoint(%d, %d) = %#04x, want %#04x", tc.x, tc.y, got, tc.want)
		}
	}
}

func TestShadow(t *testing.T) {
	for _, tc := range []struct {
		col  Color
		shad uint8
		want Color
	}{
		{ColWhite, 15, 0xD71A}, {ColWhite, 8, 0x7BEF}, {ColWhite, 1, 0x0861}, {ColWhite, 0, ColBlack}, {RGB(255, 128, 0), 8, 0x7A00},
	} {
		if got := Shadow(tc.col, tc.shad); got != tc.want {
			t.Errorf("Shadow(%#04x, %d) = %#04x, want %#04x", tc.col, tc.shad, got, tc.want)
		}
	}

	c, _, _, _ := newTest(t, Width, Height)
	c.DrawRect(0, 0, 8, 2, ColWhite)
	c.DrawRect(8, 0, 8, 2, ColRed)
	c.DrawRectShadow(4, 0, 8, 1, 8)
	// a 1-bit image 16 wide: 0xF0 0x0F
	c.DrawBlit1Shadow("\xf0\x0f", 0, 0, 0, 1, 16, 1, 16, 1)
	for _, tc := range []struct {
		x, y int
		want Color
	}{
		{3, 0, ColWhite}, {4, 0, 0x7BEF}, {11, 0, Shadow(ColRed, 8)}, {12, 0, ColRed},
		{0, 1, 0x0861}, {4, 1, ColWhite}, {11, 1, ColRed}, {12, 1, Shadow(ColRed, 1)},
	} {
		if got := c.DrawGetPoint(tc.x, tc.y); got != tc.want {
			t.Errorf("pixel %d, %d = %#04x, want %#04x", tc.x, tc.y, got, tc.want)
		}
	}
}

func TestSaveRect(t *testing.T) {
	c, _, _, _ := newTest(t, Width, Height)
	c.DrawRect(Width-4, 0, 4, 2, ColRed)
	c.DrawPoint(Width-1, 1, ColWhite)
	buf := make([]uint8, 8*2)
	c.SaveRect(buf, Width-4, 0, 8, 2) // half out of the screen
	c.DrawRect(Width-4, 0, 4, 2, ColGreen)
	c.RestoreRect(buf, Width-4, 0, 8, 2)
	if a, b := c.DrawGetPoint(Width-4, 0), c.DrawGetPoint(Width-1, 1); a != ColRed || b != ColWhite {
		t.Errorf("restored %#04x %#04x", a, b)
	}
	c.SaveRect(buf, -2, -1, 8, 2) // only the pixels on the screen
	c.RestoreRect(buf, -2, -1, 8, 2)
}

func TestDrawImg4PalMat(t *testing.T) {
	c, _, _, _ := newTest(t, Width, Height)
	pal := palette(ColBlack, ColRed, ColGreen, ColBlue)
	// 4 x 2 pixels: 1 2 3 0 / 3 0 1 2
	img := "\x12\x30" + "\x30\x12"
	var m Mat2D
	m.PrepDrawImg(4, 2, 2, 1, 4, 2, 0, 0, 0, 0, 0) // the same size
	c.DrawImg4PalMat(img, pal, 4, 2, 10, 10, 4, 2, &m, DrawImgTransp, ColBlack)
	c.DrawImg4Pal(img, pal, 0, 0, 10, 20, 4, 2, 4)
	m.PrepDrawImg(4, 2, 2, 1, 8, 4, 0, 0, 0, 0, 0) // twice the size
	c.DrawImg4PalMat(img, pal, 4, 2, 10, 30, 8, 4, &m, DrawImgNoBorder, 0)
	m.PrepDrawImg(4, 2, 2, 1, 4, 2, 0, 0, math.Pi, 0, 0) // turned around
	c.DrawImg4PalMat(img, pal, 4, 2, 20, 10, 4, 2, &m, DrawImgColor, ColWhite)
	for y := 0; y < 2; y++ {
		for x := 0; x < 4; x++ {
			want := c.DrawGetPoint(10+x, 20+y) // black is transparent: the black screen
			if got := c.DrawGetPoint(10+x, 10+y); got != want {
				t.Errorf("same size %d,%d: %#04x, want %#04x", x, y, got, want)
			}
			for _, p := range [][2]int{{10 + 2*x, 30 + 2*y}, {11 + 2*x, 31 + 2*y}} {
				if got := c.DrawGetPoint(p[0], p[1]); got != want {
					t.Errorf("twice %d,%d: %#04x, want %#04x", x, y, got, want)
				}
			}
		}
	}
	// turned by 180 degrees about the point 2,1: the source x is 2-x0, y
	// 1-y0 (x0 -2..1, y0 -1..0), out of the image white; sin(float32(pi))
	// is a little below 0, so m21 is -1/4096 and the last pixels of the
	// rows go a row up, as in the SDK
	for x, want := range []Color{ColWhite, ColWhite, ColWhite, ColBlack, ColWhite, ColGreen, ColRed, ColGreen} {
		if got := c.DrawGetPoint(20+x%4, 10+x/4); got != want {
			t.Errorf("turned %d,%d: %#04x, want %#04x", x%4, x/4, got, want)
		}
	}
}

func TestDrawTile4PalMap(t *testing.T) {
	c, _, _, _ := newTest(t, Width, Height)
	pal := palette(ColBlack, ColRed, ColGreen, ColBlue)
	// 2 tiles of 2x2 pixels: tile 0 all red (1), tile 1 all blue (3)
	tiles := "\x11\x11" + "\x33\x33"
	// a map of 2x2 tiles: 0 1 / 1 0
	tilemap := "\x00\x01\x01\x00"
	var m Mat2D
	m.PrepDrawImg(4, 4, 2, 2, 4, 4, 0, 0, 0, 0, 0) // the map 1:1, its middle in the middle
	c.DrawTile4PalMap(tiles, pal, tilemap, 1, 1, 1, 0, 0, 4, 4, &m, 0)
	for _, tc := range []struct {
		x, y int
		want Color
	}{
		{0, 0, ColRed}, {1, 1, ColRed}, {2, 0, ColBlue}, {3, 1, ColBlue}, {0, 2, ColBlue}, {2, 2, ColRed}, {3, 3, ColRed},
	} {
		if got := c.DrawGetPoint(tc.x, tc.y); got != tc.want {
			t.Errorf("pixel %d,%d: %#04x, want %#04x", tc.x, tc.y, got, tc.want)
		}
	}
}

func TestSetPalette(t *testing.T) {
	c, _, _, _ := newTest(t, Width, Height)
	c.SetPalette(palette(ColBlack, ColRed, ColBlue, ColWhite))
	c.DrawRect(0, 0, 1, 1, RGB(250, 10, 10)) // 4 entries only: a new one
	if a := c.DrawGetPoint(0, 0); a != RGB(250, 10, 10) {
		t.Errorf("4 entries: %#04x", a)
	}
	full := make([]Color, 256)
	for i := range full {
		full[i] = ColBlack
	}
	full[1], full[2] = ColRed, ColWhite
	c.SetPalette(palette(full...))
	c.DrawRect(0, 0, 1, 1, RGB(200, 20, 0)) // the palette is full: the nearest
	c.DrawRect(1, 0, 1, 1, RGB(200, 20, 0))
	if a := c.DrawGetPoint(0, 0); a != ColRed {
		t.Errorf("full: %#04x", a)
	}
	c.MapRect(0, 0, 1, 1, func(col Color) Color { return col | 0x07FF })
	if a, b := c.DrawGetPoint(0, 0), c.DrawGetPoint(1, 0); a != ColWhite || b != ColRed {
		t.Errorf("MapRect: %#04x %#04x", a, b)
	}
}

// TestClose: the next console gets the frame buffer of a closed one,
// black again.
func TestClose(t *testing.T) {
	c, d, _, _ := newTest(t, Width, Height)
	c.DrawRect(0, 0, Width, Height, ColRed)
	c.DrawRect(10, 10, 5, 5, ColBlue)
	c.DispUpdate()
	pix := &c.pix[0]
	c.Close()
	c2 := New(d, nil, 1)
	if &c2.pix[0] != pix || c2.bufs[0] == nil || c2.bufs[1] == nil {
		t.Fatal("the buffers are new")
	}
	if a, b := c2.DrawGetPoint(12, 12), at(c2, d, 12, 12); a != ColBlack || b != ColBlack {
		t.Errorf("not black: %#04x, on the display %#04x", a, b)
	}
	c2.DrawRect(0, 0, 1, 1, ColYellow)
	c2.DispUpdate()
	if a := at(c2, d, 0, 0); a != ColYellow {
		t.Errorf("drawing: %#04x", a)
	}
	if c3 := New(d, nil, 1); &c3.pix[0] == pix {
		t.Error("c2 was not closed: its buffer is used again")
	}
}

func TestDrawImgPalDbl(t *testing.T) {
	c, _, _, _ := newTest(t, Width, Height)
	pal := palette(ColBlack, ColRed, ColBlue, ColWhite)
	// 2 x 2: red blue / white black, drawn 4 x 4 from x 0 (the row wraps)
	c.DrawImgPalDbl("\x01\x02\x03\x00", pal, 0, 0, 0, 0, 4, 4, 4)
	for _, tc := range []struct {
		x, y int
		want Color
	}{
		{1, 0, ColRed}, {3, 0, ColBlue},
		{0, 0, blend2(ColBlue, ColRed)}, {2, 0, blend2(ColRed, ColBlue)},
		{1, 1, blend2(ColRed, ColWhite)}, {3, 1, blend2(ColBlue, ColBlack)},
		{0, 1, blend4(ColBlue, ColRed, ColBlack, ColWhite)},
		{1, 2, ColWhite}, {3, 3, ColBlack}, // the last line is not blended
	} {
		if got := c.DrawGetPoint(tc.x, tc.y); got != tc.want {
			t.Errorf("pixel %d,%d: %#04x, want %#04x", tc.x, tc.y, got, tc.want)
		}
	}
}

func TestDrawImg4Pal(t *testing.T) {
	c, d, _, _ := newTest(t, Width, Height)
	pal := palette(ColBlack, ColRed, ColGreen, ColBlue)
	img := "\x12\x30" // 3 pixels wide: 1 2 3, the last nibble unused
	c.DrawImg4Pal(img, pal, 1, 0, 5, 5, 2, 1, 3)
	c.DrawBlit4Pal(img, pal, 0, 0, 5, 6, 3, 1, 3, ColGreen) // green is transparent
	c.DrawRect(6, 7, 1, 1, ColWhite)
	c.DrawBlitPal("\x00\x01", palette(ColRed, ColWhite), 0, 0, 5, 7, 2, 1, 2, ColWhite)
	c.DispUpdate()
	for _, tc := range []struct {
		x, y int
		want Color
	}{
		{5, 5, ColGreen}, {6, 5, ColBlue}, {7, 5, ColBlack},
		{5, 6, ColRed}, {6, 6, ColBlack}, {7, 6, ColBlue},
		{5, 7, ColRed}, {6, 7, ColWhite},
	} {
		if got := at(c, d, tc.x, tc.y); got != tc.want {
			t.Errorf("pixel %d, %d = %#04x, want %#04x", tc.x, tc.y, got, tc.want)
		}
	}
}

func TestDrawImgRle(t *testing.T) {
	c, d, _, _ := newTest(t, Width, Height)
	pal := palette(ColBlack, ColRed, ColGreen, ColBlue)
	// 4 x 3 pixels: 1 1 1 1 / 1 2 3 0 / 2 2 2 3
	// 8 bits: repeat 1 five times (128+2), raw 3 (2), repeat 2 three times
	// (128+0), raw 1 (0)
	img8 := "\x82\x01" + "\x02\x02\x03\x00" + "\x80\x02" + "\x00\x03"
	want := []Color{ColRed, ColRed, ColRed, ColRed, ColRed, ColGreen, ColBlue, ColBlack, ColGreen, ColGreen, ColGreen, ColBlue}
	c.DrawImgRle(img8, pal, 10, 10, 4, 3)
	// 4 bits, the same: repeat 1 five times (8+2), raw 3 (2), repeat 2
	// three times (8+0), raw 1 (0)
	img4 := "\xa1" + "\x22\x30" + "\x82" + "\x03"
	c.DrawImg4Rle(img4, pal, 20, 10, 4, 3)
	// the rows above the screen are skipped
	c.DrawImgRle(img8, pal, 30, -1, 4, 3)
	c.DispUpdate()
	for i, col := range want {
		x, y := i%4, i/4
		if got := at(c, d, 10+x, 10+y); got != col {
			t.Errorf("DrawImgRle: pixel %d, %d = %#04x, want %#04x", x, y, got, col)
		}
		if got := at(c, d, 20+x, 10+y); got != col {
			t.Errorf("DrawImg4Rle: pixel %d, %d = %#04x, want %#04x", x, y, got, col)
		}
		if y > 0 {
			if got := at(c, d, 30+x, y-1); got != col {
				t.Errorf("DrawImgRle at -1: pixel %d, %d = %#04x, want %#04x", x, y, got, col)
			}
		}
	}
}

func TestDrawText(t *testing.T) {
	c, d, _, _ := newTest(t, Width, Height)
	if c.DrawFontWidth() != 8 || c.DrawFontHeight() != 16 {
		t.Errorf("default font %dx%d, want 8x16 (FontBold8x16)", c.DrawFontWidth(), c.DrawFontHeight())
	}
	c.DrawRect(0, 0, 100, 60, ColBlue)
	c.DrawText("A", 2, 3, ColYellow)
	c.SelFont8x8()
	c.DrawTextBg("AB", 10, 3, ColWhite, ColRed)
	c.SelFont6x8()
	c.DrawTextBg2("AB", 30, 3, ColWhite, ColRed) // 12 x 16 a character
	c.DrawTextW("A", 60, 3, ColGreen)
	c.DrawTextH("A", 80, 3, ColGreen)
	c.DispUpdate()
	checkGlyph(t, c, d, FontBold8x16, 'A', 2, 3, 1, 1, ColYellow, ColBlue)
	checkGlyph(t, c, d, FontBold8x8, 'A', 10, 3, 1, 1, ColWhite, ColRed)
	checkGlyph(t, c, d, FontBold8x8, 'B', 18, 3, 1, 1, ColWhite, ColRed)
	checkGlyph(t, c, d, FontCond6x8, 'A', 30, 3, 2, 2, ColWhite, ColRed)
	checkGlyph(t, c, d, FontCond6x8, 'B', 42, 3, 2, 2, ColWhite, ColRed)
	checkGlyph(t, c, d, FontCond6x8, 'A', 60, 3, 2, 1, ColGreen, ColBlue)
	checkGlyph(t, c, d, FontCond6x8, 'A', 80, 3, 1, 2, ColGreen, ColBlue)
}

// checkGlyph checks character ch of font f at x, y, sx times as wide and
// sy times as high.
func checkGlyph(t *testing.T, c *Console, d *memlcd.Display, f Font, ch byte, x, y, sx, sy int, fg, bg Color) {
	t.Helper()
	for i := 0; i < f.h*sy; i++ {
		bits := f.data[i/sy*256+int(ch)]
		for j := 0; j < f.w*sx; j++ {
			want := bg
			if bits&(0x80>>(j/sx)) != 0 {
				want = fg
			}
			if got := at(c, d, x+j, y+i); got != want {
				t.Fatalf("%c at %d, %d: pixel %d, %d = %#04x, want %#04x", ch, x, y, j, i, got, want)
			}
		}
	}
}

func TestTextClipped(t *testing.T) {
	c, _, _, _ := newTest(t, Width, Height)
	// none of these may panic
	c.DrawTextBg("XY", -4, -4, ColWhite, ColRed)
	c.DrawTextBg("XY", Width-4, Height-4, ColWhite, ColRed)
	c.DrawText("XY", -100, 500, ColWhite)
	c.DrawPoint(-1, 0, ColWhite)
	c.DrawPoint(Width, Height, ColWhite)
	c.DrawRect(-10, -10, Width+20, 15, ColWhite)
	c.DrawFrame(Width-5, Height-5, 10, 10, ColWhite)
	c.DispUpdate()
}

func TestPaletteFull(t *testing.T) {
	c, d, _, _ := newTest(t, Width, Height)
	// 300 colours: black is there already; the first 255 new ones are
	// exact, the others are the nearest.
	for i := 0; i < 300; i++ {
		c.DrawPoint(i, 0, Color(i+1)) // blues and greens
	}
	c.DispUpdate()
	for i := 0; i < 255; i++ {
		if got := at(c, d, i, 0); got != Color(i+1) {
			t.Fatalf("colour %d: %#04x, want exact", i, got)
		}
	}
	if c.ncol != 256 {
		t.Fatalf("%d colours, want 256", c.ncol)
	}
	// The palette has 0x0000 to 0x00FF (green 0 to 7, every blue): the
	// nearest of 0x0120 (green 9, blue 0) is 0x00E0 (green 7, blue 0).
	c.DrawPoint(0, 1, 0x0120)
	c.DispUpdate()
	if got := at(c, d, 0, 1); got != 0x00E0 {
		t.Errorf("nearest of 0x0120: %#04x, want 0x00e0", got)
	}

	// DrawClear frees the palette: new colours are exact again.
	c.DrawClear()
	c.DrawPoint(5, 5, 0x1234)
	c.DispUpdate()
	if got := at(c, d, 5, 5); got != 0x1234 {
		t.Errorf("after DrawClear: %#04x, want 0x1234", got)
	}
	if got := at(c, d, 100, 0); got != ColBlack {
		t.Errorf("after DrawClear: %#04x, want black", got)
	}
	if c.ncol != 2 {
		t.Errorf("after DrawClear: %d colours, want 2", c.ncol)
	}
}

func TestPaletteTableRefilled(t *testing.T) {
	c, d, _, _ := newTest(t, Width, Height)
	pal := palette(ColRed, ColGreen)
	c.DrawImgPal("\x00\x01", pal, 0, 0, 0, 0, 2, 1, 2)
	c.DrawClearCol(ColBlue) // the indices of the table are stale now
	c.DrawImgPal("\x00\x01", pal, 0, 0, 0, 0, 2, 1, 2)
	c.DispUpdate()
	if at(c, d, 0, 0) != ColRed || at(c, d, 1, 0) != ColGreen || at(c, d, 2, 0) != ColBlue {
		t.Errorf("after DrawClearCol: %#04x %#04x %#04x", at(c, d, 0, 0), at(c, d, 1, 0), at(c, d, 2, 0))
	}
}

// recorder is a display that records what DispUpdate sends.
type recorder struct {
	*framebuf.Buffer
	sent [][4]int16
}

func (r *recorder) Display() error { return nil }

func (r *recorder) DrawRGBBitmap8(x, y int16, data []uint8, w, h int16) error {
	r.sent = append(r.sent, [4]int16{x, y, w, h})
	return r.Buffer.DrawRGBBitmap8(x, y, data, w, h)
}

func TestDispUpdateSendsTheDirtyPart(t *testing.T) {
	r := &recorder{Buffer: framebuf.New(make([]uint16, 480*272), 480, 272)}
	c := New(r, nil, 1)
	c.DrawPoint(3, 4, ColWhite)
	c.DrawRect(10, 20, 5, 1, ColRed)
	c.DispUpdate()
	// The union of both (x 3 to 14, y 4 to 20) at 80 + 3, 16 + 4 on the
	// display, in bands of 8 rows.
	want := [][4]int16{{83, 20, 12, 8}, {83, 28, 12, 8}, {83, 36, 12, 1}}
	if len(r.sent) != len(want) || r.sent[0] != want[0] || r.sent[1] != want[1] || r.sent[2] != want[2] {
		t.Fatalf("sent %v, want %v", r.sent, want)
	}
	r.sent = nil
	c.DispUpdate()
	if len(r.sent) != 0 {
		t.Errorf("nothing drawn, sent %v", r.sent)
	}
	if got := framebuf.ToRGBA(r.Pix[(16+4)*480+80+3]); got.R != 255 || got.G != 255 {
		t.Errorf("pixel 3, 4 at 83, 20: %v, want white", got)
	}
	c.DispUpdateAll()
	if len(r.sent) != Height/bandRows {
		t.Errorf("DispUpdateAll: %d sends, want %d", len(r.sent), Height/bandRows)
	}
}

// keysOf returns the keys KeyGetRel returns until the buffer is empty.
func keysOf(c *Console) []uint8 {
	var ks []uint8
	for {
		k := c.KeyGetRel()
		if k == NoKey {
			return ks
		}
		ks = append(ks, k)
	}
}

func TestKeyRepeat(t *testing.T) {
	c, _, clock, p := newTest(t, Width, Height)
	p.Hold(hal.PadA, 100*time.Millisecond, 1000*time.Millisecond) // 0.1s .. 1.1s
	c.WaitMs(1500)
	// pressed at 0.1s, repeated at 0.5, 0.6 ... 1.0s, released 50ms
	// after 1.1s
	want := []uint8{KeyA, KeyA, KeyA, KeyA, KeyA, KeyA, KeyA, KeyA | KeyRelease}
	if got := keysOf(c); !equal(got, want) {
		t.Fatalf("keys %v, want %v", got, want)
	}
	p.Hold(hal.PadLeft|hal.PadOK, clock.T+10*time.Millisecond, 50*time.Millisecond)
	c.WaitMs(30)
	if !c.KeyPressed(KeyLeft) || !c.KeyPressed(KeyA) || c.KeyPressed(KeyB) {
		t.Errorf("KeyPressed: left %v A %v B %v, want left and A (the stick)", c.KeyPressed(KeyLeft), c.KeyPressed(KeyA), c.KeyPressed(KeyB))
	}
	k1, k2 := c.KeyGet(), c.KeyGet()
	if !(k1 == KeyLeft && k2 == KeyA) {
		t.Errorf("KeyGet: %d %d, want left, A", k1, k2)
	}
	c.KeyFlush()
	c.WaitMs(200)
	if k := c.KeyGet(); k != NoKey {
		t.Errorf("after the releases: KeyGet %d, want none", k)
	}
	if !c.KeyNoPressed() {
		t.Error("KeyNoPressed: false")
	}
}

func TestKeyHeldAtStart(t *testing.T) {
	c, _, _, p := newTest(t, Width, Height)
	p.Hold(hal.PadA, 0, 600*time.Millisecond) // A from the launcher
	p.Tap(hal.PadB, 300*time.Millisecond)
	p.Tap(hal.PadA, 800*time.Millisecond)
	c.WaitMs(1000)
	want := []uint8{KeyB, KeyB | KeyRelease, KeyA, KeyA | KeyRelease}
	if got := keysOf(c); !equal(got, want) {
		t.Fatalf("keys %v, want %v", got, want)
	}
}

func equal(a, b []uint8) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// touchAt is a hal.Touch pressed at one place.
type touchAt struct{ x, y int16 }

func (t *touchAt) ReadTouch() (int16, int16, bool) { return t.x, t.y, t.x >= 0 }

func TestTouchButtons(t *testing.T) {
	touch := &touchAt{-1, -1}
	c := New(memlcd.New(480, 272), touch, 1)
	c.Clock = picotest.NewClock(t, time.Hour)
	c.WaitMs(10) // the first scan: a touch now would be held from the start
	for _, tc := range []struct {
		x, y int16
		key  uint8
	}{
		{10, 10, KeyY}, {240, 10, KeyUp}, {470, 10, KeyX},
		{10, 136, KeyLeft}, {240, 136, KeyA}, {470, 136, KeyRight},
		{10, 260, KeyB}, {240, 260, KeyDown}, {470, 260, KeyA},
	} {
		touch.x, touch.y = tc.x, tc.y
		c.WaitMs(20)
		touch.x = -1
		c.WaitMs(100)
		if k := c.KeyGet(); k != tc.key {
			t.Errorf("touch at %d, %d: key %d, want %d", tc.x, tc.y, k, tc.key)
		}
		c.KeyFlush()
	}
}

func TestBusyLoopGoesOn(t *testing.T) {
	c, _, clock, p := newTest(t, Width, Height)
	p.Tap(hal.PadX, 300*time.Millisecond)
	for c.KeyGet() == NoKey { // a game waiting for a key
	}
	if clock.T < 300*time.Millisecond || clock.T > 320*time.Millisecond {
		t.Errorf("the key came at %v", clock.T)
	}
	t0 := c.Time()
	for c.Time()-t0 < 50000 { // a game waiting 50ms
	}
}

func TestRand(t *testing.T) {
	c := New(memlcd.New(Width, Height), nil, 7)
	var seen [7]bool
	for i := 0; i < 1000; i++ {
		r := c.RandU8Max(6)
		if r > 6 {
			t.Fatalf("RandU8Max(6) = %d", r)
		}
		seen[r] = true
		if r := c.RandU16Max(1000); r > 1000 {
			t.Fatalf("RandU16Max(1000) = %d", r)
		}
		if r := c.RandU8MinMax(1, 50); r < 1 || r > 50 {
			t.Fatalf("RandU8MinMax(1, 50) = %d", r)
		}
		if r := c.RandS8MinMax(-10, 10); r < -10 || r > 10 {
			t.Fatalf("RandS8MinMax(-10, 10) = %d", r)
		}
		if r := c.RandS16MinMax(-10, -2); r < -10 || r > -2 {
			t.Fatalf("RandS16MinMax(-10, -2) = %d", r)
		}
		if r := c.RandU16MinMax(100, 200); r < 100 || r > 200 {
			t.Fatalf("RandU16MinMax(100, 200) = %d", r)
		}
		if r := c.RandS16Max(-300); r < -300 || r > 0 {
			t.Fatalf("RandS16Max(-300) = %d", r)
		}
		if r := c.RandFloatMinMax(-2, 3); r < -2 || r >= 3 {
			t.Fatalf("RandFloatMinMax(-2, 3) = %v", r)
		}
	}
	for v, ok := range seen {
		if !ok {
			t.Errorf("RandU8Max(6) never returned %d", v)
		}
	}
	// the same seed, the same numbers
	a, b := New(memlcd.New(Width, Height), nil, 3), New(memlcd.New(Width, Height), nil, 3)
	for i := 0; i < 10; i++ {
		if a.RandShift() != b.RandShift() {
			t.Fatal("the same seed gave other numbers")
		}
	}
}
