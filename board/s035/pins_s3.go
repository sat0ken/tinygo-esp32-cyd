//go:build esp32_3248s035c_s3

package s035

// GPIO numbers of the board with the module replaced by an
// ESP32-S3-WROOM-1-N16R8 (16MB flash, 8MB octal PSRAM), soldered on the
// pads of the old module without other changes.
//
// The two modules have the same size and pin pitch and the same pads on
// the long sides; the S3 has 12 pads on the short side instead of 10, so
// its middle 10 sit on the old ones and IO3 and IO45 on bare board (insulate
// there). Each net of the board meets the S3 pad that lies on the old
// one. Sources: the ESP32-WROOM-32 and ESP32-S3-WROOM-1 datasheets (pin
// layout), and macsbug "Modified ESP32-3248S035R to ESP32-S3" (2025-10-23),
// whose wiring diagram shows the same numbers on the R version of the board.
//
// GND, 3V3, EN, IO0 (BOOT) and UART0 meet the same signals, so the CH340C
// still flashes the board and resets it. IO46, a strapping pin, sits on a
// GND pad: low is a valid level for both normal and download boot.

// ST7796 on the S3's SPI2 (FSPI, machine.SPI0), through the GPIO matrix.
// SCK and MISO are on IO19 and IO20, the S3's native USB pins: the
// platform package takes them away from the USB PHY first.
const (
	PinSCK  = 19 // old IO14
	PinMOSI = 9  // old IO13
	PinMISO = 20 // old IO12
	PinCS   = 47 // old IO15
	PinDC   = 48 // old IO2

	PinBacklight = 8 // old IO27; high = on
)

// GT911 on I2C0. INT is not used (old IO21 = IO42, or old IO36 = IO4).
const (
	PinTouchSDA = 16 // old IO33
	PinTouchSCL = 15 // old IO32
	PinTouchRST = 17 // old IO25
)

// microSD slot.
const (
	PinSDCS   = 38 // old IO5
	PinSDMOSI = 1  // old IO23
	PinSDSCK  = 39 // old IO18
	PinSDMISO = 40 // old IO19
)

// Other parts of the board (not used yet).
//
// The RGB LED lands on IO35..IO37, which the N16R8 module uses for its
// octal PSRAM: its resistors (R16, R17, R20) are removed on this board,
// so the LED does not light.
const (
	PinLEDRed   = 35 // old IO4
	PinLEDGreen = 36 // old IO16
	PinLEDBlue  = 37 // old IO17
	PinSpeaker  = 18 // old IO26, amplifier input
	PinLight    = 6  // old IO34, CdS light sensor (ADC)
)

// Mini joystick module on connector CN1 (see pins_esp32.go): SDA on old
// IO21, SCL on old IO22.
const (
	PinJoySDA = 42 // old IO21
	PinJoySCL = 2  // old IO22
)

// Serial console: UART0 (GPIO43 TX / GPIO44 RX) through the CH340C USB-UART.
const (
	PinUARTTX = 43
	PinUARTRX = 44
)
