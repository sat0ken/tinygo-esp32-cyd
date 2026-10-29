//go:build esp32_2432s028r

package platform

import (
	"image/color"
	"machine"

	"github.com/sat0ken/tinygo-cyd/board/s028"
	"github.com/sat0ken/tinygo-cyd/espdma"
	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/spilcd"
	"github.com/sat0ken/tinygo-cyd/touchcal"
	"github.com/sat0ken/tinygo-cyd/xpttouch"
)

// The ESP32-2432S028R: ILI9341 on SPI2 and XPT2046 on its own pins
// (package board/s028). There is no frame buffer: drawing goes straight to
// the panel (package spilcd).

var (
	screen *spilcd.Device
	touch  *xpttouch.Touch
)

// InitScreen initialises the SPI bus and the ILI9341 (320x240, landscape),
// clears the screen and turns the backlight on. The init commands are sent
// at 10MHz, drawing at SPIFrequency.
func InitScreen() (*spilcd.Device, error) {
	if screen != nil {
		return screen, nil
	}
	spi := machine.SPI2
	cfg := machine.SPIConfig{
		SCK: machine.Pin(s028.PinSCK), SDO: machine.Pin(s028.PinMOSI), SDI: machine.Pin(s028.PinMISO),
		Frequency: 10_000_000,
	}
	if err := spi.Configure(cfg); err != nil {
		return nil, err
	}
	dc := machine.Pin(s028.PinDC)
	dc.Configure(machine.PinConfig{Mode: machine.PinOutput})
	cs := machine.Pin(s028.PinCS)
	cs.Configure(machine.PinConfig{Mode: machine.PinOutput})
	cs.Set(true)

	d := spilcd.New(espdma.New(spi), dc, cs) // DrawRGBBitmap8Async sends with DMA
	d.Configure(spilcd.Config{Width: s028.Width, Height: s028.Height, MADCTL: s028.MADCTL, Invert: s028.Invert, Init: spilcd.InitILI9341})
	cfg.Frequency = s028.SPIFrequency
	if err := spi.Configure(cfg); err != nil {
		return nil, err
	}
	d.FillRectangle(0, 0, s028.Width, s028.Height, color.RGBA{0, 0, 0, 255})
	screen = d
	SetBacklight(100)
	println("ili9341: ready, 320x240, SPI", s028.SPIFrequency/1_000_000, "MHz")
	return d, nil
}

// TouchCalibration returns the touch calibration of the board.
func TouchCalibration() touchcal.Calibration {
	return touchcal.Calibration{
		Width: s028.Width, Height: s028.Height,
		RawXLeft: s028.TouchRawXLeft, RawXRight: s028.TouchRawXRight,
		RawYTop: s028.TouchRawYTop, RawYBottom: s028.TouchRawYBottom,
		SwapXY: s028.TouchSwapXY,
	}
}

// InitTouch initialises the XPT2046 (bit-banged on its own pins). The
// positions are on the whole 320x240 screen.
func InitTouch() *xpttouch.Touch {
	if touch == nil {
		touch = xpttouch.New(
			machine.Pin(s028.PinTouchSCK), machine.Pin(s028.PinTouchCS),
			machine.Pin(s028.PinTouchMOSI), machine.Pin(s028.PinTouchMISO),
			machine.Pin(s028.PinTouchIRQ),
			TouchCalibration())
	}
	return touch
}

// Init initialises the screen and the touch panel. Unlike the
// ESP32-3248S035C there is no 480x272 window: the screen is 320x240, so
// only programs that adapt to the screen size work.
func Init() (hal.Display, hal.Touch, error) {
	d, err := InitScreen()
	if err != nil {
		return nil, nil, err
	}
	return d, flushTouch{InitTouch(), d}, nil
}

var blPin = machine.Pin(s028.PinBacklight)

// SetBacklight turns the backlight (GPIO21) on (percent > 0) or off. The
// brightness cannot be set yet: TinyGo has no PWM for the ESP32.
func SetBacklight(percent int) {
	blPin.Configure(machine.PinConfig{Mode: machine.PinOutput})
	blPin.Set(percent > 0)
}
