//go:build esp32s3

package rgblcd

import (
	"device"
	"device/esp"
	"runtime/volatile"
	"unsafe"
)

// dmaOutRegs is the TX ("OUT") register block of one GDMA channel.
// Offsets from TinyGo device/esp DMA_Type (SVD): OUT_CONF0_CH0 = 0x60 ...
// OUT_PERI_SEL_CH0 = 0xA8, and OUT_CONF0_CH1 = 0x120, so channels are 0xC0
// apart.
type dmaOutRegs struct {
	CONF0            volatile.Register32 // 0x60
	CONF1            volatile.Register32 // 0x64
	INT_RAW          volatile.Register32 // 0x68
	INT_ST           volatile.Register32 // 0x6C
	INT_ENA          volatile.Register32 // 0x70
	INT_CLR          volatile.Register32 // 0x74
	OUTFIFO_STATUS   volatile.Register32 // 0x78
	PUSH             volatile.Register32 // 0x7C
	LINK             volatile.Register32 // 0x80
	STATE            volatile.Register32 // 0x84
	EOF_DES_ADDR     volatile.Register32 // 0x88
	EOF_BFR_DES_ADDR volatile.Register32 // 0x8C
	DSCR             volatile.Register32 // 0x90
	DSCR_BF0         volatile.Register32 // 0x94
	DSCR_BF1         volatile.Register32 // 0x98
	WIGHT            volatile.Register32 // 0x9C
	_                [4]byte             // 0xA0
	PRI              volatile.Register32 // 0xA4
	PERI_SEL         volatile.Register32 // 0xA8
}

const (
	dmaChannels      = 5    // SOC_GDMA_PAIRS_PER_GROUP (soc_caps.h)
	dmaChannelStride = 0xC0 // OUT_CONF0_CH1 - OUT_CONF0_CH0

	// GDMA peripheral select value for LCD_CAM.
	// Source: ESP-IDF soc/esp32s3/include/soc/gdma_channel.h
	// (SOC_GDMA_TRIG_PERIPH_LCD0 = 5).
	dmaPeriLCD = 5

	// Burst size 64 -> OUT_EXT_MEM_BK_SIZE = 2 (GDMA_LL_EXT_MEM_BK_SIZE_64B),
	// ESP-IDF's default dma_burst_size for the RGB panel. It only matters for
	// external memory, but set it like ESP-IDF does.
	dmaExtMemBkSize64 = 2
)

func dmaOut(ch uint8) *dmaOutRegs {
	return (*dmaOutRegs)(unsafe.Add(unsafe.Pointer(&esp.DMA.OUT_CONF0_CH0), uintptr(ch)*dmaChannelStride))
}

func setBits(r *volatile.Register32, mask, pos, value uint32) {
	r.Set(r.Get()&^mask | (value<<pos)&mask)
}

func getBits(r *volatile.Register32, mask, pos uint32) uint32 {
	return (r.Get() & mask) >> pos
}

// descriptor is a GDMA linked list item (ESP-IDF dma_descriptor_align4_t):
// word 0 (size/length/suc_eof/owner), buffer address, next address.
type descriptor struct {
	dw0  uint32
	buf  uint32
	next uint32
}

// maxDescriptors descriptors of up to 4095 bytes cover ~500KB, more than the
// internal SRAM can hold as a frame buffer.
const maxDescriptors = 128

// descs must be in internal SRAM, which a global is: OUTLINK_ADDR only holds
// the low 20 bits of the address (DMA_OUT_LINK_CH_OUTLINK_ADDR_Msk = 0xfffff)
// and the DMA can only reach internal RAM through it. It is a package level
// variable so the GC never frees it while the DMA is running.
var descs [maxDescriptors]descriptor

// dmaReset: gdma_ll_tx_reset_channel (out_rst = 1, then 0).
func dmaReset(ch uint8) {
	r := dmaOut(ch)
	r.CONF0.SetBits(esp.DMA_OUT_CONF0_CH_OUT_RST)
	r.CONF0.ClearBits(esp.DMA_OUT_CONF0_CH_OUT_RST)
}

// dmaInitChannel follows lcd_rgb_create_dma_channel (esp_lcd_panel_rgb.c):
//
//	gdma_connect(LCD):          gdma_ll_tx_reset_channel + peri_sel = 5
//	gdma_apply_strategy(0,0,0): out_check_owner = 0, out_auto_wrback = 0,
//	                            out_eof_mode = 0 (eof_till_data_popped = false)
//	gdma_config_transfer(64):   outdscr_burst_en = 1, out_data_burst_en = 1,
//	                            out_ext_mem_bk_size = 64B
//
// The owner check is off, so the DMA never needs the owner bits reset when it
// goes round the circular list again.
func dmaInitChannel(ch uint8) {
	enableDMABus()
	r := dmaOut(ch)
	dmaReset(ch)
	setBits(&r.PERI_SEL, esp.DMA_OUT_PERI_SEL_CH_PERI_OUT_SEL_Msk, esp.DMA_OUT_PERI_SEL_CH_PERI_OUT_SEL_Pos, dmaPeriLCD)

	r.CONF1.ClearBits(esp.DMA_OUT_CONF1_CH_OUT_CHECK_OWNER)
	r.CONF0.ClearBits(esp.DMA_OUT_CONF0_CH_OUT_AUTO_WRBACK | esp.DMA_OUT_CONF0_CH_OUT_EOF_MODE)

	r.CONF0.SetBits(esp.DMA_OUT_CONF0_CH_OUT_DATA_BURST_EN | esp.DMA_OUT_CONF0_CH_OUTDSCR_BURST_EN)
	setBits(&r.CONF1, esp.DMA_OUT_CONF1_CH_OUT_EXT_MEM_BK_SIZE_Msk, esp.DMA_OUT_CONF1_CH_OUT_EXT_MEM_BK_SIZE_Pos, dmaExtMemBkSize64)
}

// buildDescriptors mounts the frame buffer on a circular descriptor list,
// like gdma_link_mount_buffers with mark_eof = true and mark_final = false
// (stream mode): owner = DMA, size = length, suc_eof on the last item, and
// the last item's next points back to the first one.
func (d *Device) buildDescriptors() {
	lineBytes := d.cfg.Width * 2
	total := lineBytes * d.cfg.Height
	chunk := d.linesPerDesc * lineBytes
	base := uintptr(unsafe.Pointer(&d.Buffer.Pix[0]))
	for i := 0; i < d.numDescs; i++ {
		off := i * chunk
		n := chunk
		if off+n > total {
			n = total - off
		}
		last := i == d.numDescs-1
		next := &descs[0]
		if !last {
			next = &descs[i+1]
		}
		desc := &descs[i]
		volatile.StoreUint32(&desc.buf, uint32(base+uintptr(off)))
		volatile.StoreUint32(&desc.next, uint32(uintptr(unsafe.Pointer(next))))
		volatile.StoreUint32(&desc.dw0, DescWord0(n, last))
	}
	device.Asm("memw") // make sure the descriptors are in SRAM before the DMA reads them
}

// dmaStart: gdma_ahb_hal_start_with_desc -> gdma_ll_tx_set_desc_addr
// (out_link.addr = addr) then gdma_ll_tx_start (out_link.start = 1).
func dmaStart(ch uint8, head *descriptor) {
	r := dmaOut(ch)
	addr := uint32(uintptr(unsafe.Pointer(head)))
	setBits(&r.LINK, esp.DMA_OUT_LINK_CH_OUTLINK_ADDR_Msk, esp.DMA_OUT_LINK_CH_OUTLINK_ADDR_Pos, addr)
	r.LINK.SetBits(esp.DMA_OUT_LINK_CH_OUTLINK_START)
}

// dmaStop: gdma_ll_tx_stop (out_link.stop = 1).
func dmaStop(ch uint8) {
	dmaOut(ch).LINK.SetBits(esp.DMA_OUT_LINK_CH_OUTLINK_STOP)
}

// DumpDMA prints the GDMA channel configuration and the descriptor list,
// and checks that the list is circular and covers exactly one frame.
func (d *Device) DumpDMA() bool {
	ch := d.cfg.DMAChannel
	r := dmaOut(ch)
	logf("--- GDMA OUT channel %d ---\n", ch)
	logf("CONF0=%08x CONF1=%08x PERI_SEL=%d LINK=%08x STATE=%08x\n",
		r.CONF0.Get(), r.CONF1.Get(), r.PERI_SEL.Get(), r.LINK.Get(), r.STATE.Get())
	logf("OUT_DSCR=%08x INT_RAW=%08x OUTFIFO_STATUS=%08x\n", r.DSCR.Get(), r.INT_RAW.Get(), r.OUTFIFO_STATUS.Get())

	checks := []regCheck{
		{"PERI_OUT_SEL", getBits(&r.PERI_SEL, esp.DMA_OUT_PERI_SEL_CH_PERI_OUT_SEL_Msk, esp.DMA_OUT_PERI_SEL_CH_PERI_OUT_SEL_Pos), dmaPeriLCD},
		{"OUT_DATA_BURST_EN", getBits(&r.CONF0, esp.DMA_OUT_CONF0_CH_OUT_DATA_BURST_EN, esp.DMA_OUT_CONF0_CH_OUT_DATA_BURST_EN_Pos), 1},
		{"OUTDSCR_BURST_EN", getBits(&r.CONF0, esp.DMA_OUT_CONF0_CH_OUTDSCR_BURST_EN, esp.DMA_OUT_CONF0_CH_OUTDSCR_BURST_EN_Pos), 1},
		{"OUT_AUTO_WRBACK", getBits(&r.CONF0, esp.DMA_OUT_CONF0_CH_OUT_AUTO_WRBACK, esp.DMA_OUT_CONF0_CH_OUT_AUTO_WRBACK_Pos), 0},
		{"OUT_EOF_MODE", getBits(&r.CONF0, esp.DMA_OUT_CONF0_CH_OUT_EOF_MODE, esp.DMA_OUT_CONF0_CH_OUT_EOF_MODE_Pos), 0},
		{"OUT_CHECK_OWNER", getBits(&r.CONF1, esp.DMA_OUT_CONF1_CH_OUT_CHECK_OWNER, esp.DMA_OUT_CONF1_CH_OUT_CHECK_OWNER_Pos), 0},
		{"OUT_EXT_MEM_BK_SIZE", getBits(&r.CONF1, esp.DMA_OUT_CONF1_CH_OUT_EXT_MEM_BK_SIZE_Msk, esp.DMA_OUT_CONF1_CH_OUT_EXT_MEM_BK_SIZE_Pos), dmaExtMemBkSize64},
	}
	ok := true
	for _, c := range checks {
		good := c.got == c.want
		if !good {
			ok = false
		}
		logf("  %-20s %d (want %d) %s\n", c.name, c.got, c.want, okStr(good))
	}

	logf("descriptors: %d x %d lines, table at %08x\n", d.numDescs, d.linesPerDesc, uint32(uintptr(unsafe.Pointer(&descs[0]))))
	fb := uint32(uintptr(unsafe.Pointer(&d.Buffer.Pix[0])))
	total := 0
	for i := 0; i < d.numDescs; i++ {
		desc := &descs[i]
		w := volatile.LoadUint32(&desc.dw0)
		length := int(w>>12) & 0xFFF
		total += length
		// Print the first 3, the last 2 and any odd one.
		if i < 3 || i >= d.numDescs-2 {
			logf("  [%2d] @%08x len=%4d size=%4d eof=%d owner=%d buf=%08x (fb+%6d) next=%08x\n",
				i, uint32(uintptr(unsafe.Pointer(desc))), length, w&0xFFF, (w>>30)&1, w>>31,
				desc.buf, desc.buf-fb, desc.next)
		} else if i == 3 {
			logf("  ...\n")
		}
	}

	// Walk the list from the head: it must come back to the head after
	// exactly numDescs steps.
	head := uint32(uintptr(unsafe.Pointer(&descs[0])))
	addr := head
	steps := 0
	for steps = 1; steps <= maxDescriptors+1; steps++ {
		idx := int(addr-head) / int(unsafe.Sizeof(descriptor{}))
		if idx < 0 || idx >= maxDescriptors {
			break
		}
		addr = descs[idx].next
		if addr == head {
			break
		}
	}
	circular := addr == head && steps == d.numDescs
	want := d.cfg.Width * d.cfg.Height * 2
	logf("  circular: %v (%d steps)  total length: %d (want %d) %s\n",
		circular, steps, total, want, okStr(circular && total == want))
	if !circular || total != want {
		ok = false
	}

	if d.started {
		head20 := getBits(&r.LINK, esp.DMA_OUT_LINK_CH_OUTLINK_ADDR_Msk, 0)
		logf("  OUTLINK_ADDR=%05x (head & 0xfffff = %05x) %s\n", head20, head&0xFFFFF, okStr(head20 == head&0xFFFFF))
		if head20 != head&0xFFFFF {
			ok = false
		}
	}
	return ok
}
