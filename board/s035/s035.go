// Package s035 holds every board-specific value of the Sunton
// ESP32-3248S035C (ESP32, 3.5" 320x480 ST7796 SPI TFT, GT911 capacitive
// touch). Package board is the ESP32-4827S043.
//
// The CPU is an ESP-32S module: the original ESP32 chip (not the ESP32-S2
// or S3) with 4MB flash and no PSRAM, so the TinyGo target inherits
// esp32-generic. On the V1.4 board the footprint of the extra flash chip
// (U4) is empty. Connectors on the back: P1 (VIN, TX, RX, GND), P3 (GND,
// IO35, IO22, IO21), CN1 (GND, IO22, IO21, 3.3V), P4 speaker, and TP-C
// for the touch panel (GND, SDA, SCL, RST, INT, VDD).
//
// The panel is used in landscape, 480x320. The ESP32 has no PSRAM and too
// little RAM for a frame buffer, so drawing goes straight to the panel over
// SPI (package st7796).
//
// Nothing here depends on the machine package, so the constants can be used
// from host tests as well.
//
// Sources:
//   - rzeldent/platformio-espressif32-sunton esp32-3248S035C.json
//   - lang-ship.com "ESP32-3248S035" (LovyanGFX settings of the C version)
package s035

// Screen size in landscape.
const (
	Width  = 480
	Height = 320
)

// The 480x272 games and apps written for the ESP32-4827S043 run in a
// window in the middle of the screen (see platform.Init).
const (
	WindowX = 0
	WindowY = (Height - 272) / 2
	WindowW = 480
	WindowH = 272
)

// ST7796 on SPI2 (HSPI). RST is not connected: the driver resets the
// panel with the software reset command.
const (
	PinSCK  = 14
	PinMOSI = 13
	PinMISO = 12
	PinCS   = 15
	PinDC   = 2

	PinBacklight = 27 // high = on
)

// SPIFrequency is the SPI clock for drawing. LovyanGFX uses 40MHz on this
// board; if the picture shows noise, lower it to 20MHz.
const SPIFrequency = 40_000_000

// MADCTL is the ST7796 memory access control value: row/column exchange
// for landscape (MV, 0x20) and BGR colour order (0x08). Checked on the
// board (2026-10-18, examples/s035_check): colours right, "TOP LEFT" at the
// top left with the USB connector on the right. 0xE8 (MY|MX|MV|BGR)
// turns the picture by 180 degrees; then the touch calibration below must
// be measured again (examples/s035_check).
const MADCTL = 0x28

// Invert turns on the display inversion (INVON). LovyanGFX: invert = false
// (checked on the board).
const Invert = false

// GT911 capacitive touch controller on I2C0. INT is GPIO21 according to
// rzeldent and GPIO36 according to lang-ship; it is not used (the driver
// polls), and neither is RST (GPIO25), so the controller keeps the I2C
// address it chose at power-on (0x5D or 0x14; both are tried).
// On the board (2026-10-18): address 0x5D, product "911", firmware 0x1060,
// resolution 320x480, config version 0x69.
const (
	PinTouchSDA = 33
	PinTouchSCL = 32
	PinTouchRST = 25

	TouchI2CFrequency = 400_000
)

// Touch calibration: raw GT911 readings at the left/right and top/bottom
// edges of the landscape screen (see touchcal.Calibration).
//
// The GT911 reports portrait coordinates in its own resolution, 320x480,
// which is the panel's: with MADCTL 0x28 its Y is the screen X and its X
// runs from the bottom (0) to the top (319), so no scaling is needed.
//
// Measured on the board (2026-10-18, examples/s035_check): touching the
// corners gave raw X 2..316 and raw Y 8..475, and the calibration targets
// 20 pixels from the edges matched within 3 pixels, except the two at the
// bottom, which read 12..19 pixels low (raw X 1 and 8 instead of 20). The
// 4-corner fit (0, 480, 319, -17) would make the bottom 16 pixels of the
// screen unreachable, so the GT911's own coordinates are used. (The
// LovyanGFX values for this board, x 14..310 / y 5..448, are 30 pixels off
// at the right edge on this one.)
const (
	TouchRawXLeft   = 0
	TouchRawXRight  = 479
	TouchRawYTop    = 319
	TouchRawYBottom = 0
	TouchSwapXY     = true
)

// microSD slot on its own SPI bus (VSPI), not shared with the display.
const (
	PinSDCS   = 5
	PinSDMOSI = 23
	PinSDSCK  = 18
	PinSDMISO = 19
)

// Other parts of the board (not used yet).
const (
	PinLEDRed   = 4 // RGB LED, low = on
	PinLEDGreen = 16
	PinLEDBlue  = 17
	PinSpeaker  = 26 // amplifier input
	PinLight    = 34 // CdS light sensor (ADC)
)

// Serial console: UART0 (GPIO1 TX / GPIO3 RX) through the CH340C USB-UART.
const (
	PinUARTTX = 1
	PinUARTRX = 3
)
