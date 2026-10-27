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
//	d  read the panel ID over MISO (GPIO12), without and with USR_MISO
//
// Flash: make flash-s035 PKG=./examples/s035_check
package main

import (
	"device/esp"
	"machine"
	"time"

	"github.com/sat0ken/tinygo-cyd/board/s035"
	"github.com/sat0ken/tinygo-cyd/gt911"
	"github.com/sat0ken/tinygo-cyd/internal/panelcheck"
	"github.com/sat0ken/tinygo-cyd/platform"
	"github.com/sat0ken/tinygo-cyd/spilcd"
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
		Serial:    machine.Serial,
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

// readID reads the panel ID at 10MHz. TinyGo's ESP32 SPI driver enables
// full duplex (DOUTDIN) and the write phase (USR_MOSI) but not the read
// phase (USR_MISO), which ESP-IDF also sets for full duplex; the second
// try sets it to see whether the bytes then come in. (On this board every
// read gives zeros: MISO does not seem to reach the panel.)
func readID(d *spilcd.Device) {
	spi := machine.SPI2
	cfg := machine.SPIConfig{
		SCK: machine.Pin(s035.PinSCK), SDO: machine.Pin(s035.PinMOSI), SDI: machine.Pin(s035.PinMISO),
		Frequency: 10_000_000,
	}
	spi.Configure(cfg)
	for _, miso := range []bool{false, true} {
		if miso {
			esp.SPI2.USER.SetBits(esp.SPI_USER_USR_MISO)
		}
		a := d.Read(spilcd.RDDID, 4)
		b := d.Read(spilcd.RDID4, 4)
		println("USR_MISO", miso, " RDDID(04)", panelcheck.HexBytes(a[:]), " RDID4(d3)", panelcheck.HexBytes(b[:]))
	}
	cfg.Frequency = s035.SPIFrequency
	spi.Configure(cfg) // also clears USR_MISO again
	println("(ST7796: RDDID 7796xx / RDID4 007796, maybe shifted by a dummy bit)")
}
