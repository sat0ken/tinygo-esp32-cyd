//go:build esp32_3248s035c && esp32

package main

import (
	"device/esp"
	"machine"

	"github.com/sat0ken/tinygo-cyd/board/s035"
	"github.com/sat0ken/tinygo-cyd/internal/panelcheck"
	"github.com/sat0ken/tinygo-cyd/spilcd"
)

// readID reads the panel ID at 10MHz. TinyGo's ESP32 SPI driver enables
// full duplex (DOUTDIN) and the write phase (USR_MOSI) but not the read
// phase (USR_MISO), which ESP-IDF also sets for full duplex; the second
// try sets it to see whether the bytes then come in. (On this board every
// read gives zeros: MISO does not seem to reach the panel.)
func readID(d *spilcd.Device) {
	spi := machine.SPI2
	cfg := machine.SPIConfig{
		SCK: machine.Pin(s035.PinSCK), SDO: machine.Pin(s035.PinMOSI), SDI: machine.Pin(s035.PinMISO),
		Frequency: 10_000_000,
	}
	spi.Configure(cfg)
	for _, miso := range []bool{false, true} {
		if miso {
			esp.SPI2.USER.SetBits(esp.SPI_USER_USR_MISO)
		}
		a := d.Read(spilcd.RDDID, 4)
		b := d.Read(spilcd.RDID4, 4)
		println("USR_MISO", miso, " RDDID(04)", panelcheck.HexBytes(a[:]), " RDID4(d3)", panelcheck.HexBytes(b[:]))
	}
	cfg.Frequency = s035.SPIFrequency
	spi.Configure(cfg) // also clears USR_MISO again
	println("(ST7796: RDDID 7796xx / RDID4 007796, maybe shifted by a dummy bit)")
}
