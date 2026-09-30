//go:build esp32s3

package platform

import (
	"machine"
	"unsafe"

	"github.com/sat0ken/tinygo-cyd/board"
	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/rgblcd"
	"github.com/sat0ken/tinygo-cyd/xpttouch"
)

// fbMem is the frame buffer (480*272*2 = 261,120 bytes) in internal SRAM.
// It is declared as uint32 so that it is 4-byte aligned for the GDMA.
var fbMem [board.Width * board.Height / 2]uint32

// FrameBuffer returns fbMem as RGB565 pixels.
func FrameBuffer() []uint16 {
	return unsafe.Slice((*uint16)(unsafe.Pointer(&fbMem[0])), board.Width*board.Height)
}

var (
	lcd      *rgblcd.Device
	blPWM    = machine.PWM0
	blCh     uint8
	blConfig bool
)

// InitLCD initialises and starts the RGB panel and turns the backlight on.
func InitLCD() (*rgblcd.Device, error) {
	if lcd != nil {
		return lcd, nil
	}
	d, err := rgblcd.New(RGBConfig(), FrameBuffer())
	if err != nil {
		return nil, err
	}
	d.Start()
	lcd = d
	SetBacklight(100)
	return d, nil
}

// SetBacklight sets the backlight (GPIO2) brightness in percent with PWM.
func SetBacklight(percent int) {
	if !blConfig {
		if err := blPWM.Configure(machine.PWMConfig{Period: 1e9 / 5000}); err != nil {
			println("backlight pwm:", err.Error())
			return
		}
		ch, err := blPWM.Channel(machine.Pin(board.PinBacklight))
		if err != nil {
			println("backlight channel:", err.Error())
			return
		}
		blCh = ch
		blConfig = true
	}
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	blPWM.Set(blCh, blPWM.Top()*uint32(percent)/100)
}

// InitTouch initialises the XPT2046 with the calibration from package board.
func InitTouch() *xpttouch.Touch {
	return xpttouch.New(
		machine.Pin(board.PinTouchSCK), machine.Pin(board.PinTouchCS),
		machine.Pin(board.PinTouchMOSI), machine.Pin(board.PinTouchMISO),
		machine.Pin(board.PinTouchINT),
		TouchCalibration())
}

// Init initialises the display and touch panel of the board.
func Init() (hal.Display, hal.Touch, error) {
	d, err := InitLCD()
	if err != nil {
		return nil, nil, err
	}
	return d, InitTouch(), nil
}
