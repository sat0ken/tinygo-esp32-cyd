//go:build esp32s3

package rgblcd

import "device/esp"

// enableLCDBus turns on the LCD_CAM APB clock and pulses its reset.
// Source: ESP-IDF lcd_ll_enable_bus_clock / lcd_ll_reset_register
// (SYSTEM.perip_clk_en1.lcd_cam_clk_en, SYSTEM.perip_rst_en1.lcd_cam_rst).
func enableLCDBus() {
	esp.SYSTEM.SetPERIP_CLK_EN1_LCD_CAM_CLK_EN(1)
	esp.SYSTEM.SetPERIP_RST_EN1_LCD_CAM_RST(1)
	esp.SYSTEM.SetPERIP_RST_EN1_LCD_CAM_RST(0)
}

// enableDMABus turns on the GDMA APB clock and pulses its reset, unless
// another user already enabled it (ESP-IDF also resets only when the
// reference count is 0; resetting would break channels already in use).
// Source: ESP-IDF _gdma_ll_enable_bus_clock / _gdma_ll_reset_register
// (SYSTEM.perip_clk_en1.dma_clk_en, SYSTEM.perip_rst_en1.dma_rst).
func enableDMABus() {
	if esp.SYSTEM.GetPERIP_CLK_EN1_DMA_CLK_EN() != 0 && esp.SYSTEM.GetPERIP_RST_EN1_DMA_RST() == 0 {
		return
	}
	esp.SYSTEM.SetPERIP_CLK_EN1_DMA_CLK_EN(1)
	esp.SYSTEM.SetPERIP_RST_EN1_DMA_RST(1)
	esp.SYSTEM.SetPERIP_RST_EN1_DMA_RST(0)
}

// setClockSource enables the LCD module clock and selects its source.
// Source: ESP-IDF lcd_ll_enable_clock (lcd_clock.clk_en) and
// lcd_ll_select_clk_src (lcd_clock.lcd_clk_sel).
func setClockSource(src ClockSource) {
	esp.LCD_CAM.SetLCD_CLOCK_CLK_EN(1)
	esp.LCD_CAM.SetLCD_CLOCK_LCD_CLK_SEL(uint32(src))
}

// setPixelClock writes the dividers computed by CalcClock.
// Source: ESP-IDF lcd_ll_set_pixel_clock_prescale (called from
// lcd_hal_cal_pclk_freq) followed by lcd_ll_set_group_clock_coeff.
func setPixelClock(d ClockDiv) {
	// pixel_clk = lcd_clk / (1 + clkcnt_n); clkcnt_n can't be zero, so a
	// prescale of 1 is done with lcd_clk_equ_sysclk instead.
	if d.MO == 1 {
		esp.LCD_CAM.SetLCD_CLOCK_LCD_CLK_EQU_SYSCLK(1)
		esp.LCD_CAM.SetLCD_CLOCK_LCD_CLKCNT_N(1)
	} else {
		esp.LCD_CAM.SetLCD_CLOCK_LCD_CLK_EQU_SYSCLK(0)
		esp.LCD_CAM.SetLCD_CLOCK_LCD_CLKCNT_N(d.MO - 1)
	}
	// lcd_clk = source / (div_num + div_b / div_a); div_num == 0 means 256.
	n := d.N
	if n >= lcdClkFracDivNMax {
		n = 0
	}
	esp.LCD_CAM.SetLCD_CLOCK_LCD_CLKM_DIV_NUM(n)
	esp.LCD_CAM.SetLCD_CLOCK_LCD_CLKM_DIV_A(d.A)
	esp.LCD_CAM.SetLCD_CLOCK_LCD_CLKM_DIV_B(d.B)
}

// SelfTest enables the LCD_CAM and GDMA clocks and checks that registers of
// both keep the values written to them (phase 1 of the bring-up). It uses
// registers with no side effect: LCD_CAM.LCD_CTRL1 (timing only) and the
// TX priority of the given GDMA channel. Both are restored afterwards.
func SelfTest(dmaCh uint8) bool {
	enableLCDBus()
	enableDMABus()
	ok := DumpClocks()

	logf("LCD_CAM LC_REG_DATE=%08x\n", esp.LCD_CAM.LC_REG_DATE.Get())
	saved := esp.LCD_CAM.LCD_CTRL1.Get()
	for _, v := range []uint32{0x12345678, 0xA5A5A5A5, 0x5A5A5A5A, 0} {
		esp.LCD_CAM.LCD_CTRL1.Set(v)
		got := esp.LCD_CAM.LCD_CTRL1.Get()
		logf("LCD_CAM.LCD_CTRL1 write %08x read %08x %s\n", v, got, okStr(got == v))
		ok = ok && got == v
	}
	esp.LCD_CAM.LCD_CTRL1.Set(saved)

	if dmaCh < dmaChannels {
		r := dmaOut(dmaCh)
		savedPri := r.PRI.Get()
		for _, v := range []uint32{5, 0xA, 3, 0} {
			r.PRI.Set(v)
			got := r.PRI.Get() & 0xF // OUT_PRI_CHn_TX_PRI is 4 bits
			logf("DMA.OUT_PRI_CH%d write %x read %x %s\n", dmaCh, v, got, okStr(got == v))
			ok = ok && got == v
		}
		r.PRI.Set(savedPri)
	}
	return ok
}
