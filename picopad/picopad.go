// Package picopad runs games ported from PicoLibSDK, the C SDK of the
// PicoPad (an RP2040 game console with a 320x240 screen and eight buttons,
// https://github.com/Panda381/PicoLibSDK). Console has the SDK functions
// the games use, with the same names and arguments: drawing into a frame
// buffer (DrawRect, DrawImgPal, DrawText ...), DispUpdate to send what
// changed to the panel, the buttons (KeyGet, KeyPressed ...), the time
// (Time, WaitMs) and the random numbers (RandU8 ...). A port keeps the
// game's own loop: the functions that wait read the buttons and sleep.
//
// The PicoPad's frame buffer has 16-bit colours (150KB), which leaves too
// little of the ESP32's RAM for a game. Here it holds 8-bit palette
// indices (75KB): an RGB565 colour gets an index the first time it is
// drawn, and DispUpdate turns the indices back into RGB565 a few rows at a
// time. Up to 256 colours are exact; after that, a new colour gets the
// nearest one. Filling the whole screen with one colour (DrawClear) frees
// the palette again.
//
// The 320x240 screen is in the middle of the display (on 480x272, 80
// pixels from the left and 16 from the top).
package picopad

import (
	"image/color"
	"unsafe"

	"github.com/sat0ken/tinygo-cyd/hal"
)

//go:generate go run ../tools/picoimg -pkg picopad -root $PICOLIBSDK -o font.go _font/font_bold_8x8.c _font/font_bold_8x14.c _font/font_bold_8x16.c _font/font_cond_6x8.c _font/font_tiny_5x8.c

// The PicoPad's screen size (WIDTH, HEIGHT).
const (
	Width  = 320
	Height = 240
)

// bandRows is how many rows DispUpdate converts and sends at a time.
const bandRows = 8

// Color is an RGB565 colour (COLTYPE of PicoLibSDK).
type Color uint16

// RGB is COLOR(r, g, b) of PicoLibSDK.
func RGB(r, g, b uint8) Color {
	return Color(r&0xF8)<<8 | Color(g&0xFC)<<3 | Color(b)>>3
}

// The base colours of PicoLibSDK (COL_*).
const (
	ColBlack   Color = 0x0000
	ColBlue    Color = 0x001F
	ColGreen   Color = 0x07E0
	ColCyan    Color = 0x07FF
	ColRed     Color = 0xF800
	ColMagenta Color = 0xF81F
	ColYellow  Color = 0xFFE0
	ColWhite   Color = 0xFFFF
	ColGray    Color = 0x8410
)

// Palette is the palette of an image: RGB565 colours, two bytes each,
// little endian (as tools/picoimg writes the u16 arrays of PicoLibSDK).
type Palette string

// color returns entry e.
func (p Palette) color(e int) Color { return Color(p[2*e]) | Color(p[2*e+1])<<8 }

// palTable maps the entries of a palette to frame buffer indices.
type palTable struct {
	pal Palette // its data pointer and length identify the palette
	gen uint32  // Console.gen when inx was filled
	inx [256]uint8
}

// maxTables is how many palettes the console remembers (GHOSTRAC, the
// game with the most, has 27).
const maxTables = 32

type bitmapDrawer interface {
	DrawRGBBitmap8(x, y int16, data []uint8, w, h int16) error
}

// asyncDrawer is spilcd.Device (and viewport.Display over one).
type asyncDrawer interface {
	DrawRGBBitmap8Async(x, y int16, data []uint8, w, h int16) error
	Wait()
}

// Console is the PicoPad for a ported game: its frame buffer, buttons,
// time and random numbers.
type Console struct {
	d      hal.Display
	bd     bitmapDrawer
	ad     asyncDrawer // nil: send at once
	ox, oy int16       // where the screen is on the display

	pix    []uint8 // Width*Height palette indices
	pal    [256]uint16
	ncol   int    // entries of pal in use
	gen    uint32 // changes when the palette is freed
	tables []*palTable
	nextT  int // the table to reuse when there are maxTables

	dirtyX1, dirtyY1, dirtyX2, dirtyY2 int

	bufs [2][]byte // bandRows rows of big endian RGB565
	next int       // the buffer to fill: the other may be going out

	font    Font
	shadows [16]*shadowTab // the shadow levels used, made when needed
	colors  *colorCache    // the colours made while drawing, made when needed

	keys
	rand uint64
}

// New returns the console on d (at least 320x240) with the touch panel t
// (nil: none) and the random seed seed. It clears the display. Set Pad to
// use a game controller.
func New(d hal.Display, t hal.Touch, seed uint32) *Console {
	w, h := d.Size()
	c := &Console{
		d:    d,
		ox:   (w - Width) / 2,
		oy:   (h - Height) / 2,
		pix:  make([]uint8, Width*Height),
		ncol: 1,            // the frame buffer starts black: index 0
		font: FontBold8x16, // the default of PicoLibSDK
	}
	c.bd, _ = d.(bitmapDrawer)
	c.ad, _ = d.(asyncDrawer)
	for k := range c.bufs {
		c.bufs[k] = make([]byte, Width*bandRows*2)
	}
	c.keys.init(t, w, h)
	c.randInit(seed)
	c.DispDirtyNone()
	d.FillRectangle(0, 0, w, h, color.RGBA{0, 0, 0, 255})
	d.Display()
	return c
}

// index returns the frame buffer index of col, giving it one if it has
// none yet (the nearest colour if the palette is full).
func (c *Console) index(col Color) uint8 {
	for i := 0; i < c.ncol; i++ {
		if c.pal[i] == uint16(col) {
			return uint8(i)
		}
	}
	if c.ncol < len(c.pal) {
		c.pal[c.ncol] = uint16(col)
		c.ncol++
		return uint8(c.ncol - 1)
	}
	best, bestD := 0, -1
	for i, p := range c.pal {
		if d := distance(Color(p), col); bestD < 0 || d < bestD {
			best, bestD = i, d
		}
	}
	return uint8(best)
}

// distance compares two colours, with the 5-bit red and blue scaled to the
// 6 bits of green.
func distance(a, b Color) int {
	dr := int(a>>11) - int(b>>11)
	dg := int(a>>5&0x3F) - int(b>>5&0x3F)
	db := int(a&0x1F) - int(b&0x1F)
	return 4*dr*dr + dg*dg + 4*db*db
}

// freePalette forgets every colour: call it only when the whole frame
// buffer is about to be overwritten.
func (c *Console) freePalette() {
	c.ncol = 0
	c.gen++
}

// PrepPalette gives the colours of pal their frame buffer indices now. A
// game that draws more than 256 colours chooses so which colours are
// exact: the ones that come when the palette is full get the nearest
// colour. (The PicoPad has a 16-bit frame buffer and no such function.)
func (c *Console) PrepPalette(pal Palette) { c.table(pal) }

// table returns the frame buffer indices of the entries of pal.
func (c *Console) table(pal Palette) *[256]uint8 {
	var t *palTable
	for _, u := range c.tables {
		if len(u.pal) == len(pal) && unsafe.StringData(string(u.pal)) == unsafe.StringData(string(pal)) {
			t = u
			break
		}
	}
	if t == nil {
		if len(c.tables) < maxTables {
			t = new(palTable)
			c.tables = append(c.tables, t)
		} else {
			t = c.tables[c.nextT]
			c.nextT = (c.nextT + 1) % maxTables
		}
		t.pal = pal
		t.gen = c.gen - 1 // not filled
	}
	if t.gen != c.gen {
		for e := 0; e < len(pal)/2 && e < len(t.inx); e++ {
			t.inx[e] = c.index(pal.color(e))
		}
		t.gen = c.gen
	}
	return &t.inx
}

// DispDirtyRect adds a rectangle to the part of the screen DispUpdate
// sends.
func (c *Console) DispDirtyRect(x, y, w, h int) {
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
	c.dirtyX1 = min(c.dirtyX1, x)
	c.dirtyX2 = max(c.dirtyX2, x+w)
	c.dirtyY1 = min(c.dirtyY1, y)
	c.dirtyY2 = max(c.dirtyY2, y+h)
}

// DispDirtyAll makes DispUpdate send the whole screen.
func (c *Console) DispDirtyAll() {
	c.dirtyX1, c.dirtyX2, c.dirtyY1, c.dirtyY2 = 0, Width, 0, Height
}

// DispDirtyNone makes DispUpdate send nothing.
func (c *Console) DispDirtyNone() {
	c.dirtyX1, c.dirtyX2, c.dirtyY1, c.dirtyY2 = Width, 0, Height, 0
}

// DispUpdate sends the part of the screen drawn since the last update.
func (c *Console) DispUpdate() {
	x1, x2, y1, y2 := c.dirtyX1, c.dirtyX2, c.dirtyY1, c.dirtyY2
	if x1 >= x2 || y1 >= y2 {
		return
	}
	c.DispDirtyNone()
	w := x2 - x1
	for y0 := y1; y0 < y2; y0 += bandRows {
		rows := min(bandRows, y2-y0)
		buf := c.bufs[c.next]
		i := 0
		for y := y0; y < y0+rows; y++ {
			for _, p := range c.pix[y*Width+x1 : y*Width+x2] {
				v := c.pal[p]
				buf[i], buf[i+1] = byte(v>>8), byte(v)
				i += 2
			}
		}
		c.send(int16(x1), int16(y0), buf[:i], int16(w), int16(rows))
		c.next ^= 1
	}
	c.d.Display()
}

// DispUpdateAll sends the whole screen.
func (c *Console) DispUpdateAll() {
	c.DispDirtyAll()
	c.DispUpdate()
}

// send draws w x h pixels of big endian RGB565 at x, y of the screen.
func (c *Console) send(x, y int16, data []byte, w, h int16) {
	x, y = x+c.ox, y+c.oy
	switch {
	case c.ad != nil:
		c.ad.DrawRGBBitmap8Async(x, y, data, w, h) // waits for the other buffer
	case c.bd != nil:
		c.bd.DrawRGBBitmap8(x, y, data, w, h)
	default:
		for i := 0; i < int(w)*int(h); i++ {
			v := uint16(data[2*i])<<8 | uint16(data[2*i+1])
			c.d.SetPixel(x+int16(i)%w, y+int16(i)/w, color.RGBA{
				uint8(v>>11) << 3, uint8(v>>5) << 2, uint8(v) << 3, 255})
		}
	}
}
