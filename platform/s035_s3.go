//go:build esp32_3248s035c && esp32s3

package platform

import (
	"device/esp"
	"machine"

	"github.com/sat0ken/tinygo-cyd/spilcd"
)

// The ST7796 of the ESP32-3248S035C with an ESP32-S3 module is on the
// S3's SPI2 (FSPI), which TinyGo calls machine.SPI0. It is driven without
// DMA (package espdma is for the original ESP32): DrawRGBBitmap8Async
// draws at once.
var lcdSPI = machine.SPI0

// prepareLCDPins takes IO19 and IO20 (the panel's SCK and MISO) away from
// the USB PHY. After reset the pads belong to the USB Serial/JTAG
// controller (USB_PAD_ENABLE, with the pull-up on D+ = IO20), whatever the
// IO MUX says. TinyGo's Pin.Configure does not change that; ESP-IDF's
// gpio_ll_iomux_func_sel clears USB_PAD_ENABLE for these two pins, and so
// does this. The board has no USB on these pins: the serial console goes
// through the CH340C on UART0.
func prepareLCDPins() {
	esp.USB_DEVICE.SetCONF0_USB_PAD_ENABLE(0)
}

// lcdBus returns spi as it is.
func lcdBus(spi *machine.SPI) spilcd.Bus { return spi }
