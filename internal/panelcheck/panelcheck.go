// Package panelcheck is the common part of the bring-up programs of the SPI
// panel boards (examples/s035_check, examples/s028_check): a test screen,
// serial commands to try panel settings, touch logging, touch calibration
// and drawing speed.
//
// It does not import machine: the board program passes the serial input,
// the panel and the touch panel in.
package panelcheck

import (
	"image/color"
	"strconv"
	"time"

	"github.com/sat0ken/tinygo-cyd/romfont/freesans"
	"github.com/sat0ken/tinygo-cyd/romfont/proggy"
	"github.com/sat0ken/tinygo-cyd/spilcd"
	"github.com/sat0ken/tinygo-cyd/touchcal"
	"tinygo.org/x/tinydraw"
	"tinygo.org/x/tinyfont"
)

// Touch is a touch panel that also gives its raw readings.
type Touch interface {
	ReadTouch() (x, y int16, pressed bool)
	// Raw returns the raw reading of the last ReadTouch that was pressed.
	Raw() (x, y int32)
	SetCalibration(touchcal.Calibration)
}

// Serial is the serial input (machine.Serial).
type Serial interface {
	Buffered() int
	ReadByte() (byte, error)
}

// Config describes the board.
type Config struct {
	Name      string // "s035_check"
	BoardFile string // "board/s035/s035.go", for the calibration printout
	Panel     *spilcd.Device
	Touch     Touch // nil if there is none
	Serial    Serial
	// Window, if not zero, is drawn dotted on the test screen (the 480x272
	// window of the games on a 480x320 panel).
	Window [4]int16
	// MinSpanX/Y reject calibration taps that cannot be the targets (in raw
	// units between the targets).
	MinSpanX, MinSpanY int32
	// ReadID, if set, is run by the "d" command.
	ReadID func()
}

var (
	black  = color.RGBA{0, 0, 0, 255}
	white  = color.RGBA{255, 255, 255, 255}
	yellow = color.RGBA{255, 255, 0, 255}
	gray   = color.RGBA{128, 128, 128, 255}
	bars   = []color.RGBA{
		{255, 0, 0, 255}, {0, 255, 0, 255}, {0, 0, 255, 255}, {0, 255, 255, 255},
		{255, 0, 255, 255}, {255, 255, 0, 255}, white, black,
	}
	barNames = []string{"RED", "GREEN", "BLUE", "CYAN", "MAGENTA", "YELLOW", "WHITE", "BLACK"}
)

// Run draws the test screen and then handles serial commands and touches
// forever.
func Run(c Config) {
	c.drawTest()
	c.help()
	var lastLog time.Time
	for {
		for c.Serial.Buffered() > 0 {
			b, _ := c.Serial.ReadByte()
			c.command(b)
		}
		if c.Touch != nil {
			if x, y, pressed := c.Touch.ReadTouch(); pressed {
				c.Panel.FillRectangle(x-2, y-2, 5, 5, yellow)
				if time.Since(lastLog) > 100*time.Millisecond {
					rx, ry := c.Touch.Raw()
					println("touch raw", rx, ry, "-> screen", x, y)
					lastLog = time.Now()
				}
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (c *Config) help() {
	println("commands: r = turn 180, o = swap red/blue, i = inversion, c = redraw,")
	println("          k = calibrate touch, f = fill speed, t = text speed, d = read panel ID")
}

func (c *Config) command(b byte) {
	d := c.Panel
	cfg := d.Config()
	w, h := d.Size()
	switch b {
	case 'r':
		d.SetMADCTL(cfg.MADCTL ^ 0xC0) // MY and MX
		println("MADCTL", HexByte(cfg.MADCTL^0xC0), "(the touch calibration is for the old one - use k)")
		c.drawTest()
	case 'o':
		d.SetMADCTL(cfg.MADCTL ^ 0x08)
		println("MADCTL", HexByte(cfg.MADCTL^0x08))
		c.drawTest()
	case 'i':
		d.SetInvert(!cfg.Invert)
		println("invert", !cfg.Invert)
	case 'c':
		c.drawTest()
	case 'k':
		if c.Touch == nil {
			println("no touch panel")
			return
		}
		c.calibrate()
		c.drawTest()
	case 'f':
		start := time.Now()
		for i := 0; i < 10; i++ {
			d.FillRectangle(0, 0, w, h, bars[i%len(bars)])
		}
		ms := time.Since(start).Milliseconds()
		println("fill", w, "x", h, "x10:", ms, "ms =", ms/10, "ms per screen")
		c.drawTest()
	case 't':
		d.FillRectangle(0, 0, w, h, black)
		start := time.Now()
		for row := int16(0); row < 10; row++ {
			tinyfont.WriteLine(d, &freesans.Bold9pt7b, 4, 22+row*22, "The quick brown fox jumps 0123", white)
		}
		d.Display()
		println("300 characters (freesans bold 9pt):", time.Since(start).Milliseconds(), "ms")
		time.Sleep(2 * time.Second)
		c.drawTest()
	case 'd':
		if c.ReadID != nil {
			c.ReadID()
		}
	case '\r', '\n':
	default:
		c.help()
	}
}

// drawTest draws eight colour bars (red, green, blue, cyan, magenta,
// yellow, white, black from the left), a white border, "TOP LEFT" in the
// top-left corner and the size and MADCTL at the bottom.
func (c *Config) drawTest() {
	d := c.Panel
	w, h := d.Size()
	bw := w / int16(len(bars))
	for i, col := range bars {
		d.FillRectangle(int16(i)*bw, 0, bw, h, col)
	}
	// Names in the middle of the bars, on a gray band.
	d.FillRectangle(0, h/2-12, w, 24, gray)
	for i, n := range barNames {
		_, tw := tinyfont.LineWidth(&proggy.TinySZ8pt7b, n)
		tinyfont.WriteLine(d, &proggy.TinySZ8pt7b, int16(i)*bw+(bw-int16(tw))/2, h/2+4, n, white)
	}
	// White border on the outermost pixels.
	d.FillRectangle(0, 0, w, 1, white)
	d.FillRectangle(0, h-1, w, 1, white)
	d.FillRectangle(0, 0, 1, h, white)
	d.FillRectangle(w-1, 0, 1, h, white)
	if win := c.Window; win[2] > 0 {
		for x := int16(0); x < w; x += 8 {
			d.FillRectangle(x, win[1], 4, 1, white)
			d.FillRectangle(x, win[1]+win[3]-1, 4, 1, white)
		}
	}
	d.FillRectangle(4, 4, 96, 20, black)
	tinyfont.WriteLine(d, &freesans.Bold9pt7b, 8, 19, "TOP LEFT", white)
	tinydraw.Line(d, 2, 2, 12, 2, yellow)
	tinydraw.Line(d, 2, 2, 2, 12, yellow)
	label := strconv.Itoa(int(w)) + "x" + strconv.Itoa(int(h)) + "  MADCTL " + HexByte(d.Config().MADCTL)
	_, lw := tinyfont.LineWidth(&proggy.TinySZ8pt7b, label)
	d.FillRectangle((w-int16(lw))/2-6, h-60, int16(lw)+12, 22, black)
	tinyfont.WriteLine(d, &proggy.TinySZ8pt7b, (w-int16(lw))/2, h-45, label, white)
	d.Display()
}

// ---- touch calibration ----

const inset = 20

func (c *Config) calibrate() {
	d := c.Panel
	w, h := d.Size()
	targets := [4][2]int16{{inset, inset}, {w - 1 - inset, inset}, {w - 1 - inset, h - 1 - inset}, {inset, h - 1 - inset}}
	var raw [4][2]int32
	d.FillRectangle(0, 0, w, h, black)
	tinyfont.WriteLine(d, &freesans.Bold9pt7b, w/2-70, h/2, "touch the targets", white)
	for i, p := range targets {
		cross(d, p[0], p[1], yellow)
		raw[i] = c.waitTap()
		println("target", i, "at", p[0], p[1], "raw", raw[i][0], raw[i][1])
		cross(d, p[0], p[1], black)
	}
	cal, err := touchcal.FromCorners(raw, w, h, inset, c.MinSpanX, c.MinSpanY)
	if err != nil {
		println("calibration NG:", err.Error(), "- k to try again")
		return
	}
	c.Touch.SetCalibration(cal)
	println("calibration OK. For", c.BoardFile, "(MADCTL", HexByte(d.Config().MADCTL)+"):")
	println("TouchRawXLeft   =", cal.RawXLeft)
	println("TouchRawXRight  =", cal.RawXRight)
	println("TouchRawYTop    =", cal.RawYTop)
	println("TouchRawYBottom =", cal.RawYBottom)
	println("TouchSwapXY     =", cal.SwapXY)
}

func cross(d *spilcd.Device, x, y int16, c color.RGBA) {
	d.FillRectangle(x-10, y, 21, 1, c)
	d.FillRectangle(x, y-10, 1, 21, c)
	d.Display()
}

// waitTap waits for a touch and its release and returns the average raw
// position while touched.
func (c *Config) waitTap() [2]int32 {
	for {
		var sx, sy, n int32
		for {
			if _, _, pressed := c.Touch.ReadTouch(); pressed {
				rx, ry := c.Touch.Raw()
				sx += rx
				sy += ry
				n++
			} else if n > 0 {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		if n >= 3 {
			return [2]int32{sx / n, sy / n}
		}
	}
}

// HexByte returns b as "0x2a".
func HexByte(b byte) string {
	s := strconv.FormatUint(uint64(b), 16)
	if len(s) == 1 {
		s = "0" + s
	}
	return "0x" + s
}

// HexBytes returns b as "00 77 96".
func HexBytes(b []byte) string {
	s := ""
	for i, v := range b {
		if i > 0 {
			s += " "
		}
		s += HexByte(v)[2:]
	}
	return s
}
