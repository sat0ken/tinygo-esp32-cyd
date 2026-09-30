//go:build esp32s3

// 04_touch calibrates the XPT2046 touch panel and then draws a dot where the
// screen is touched.
//
//  1. Touch the four targets in the order shown (top-left, top-right,
//     bottom-right, bottom-left). Raw values are printed on the serial log.
//  2. The calibration is computed and printed as constants for board.go.
//  3. Afterwards every touch draws a dot at the converted position; the
//     dot should be under the stylus. Touch the top-left corner area
//     ("RECAL") to calibrate again.
//
// GPIO18 (touch INT) is shared with connectors P3/P4: keep it free there.
//
// Flash: tinygo flash -target=./targets/esp32-4827s043.json -monitor ./examples/04_touch
package main

import (
	"strconv"
	"time"

	"github.com/sat0ken/tinygo-cyd/app"
	"github.com/sat0ken/tinygo-cyd/board"
	"github.com/sat0ken/tinygo-cyd/platform"
	"github.com/sat0ken/tinygo-cyd/rgblcd"
	"github.com/sat0ken/tinygo-cyd/xpttouch"
	"tinygo.org/x/tinydraw"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/proggy"
)

// Targets are inset from the corners so they can be hit reliably.
const inset = 20

var targets = [4][2]int16{
	{inset, inset},
	{board.Width - 1 - inset, inset},
	{board.Width - 1 - inset, board.Height - 1 - inset},
	{inset, board.Height - 1 - inset},
}

func main() {
	time.Sleep(time.Second)
	println("\n=== 04_touch ===")
	d, err := platform.InitLCD()
	if err != nil {
		println("InitLCD:", err.Error())
		select {}
	}
	t := platform.InitTouch()
	println("default calibration:", calString(t.Cal))

	for {
		t.Cal = calibrate(d, t)
		println("calibration:", calString(t.Cal))
		println("// paste into board/board.go:")
		println("TouchRawXLeft   =", t.Cal.RawXLeft)
		println("TouchRawXRight  =", t.Cal.RawXRight)
		println("TouchRawYTop    =", t.Cal.RawYTop)
		println("TouchRawYBottom =", t.Cal.RawYBottom)
		println("TouchSwapXY     =", t.Cal.SwapXY)
		paint(d, t)
	}
}

// waitTap waits for a press and release and returns the average raw
// reading while pressed.
func waitTap(t *xpttouch.Touch) (int32, int32) {
	for {
		var sx, sy, n int32
		for {
			x, y, ok := t.ReadRaw()
			if ok {
				sx += x
				sy += y
				n++
			} else if n > 0 {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		if n >= 3 { // ignore very short glitches
			return sx / n, sy / n
		}
	}
}

func calibrate(d *rgblcd.Device, t *xpttouch.Touch) xpttouch.Calibration {
	var raw [4][2]int32
	for i, tg := range targets {
		d.FillScreen(app.Black)
		tinyfont.WriteLine(d, &proggy.TinySZ8pt7b, 150, 130, "touch the target "+strconv.Itoa(i+1)+"/4", app.White)
		tinydraw.Circle(d, tg[0], tg[1], 8, app.Red)
		d.DrawFastHLine(tg[0]-14, tg[0]+14, tg[1], app.White)
		d.DrawFastVLine(tg[0], tg[1]-14, tg[1]+14, app.White)
		x, y := waitTap(t)
		raw[i] = [2]int32{x, y}
		println("target", i+1, "screen", tg[0], tg[1], "raw", x, y)
	}

	// If raw X changes more between top-left and bottom-left than between
	// top-left and top-right, the axes are swapped.
	swap := abs(raw[3][0]-raw[0][0]) > abs(raw[1][0]-raw[0][0])
	if swap {
		for i := range raw {
			raw[i][0], raw[i][1] = raw[i][1], raw[i][0]
		}
	}
	// Average the two readings on each edge, then extrapolate from the
	// target position (inset) to the screen edge (0 and size-1).
	left := (raw[0][0] + raw[3][0]) / 2
	right := (raw[1][0] + raw[2][0]) / 2
	top := (raw[0][1] + raw[1][1]) / 2
	bottom := (raw[2][1] + raw[3][1]) / 2
	left, right = extrapolate(left, right, inset, board.Width-1-inset, board.Width-1)
	top, bottom = extrapolate(top, bottom, inset, board.Height-1-inset, board.Height-1)
	return xpttouch.Calibration{
		Width: board.Width, Height: board.Height,
		RawXLeft: left, RawXRight: right, RawYTop: top, RawYBottom: bottom,
		SwapXY: swap,
	}
}

// extrapolate maps raw readings a (at pixel pa) and b (at pixel pb) to the
// raw values at pixel 0 and pixel max.
func extrapolate(a, b int32, pa, pb, max int32) (int32, int32) {
	perPixel := float32(b-a) / float32(pb-pa)
	lo := a - int32(perPixel*float32(pa))
	hi := b + int32(perPixel*float32(max-pb))
	return lo, hi
}

func paint(d *rgblcd.Device, t *xpttouch.Touch) {
	d.FillScreen(app.Black)
	d.FillRectangle(0, 0, 60, 24, app.Red)
	tinyfont.WriteLine(d, &proggy.TinySZ8pt7b, 8, 16, "RECAL", app.White)
	tinyfont.WriteLine(d, &proggy.TinySZ8pt7b, 150, 130, "draw with the stylus", app.White)
	// Crosses at the calibration targets to check the accuracy.
	for _, tg := range targets {
		d.DrawFastHLine(tg[0]-6, tg[0]+6, tg[1], app.Cyan)
		d.DrawFastVLine(tg[0], tg[1]-6, tg[1]+6, app.Cyan)
	}
	last := time.Now()
	for {
		x, y, ok := t.ReadTouch()
		if ok {
			if x < 60 && y < 24 {
				for {
					if _, _, ok := t.ReadTouch(); !ok {
						break
					}
					time.Sleep(10 * time.Millisecond)
				}
				return
			}
			d.FillRectangle(x-1, y-1, 3, 3, app.Yellow)
			if time.Since(last) > 200*time.Millisecond {
				rx, ry, _ := t.ReadRaw()
				println("touch", x, y, "raw", rx, ry)
				last = time.Now()
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func calString(c xpttouch.Calibration) string {
	return "x " + strconv.Itoa(int(c.RawXLeft)) + ".." + strconv.Itoa(int(c.RawXRight)) +
		" y " + strconv.Itoa(int(c.RawYTop)) + ".." + strconv.Itoa(int(c.RawYBottom)) +
		" swap=" + strconv.FormatBool(c.SwapXY)
}

func abs(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}
