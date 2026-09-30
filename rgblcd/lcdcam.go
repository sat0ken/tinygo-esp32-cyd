//go:build esp32s3

package rgblcd

import "device/esp"

// Interrupt masks. Source: ESP-IDF lcd_ll.h LCD_LL_EVENT_VSYNC_END /
// LCD_LL_EVENT_TRANS_DONE, TinyGo device/esp LC_DMA_INT_* bit 0/1.
const (
	lcdEventVSyncEnd  = 1 << 0
	lcdEventTransDone = 1 << 1
)

func b2u(b bool) uint32 {
	if b {
		return 1
	}
	return 0
}

// lcdFifoReset: lcd_ll_fifo_reset (lcd_misc.lcd_afifo_reset = 1, self clear).
func lcdFifoReset() {
	esp.LCD_CAM.SetLCD_MISC_LCD_AFIFO_RESET(1)
}

// lcdReset: lcd_ll_reset (lcd_user.lcd_reset = 1, self clear).
func lcdReset() {
	esp.LCD_CAM.SetLCD_USER_LCD_RESET(1)
}

// lcdStart: lcd_ll_start. Update the parameters, then start.
func lcdStart() {
	esp.LCD_CAM.SetLCD_USER_LCD_UPDATE(1)
	esp.LCD_CAM.SetLCD_USER_LCD_START(1)
}

// lcdStop: lcd_ll_stop.
func lcdStop() {
	esp.LCD_CAM.SetLCD_USER_LCD_START(0)
	esp.LCD_CAM.SetLCD_USER_LCD_UPDATE(1) // self clear
}

// lcdInitController prepares the controller up to (not including) the RGB
// set-up. It follows esp_lcd_new_rgb_panel (esp_lcd_panel_rgb.c, v5.4):
// bus clock + reset, module clock on, clock source, FIFO/controller reset,
// interrupts off and cleared.
func (d *Device) lcdInitController() {
	enableLCDBus()
	setClockSource(d.clkSrc)
	lcdFifoReset()
	lcdReset()
	// lcd_ll_enable_interrupt(LCD_LL_EVENT_VSYNC_END, false) and
	// lcd_ll_clear_interrupt_status(UINT32_MAX): only bits 0..1 are LCD's.
	esp.LCD_CAM.LC_DMA_INT_ENA.ClearBits(lcdEventVSyncEnd | lcdEventTransDone)
	esp.LCD_CAM.LC_DMA_INT_CLR.Set(lcdEventVSyncEnd | lcdEventTransDone)
}

// lcdSetupRGB is the port of rgb_panel_init (esp_lcd_panel_rgb.c, v5.4),
// minus starting the transmission and the interrupt handler.
func (d *Device) lcdSetupRGB() {
	t := &d.cfg.Timing
	r := &d.timing

	// Pixel clock frequency (lcd_hal_cal_pclk_freq + lcd_ll_set_group_clock_coeff).
	setPixelClock(d.clk)

	// Pixel clock phase and polarity.
	esp.LCD_CAM.SetLCD_CLOCK_LCD_CK_IDLE_EDGE(b2u(t.PclkIdleHigh)) // lcd_ll_set_clock_idle_level
	esp.LCD_CAM.SetLCD_CLOCK_LCD_CK_OUT_EDGE(b2u(t.PclkActiveNeg)) // lcd_ll_set_pixel_clock_edge

	// lcd_ll_enable_rgb_mode(true)
	esp.LCD_CAM.SetLCD_CTRL_LCD_RGB_MODE_EN(1)
	// lcd_ll_set_dma_read_stride(16): 2 bytes per cycle.
	esp.LCD_CAM.SetLCD_USER_LCD_2BYTE_EN(1)

	// lcd_ll_set_phase_cycles(0, 0, 1): data phase only.
	// The C code writes (cycles - 1) into the bit fields, so a dummy phase of
	// 0 cycles leaves 0b11 in the 2-bit LCD_DUMMY_CYCLELEN; do the same.
	esp.LCD_CAM.SetLCD_USER_LCD_CMD(0)
	esp.LCD_CAM.SetLCD_USER_LCD_DUMMY(0)
	esp.LCD_CAM.SetLCD_USER_LCD_DOUT(1)
	esp.LCD_CAM.SetLCD_USER_LCD_CMD_2_CYCLE_EN(0)
	esp.LCD_CAM.SetLCD_USER_LCD_DUMMY_CYCLELEN(lcdDummyCycleLen)
	esp.LCD_CAM.SetLCD_USER_LCD_DOUT_CYCLELEN(0)

	// lcd_ll_enable_output_always_on(true): the number of data cycles is
	// controlled by the DMA, not by lcd_dout_cyclelen.
	esp.LCD_CAM.SetLCD_USER_LCD_ALWAYS_OUT_EN(1)

	// lcd_ll_set_idle_level(!hsync_idle_low, !vsync_idle_low, de_idle_high)
	esp.LCD_CAM.SetLCD_CTRL2_LCD_HSYNC_IDLE_POL(b2u(!t.HSyncIdleLow))
	esp.LCD_CAM.SetLCD_CTRL2_LCD_VSYNC_IDLE_POL(b2u(!t.VSyncIdleLow))
	esp.LCD_CAM.SetLCD_CTRL2_LCD_DE_IDLE_POL(b2u(t.DEIdleHigh))

	// lcd_ll_set_blank_cycles(1, 1): RGB panels always have porches.
	//   lcd_bk_en = 1, lcd_vfk_cyclelen = 1-1, lcd_vbk_cyclelen = 1-1
	esp.LCD_CAM.SetLCD_MISC_LCD_BK_EN(1)
	esp.LCD_CAM.SetLCD_MISC_LCD_VFK_CYCLELEN(0)
	esp.LCD_CAM.SetLCD_MISC_LCD_VBK_CYCLELEN(0)

	// lcd_ll_set_horizontal_timing / lcd_ll_set_vertical_timing, values from
	// CalcTiming.
	esp.LCD_CAM.SetLCD_CTRL2_LCD_HSYNC_WIDTH(r.HSyncWidth)
	esp.LCD_CAM.SetLCD_CTRL_LCD_HB_FRONT(r.HBFront)
	esp.LCD_CAM.SetLCD_CTRL1_LCD_HA_WIDTH(r.HAWidth)
	esp.LCD_CAM.SetLCD_CTRL1_LCD_HT_WIDTH(r.HTWidth)
	esp.LCD_CAM.SetLCD_CTRL2_LCD_VSYNC_WIDTH(r.VSyncWidth)
	esp.LCD_CAM.SetLCD_CTRL1_LCD_VB_FRONT(r.VBFront)
	esp.LCD_CAM.SetLCD_CTRL_LCD_VA_HEIGHT(r.VAHeight)
	esp.LCD_CAM.SetLCD_CTRL_LCD_VT_HEIGHT(r.VTHeight)

	// lcd_ll_enable_output_hsync_in_porch_region(true)
	esp.LCD_CAM.SetLCD_CTRL2_LCD_HS_BLANK_EN(1)
	// lcd_ll_set_hsync_position(0): HSYNC at the very beginning of the line.
	esp.LCD_CAM.SetLCD_CTRL2_LCD_HSYNC_POSITION(0)
	// lcd_ll_enable_auto_next_frame(stream_mode = true): keep sending frames.
	esp.LCD_CAM.SetLCD_MISC_LCD_NEXT_FRAME_EN(1)
}

// lcdDummyCycleLen is (0 - 1) truncated to the 2-bit LCD_DUMMY_CYCLELEN
// field, see lcdSetupRGB.
const lcdDummyCycleLen = 3

// regCheck is one field read back from a register and its expected value.
type regCheck struct {
	name      string
	got, want uint32
}

func (d *Device) lcdChecks() []regCheck {
	t := &d.cfg.Timing
	r := &d.timing
	c := d.clk
	n := c.N
	if n >= lcdClkFracDivNMax {
		n = 0
	}
	equ, cnt := uint32(0), c.MO-1
	if c.MO == 1 {
		equ, cnt = 1, 1
	}
	l := esp.LCD_CAM
	return []regCheck{
		{"LCD_CLOCK.CLK_EN", l.GetLCD_CLOCK_CLK_EN(), 1},
		{"LCD_CLOCK.LCD_CLK_SEL", l.GetLCD_CLOCK_LCD_CLK_SEL(), uint32(d.clkSrc)},
		{"LCD_CLOCK.LCD_CLKM_DIV_NUM", l.GetLCD_CLOCK_LCD_CLKM_DIV_NUM(), n},
		{"LCD_CLOCK.LCD_CLKM_DIV_A", l.GetLCD_CLOCK_LCD_CLKM_DIV_A(), c.A},
		{"LCD_CLOCK.LCD_CLKM_DIV_B", l.GetLCD_CLOCK_LCD_CLKM_DIV_B(), c.B},
		{"LCD_CLOCK.LCD_CLK_EQU_SYSCLK", l.GetLCD_CLOCK_LCD_CLK_EQU_SYSCLK(), equ},
		{"LCD_CLOCK.LCD_CLKCNT_N", l.GetLCD_CLOCK_LCD_CLKCNT_N(), cnt},
		{"LCD_CLOCK.LCD_CK_IDLE_EDGE", l.GetLCD_CLOCK_LCD_CK_IDLE_EDGE(), b2u(t.PclkIdleHigh)},
		{"LCD_CLOCK.LCD_CK_OUT_EDGE", l.GetLCD_CLOCK_LCD_CK_OUT_EDGE(), b2u(t.PclkActiveNeg)},
		{"LCD_USER.LCD_2BYTE_EN", l.GetLCD_USER_LCD_2BYTE_EN(), 1},
		{"LCD_USER.LCD_CMD", l.GetLCD_USER_LCD_CMD(), 0},
		{"LCD_USER.LCD_DUMMY", l.GetLCD_USER_LCD_DUMMY(), 0},
		{"LCD_USER.LCD_DOUT", l.GetLCD_USER_LCD_DOUT(), 1},
		{"LCD_USER.LCD_DUMMY_CYCLELEN", l.GetLCD_USER_LCD_DUMMY_CYCLELEN(), lcdDummyCycleLen},
		{"LCD_USER.LCD_DOUT_CYCLELEN", l.GetLCD_USER_LCD_DOUT_CYCLELEN(), 0},
		{"LCD_USER.LCD_ALWAYS_OUT_EN", l.GetLCD_USER_LCD_ALWAYS_OUT_EN(), 1},
		{"LCD_MISC.LCD_BK_EN", l.GetLCD_MISC_LCD_BK_EN(), 1},
		{"LCD_MISC.LCD_VFK_CYCLELEN", l.GetLCD_MISC_LCD_VFK_CYCLELEN(), 0},
		{"LCD_MISC.LCD_VBK_CYCLELEN", l.GetLCD_MISC_LCD_VBK_CYCLELEN(), 0},
		{"LCD_MISC.LCD_NEXT_FRAME_EN", l.GetLCD_MISC_LCD_NEXT_FRAME_EN(), 1},
		{"LCD_CTRL.LCD_RGB_MODE_EN", l.GetLCD_CTRL_LCD_RGB_MODE_EN(), 1},
		{"LCD_CTRL.LCD_HB_FRONT", l.GetLCD_CTRL_LCD_HB_FRONT(), r.HBFront},
		{"LCD_CTRL.LCD_VA_HEIGHT", l.GetLCD_CTRL_LCD_VA_HEIGHT(), r.VAHeight},
		{"LCD_CTRL.LCD_VT_HEIGHT", l.GetLCD_CTRL_LCD_VT_HEIGHT(), r.VTHeight},
		{"LCD_CTRL1.LCD_VB_FRONT", l.GetLCD_CTRL1_LCD_VB_FRONT(), r.VBFront},
		{"LCD_CTRL1.LCD_HA_WIDTH", l.GetLCD_CTRL1_LCD_HA_WIDTH(), r.HAWidth},
		{"LCD_CTRL1.LCD_HT_WIDTH", l.GetLCD_CTRL1_LCD_HT_WIDTH(), r.HTWidth},
		{"LCD_CTRL2.LCD_VSYNC_WIDTH", l.GetLCD_CTRL2_LCD_VSYNC_WIDTH(), r.VSyncWidth},
		{"LCD_CTRL2.LCD_VSYNC_IDLE_POL", l.GetLCD_CTRL2_LCD_VSYNC_IDLE_POL(), b2u(!t.VSyncIdleLow)},
		{"LCD_CTRL2.LCD_DE_IDLE_POL", l.GetLCD_CTRL2_LCD_DE_IDLE_POL(), b2u(t.DEIdleHigh)},
		{"LCD_CTRL2.LCD_HS_BLANK_EN", l.GetLCD_CTRL2_LCD_HS_BLANK_EN(), 1},
		{"LCD_CTRL2.LCD_HSYNC_WIDTH", l.GetLCD_CTRL2_LCD_HSYNC_WIDTH(), r.HSyncWidth},
		{"LCD_CTRL2.LCD_HSYNC_IDLE_POL", l.GetLCD_CTRL2_LCD_HSYNC_IDLE_POL(), b2u(!t.HSyncIdleLow)},
		{"LCD_CTRL2.LCD_HSYNC_POSITION", l.GetLCD_CTRL2_LCD_HSYNC_POSITION(), 0},
	}
}

// DumpLCD prints the LCD_CAM registers and checks every field the driver
// set against the value computed from the configuration (the same value
// ESP-IDF would write). It returns false on any mismatch.
func (d *Device) DumpLCD() bool {
	l := esp.LCD_CAM
	logf("--- LCD_CAM ---\n")
	logf("LCD_CLOCK=%08x LCD_USER=%08x LCD_MISC=%08x\n", l.LCD_CLOCK.Get(), l.LCD_USER.Get(), l.LCD_MISC.Get())
	logf("LCD_CTRL =%08x LCD_CTRL1=%08x LCD_CTRL2=%08x\n", l.LCD_CTRL.Get(), l.LCD_CTRL1.Get(), l.LCD_CTRL2.Get())
	logf("LCD_USER.LCD_START=%d LC_DMA_INT_RAW=%08x LC_REG_DATE=%08x\n",
		l.GetLCD_USER_LCD_START(), l.LC_DMA_INT_RAW.Get(), l.LC_REG_DATE.Get())
	ok := true
	for _, c := range d.lcdChecks() {
		good := c.got == c.want
		if !good {
			ok = false
		}
		logf("  %-30s %5d (want %5d) %s\n", c.name, c.got, c.want, okStr(good))
	}
	logf("pclk: src=%dHz N=%d A=%d B=%d MO=%d -> %dHz, %d.%02d fps\n",
		d.clkSrc.Hz(), d.clk.N, d.clk.A, d.clk.B, d.clk.MO, d.clk.PclkHz,
		int(d.refresh), int(d.refresh*100)%100)
	return ok
}
