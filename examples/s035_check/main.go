//go:build esp32_3248s035c

// s035_check brings up the ESP32-3248S035C: the ST7796 panel (480x320
// landscape) and the GT911 touch controller.
//
// The log shows the panel ID and the GT911 product, firmware and
// resolution. The screen shows eight colour bars (red, green, blue, cyan,
// magenta, yellow, white, black from the left), a white border, "TOP LEFT"
// in the top-left corner and the 480x272 window the games run in (dotted).
// Touching draws dots and logs the raw and screen positions.
//
// Serial commands (115200 baud):
//
//	r  turn the picture by 180 degrees (MADCTL 0x28 <-> 0xE8)
//	o  swap red and blue (MADCTL BGR bit)
//	i  display inversion on/off
//	c  draw the test screen again
//	k  calibrate the touch panel: touch the four targets
//	f  speed: fill the screen 10 times
//	t  speed: draw 300 characters
//
// Flash: make flash TARGET=./targets/esp32-3248s035c.json PKG=./examples/s035_check
package main

import (
	"image/color"
	"machine"
	"strconv"
	"time"

	"github.com/sat0ken/tinygo-cyd/board/s035"
	"github.com/sat0ken/tinygo-cyd/gt911"
	"github.com/sat0ken/tinygo-cyd/platform"
	"github.com/sat0ken/tinygo-cyd/st7796"
	"github.com/sat0ken/tinygo-cyd/touchcal"
	"tinygo.org/x/tinydraw"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/freesans"
	"tinygo.org/x/tinyfont/proggy"
)

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

func main() {
	time.Sleep(time.Second)
	println("\n=== s035_check ===")
	d, err := platform.InitScreen()
	if err != nil {
		println("InitScreen:", err.Error())
		platform.Halt()
	}
	t, err := platform.InitTouch()
	if err != nil {
		println("InitTouch:", err.Error())
	}
	drawTest(d)
	help()

	uart := machine.Serial
	var lastLog time.Time
	for {
		for uart.Buffered() > 0 {
			b, _ := uart.ReadByte()
			command(d, t, b)
		}
		if t != nil {
			if x, y, pressed := t.ReadTouch(); pressed {
				d.FillRectangle(x-2, y-2, 5, 5, yellow)
				if time.Since(lastLog) > 100*time.Millisecond {
					raw := t.Raw()
					println("touch raw", raw.X, raw.Y, "size", raw.Size, "-> screen", x, y)
					lastLog = time.Now()
				}
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func help() {
	println("commands: r = turn 180, o = swap red/blue, i = inversion, c = redraw,")
	println("          k = calibrate touch, f = fill speed, t = text speed")
}

func command(d *st7796.Device, t *gt911.Touch, b byte) {
	cfg := d.Config()
	switch b {
	case 'r':
		d.SetMADCTL(cfg.MADCTL ^ 0xC0) // MY and MX
		println("MADCTL", hexByte(cfg.MADCTL^0xC0), "(touch calibration is for", hexByte(s035.MADCTL), "- use k)")
		drawTest(d)
	case 'o':
		d.SetMADCTL(cfg.MADCTL ^ 0x08)
		println("MADCTL", hexByte(cfg.MADCTL^0x08))
		drawTest(d)
	case 'i':
		d.SetInvert(!cfg.Invert)
		println("invert", !cfg.Invert)
	case 'c':
		drawTest(d)
	case 'k':
		if t == nil {
			println("no touch panel")
			return
		}
		calibrate(d, t)
		drawTest(d)
	case 'f':
		start := time.Now()
		for i := 0; i < 10; i++ {
			d.FillRectangle(0, 0, s035.Width, s035.Height, bars[i%len(bars)])
		}
		ms := time.Since(start).Milliseconds()
		println("fill 480x320 x10:", ms, "ms =", ms/10, "ms per screen")
		drawTest(d)
	case 't':
		d.FillRectangle(0, 0, s035.Width, s035.Height, black)
		start := time.Now()
		for row := int16(0); row < 10; row++ {
			tinyfont.WriteLine(d, &freesans.Bold9pt7b, 4, 24+row*28, "The quick brown fox jumps 0123", white)
		}
		d.Display()
		println("300 characters (freesans bold 9pt):", time.Since(start).Milliseconds(), "ms")
		time.Sleep(2 * time.Second)
		drawTest(d)
	case '\r', '\n':
	default:
		help()
	}
}

func drawTest(d *st7796.Device) {
	w, h := d.Size()
	bw := w / int16(len(bars))
	for i, c := range bars {
		d.FillRectangle(int16(i)*bw, 0, bw, h, c)
	}
	// Names under the middle of the bars, on a gray band.
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
	// The 480x272 window of the games, dotted.
	for x := int16(0); x < w; x += 8 {
		d.FillRectangle(x, s035.WindowY, 4, 1, white)
		d.FillRectangle(x, s035.WindowY+s035.WindowH-1, 4, 1, white)
	}
	d.FillRectangle(4, 4, 96, 20, black)
	tinyfont.WriteLine(d, &freesans.Bold9pt7b, 8, 19, "TOP LEFT", white)
	tinydraw.Line(d, 2, 2, 12, 2, yellow)
	tinydraw.Line(d, 2, 2, 2, 12, yellow)
	label := "480x320  MADCTL " + hexByte(d.Config().MADCTL)
	d.FillRectangle(w/2-90, h-60, 180, 22, black)
	tinyfont.WriteLine(d, &proggy.TinySZ8pt7b, w/2-84, h-45, label, white)
	d.Display()
}

// ---- touch calibration ----

const inset = 20

func calibrate(d *st7796.Device, t *gt911.Touch) {
	w, h := d.Size()
	targets := [4][2]int16{{inset, inset}, {w - 1 - inset, inset}, {w - 1 - inset, h - 1 - inset}, {inset, h - 1 - inset}}
	var raw [4][2]int32
	d.FillRectangle(0, 0, w, h, black)
	tinyfont.WriteLine(d, &freesans.Bold9pt7b, w/2-90, h/2, "touch the targets", white)
	for i, p := range targets {
		cross(d, p[0], p[1], yellow)
		raw[i] = waitTap(t)
		println("target", i, "at", p[0], p[1], "raw", raw[i][0], raw[i][1])
		cross(d, p[0], p[1], black)
	}
	cal, err := touchcal.FromCorners(raw, w, h, inset, 100, 60)
	if err != nil {
		println("calibration NG:", err.Error(), "- k to try again")
		return
	}
	t.Cal = cal
	println("calibration OK. For board/s035/s035.go (MADCTL", hexByte(d.Config().MADCTL)+"):")
	println("TouchRawXLeft   =", cal.RawXLeft)
	println("TouchRawXRight  =", cal.RawXRight)
	println("TouchRawYTop    =", cal.RawYTop)
	println("TouchRawYBottom =", cal.RawYBottom)
	println("TouchSwapXY     =", cal.SwapXY)
}

func cross(d *st7796.Device, x, y int16, c color.RGBA) {
	d.FillRectangle(x-10, y, 21, 1, c)
	d.FillRectangle(x, y-10, 1, 21, c)
	d.Display()
}

// waitTap waits for a touch and its release and returns the average raw
// position while touched.
func waitTap(t *gt911.Touch) [2]int32 {
	for {
		var sx, sy, n int32
		for {
			_, _, pressed := t.ReadTouch()
			if pressed {
				r := t.Raw()
				sx += int32(r.X)
				sy += int32(r.Y)
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

func hexByte(b byte) string {
	s := strconv.FormatUint(uint64(b), 16)
	if len(s) == 1 {
		s = "0" + s
	}
	return "0x" + s
}
