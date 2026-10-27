//go:build esp32_2432s028r

// s028_check brings up the ESP32-2432S028R: the ILI9341 panel (320x240
// landscape) and the XPT2046 touch controller.
//
// The screen shows eight colour bars (red, green, blue, cyan, magenta,
// yellow, white, black from the left), a white border and "TOP LEFT" in the
// top-left corner. Touching draws dots and logs the raw and screen
// positions. The touch calibration in board/s028 is only a guess: measure
// it with "k".
//
// Serial commands (115200 baud), see internal/panelcheck:
//
//	r  turn the picture by 180 degrees (MADCTL 0x28 <-> 0xE8)
//	o  swap red and blue (MADCTL BGR bit)
//	i  display inversion on/off (the two-USB version needs it on)
//	c  draw the test screen again
//	k  calibrate the touch panel: touch the four targets
//	f  speed: fill the screen 10 times
//	t  speed: draw 300 characters
//	d  read the panel ID over MISO (GPIO12), without and with USR_MISO
//
// Flash: make flash-s028 PKG=./examples/s028_check
package main

import (
	"device/esp"
	"machine"
	"time"

	"github.com/sat0ken/tinygo-cyd/board/s028"
	"github.com/sat0ken/tinygo-cyd/internal/panelcheck"
	"github.com/sat0ken/tinygo-cyd/platform"
	"github.com/sat0ken/tinygo-cyd/spilcd"
	"github.com/sat0ken/tinygo-cyd/touchcal"
	"github.com/sat0ken/tinygo-cyd/xpttouch"
)

func main() {
	time.Sleep(time.Second)
	println("\n=== s028_check ===")
	d, err := platform.InitScreen()
	if err != nil {
		println("InitScreen:", err.Error())
		platform.Halt()
	}
	readID(d)
	panelcheck.Run(panelcheck.Config{
		Name:      "s028_check",
		BoardFile: "board/s028/s028.go",
		Panel:     d,
		Touch:     &touch{t: platform.InitTouch()},
		Serial:    machine.Serial,
		MinSpanX:  1000, MinSpanY: 500, // 12-bit XPT2046 readings
		ReadID: func() { readID(d) },
	})
}

// touch adds Raw and SetCalibration to the XPT2046, and reads the raw
// value once per ReadTouch.
type touch struct {
	t      *xpttouch.Touch
	rx, ry int32
}

func (t *touch) ReadTouch() (x, y int16, pressed bool) {
	rx, ry, ok := t.t.ReadRaw()
	if !ok {
		return 0, 0, false
	}
	t.rx, t.ry = rx, ry
	x, y = t.t.Cal.Map(rx, ry)
	return x, y, true
}

func (t *touch) Raw() (x, y int32) { return t.rx, t.ry }

func (t *touch) SetCalibration(c touchcal.Calibration) { t.t.Cal = c }

// readID reads the panel ID at 10MHz, without and with the SPI read phase
// (USR_MISO, which TinyGo's ESP32 SPI driver does not set). On this board
// MISO is wired to the panel, so an ILI9341 should answer RDDID with
// 00 93 41 (after a dummy bit or byte) and an ST7789 with 85 85 52.
func readID(d *spilcd.Device) {
	spi := machine.SPI2
	cfg := machine.SPIConfig{
		SCK: machine.Pin(s028.PinSCK), SDO: machine.Pin(s028.PinMOSI), SDI: machine.Pin(s028.PinMISO),
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
	cfg.Frequency = s028.SPIFrequency
	spi.Configure(cfg) // also clears USR_MISO again
}
