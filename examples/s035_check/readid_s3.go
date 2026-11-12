//go:build esp32_3248s035c && esp32s3

package main

import (
	"machine"

	"github.com/sat0ken/tinygo-cyd/board/s035"
	"github.com/sat0ken/tinygo-cyd/internal/panelcheck"
	"github.com/sat0ken/tinygo-cyd/spilcd"
)

// readID reads the panel ID at 10MHz. TinyGo's ESP32-S3 SPI driver turns
// on the read phase (USR_MISO) in Configure, so one try is enough. MISO is
// IO20 on this board (old IO12, which did not seem to reach the panel).
func readID(d *spilcd.Device) {
	spi := machine.SPI0
	cfg := machine.SPIConfig{
		SCK: machine.Pin(s035.PinSCK), SDO: machine.Pin(s035.PinMOSI), SDI: machine.Pin(s035.PinMISO),
		Frequency: 10_000_000,
	}
	spi.Configure(cfg)
	a := d.Read(spilcd.RDDID, 4)
	b := d.Read(spilcd.RDID4, 4)
	println("RDDID(04)", panelcheck.HexBytes(a[:]), " RDID4(d3)", panelcheck.HexBytes(b[:]))
	cfg.Frequency = s035.SPIFrequency
	spi.Configure(cfg)
	println("(ST7796: RDDID 7796xx / RDID4 007796, maybe shifted by a dummy bit)")
}
