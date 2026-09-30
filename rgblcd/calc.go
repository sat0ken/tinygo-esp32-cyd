package rgblcd

// ClockDiv is the result of the pixel clock calculation.
//
//	lcd_clk   = source / (N + B/A)
//	pixel_clk = lcd_clk / MO
type ClockDiv struct {
	N, A, B uint32 // LCD_CLKM_DIV_NUM, LCD_CLKM_DIV_A, LCD_CLKM_DIV_B
	MO      uint32 // pixel clock prescale; LCD_CLKCNT_N = MO-1 (or CLK_EQU_SYSCLK if MO == 1)
	PclkHz  uint32 // resulting pixel clock
}

// Limits from ESP-IDF hal/esp32s3/include/hal/lcd_ll.h.
const (
	lcdClkFracDivNMax  = 256 // LCD_LL_CLK_FRAC_DIV_N_MAX
	lcdClkFracDivABMax = 64  // LCD_LL_CLK_FRAC_DIV_AB_MAX
	lcdPclkDivMax      = 64  // LCD_LL_PCLK_DIV_MAX
)

// CalcClock ports ESP-IDF lcd_hal_cal_pclk_freq (hal/lcd_hal.c) and
// hal_utils_calc_clk_div_frac_fast (hal/hal_utils.c), v5.4.
//
// allowEqualSysclk corresponds to LCD_HAL_PCLK_FLAG_ALLOW_EQUAL_SYSCLK, which
// esp_lcd_new_rgb_panel sets only when neither pclk_active_neg nor
// pclk_idle_high is used (the PCLK polarity cannot be changed when
// LCD_PCLK == LCD_CLK).
func CalcClock(srcHz, wantHz uint32, allowEqualSysclk bool) (ClockDiv, bool) {
	if wantHz == 0 {
		return ClockDiv{}, false
	}
	mo := srcHz/wantHz/lcdClkFracDivNMax + 1
	if mo == 1 && !allowEqualSysclk {
		mo = 2
	}
	if mo > lcdPclkDivMax {
		return ClockDiv{}, false
	}
	expHz := wantHz * mo

	// hal_utils_calc_clk_div_frac_fast with max_integ = 256, min_integ = 2,
	// max_fract = 64.
	const maxFract = lcdClkFracDivABMax
	denom := uint32(2)
	numer := uint32(0)
	integ := srcHz / expHz
	freqErr := srcHz % expHz
	if freqErr != 0 {
		// Carry if the fraction is greater than 1.0 - 1.0/((max_fract-1)*2).
		// (Integer arithmetic exactly as in the C code.)
		if freqErr < expHz-expHz/(maxFract-1)*2 {
			g := gcd(expHz, freqErr)
			denom = expHz / g
			numer = freqErr / g
			d := denom/maxFract + 1
			denom /= d
			numer /= d
		} else {
			integ++
		}
	}
	if integ < 2 || integ >= lcdClkFracDivNMax || integ == 0 {
		return ClockDiv{}, false
	}
	var real uint32
	if numer != 0 {
		temp := integ*denom + numer
		real = uint32((uint64(srcHz)*uint64(denom) + uint64(temp/2)) / uint64(temp))
	} else {
		real = srcHz / integ
	}
	return ClockDiv{N: integ, A: denom, B: numer, MO: mo, PclkHz: real / mo}, true
}

func gcd(a, b uint32) uint32 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// TimingRegs holds the values written to the LCD_CTRL/CTRL1/CTRL2 timing
// fields.
type TimingRegs struct {
	HSyncWidth uint32 // LCD_CTRL2.LCD_HSYNC_WIDTH (7 bit)
	HBFront    uint32 // LCD_CTRL.LCD_HB_FRONT     (11 bit)
	HAWidth    uint32 // LCD_CTRL1.LCD_HA_WIDTH    (12 bit)
	HTWidth    uint32 // LCD_CTRL1.LCD_HT_WIDTH    (12 bit)
	VSyncWidth uint32 // LCD_CTRL2.LCD_VSYNC_WIDTH (7 bit)
	VBFront    uint32 // LCD_CTRL1.LCD_VB_FRONT    (8 bit)
	VAHeight   uint32 // LCD_CTRL.LCD_VA_HEIGHT    (10 bit)
	VTHeight   uint32 // LCD_CTRL.LCD_VT_HEIGHT    (10 bit)
}

// CalcTiming converts the panel timing to register values with the same
// formulas as ESP-IDF lcd_ll_set_horizontal_timing and
// lcd_ll_set_vertical_timing (hal/esp32s3/include/hal/lcd_ll.h, v5.4):
//
//	lcd_hsync_width = hsw - 1
//	lcd_hb_front    = hbp + hsw - 1
//	lcd_ha_width    = active_width - 1
//	lcd_ht_width    = hsw + hbp + active_width + hfp - 1
//	lcd_vsync_width = vsw - 1
//	lcd_vb_front    = vbp + vsw - 1
//	lcd_va_height   = active_height - 1
//	lcd_vt_height   = vsw + vbp + active_height + vfp - 1
//
// The field widths come from the TinyGo device/esp SVD definitions of
// LCD_CAM.LCD_CTRL/LCD_CTRL1/LCD_CTRL2.
func CalcTiming(width, height int, t Timing) (TimingRegs, error) {
	if width <= 0 || height <= 0 {
		return TimingRegs{}, ErrBadSize
	}
	if t.HSyncPulseWidth == 0 || t.VSyncPulseWidth == 0 {
		return TimingRegs{}, ErrBadTiming
	}
	w, h := uint32(width), uint32(height)
	r := TimingRegs{
		HSyncWidth: t.HSyncPulseWidth - 1,
		HBFront:    t.HSyncBackPorch + t.HSyncPulseWidth - 1,
		HAWidth:    w - 1,
		HTWidth:    t.HSyncPulseWidth + t.HSyncBackPorch + w + t.HSyncFrontPorch - 1,
		VSyncWidth: t.VSyncPulseWidth - 1,
		VBFront:    t.VSyncBackPorch + t.VSyncPulseWidth - 1,
		VAHeight:   h - 1,
		VTHeight:   t.VSyncPulseWidth + t.VSyncBackPorch + h + t.VSyncFrontPorch - 1,
	}
	if r.HSyncWidth >= 1<<7 || r.HBFront >= 1<<11 || r.HAWidth >= 1<<12 || r.HTWidth >= 1<<12 ||
		r.VSyncWidth >= 1<<7 || r.VBFront >= 1<<8 || r.VAHeight >= 1<<10 || r.VTHeight >= 1<<10 {
		return TimingRegs{}, ErrBadTiming
	}
	return r, nil
}

// RefreshHz returns the frame rate for a pixel clock and timing.
func RefreshHz(pclkHz uint32, width, height int, t Timing) float32 {
	ht := t.HSyncPulseWidth + t.HSyncBackPorch + uint32(width) + t.HSyncFrontPorch
	vt := t.VSyncPulseWidth + t.VSyncBackPorch + uint32(height) + t.VSyncFrontPorch
	return float32(pclkHz) / float32(ht*vt)
}

// DMA descriptor constants.
// Source: ESP-IDF hal/include/hal/dma_types.h (dma_descriptor_t) and
// ESP32-S3 TRM, GDMA chapter "Linked List".
//
//	dw0[11:0]  size
//	dw0[23:12] length
//	dw0[28]    err_eof
//	dw0[30]    suc_eof
//	dw0[31]    owner (1 = DMA)
const (
	DescMaxSize  = 4095 // DMA_DESCRIPTOR_BUFFER_MAX_SIZE
	descOwnerDMA = 1 << 31
	descSucEOF   = 1 << 30
)

// DescWord0 builds word 0 of a DMA descriptor. Like ESP-IDF
// gdma_link_mount_buffers, size is set equal to length and the owner is DMA.
func DescWord0(length int, eof bool) uint32 {
	w := uint32(length)&0xFFF | (uint32(length)&0xFFF)<<12 | descOwnerDMA
	if eof {
		w |= descSucEOF
	}
	return w
}

// DescLayout returns how a frame of height lines of lineBytes each is
// split into descriptors: linesPerDesc whole lines per descriptor (the last
// one may have fewer) and the number of descriptors.
//
// Each descriptor covers whole lines so that a wrong length shows up as a
// clean line shift instead of a tear inside a line. For 480x272 RGB565 this
// gives 4 lines (3840 bytes) x 68 descriptors.
func DescLayout(lineBytes, height int) (linesPerDesc, count int, err error) {
	if lineBytes <= 0 || height <= 0 {
		return 0, 0, ErrBadSize
	}
	linesPerDesc = DescMaxSize / lineBytes
	if linesPerDesc == 0 {
		return 0, 0, ErrLineTooLong
	}
	count = (height + linesPerDesc - 1) / linesPerDesc
	return linesPerDesc, count, nil
}
