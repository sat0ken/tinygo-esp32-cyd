//go:build !esp32_3248s035c

// Package platform initialises the display and touch for the current build
// target: the ESP32-4827S043 (esp32s3), the ESP32-3248S035C (build tag
// esp32_3248s035c), the browser (js/wasm) or the host (tests).
//
// config.go is the ESP32-4827S043 configuration; it also builds on the
// host and in the browser, where the values are used for the screen size.
package platform

import (
	"github.com/sat0ken/tinygo-cyd/board"
	"github.com/sat0ken/tinygo-cyd/rgblcd"
	"github.com/sat0ken/tinygo-cyd/xpttouch"
)

// RGBConfig returns the rgblcd configuration of the ESP32-4827S043.
func RGBConfig() rgblcd.Config {
	return rgblcd.Config{
		Width:    board.Width,
		Height:   board.Height,
		DataPins: board.DataPins(),
		DE:       board.PinDE,
		VSync:    board.PinVSync,
		HSync:    board.PinHSync,
		PCLK:     board.PinPCLK,
		Timing: rgblcd.Timing{
			PclkHz:          board.PclkHz,
			HSyncPulseWidth: board.HSyncPulseWidth,
			HSyncBackPorch:  board.HSyncBackPorch,
			HSyncFrontPorch: board.HSyncFrontPorch,
			VSyncPulseWidth: board.VSyncPulseWidth,
			VSyncBackPorch:  board.VSyncBackPorch,
			VSyncFrontPorch: board.VSyncFrontPorch,
			HSyncIdleLow:    board.HSyncIdleLow,
			VSyncIdleLow:    board.VSyncIdleLow,
			DEIdleHigh:      board.DEIdleHigh,
			PclkActiveNeg:   board.PclkActiveNeg,
			PclkIdleHigh:    board.PclkIdleHigh,
		},
		ClockSource: rgblcd.ClockPLL160M,
		DMAChannel:  0,
	}
}

// TouchCalibration returns the touch calibration of the board.
func TouchCalibration() xpttouch.Calibration {
	return xpttouch.Calibration{
		Width: board.Width, Height: board.Height,
		RawXLeft: board.TouchRawXLeft, RawXRight: board.TouchRawXRight,
		RawYTop: board.TouchRawYTop, RawYBottom: board.TouchRawYBottom,
		SwapXY: board.TouchSwapXY,
	}
}
