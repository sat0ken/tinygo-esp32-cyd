//go:build esp32s3

package rgblcd

import (
	"device/esp"
	"fmt"
	"time"
	"unsafe"

	"github.com/sat0ken/tinygo-cyd/framebuf"
)

// Internal SRAM that the GDMA can reach.
// Source: ESP-IDF soc/esp32s3/include/soc/soc.h (SOC_DMA_LOW, SOC_DMA_HIGH).
const (
	dmaMemLow  = 0x3FC88000
	dmaMemHigh = 0x3FD00000
)

// Device is an RGB panel driven by LCD_CAM + GDMA.
//
// The drawing methods (SetPixel, FillRectangle, DrawBitmap, ...) come from
// the embedded framebuf.Buffer: they only write to RAM, the DMA continuously
// sends the buffer to the panel.
type Device struct {
	*framebuf.Buffer

	cfg          Config
	clkSrc       ClockSource
	clk          ClockDiv
	timing       TimingRegs
	refresh      float32
	linesPerDesc int
	numDescs     int
	started      bool
}

// New validates cfg, then initialises LCD_CAM, GDMA and the GPIO matrix in
// the order of ESP-IDF esp_lcd_new_rgb_panel + esp_lcd_panel_reset +
// esp_lcd_panel_init (esp_lcd_panel_rgb.c, v5.4). It does not start the
// output; call Start for that.
//
// buf must hold at least Width*Height pixels, be 4-byte aligned and be in
// internal SRAM (a global array, not PSRAM).
func New(cfg Config, buf []uint16) (*Device, error) {
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > 0x7FFF || cfg.Height > 0x7FFF {
		return nil, ErrBadSize
	}
	if len(buf) < cfg.Width*cfg.Height {
		return nil, ErrBufferSize
	}
	addr := uintptr(unsafe.Pointer(&buf[0]))
	if addr&3 != 0 {
		return nil, ErrBufferAlign
	}
	if addr < dmaMemLow || addr+uintptr(cfg.Width*cfg.Height*2) > dmaMemHigh {
		return nil, ErrBufferMemory
	}
	if cfg.DMAChannel >= dmaChannels {
		return nil, ErrBadDMA
	}

	d := &Device{cfg: cfg, clkSrc: cfg.ClockSource}
	if d.clkSrc == 0 {
		d.clkSrc = ClockPLL160M
	}
	t := cfg.Timing
	// esp_lcd_new_rgb_panel: LCD_PCLK == LCD_CLK is only allowed when the
	// PCLK polarity does not need to be changed.
	allowEqual := !(t.PclkActiveNeg || t.PclkIdleHigh)
	var ok bool
	d.clk, ok = CalcClock(d.clkSrc.Hz(), t.PclkHz, allowEqual)
	if !ok {
		return nil, ErrBadPclk
	}
	var err error
	d.timing, err = CalcTiming(cfg.Width, cfg.Height, t)
	if err != nil {
		return nil, err
	}
	d.linesPerDesc, d.numDescs, err = DescLayout(cfg.Width*2, cfg.Height)
	if err != nil {
		return nil, err
	}
	if d.numDescs > maxDescriptors {
		return nil, ErrTooManyDescs
	}
	d.refresh = RefreshHz(d.clk.PclkHz, cfg.Width, cfg.Height, t)
	d.Buffer = framebuf.New(buf, int16(cfg.Width), int16(cfg.Height))

	// esp_lcd_new_rgb_panel
	d.lcdInitController()
	dmaInitChannel(cfg.DMAChannel)
	d.buildDescriptors()
	d.routeSignals()
	// esp_lcd_panel_reset -> rgb_panel_reset
	lcdFifoReset()
	lcdReset()
	// esp_lcd_panel_init -> rgb_panel_init (without starting)
	d.lcdSetupRGB()
	return d, nil
}

// Start begins continuous refresh. Port of
// lcd_rgb_panel_start_transmission (esp_lcd_panel_rgb.c): reset DMA and the
// LCD controller, start the DMA first, wait a moment for the DMA to fill the
// LCD FIFO, then start the LCD engine.
func (d *Device) Start() {
	ch := d.cfg.DMAChannel
	dmaReset(ch)
	lcdStop()
	lcdReset()
	lcdFifoReset()
	dmaStart(ch, &descs[0])
	delay1us()
	lcdStart()
	d.started = true
}

// Stop stops the output. The panel keeps the last state of the lines.
func (d *Device) Stop() {
	lcdStop()
	dmaStop(d.cfg.DMAChannel)
	d.started = false
}

// delay1us busy-waits for at least 1us ("delay 1us is sufficient for DMA to
// pass data to LCD FIFO"). Each APB register read takes several 80MHz APB
// cycles, so 100 reads are well over 1us even at 240MHz.
func delay1us() {
	for i := 0; i < 100; i++ {
		_ = esp.LCD_CAM.LC_REG_DATE.Get()
	}
}

// Display does nothing: the panel is refreshed continuously from the frame
// buffer. It exists to satisfy drivers.Displayer.
func (d *Device) Display() error {
	return nil
}

// WaitVSync blocks until the LCD controller finishes the current frame
// (LCD_VSYNC_INT_RAW), or until timeout. Drawing right after it returns
// gives the vertical blanking period plus the time the scan needs to reach
// the area being drawn, which removes most tearing.
//
// This polls the raw interrupt bit, so no interrupt handler is needed.
// It returns false on timeout (e.g. the output is not started).
func (d *Device) WaitVSync(timeout time.Duration) bool {
	if !d.started {
		return false
	}
	esp.LCD_CAM.LC_DMA_INT_CLR.Set(lcdEventVSyncEnd)
	deadline := time.Now().Add(timeout)
	for esp.LCD_CAM.LC_DMA_INT_RAW.Get()&lcdEventVSyncEnd == 0 {
		if time.Now().After(deadline) {
			return false
		}
	}
	return true
}

// PclkHz returns the actual pixel clock.
func (d *Device) PclkHz() uint32 { return d.clk.PclkHz }

// RefreshHz returns the actual frame rate.
func (d *Device) RefreshHz() float32 { return d.refresh }

// Dump prints every register the driver configured, read back from the
// hardware, and whether it matches. It returns false if anything is wrong.
func (d *Device) Dump() bool {
	ok := DumpClocks()
	ok = d.DumpGPIO() && ok
	ok = d.DumpLCD() && ok
	ok = d.DumpDMA() && ok
	fb := uint32(uintptr(unsafe.Pointer(&d.Buffer.Pix[0])))
	logf("frame buffer @%08x, %d bytes, started=%v\n", fb, len(d.Buffer.Pix)*2, d.started)
	logf("=== register check: %s ===\n", okStr(ok))
	return ok
}

// DumpClocks prints the peripheral clock/reset bits for LCD_CAM and GDMA.
func DumpClocks() bool {
	s := esp.SYSTEM
	lcdClk, lcdRst := s.GetPERIP_CLK_EN1_LCD_CAM_CLK_EN(), s.GetPERIP_RST_EN1_LCD_CAM_RST()
	dmaClk, dmaRst := s.GetPERIP_CLK_EN1_DMA_CLK_EN(), s.GetPERIP_RST_EN1_DMA_RST()
	ok := lcdClk == 1 && lcdRst == 0 && dmaClk == 1 && dmaRst == 0
	logf("--- SYSTEM ---\n")
	logf("PERIP_CLK_EN1=%08x PERIP_RST_EN1=%08x\n", s.PERIP_CLK_EN1.Get(), s.PERIP_RST_EN1.Get())
	logf("  LCD_CAM clk_en=%d rst=%d  DMA clk_en=%d rst=%d %s\n", lcdClk, lcdRst, dmaClk, dmaRst, okStr(ok))
	return ok
}

func okStr(ok bool) string {
	if ok {
		return "OK"
	}
	return "NG"
}

func logf(format string, args ...any) {
	fmt.Printf(format, args...)
}

// DMAStatus returns the address of the descriptor the DMA is working on
// (OUT_DSCR) and the OUT_STATE register. If the DMA is running, OUT_DSCR
// changes between calls.
func (d *Device) DMAStatus() (dscr, state uint32) {
	r := dmaOut(d.cfg.DMAChannel)
	return r.DSCR.Get(), r.STATE.Get()
}
