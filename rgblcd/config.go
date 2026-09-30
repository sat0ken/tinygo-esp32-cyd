// Package rgblcd drives an RGB parallel TFT from the ESP32-S3 LCD_CAM
// peripheral with GDMA, without a CPU in the loop.
//
// The frame buffer lives in internal SRAM. GDMA reads it through a circular
// descriptor list and LCD_CAM, with "next frame" enabled, keeps requesting
// frames forever, the same set-up ESP-IDF's esp_lcd RGB driver uses when the
// frame buffer is in internal RAM and no bounce buffer is used
// (components/esp_lcd/rgb/esp_lcd_panel_rgb.c, v5.4).
//
// This file and calc.go have no build tags: they are pure computations
// that are unit tested on the host. Everything that touches registers is in
// the esp32s3-only files.
package rgblcd

import "errors"

// Timing describes the panel timing, using the same terms as ESP-IDF's
// esp_lcd_rgb_timing_t.
type Timing struct {
	PclkHz uint32 // wanted pixel clock

	HSyncPulseWidth uint32
	HSyncBackPorch  uint32
	HSyncFrontPorch uint32
	VSyncPulseWidth uint32
	VSyncBackPorch  uint32
	VSyncFrontPorch uint32

	HSyncIdleLow  bool // HSYNC is low when idle (active high pulse)
	VSyncIdleLow  bool // VSYNC is low when idle (active high pulse)
	DEIdleHigh    bool // DE is high when idle
	PclkActiveNeg bool // data is output on the falling edge of PCLK
	PclkIdleHigh  bool // PCLK is high when idle
}

// ClockSource selects the LCD_CAM module clock.
type ClockSource uint8

// Values from ESP-IDF hal/esp32s3/include/hal/lcd_ll.h lcd_ll_select_clk_src
// (LCD_CLK_SEL: 1 = XTAL, 2 = PLL_D2 (240MHz), 3 = PLL_F160M).
const (
	ClockPLL160M ClockSource = 3 // default, same as ESP-IDF LCD_CLK_SRC_DEFAULT
	ClockPLL240M ClockSource = 2
	ClockXTAL    ClockSource = 1
)

// Hz returns the frequency of the clock source.
// Source: ESP-IDF soc/esp32s3/include/soc/clk_tree_defs.h.
func (c ClockSource) Hz() uint32 {
	switch c {
	case ClockPLL240M:
		return 240_000_000
	case ClockXTAL:
		return 40_000_000
	default:
		return 160_000_000
	}
}

// NoPin marks an unused signal.
const NoPin = 0xFF

// Config is everything the driver needs to know about a board.
type Config struct {
	Width  int
	Height int

	// DataPins[i] is the GPIO for LCD_DATA_OUTi. For RGB565 with the
	// frame buffer in standard RGB565, use B0-B4, G0-G5, R0-R4.
	DataPins [16]uint8

	DE    uint8
	VSync uint8
	HSync uint8
	PCLK  uint8

	Timing Timing

	// ClockSource defaults to ClockPLL160M when zero.
	ClockSource ClockSource

	// DMAChannel is the GDMA TX channel to use (0..4).
	DMAChannel uint8
}

var (
	ErrBadSize      = errors.New("rgblcd: bad width/height")
	ErrBadTiming    = errors.New("rgblcd: timing value does not fit its register field")
	ErrBadPclk      = errors.New("rgblcd: pixel clock cannot be generated from the clock source")
	ErrBadDMA       = errors.New("rgblcd: bad DMA channel")
	ErrBufferSize   = errors.New("rgblcd: frame buffer is smaller than width*height")
	ErrBufferAlign  = errors.New("rgblcd: frame buffer must be 4-byte aligned")
	ErrBufferMemory = errors.New("rgblcd: frame buffer must be in internal SRAM")
	ErrLineTooLong  = errors.New("rgblcd: one line does not fit a DMA descriptor")
	ErrTooManyDescs = errors.New("rgblcd: frame needs more DMA descriptors than available")
)
