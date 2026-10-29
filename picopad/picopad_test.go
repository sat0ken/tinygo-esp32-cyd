package picopad

import (
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

func TestDrawText(t *testing.T) {
	c, d, _, _ := newTest(t, Width, Height)
	c.DrawRect(0, 0, 40, 20, ColBlue)
	c.DrawText("A", 2, 3, ColYellow)
	c.DrawTextBg("AB", 10, 3, ColWhite, ColRed)
	c.DispUpdate()
	checkGlyph(t, c, d, 'A', 2, 3, ColYellow, ColBlue)
	checkGlyph(t, c, d, 'A', 10, 3, ColWhite, ColRed)
	checkGlyph(t, c, d, 'B', 18, 3, ColWhite, ColRed)
}

// checkGlyph checks character ch of the font at x, y.
func checkGlyph(t *testing.T, c *Console, d *memlcd.Display, ch byte, x, y int, fg, bg Color) {
	t.Helper()
	for i := 0; i < 8; i++ {
		bits := fontBold8x8[i*256+int(ch)]
		for j := 0; j < 8; j++ {
			want := bg
			if bits&(0x80>>j) != 0 {
				want = fg
			}
			if got := at(c, d, x+j, y+i); got != want {
				t.Fatalf("%c: pixel %d, %d = %#04x, want %#04x", ch, j, i, got, want)
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
