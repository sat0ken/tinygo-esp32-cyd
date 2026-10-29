package spilcd

import (
	"image/color"
	"math/rand"
	"testing"
	"time"

	"github.com/sat0ken/tinygo-cyd/framebuf"
	"github.com/sat0ken/tinygo-cyd/memlcd"
	"github.com/sat0ken/tinygo-cyd/romfont/freesans"
	"tinygo.org/x/tinyfont"
)

// panel simulates the controller side of the bus: it decodes commands and
// keeps the pixels written with RAMWR (in screen coordinates; MADCTL is
// not simulated).
type panel struct {
	w, h           int
	pix            []uint16
	dc             bool // true = data
	cs             bool // true = not selected
	cmd            byte
	param          []byte
	x0, x1, y0, y1 int
	x, y           int
	half           int // -1, or the high byte of a pixel being written
	cmds           []byte
	txs            int
	csErr          bool
}

func newPanel(w, h int) *panel {
	return &panel{w: w, h: h, pix: make([]uint16, w*h), cs: true, half: -1}
}

type pinFunc func(bool)

func (f pinFunc) Set(v bool) { f(v) }

func (p *panel) Tx(w, r []byte) error {
	p.txs++
	if p.cs {
		p.csErr = true
	}
	if !p.dc {
		for _, b := range w {
			p.cmd = b
			p.param = p.param[:0]
			p.cmds = append(p.cmds, b)
			if b == RAMWR {
				p.x, p.y, p.half = p.x0, p.y0, -1
			}
		}
		return nil
	}
	for _, b := range w {
		switch p.cmd {
		case CASET, RASET:
			p.param = append(p.param, b)
			if len(p.param) == 4 {
				a := int(p.param[0])<<8 | int(p.param[1])
				z := int(p.param[2])<<8 | int(p.param[3])
				if p.cmd == CASET {
					p.x0, p.x1 = a, z
				} else {
					p.y0, p.y1 = a, z
				}
			}
		case RAMWR:
			if p.half < 0 {
				p.half = int(b)
				continue
			}
			if p.y <= p.y1 && p.x < p.w && p.y < p.h {
				p.pix[p.y*p.w+p.x] = uint16(p.half)<<8 | uint16(b)
			}
			p.half = -1
			p.x++
			if p.x > p.x1 {
				p.x = p.x0
				p.y++
			}
		}
	}
	return nil
}

func newTestDevice(w, h int16) (*Device, *panel) {
	p := newPanel(int(w), int(h))
	d := New(p, pinFunc(func(v bool) { p.dc = v }), pinFunc(func(v bool) { p.cs = v }))
	d.Sleep = func(time.Duration) {}
	d.Configure(Config{Width: w, Height: h, MADCTL: 0x28, Init: InitST7796})
	return d, p
}

func compare(t *testing.T, p *panel, b *memlcd.Display, what string) {
	t.Helper()
	bad := 0
	for y := 0; y < p.h; y++ {
		for x := 0; x < p.w; x++ {
			if got, want := p.pix[y*p.w+x], b.Pixel(int16(x), int16(y)); got != want {
				if bad < 5 {
					t.Errorf("%s: pixel %d,%d = %04x, want %04x", what, x, y, got, want)
				}
				bad++
			}
		}
	}
	if bad > 0 {
		t.Errorf("%s: %d pixels differ", what, bad)
	}
	if p.csErr {
		t.Errorf("%s: data sent with CS high", what)
	}
}

func TestILI9341Init(t *testing.T) {
	p := newPanel(320, 240)
	d := New(p, pinFunc(func(v bool) { p.dc = v }), pinFunc(func(v bool) { p.cs = v }))
	d.Sleep = func(time.Duration) {}
	d.Configure(Config{Width: 320, Height: 240, MADCTL: 0x28, Init: InitILI9341})
	n := len(p.cmds)
	if p.cmds[0] != SWRESET || p.cmds[n-4] != SLPOUT || p.cmds[n-1] != DISPON {
		t.Errorf("commands %x", p.cmds)
	}
	white := color.RGBA{255, 255, 255, 255}
	d.FillRectangle(310, 230, 20, 20, white) // clipped at 320x240
	if p.pix[239*320+319] != 0xFFFF || p.pix[230*320+309] != 0 {
		t.Error("fill not clipped to 320x240")
	}
}

func TestInitCommands(t *testing.T) {
	_, p := newTestDevice(480, 320)
	want := []byte{SWRESET, SLPOUT}
	if p.cmds[0] != want[0] || p.cmds[1] != want[1] {
		t.Errorf("starts with %x", p.cmds[:2])
	}
	n := len(p.cmds)
	if p.cmds[n-3] != MADCTL || p.cmds[n-2] != INVOFF || p.cmds[n-1] != DISPON {
		t.Errorf("ends with %x", p.cmds[n-3:])
	}
}

// TestSameAsFrameBuffer draws the same things on the device and on a frame
// buffer and compares the pixels.
func TestSameAsFrameBuffer(t *testing.T) {
	const w, h = 120, 80
	d, p := newTestDevice(w, h)
	b := memlcd.New(w, h)
	r := rand.New(rand.NewSource(1))
	col := func() color.RGBA { return color.RGBA{uint8(r.Intn(256)), uint8(r.Intn(256)), uint8(r.Intn(256)), 255} }
	for i := 0; i < 200; i++ {
		x, y := int16(r.Intn(w+40)-20), int16(r.Intn(h+40)-20)
		rw, rh := int16(r.Intn(50)), int16(r.Intn(40))
		c := col()
		switch r.Intn(4) {
		case 0:
			d.FillRectangle(x, y, rw, rh, c)
			b.FillRectangle(x, y, rw, rh, c)
		case 1:
			// A row of pixels with a colour change and a gap (runs).
			c2 := col()
			for k := int16(0); k < rw; k++ {
				cc := c
				if k > rw/2 {
					cc = c2
				}
				if k == rw/3 {
					continue
				}
				d.SetPixel(x+k, y, cc)
				b.SetPixel(x+k, y, cc)
			}
		case 2:
			data := make([]byte, int(rw)*int(rh)*2)
			r.Read(data)
			d.DrawRGBBitmap8(x, y, data, rw, rh) // clipped by the driver
			for i := 0; i < int(rw)*int(rh); i++ {
				v := uint16(data[2*i])<<8 | uint16(data[2*i+1])
				b.SetPixel(x+int16(i%int(rw)), y+int16(i/int(rw)), framebuf.ToRGBA(v))
			}
		case 3:
			tinyfont.WriteLine(d, &freesans.Bold9pt7b, x, y, "Ag9", c)
			tinyfont.WriteLine(b, &freesans.Bold9pt7b, x, y, "Ag9", c)
		}
	}
	d.Display()
	compare(t, p, b, "random drawing")
}

func TestRunsAreJoined(t *testing.T) {
	d, p := newTestDevice(480, 320)
	before := p.txs
	white := color.RGBA{255, 255, 255, 255}
	for x := int16(10); x < 110; x++ {
		d.SetPixel(x, 20, white)
	}
	d.Display()
	// One run: CASET + RASET + RAMWR (3 commands, 2 parameter blocks) and
	// one pixel block.
	if got := p.txs - before; got != 6 {
		t.Errorf("%d transfers for one run of 100 pixels", got)
	}
	// The next run on the same row does not send RASET again.
	before = p.txs
	d.SetPixel(200, 20, white)
	d.Display()
	if got := p.txs - before; got != 4 {
		t.Errorf("%d transfers for a run on the same row", got)
	}
}

func TestRGB565(t *testing.T) {
	hi, lo := RGB565(color.RGBA{0xFF, 0x00, 0x00, 255})
	if hi != 0xF8 || lo != 0x00 {
		t.Errorf("red %02x%02x", hi, lo)
	}
	hi, lo = RGB565(color.RGBA{0x00, 0xFF, 0x00, 255})
	if hi != 0x07 || lo != 0xE0 {
		t.Errorf("green %02x%02x", hi, lo)
	}
	hi, lo = RGB565(color.RGBA{0x00, 0x00, 0xFF, 255})
	if hi != 0x00 || lo != 0x1F {
		t.Errorf("blue %02x%02x", hi, lo)
	}
	for _, c := range []color.RGBA{{1, 2, 3, 255}, {200, 100, 50, 255}} {
		hi, lo := RGB565(c)
		if uint16(hi)<<8|uint16(lo) != framebuf.RGB565(c) {
			t.Errorf("%v differs from framebuf", c)
		}
	}
}

// asyncPanel sends StartTx blocks only when Wait is called (the latest
// a DMA transfer can end), keeping a copy, so that a block overwritten
// too early would show.
type asyncPanel struct {
	*panel
	block   []byte
	pending bool
	csLow   bool // CS was low all the time a block was pending
}

func (a *asyncPanel) Tx(w, r []byte) error {
	a.Wait()
	return a.panel.Tx(w, r)
}

func (a *asyncPanel) StartTx(w []byte) {
	a.Wait()
	a.block = append(a.block[:0], w...)
	a.pending, a.csLow = true, !a.cs
}

func (a *asyncPanel) Wait() {
	if !a.pending {
		return
	}
	a.pending = false
	if a.cs {
		a.csLow = false
	}
	a.panel.Tx(a.block, nil)
}

// TestAsync draws bands with DrawRGBBitmap8Async from two buffers, like
// games/kart, then text over them: the picture is as with the frame
// buffer, CS stays low while a band goes out.
func TestAsync(t *testing.T) {
	p := newPanel(64, 48)
	a := &asyncPanel{panel: p}
	d := New(a, pinFunc(func(v bool) { p.dc = v }), pinFunc(func(v bool) { p.cs = v }))
	d.Sleep = func(time.Duration) {}
	d.Configure(Config{Width: 64, Height: 48, MADCTL: 0x28, Init: InitST7796})
	fb := memlcd.New(64, 48)

	var bufs [2][64 * 8 * 2]byte
	for band := 0; band < 6; band++ {
		buf := &bufs[band%2]
		for i := range buf {
			buf[i] = byte(band*31 + i)
		}
		d.DrawRGBBitmap8Async(0, int16(band*8), buf[:], 64, 8)
		fb.DrawRGBBitmap8(0, int16(band*8), buf[:], 64, 8)
		if !a.csLow || !d.sending {
			t.Fatalf("band %d: CS not held low while it is sent", band)
		}
	}
	white := color.RGBA{255, 255, 255, 255}
	tinyfont.WriteLine(d, &freesans.Bold9pt7b, 2, 20, "Hi", white)
	tinyfont.WriteLine(fb, &freesans.Bold9pt7b, 2, 20, "Hi", white)
	d.Display()
	compare(t, p, fb, "async")
	if a.pending || d.sending || !p.cs {
		t.Error("a block still pending after drawing on top")
	}
}
