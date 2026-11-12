//go:build !esp32_3248s035c_s3

package s035

// GPIO numbers of the board with its original ESP-32S module.

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

// GT911 on I2C0. INT is GPIO21 according to rzeldent and GPIO36 according
// to lang-ship; it is not used.
const (
	PinTouchSDA = 33
	PinTouchSCL = 32
	PinTouchRST = 25
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

// Mini joystick module (package minijoy) as a game controller: I2C on
// connector CN1 (GND, IO22, IO21, 3.3V from the top), SDA on IO21 and SCL
// on IO22 (checked on the board with a bit-banged scan; the other way
// round finds nothing). The module's order is G, V, SDA, SCL, so the wires
// cross. IO21 is also the GT911's INT according to rzeldent; the touch
// still works, and the GT911's address is found either way.
const (
	PinJoySDA = 21
	PinJoySCL = 22
)

// Serial console: UART0 (GPIO1 TX / GPIO3 RX) through the CH340C USB-UART.
const (
	PinUARTTX = 1
	PinUARTRX = 3
)
