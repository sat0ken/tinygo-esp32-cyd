// Package spilcd drives SPI TFT controllers that use the MIPI DCS commands
// (CASET, RASET, RAMWR, MADCTL...), such as the ST7796S (ESP32-3248S035C)
// and the ILI9341 (ESP32-2432S028R), over 4-wire SPI without a frame
// buffer: every drawing call goes to the panel at once. Only the init
// sequence differs between controllers (Config.Init).
//
// SetPixel is the slow path (each pixel needs its own address window), and
// tinyfont draws text pixel by pixel, so the driver joins pixels of the
// same colour next to each other on a row into one run and sends the run
// when the next pixel does not continue it. Display (and every other
// drawing call) sends the last run.
//
// The package does not import machine: the SPI bus and the pins are small
// interfaces, so the command stream can be tested on the host.
package spilcd

import (
	"image/color"
	"time"
)

// Bus is an SPI bus (machine.SPI).
type Bus interface {
	Tx(w, r []byte) error
}

// Pin is an output pin (machine.Pin).
type Pin interface {
	Set(high bool)
}

// Commands.
const (
	SWRESET = 0x01
	RDDID   = 0x04
	SLPOUT  = 0x11
	INVOFF  = 0x20
	INVON   = 0x21
	DISPON  = 0x29
	CASET   = 0x2A
	RASET   = 0x2B
	RAMWR   = 0x2C
	MADCTL  = 0x36
	COLMOD  = 0x3A
	RDID4   = 0xD3
)

// Config is the panel setting.
type Config struct {
	// Width and Height after MADCTL (480x320 or 320x240 in landscape).
	Width, Height int16
	MADCTL        byte
	Invert        bool
	// Init is the controller's init sequence, sent after the software
	// reset: InitST7796 or InitILI9341. It must leave sleep mode and set
	// 16-bit colour (COLMOD 0x55); MADCTL, the inversion and DISPON follow.
	Init []Command
}

// Command is one step of an init sequence. Data is a string so that the
// sequences stay in flash.
type Command struct {
	Cmd  byte
	Data string
	Wait time.Duration // after the command
}

// pixel bytes sent at a time (a multiple of 64, the ESP32 SPI FIFO).
const bufSize = 1024

// Device is a panel.
type Device struct {
	bus    Bus
	dc, cs Pin
	cfg    Config

	// Sleep waits between init commands; tests replace it.
	Sleep func(time.Duration)

	// The address window last sent, so that an unchanged CASET or RASET is
	// not sent again (runs on one row only change the columns).
	winX0, winX1, winY0, winY1 int16
	winOK                      bool

	// The pending run of SetPixel calls.
	runX, runY, runN int16
	runC             color.RGBA

	buf [bufSize]byte
	cmd [4]byte
}

// New returns a device on bus with the data/command and chip select pins
// (already configured as outputs).
func New(bus Bus, dc, cs Pin) *Device {
	return &Device{bus: bus, dc: dc, cs: cs, Sleep: time.Sleep}
}

// InitST7796 is the ST7796S init sequence of TFT_eSPI (ST7796_Init.h).
var InitST7796 = []Command{
	{SLPOUT, "", 120 * time.Millisecond},
	{0xF0, "\xC3", 0}, // enable extension command 2 part I
	{0xF0, "\x96", 0}, // part II
	{COLMOD, "\x55", 0},
	{0xB4, "\x01", 0},         // 1-dot inversion
	{0xB6, "\x80\x02\x3B", 0}, // display function control
	{0xE8, "\x40\x8A\x00\x00\x29\x19\xA5\x33", 0},
	{0xC1, "\x06", 0},                      // power control 2
	{0xC2, "\xA7", 0},                      // power control 3
	{0xC5, "\x18", 120 * time.Millisecond}, // VCOM
	{0xE0, "\xF0\x09\x0B\x06\x04\x15\x2F\x54\x42\x3C\x17\x14\x18\x1B", 0},
	{0xE1, "\xE0\x09\x0B\x06\x04\x03\x2B\x43\x42\x3B\x16\x14\x17\x1B", 120 * time.Millisecond},
	{0xF0, "\x3C", 0}, // disable extension command 2 part I
	{0xF0, "\x69", 120 * time.Millisecond},
}

// InitILI9341 is the ILI9341 init sequence of Adafruit_ILI9341 (initcmd).
var InitILI9341 = []Command{
	{0xEF, "\x03\x80\x02", 0},
	{0xCF, "\x00\xC1\x30", 0},     // power control B
	{0xED, "\x64\x03\x12\x81", 0}, // power on sequence control
	{0xE8, "\x85\x00\x78", 0},     // driver timing control A
	{0xCB, "\x39\x2C\x00\x34\x02", 0},
	{0xF7, "\x20", 0},     // pump ratio control
	{0xEA, "\x00\x00", 0}, // driver timing control B
	{0xC0, "\x23", 0},     // power control 1
	{0xC1, "\x10", 0},     // power control 2
	{0xC5, "\x3E\x28", 0}, // VCOM control 1
	{0xC7, "\x86", 0},     // VCOM control 2
	{0x37, "\x00", 0},     // vertical scroll start
	{COLMOD, "\x55", 0},
	{0xB1, "\x00\x18", 0},     // frame rate 79Hz
	{0xB6, "\x08\x82\x27", 0}, // display function control
	{0xF2, "\x00", 0},         // 3-gamma off
	{0x26, "\x01", 0},         // gamma curve 1
	{0xE0, "\x0F\x31\x2B\x0C\x0E\x08\x4E\xF1\x37\x07\x10\x03\x0E\x09\x00", 0},
	{0xE1, "\x00\x0E\x14\x03\x11\x07\x31\xC1\x48\x08\x0F\x0C\x31\x36\x0F", 0},
	{SLPOUT, "", 150 * time.Millisecond},
}

// Configure resets and initialises the panel and turns it on.
func (d *Device) Configure(cfg Config) {
	d.cfg = cfg
	d.Command(SWRESET)
	d.Sleep(120 * time.Millisecond)
	for _, c := range cfg.Init {
		n := copy(d.buf[:], c.Data)
		d.Command(c.Cmd, d.buf[:n]...)
		if c.Wait > 0 {
			d.Sleep(c.Wait)
		}
	}
	d.SetMADCTL(cfg.MADCTL)
	d.SetInvert(cfg.Invert)
	d.Command(DISPON)
	d.Sleep(20 * time.Millisecond)
}

// SetMADCTL changes the memory access control (rotation and colour order).
// It does not redraw anything.
func (d *Device) SetMADCTL(v byte) {
	d.flush()
	d.cfg.MADCTL = v
	d.Command(MADCTL, v)
	d.winOK = false
}

// SetInvert turns the display inversion on or off.
func (d *Device) SetInvert(on bool) {
	d.flush()
	d.cfg.Invert = on
	if on {
		d.Command(INVON)
	} else {
		d.Command(INVOFF)
	}
}

// Config returns the current setting.
func (d *Device) Config() Config { return d.cfg }

// Command sends a command and its parameters.
func (d *Device) Command(cmd byte, data ...byte) {
	d.cs.Set(false)
	d.sendCmd(cmd)
	if len(data) > 0 {
		d.bus.Tx(data, nil)
	}
	d.cs.Set(true)
}

// Read sends a command and reads n bytes back (n <= 4). The panel needs a
// slower SPI clock for reading (about 10MHz); the first byte is a dummy
// byte for most commands.
func (d *Device) Read(cmd byte, n int) [4]byte {
	d.flush()
	var r [4]byte
	d.cs.Set(false)
	d.sendCmd(cmd)
	d.bus.Tx(nil, r[:n])
	d.cs.Set(true)
	return r
}

// sendCmd sends one command byte with DC low and leaves DC high for data.
func (d *Device) sendCmd(cmd byte) {
	d.dc.Set(false)
	d.cmd[0] = cmd
	d.bus.Tx(d.cmd[:1], nil)
	d.dc.Set(true)
}

// window sets the address window and starts a memory write. Called with
// CS low.
func (d *Device) window(x0, y0, x1, y1 int16) {
	if !d.winOK || x0 != d.winX0 || x1 != d.winX1 {
		d.sendCmd(CASET)
		d.cmd = [4]byte{byte(x0 >> 8), byte(x0), byte(x1 >> 8), byte(x1)}
		d.bus.Tx(d.cmd[:], nil)
	}
	if !d.winOK || y0 != d.winY0 || y1 != d.winY1 {
		d.sendCmd(RASET)
		d.cmd = [4]byte{byte(y0 >> 8), byte(y0), byte(y1 >> 8), byte(y1)}
		d.bus.Tx(d.cmd[:], nil)
	}
	d.winX0, d.winX1, d.winY0, d.winY1, d.winOK = x0, x1, y0, y1, true
	d.sendCmd(RAMWR)
}

// RGB565 returns c as the two bytes the panel expects (big endian).
func RGB565(c color.RGBA) (hi, lo byte) {
	return c.R&0xF8 | c.G>>5, (c.G&0x1C)<<3 | c.B>>3
}

// ---- drivers.Displayer and hal.Display ----

// Size returns the screen size after MADCTL.
func (d *Device) Size() (int16, int16) { return d.cfg.Width, d.cfg.Height }

// SetPixel draws one pixel (joined into a run with the pixels around it).
func (d *Device) SetPixel(x, y int16, c color.RGBA) {
	if x < 0 || y < 0 || x >= d.cfg.Width || y >= d.cfg.Height {
		return
	}
	if d.runN > 0 && y == d.runY && x == d.runX+d.runN && c == d.runC && d.runN < bufSize/2 {
		d.runN++
		return
	}
	d.flush()
	d.runX, d.runY, d.runN, d.runC = x, y, 1, c
}

// flush sends the pending run of pixels.
func (d *Device) flush() {
	if d.runN == 0 {
		return
	}
	n := d.runN
	d.runN = 0
	d.fill(d.runX, d.runY, n, 1, d.runC)
}

// Display sends the pending pixels. Nothing else is buffered.
func (d *Device) Display() error {
	d.flush()
	return nil
}

// FillRectangle fills a rectangle (clipped to the screen).
func (d *Device) FillRectangle(x, y, w, h int16, c color.RGBA) error {
	d.flush()
	x, y, w, h, ok := d.clip(x, y, w, h)
	if ok {
		d.fill(x, y, w, h, c)
	}
	return nil
}

func (d *Device) clip(x, y, w, h int16) (int16, int16, int16, int16, bool) {
	if x < 0 {
		w += x
		x = 0
	}
	if y < 0 {
		h += y
		y = 0
	}
	if x+w > d.cfg.Width {
		w = d.cfg.Width - x
	}
	if y+h > d.cfg.Height {
		h = d.cfg.Height - y
	}
	return x, y, w, h, w > 0 && h > 0
}

// fill sends a rectangle of one colour (already clipped).
func (d *Device) fill(x, y, w, h int16, c color.RGBA) {
	hi, lo := RGB565(c)
	total := int(w) * int(h) * 2
	n := total
	if n > bufSize {
		n = bufSize
	}
	for i := 0; i < n; i += 2 {
		d.buf[i], d.buf[i+1] = hi, lo
	}
	d.cs.Set(false)
	d.window(x, y, x+w-1, y+h-1)
	for total > 0 {
		k := total
		if k > bufSize {
			k = bufSize
		}
		d.bus.Tx(d.buf[:k], nil)
		total -= k
	}
	d.cs.Set(true)
}

// DrawRGBBitmap8 draws w x h pixels of big endian RGB565 bytes (the
// format the panel takes, so they are sent as they are).
func (d *Device) DrawRGBBitmap8(x, y int16, data []uint8, w, h int16) error {
	d.flush()
	if len(data) < int(w)*int(h)*2 {
		return errShortData
	}
	cx, cy, cw, ch, ok := d.clip(x, y, w, h)
	if !ok {
		return nil
	}
	d.cs.Set(false)
	d.window(cx, cy, cx+cw-1, cy+ch-1)
	if cw == w {
		// Whole rows: one transfer.
		start := int(cy-y) * int(w) * 2
		d.bus.Tx(data[start:start+int(cw)*int(ch)*2], nil)
	} else {
		for row := cy - y; row < cy-y+ch; row++ {
			start := (int(row)*int(w) + int(cx-x)) * 2
			d.bus.Tx(data[start:start+int(cw)*2], nil)
		}
	}
	d.cs.Set(true)
	return nil
}

type errorString string

func (e errorString) Error() string { return string(e) }

const errShortData = errorString("spilcd: bitmap data too short")
