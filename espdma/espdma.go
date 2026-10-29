//go:build esp32

// Package espdma sends SPI data with DMA on the ESP32 (the original chip:
// SPI2 = HSPI on DMA channel 1, SPI3 = VSPI on channel 2), so that the CPU
// can work while a block goes out. TinyGo's machine.SPI fills the 64-byte
// buffer of the SPI peripheral and waits for each chunk.
//
// SPI wraps a configured machine.SPI: Tx (and so everything else) first
// waits for a transfer started with StartTx. It is a spilcd.AsyncBus.
package espdma

import (
	"device/esp"
	"machine"
	"unsafe"
)

// A DMA descriptor (lldesc_t) points to at most 4095 bytes; keep it a
// multiple of 4.
const (
	chunk    = 4092
	maxDescs = 32 // 128KB at a time

	descOwnerDMA = 1 << 31
	descEOF      = 1 << 30

	// fillReads is how many times an SPI register is read between starting
	// the DMA and starting the SPI (about 3µs): the DMA needs a moment to
	// fill the 32-byte FIFO of the SPI. Started at once, at 40MHz the SPI
	// sent the 32 bytes left in the FIFO from the last transfer first, and
	// the rest 32 bytes late (seen by looping MOSI back into MISO: correct
	// with 20 or 100 reads, wrong with none; on the panel, a band of 16
	// pixels at the left with the end of the row above).
	fillReads = 64
)

// SPI is an SPI bus that can send a block in the background.
type SPI struct {
	*machine.SPI
	desc []uint32 // descriptors: size and length, buffer, next
	busy bool
	user uint32 // the USER register before the transfer
}

// New sets up the DMA channel of spi (machine.SPI2 or machine.SPI3).
func New(spi *machine.SPI) *SPI {
	esp.DPORT.SetPERIP_CLK_EN_SPI_DMA_CLK_EN(1)
	esp.DPORT.SetPERIP_RST_EN_SPI_DMA_RST(1)
	esp.DPORT.SetPERIP_RST_EN_SPI_DMA_RST(0)
	if spi.Bus == esp.SPI3 {
		esp.DPORT.SetSPI_DMA_CHAN_SEL_SPI3_DMA_CHAN_SEL(2)
	} else {
		esp.DPORT.SetSPI_DMA_CHAN_SEL_SPI2_DMA_CHAN_SEL(1)
	}
	return &SPI{SPI: spi, desc: make([]uint32, 3*maxDescs)}
}

// Tx waits for a background transfer and then does a normal one.
func (s *SPI) Tx(w, r []byte) error {
	s.Wait()
	return s.SPI.Tx(w, r)
}

// StartTx starts sending w and returns at once. w must be in internal RAM
// (not a string in flash), start on a 4-byte boundary and stay unchanged
// until Wait returns. Longer blocks than maxDescs*chunk bytes are sent
// with Tx.
func (s *SPI) StartTx(w []byte) {
	s.Wait()
	n := len(w)
	if n == 0 {
		return
	}
	if n > maxDescs*chunk || uintptr(unsafe.Pointer(&w[0]))%4 != 0 {
		s.SPI.Tx(w, nil)
		return
	}
	buf := uint32(uintptr(unsafe.Pointer(&w[0])))
	for i, off := 0, 0; off < n; i, off = i+1, off+chunk {
		size := n - off
		if size > chunk {
			size = chunk
		}
		d := s.desc[3*i : 3*i+3]
		d[0] = descOwnerDMA | uint32(size)<<12 | uint32(size+3)&^3
		d[1] = buf + uint32(off)
		d[2] = 0
		if off+size < n {
			d[2] = uint32(uintptr(unsafe.Pointer(&s.desc[3*i+3])))
		} else {
			d[0] |= descEOF
		}
	}

	bus := s.Bus
	const resets = esp.SPI_DMA_CONF_OUT_RST | esp.SPI_DMA_CONF_AHBM_RST | esp.SPI_DMA_CONF_AHBM_FIFO_RST
	bus.DMA_CONF.SetBits(resets)
	bus.DMA_CONF.ClearBits(resets)
	bus.DMA_CONF.SetBits(esp.SPI_DMA_CONF_OUTDSCR_BURST_EN | esp.SPI_DMA_CONF_OUT_DATA_BURST_EN)
	s.user = bus.USER.Get()
	bus.USER.Set(s.user&^(esp.SPI_USER_DOUTDIN|esp.SPI_USER_USR_MISO) | esp.SPI_USER_USR_MOSI)
	bus.MOSI_DLEN.Set(uint32(n)*8 - 1)
	bus.MISO_DLEN.Set(0)
	desc := uint32(uintptr(unsafe.Pointer(&s.desc[0])))
	bus.DMA_OUT_LINK.Set(desc&esp.SPI_DMA_OUT_LINK_OUTLINK_ADDR_Msk | esp.SPI_DMA_OUT_LINK_OUTLINK_START)
	for i := 0; i < fillReads; i++ {
		bus.DMA_STATUS.Get()
	}
	bus.CMD.Set(esp.SPI_CMD_USR)
	s.busy = true
}

// Wait waits for the transfer started with StartTx to end.
func (s *SPI) Wait() {
	if !s.busy {
		return
	}
	for s.Bus.CMD.Get()&esp.SPI_CMD_USR != 0 {
	}
	s.Bus.USER.Set(s.user)
	s.busy = false
}
