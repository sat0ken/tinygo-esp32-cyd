// Package s028 holds every board-specific value of the Sunton
// ESP32-2432S028R, the "Cheap Yellow Display" (ESP32, 2.8" 240x320 ILI9341
// SPI TFT, XPT2046 resistive touch). The version with one micro USB
// connector; the version with two USB connectors is reported to need the
// display inversion, or to have an ST7789.
//
// The CPU is the original ESP32 (ESP-WROOM-32, 4MB flash, no PSRAM), so
// the TinyGo target inherits esp32-generic and drawing goes straight to
// the panel over SPI (package spilcd), as on the ESP32-3248S035C.
//
// The panel is used in landscape, 320x240. That is smaller than the 480x272
// the games were made for, so only programs that adapt to the screen size
// run on it (examples/s028_check, cmd/app).
//
// Nothing here depends on the machine package, so the constants can be used
// from host tests as well.
//
// The LCDwiki "2.8inch ESP32-32E Display" (E32R28T: ESP32-32E module,
// ILI9341V, XPT2046, USB-C, battery charger) has the same panel, touch and
// microSD pins, so it uses this package too. Only parts not used yet
// differ: see E32R28T below.
//
// Sources:
//   - rzeldent/platformio-espressif32-sunton esp32-2432S028R.json
//   - witnessmenow/ESP32-Cheap-Yellow-Display PINS.md
//   - lcdwiki.com 2.8inch_ESP32-32E_Display (E32R28T)
package s028

// Screen size in landscape.
const (
	Width  = 320
	Height = 240
)

// ILI9341 on SPI2 (HSPI). RST is not connected: the driver resets the
// panel with the software reset command. MISO (GPIO12) is wired to the
// panel's SDO.
const (
	PinSCK  = 14
	PinMOSI = 13
	PinMISO = 12
	PinCS   = 15
	PinDC   = 2

	PinBacklight = 21 // high = on (also on connector P3)
)

// SPIFrequency is the SPI clock for drawing (rzeldent: 24MHz, TFT_eSPI
// setups for this board: 40 or 55MHz). If the picture shows noise, lower
// it to 20MHz.
const SPIFrequency = 40_000_000

// MADCTL is the ILI9341 memory access control value: row/column exchange
// for landscape (MV, 0x20) and BGR colour order (0x08). rzeldent: BGR,
// portrait mirrored in X (0x48), so 0x28 or 0xE8 is landscape. Checked on
// the ESP32-2432S028R and the E32R28T (2026-10-27, examples/s028_check):
// the test screen looks right on both. Measured on the E32R28T: a full
// 320x240 fill takes 39ms at 40MHz SPI, 300 characters 139ms. Reading the
// panel ID gives zeros although MISO is wired to the panel on these boards,
// also with USR_MISO set, so TinyGo's ESP32 SPI does not seem to read.
const MADCTL = 0x28

// Invert turns on the display inversion (INVON): off on the one-USB
// version (TFT_eSPI setups for the two-USB version turn it on).
const Invert = false

// XPT2046 resistive touch controller on its own pins (bit-banged by the
// tinygo drivers xpt2046 package). GPIO36 (IRQ, low while touched) and
// GPIO39 are input-only pins.
const (
	PinTouchSCK  = 25
	PinTouchMOSI = 32
	PinTouchMISO = 39
	PinTouchCS   = 33
	PinTouchIRQ  = 36
)

// Touch calibration: raw 12-bit XPT2046 readings (xpttouch.ReadRaw) at the
// left/right and top/bottom edges of the landscape screen (see
// touchcal.Calibration). These are a first guess from the usual CYD
// examples (x 200..3700, y 240..3800); measure them on the board with
// examples/s028_check ("k").
const (
	TouchRawXLeft   = 200
	TouchRawXRight  = 3700
	TouchRawYTop    = 240
	TouchRawYBottom = 3800
	TouchSwapXY     = false
)

// microSD slot on VSPI.
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
	PinSpeaker  = 26 // amplifier input (connector P4)
	PinLight    = 34 // CdS light sensor (ADC)
)

// E32R28T (LCDwiki 2.8inch ESP32-32E Display): the pins that differ from
// the ESP32-2432S028R. Its panel reset is wired to EN.
const (
	E32R28TPinLEDRed    = 22 // RGB LED (common anode), low = on; green 16, blue 17 as above
	E32R28TPinAmpEnable = 4  // speaker amplifier, low = enabled (the red LED on the S028R)
	E32R28TPinBattery   = 34 // battery voltage (ADC), not a light sensor
)

// Connectors: P1 (VIN, TX, RX, GND), P3 (GND, IO35, IO22, IO21 = backlight),
// CN1 (GND, IO22, IO27, 3.3V). Serial console: UART0 (GPIO1 TX / GPIO3 RX)
// through the CH340 USB-UART.
const (
	PinUARTTX = 1
	PinUARTRX = 3
)
