//go:build esp32_3248s035c

package platform

import (
	"device/esp"
	"errors"
	"image/color"
	"machine"
	"time"

	"github.com/sat0ken/tinygo-cyd/board/s035"
	"github.com/sat0ken/tinygo-cyd/gt911"
	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/st7796"
	"github.com/sat0ken/tinygo-cyd/touchcal"
	"github.com/sat0ken/tinygo-cyd/viewport"
)

// The ESP32-3248S035C: ST7796 on SPI2 and GT911 on I2C0 (package
// board/s035). There is no frame buffer: drawing goes straight to the
// panel.

var (
	screen *st7796.Device
	touch  *gt911.Touch
)

// InitScreen initialises the SPI bus and the ST7796 (480x320, landscape),
// clears the screen and turns the backlight on. It reads the panel ID at a
// slow SPI clock first and prints it (only for the log: not every panel
// answers).
func InitScreen() (*st7796.Device, error) {
	if screen != nil {
		return screen, nil
	}
	spi := machine.SPI2
	cfg := machine.SPIConfig{
		SCK: machine.Pin(s035.PinSCK), SDO: machine.Pin(s035.PinMOSI), SDI: machine.Pin(s035.PinMISO),
		Frequency: 10_000_000,
	}
	if err := spi.Configure(cfg); err != nil {
		return nil, err
	}
	dc := machine.Pin(s035.PinDC)
	dc.Configure(machine.PinConfig{Mode: machine.PinOutput})
	cs := machine.Pin(s035.PinCS)
	cs.Configure(machine.PinConfig{Mode: machine.PinOutput})
	cs.Set(true)

	d := st7796.New(spi, dc, cs)
	d.Configure(st7796.Config{Width: s035.Width, Height: s035.Height, MADCTL: s035.MADCTL, Invert: s035.Invert})
	id := d.Read(st7796.RDID4, 4)
	println("st7796: RDID4", hex(id[:]), "(expected 00 77 96 after a dummy byte or bit)")

	cfg.Frequency = s035.SPIFrequency
	if err := spi.Configure(cfg); err != nil {
		return nil, err
	}
	d.FillRectangle(0, 0, s035.Width, s035.Height, color.RGBA{0, 0, 0, 255})
	screen = d
	SetBacklight(100)
	return d, nil
}

// TouchCalibration returns the touch calibration of the board.
func TouchCalibration() touchcal.Calibration {
	return touchcal.Calibration{
		Width: s035.Width, Height: s035.Height,
		RawXLeft: s035.TouchRawXLeft, RawXRight: s035.TouchRawXRight,
		RawYTop: s035.TouchRawYTop, RawYBottom: s035.TouchRawYBottom,
		SwapXY: s035.TouchSwapXY,
	}
}

// InitTouch initialises I2C0 and finds the GT911. The positions are on the
// whole 480x320 screen. If the controller does not answer, it resets it
// with its RST pin and tries again.
func InitTouch() (*gt911.Touch, error) {
	if touch != nil {
		return touch, nil
	}
	i2c := machine.I2C0
	err := i2c.Configure(machine.I2CConfig{
		SCL: machine.Pin(s035.PinTouchSCL), SDA: machine.Pin(s035.PinTouchSDA),
		Frequency: s035.TouchI2CFrequency,
	})
	if err != nil {
		return nil, err
	}
	enableHighPins(s035.PinTouchSCL, s035.PinTouchSDA)
	dev := gt911.New(i2c)
	info, ok := dev.Probe()
	if !ok {
		println("gt911: no answer, resetting")
		rst := machine.Pin(s035.PinTouchRST)
		rst.Configure(machine.PinConfig{Mode: machine.PinOutput})
		rst.Set(false)
		time.Sleep(10 * time.Millisecond)
		rst.Set(true)
		time.Sleep(100 * time.Millisecond)
		info, ok = dev.Probe()
	}
	if !ok {
		return nil, errors.New("gt911: not found at 0x5D or 0x14")
	}
	println("gt911: address", hex([]byte{byte(dev.Addr)}), "product", info.ProductID,
		"firmware", hex([]byte{byte(info.Firmware >> 8), byte(info.Firmware)}),
		"resolution", info.XRes, "x", info.YRes, "config", info.Config)
	touch = gt911.NewTouch(dev, TouchCalibration())
	return touch, nil
}

// enableHighPins sets the output enable of GPIO32..39. TinyGo's ESP32 I2C
// driver (machine_esp32_i2c.go, initPins) enables the output with
// GPIO.ENABLE_W1TS.Set(1 << pin), which is 0 for pin 32 and up (they are
// in ENABLE1), so SCL (32) and SDA (33) of this board would never be
// driven. The bits stay set when the driver configures the pins again.
func enableHighPins(pins ...uint8) {
	for _, p := range pins {
		if p >= 32 {
			esp.GPIO.ENABLE1_W1TS.Set(1 << (p - 32))
		}
	}
}

// Init initialises the screen and the touch panel and returns them as a
// 480x272 window in the middle of the screen, so the code written for the
// ESP32-4827S043 runs as it is. Without touch it still returns the
// display (and a touch panel that is never pressed).
func Init() (hal.Display, hal.Touch, error) {
	d, err := InitScreen()
	if err != nil {
		return nil, nil, err
	}
	var t hal.Touch = hal.NoTouch{}
	if gt, err := InitTouch(); err != nil {
		println(err.Error())
	} else {
		t = gt
	}
	win := viewport.New(d, s035.WindowX, s035.WindowY, s035.WindowW, s035.WindowH)
	wt := viewport.NewTouch(t, s035.WindowX, s035.WindowY, s035.WindowW, s035.WindowH)
	return win, flushTouch{wt, d}, nil
}

// flushTouch sends the pixels the display still holds (see st7796) before
// reading the touch panel. Every game loop reads the touch panel once a
// frame, so whatever was drawn in the last frame is on the screen even if
// the code does not call Display.
type flushTouch struct {
	t hal.Touch
	d *st7796.Device
}

func (f flushTouch) ReadTouch() (x, y int16, pressed bool) {
	f.d.Display()
	return f.t.ReadTouch()
}

var blPin = machine.Pin(s035.PinBacklight)

// SetBacklight turns the backlight (GPIO27) on (percent > 0) or off. The
// brightness cannot be set yet: TinyGo has no PWM for the ESP32.
func SetBacklight(percent int) {
	blPin.Configure(machine.PinConfig{Mode: machine.PinOutput})
	blPin.Set(percent > 0)
}

func hex(b []byte) string {
	const digits = "0123456789abcdef"
	s := make([]byte, 0, len(b)*3)
	for i, v := range b {
		if i > 0 {
			s = append(s, ' ')
		}
		s = append(s, digits[v>>4], digits[v&15])
	}
	return string(s)
}
