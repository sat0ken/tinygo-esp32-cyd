// Package board holds every board-specific value of the Sunton
// ESP32-4827S043 (ESP32-S3, 4.3" 480x272 RGB565 parallel TFT).
//
// Nothing here depends on the machine package, so the constants can be used
// from host tests and from the wasm build as well.
package board

// Panel resolution.
const (
	Width  = 480
	Height = 272
)

// GPIO numbers of the RGB panel.
//
// Source: Arduino_GFX ESP32-4827S043 setting
// (Arduino_ESP32RGBPanel(40, 41, 39, 42, R:45,48,47,21,14, G:5,6,7,15,16,4,
// B:8,3,46,9,1, ...)).
const (
	PinDE    = 40
	PinVSync = 41
	PinHSync = 39
	PinPCLK  = 42

	PinBacklight = 2
)

// Colour data pins.
var (
	PinsR = [5]uint8{45, 48, 47, 21, 14} // R0..R4
	PinsG = [6]uint8{5, 6, 7, 15, 16, 4} // G0..G5
	PinsB = [5]uint8{8, 3, 46, 9, 1}     // B0..B4
)

// DataPins returns the GPIO for LCD_DATA_OUT0..15.
//
// The order is B0-B4, G0-G5, R0-R4 so that a uint16 in the frame buffer is a
// standard RGB565 value (red in the top 5 bits).
// Source: Arduino_GFX src/databus/Arduino_ESP32RGBPanel.cpp (non big-endian
// branch: data_gpio_nums[0..4]=b0..b4, [5..10]=g0..g5, [11..15]=r0..r4).
func DataPins() [16]uint8 {
	var p [16]uint8
	copy(p[0:5], PinsB[:])
	copy(p[5:11], PinsG[:])
	copy(p[11:16], PinsR[:])
	return p
}

// Timing of the panel.
//
// Source: Arduino_GFX ESP32-4827S043 setting (hsync_polarity 0, hfp 8, hpw 4,
// hbp 43, vsync_polarity 0, vfp 8, vpw 4, vbp 12, pclk_active_neg 1, 9MHz).
// Arduino_GFX maps polarity 0 to ESP-IDF "hsync_idle_low/vsync_idle_low = 1".
//
// The alternative hsync 1/1/43, vsync 3/1/12 is reported to cause jitter;
// do not use it. If the picture is unstable, lower PclkHz to 8MHz, then 7MHz.
const (
	PclkHz = 9_000_000

	HSyncFrontPorch = 8
	HSyncPulseWidth = 4
	HSyncBackPorch  = 43

	VSyncFrontPorch = 8
	VSyncPulseWidth = 4
	VSyncBackPorch  = 12

	HSyncIdleLow  = true // hsync_polarity = 0
	VSyncIdleLow  = true // vsync_polarity = 0
	DEIdleHigh    = false
	PclkActiveNeg = true // data changes on the falling edge
	PclkIdleHigh  = false
)

// XPT2046 resistive touch controller (R version of the board).
// It shares its SPI bus with the microSD slot.
//
// GPIO18 (INT) is also routed to the P3/P4 connectors: do not use it for
// anything else on those connectors while touch is in use.
const (
	PinTouchSCK  = 12
	PinTouchMISO = 13
	PinTouchMOSI = 11
	PinTouchCS   = 38
	PinTouchINT  = 18
)

// Serial console: UART0 (GPIO43 TX / GPIO44 RX) through the CH340C USB-UART.
const (
	PinUARTTX = 43
	PinUARTRX = 44
)

// Touch calibration: raw 12-bit XPT2046 readings at the left/right and
// top/bottom edges of the screen, as returned by xpttouch.ReadRaw.
//
// Measured with examples/04_touch on the board (2026-09-30): raw X grows
// from left to right, raw Y grows from bottom to top, axes not swapped.
// Panels differ slightly; run examples/04_touch again to re-measure.
const (
	TouchRawXLeft   = 172
	TouchRawXRight  = 3940
	TouchRawYTop    = 3884
	TouchRawYBottom = 329
	TouchSwapXY     = false
)

// Flash region for data that is too large for the program image (see
// package flashmap): examples/05_slideshow reads its slides here.
// The program itself starts at 0 and is far smaller than 8MB.
// Written with `make flash-slides`.
const (
	SlidesFlashOffset = 0x800000 // 8MB, 64KB aligned
	SlidesFlashSize   = 0x800000 // up to the end of the 16MB flash
)
