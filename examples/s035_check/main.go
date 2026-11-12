//go:build esp32_3248s035c

// s035_check brings up the ESP32-3248S035C: the ST7796 panel (480x320
// landscape) and the GT911 touch controller.
//
// The log shows the GT911 product, firmware and resolution. The screen
// shows eight colour bars (red, green, blue, cyan, magenta, yellow, white,
// black from the left), a white border, "TOP LEFT" in the top-left corner
// and the 480x272 window the games run in (dotted). Touching draws dots and
// logs the raw and screen positions.
//
// Serial commands (115200 baud), see internal/panelcheck:
//
//	r  turn the picture by 180 degrees (MADCTL 0x28 <-> 0xE8)
//	o  swap red and blue (MADCTL BGR bit)
//	i  display inversion on/off
//	c  draw the test screen again
//	k  calibrate the touch panel: touch the four targets
//	f  speed: fill the screen 10 times
//	t  speed: draw 300 characters
//	d  read the panel ID over MISO (s035.PinMISO), see readid_*.go
//
// Flash: make flash-s035 PKG=./examples/s035_check
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/board/s035"
	"github.com/sat0ken/tinygo-cyd/gt911"
	"github.com/sat0ken/tinygo-cyd/internal/panelcheck"
	"github.com/sat0ken/tinygo-cyd/platform"
	"github.com/sat0ken/tinygo-cyd/touchcal"
)

func main() {
	time.Sleep(time.Second)
	println("\n=== s035_check ===")
	d, err := platform.InitScreen()
	if err != nil {
		println("InitScreen:", err.Error())
		platform.Halt()
	}
	c := panelcheck.Config{
		Name:      "s035_check",
		BoardFile: "board/s035/s035.go",
		Panel:     d,
		Serial:    platform.SerialInput(),
		Window:    [4]int16{s035.WindowX, s035.WindowY, s035.WindowW, s035.WindowH},
		MinSpanX:  100, MinSpanY: 60, // the GT911 reports pixels
		ReadID: func() { readID(d) },
	}
	if t, err := platform.InitTouch(); err != nil {
		println("InitTouch:", err.Error())
	} else {
		c.Touch = touch{t}
	}
	panelcheck.Run(c)
}

// touch adds Raw and SetCalibration to the GT911.
type touch struct{ t *gt911.Touch }

func (t touch) ReadTouch() (x, y int16, pressed bool) { return t.t.ReadTouch() }

func (t touch) Raw() (x, y int32) {
	r := t.t.Raw()
	return int32(r.X), int32(r.Y)
}

func (t touch) SetCalibration(c touchcal.Calibration) { t.t.Cal = c }
