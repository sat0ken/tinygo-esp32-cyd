//go:build esp32s3

package main

import (
	"machine"

	"github.com/sat0ken/tinygo-cyd/board"
	"github.com/sat0ken/tinygo-cyd/xpttouch"
)

// sdFrequency is the SPI clock for reading the card. The tinygo sdcard
// driver uses 4MHz after initialisation; SD cards allow 25MHz in SPI mode.
// The driver does not check the data CRC, so if pictures show wrong pixels
// lower this back to 4MHz.
const sdFrequency = 10_000_000

// sharedBus arbitrates the SPI pins shared by the microSD card and the
// XPT2046 touch controller (SCK=12, MOSI=11, MISO=13):
//
//   - SD card: the hardware SPI (machine.SPI0 = SPI2/FSPI) drives the pins
//     through the GPIO matrix.
//   - touch: the tinygo xpt2046 driver bit-bangs the same pins as GPIO.
//
// Before each use the pins are switched to the right function and the
// other device's chip select is held high. Switching happens at most once
// per slide, because the slide show only reads the card while drawing.
type sharedBus struct {
	spi  *machine.SPI
	sck  machine.Pin
	mosi machine.Pin
	miso machine.Pin
	sdCS machine.Pin
	tCS  machine.Pin
	mode int // 0 = unknown, modeSD, modeTouch
}

const (
	modeSD = iota + 1
	modeTouch
)

func newSharedBus() *sharedBus {
	b := &sharedBus{
		spi:  machine.SPI0,
		sck:  machine.Pin(board.PinTouchSCK),
		mosi: machine.Pin(board.PinTouchMOSI),
		miso: machine.Pin(board.PinTouchMISO),
		sdCS: machine.Pin(board.PinSDCS),
		tCS:  machine.Pin(board.PinTouchCS),
	}
	b.sdCS.Configure(machine.PinConfig{Mode: machine.PinOutput})
	b.sdCS.High()
	b.tCS.Configure(machine.PinConfig{Mode: machine.PinOutput})
	b.tCS.High()
	return b
}

// useSD routes the pins to the SPI peripheral.
func (b *sharedBus) useSD() {
	if b.mode == modeSD {
		return
	}
	b.tCS.High()
	b.spi.Configure(machine.SPIConfig{
		SCK: b.sck, SDO: b.mosi, SDI: b.miso,
		Frequency: sdFrequency,
		Mode:      0,
	})
	b.mode = modeSD
}

// useTouch gives the pins back to GPIO for the bit-banged touch driver
// (same directions as xpt2046.Device.Configure: clock/data out low, data in).
func (b *sharedBus) useTouch() {
	if b.mode == modeTouch {
		return
	}
	b.sdCS.High()
	b.sck.Configure(machine.PinConfig{Mode: machine.PinOutput})
	b.sck.Low()
	b.mosi.Configure(machine.PinConfig{Mode: machine.PinOutput})
	b.mosi.Low()
	b.miso.Configure(machine.PinConfig{Mode: machine.PinInput})
	b.mode = modeTouch
}

// busTouch is a hal.Touch that takes the bus before reading.
type busTouch struct {
	bus *sharedBus
	t   *xpttouch.Touch
}

func (bt busTouch) ReadTouch() (x, y int16, pressed bool) {
	bt.bus.useTouch()
	return bt.t.ReadTouch()
}
