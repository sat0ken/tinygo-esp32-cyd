//go:build esp32_3248s035c && esp32

package platform

import (
	"machine"

	"github.com/sat0ken/tinygo-cyd/espdma"
	"github.com/sat0ken/tinygo-cyd/spilcd"
)

// The ST7796 of the ESP32-3248S035C with its original ESP32 module is on
// SPI2 (HSPI).
var lcdSPI = machine.SPI2

// prepareLCDPins does nothing: the pins are plain GPIOs.
func prepareLCDPins() {}

// lcdBus returns spi with DMA, so that DrawRGBBitmap8Async sends in the
// background.
func lcdBus(spi *machine.SPI) spilcd.Bus { return espdma.New(spi) }
