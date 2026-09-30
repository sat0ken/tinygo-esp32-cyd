//go:build esp32s3

package rgblcd

import (
	"device/esp"
	"runtime/volatile"
	"unsafe"
)

// GPIO matrix output signal numbers.
// Source: ESP-IDF soc/esp32s3/include/soc/gpio_sig_map.h and
// soc/esp32s3/lcd_periph.c (lcd_periph_rgb_signals).
const (
	sigLCDDataOut0 = 133 // LCD_DATA_OUT0_IDX (LCD_DATA_OUTn_IDX = 133 + n, n = 0..15)
	sigLCDHEnable  = 150 // LCD_H_ENABLE_IDX, used as DE
	sigLCDHSync    = 151 // LCD_H_SYNC_IDX
	sigLCDVSync    = 152 // LCD_V_SYNC_IDX
	sigLCDPclk     = 154 // LCD_PCLK_IDX
)

const (
	// PIN_FUNC_GPIO: IO_MUX function 1 routes the pad through the GPIO matrix.
	// Source: TinyGo machine_esp32s3.go ("MCU_SEL: Function 1 is always GPIO"),
	// ESP32-S3 TRM "IO MUX Pin Functions".
	ioMuxFuncGPIO = 1
	// Default pad drive strength (GPIO_DRIVE_CAP_DEFAULT = 2, also the
	// IO_MUX_GPIOn_REG reset value). The ESP-IDF RGB driver does not change
	// it. On GPIO17/18 the register encoding of 1 and 2 is swapped
	// (gpio_ll_set_drive_capability); routePin handles that.
	ioMuxDrvDefault = 2
)

// pinSignal is one GPIO/signal pair to route.
type pinSignal struct {
	gpio   uint8
	signal uint32
	name   string
}

func (d *Device) pinSignals() []pinSignal {
	c := &d.cfg
	list := make([]pinSignal, 0, 20)
	for i, p := range c.DataPins {
		if p != NoPin {
			list = append(list, pinSignal{p, sigLCDDataOut0 + uint32(i), dataName(i)})
		}
	}
	if c.HSync != NoPin {
		list = append(list, pinSignal{c.HSync, sigLCDHSync, "HSYNC"})
	}
	if c.VSync != NoPin {
		list = append(list, pinSignal{c.VSync, sigLCDVSync, "VSYNC"})
	}
	if c.PCLK != NoPin {
		list = append(list, pinSignal{c.PCLK, sigLCDPclk, "PCLK"})
	}
	if c.DE != NoPin {
		list = append(list, pinSignal{c.DE, sigLCDHEnable, "DE"})
	}
	return list
}

func dataName(i int) string {
	const digits = "0123456789"
	if i < 10 {
		return "DATA" + digits[i:i+1]
	}
	return "DATA1" + digits[i-10:i-9]
}

func ioMuxReg(gpio uint8) *volatile.Register32 {
	// IO_MUX_GPIOn_REG = IO_MUX base + 0x4 + 4*n (TinyGo device/esp IO_MUX.GPIO0 is at 0x4).
	return (*volatile.Register32)(unsafe.Add(unsafe.Pointer(&esp.IO_MUX.GPIO0), uintptr(gpio)*4))
}

func outSelReg(gpio uint8) *volatile.Register32 {
	// GPIO_FUNCn_OUT_SEL_CFG_REG = GPIO base + 0x554 + 4*n.
	return (*volatile.Register32)(unsafe.Add(unsafe.Pointer(&esp.GPIO.FUNC0_OUT_SEL_CFG), uintptr(gpio)*4))
}

// routePin connects a peripheral output signal to a pad.
// Source: ESP-IDF lcd_rgb_panel_configure_gpio, which calls
//
//	gpio_func_sel(gpio, PIN_FUNC_GPIO)                    // IO_MUX MCU_SEL = 1
//	esp_rom_gpio_connect_out_signal(gpio, sig, false, false)
//
// and esp_rom_gpio_connect_out_signal (esp_rom/patches/esp_rom_gpio.c):
// FUNCn_OUT_SEL_CFG = sig (no inversion, OEN_SEL = 0: the output enable comes
// from the peripheral), then GPIO_ENABLE_W1TS = 1 << gpio.
//
// GPIO19/20 (USB D-/D+) would additionally need the USB pad disabled; they
// are not used by this driver's boards.
func routePin(gpio uint8, signal uint32) {
	mux := ioMuxReg(gpio)
	drv := uint32(ioMuxDrvDefault)
	if gpio == 17 || gpio == 18 {
		drv = 1 // bits 0/1 swapped on these pads (gpio_ll_set_drive_capability)
	}
	v := mux.Get()
	v &^= esp.IO_MUX_GPIO_MCU_SEL_Msk | esp.IO_MUX_GPIO_FUN_DRV_Msk
	v |= ioMuxFuncGPIO<<esp.IO_MUX_GPIO_MCU_SEL_Pos | drv<<esp.IO_MUX_GPIO_FUN_DRV_Pos
	mux.Set(v)

	outSelReg(gpio).Set(signal << esp.GPIO_FUNC_OUT_SEL_CFG_OUT_SEL_Pos)

	if gpio < 32 {
		esp.GPIO.ENABLE_W1TS.Set(1 << gpio)
	} else {
		esp.GPIO.ENABLE1_W1TS.Set(1 << (gpio - 32))
	}
}

func (d *Device) routeSignals() {
	for _, ps := range d.pinSignals() {
		routePin(ps.gpio, ps.signal)
	}
}

func outputEnabled(gpio uint8) bool {
	if gpio < 32 {
		return esp.GPIO.ENABLE.Get()&(1<<gpio) != 0
	}
	return esp.GPIO.ENABLE1.Get()&(1<<(gpio-32)) != 0
}

// DumpGPIO prints the routing of every pin and checks it against the
// configuration. It returns false if any pin is wrong.
func (d *Device) DumpGPIO() bool {
	logf("--- GPIO matrix (FUNCn_OUT_SEL_CFG) ---\n")
	ok := true
	for _, ps := range d.pinSignals() {
		sel := outSelReg(ps.gpio).Get() & esp.GPIO_FUNC_OUT_SEL_CFG_OUT_SEL_Msk
		mux := ioMuxReg(ps.gpio).Get()
		mcuSel := (mux & esp.IO_MUX_GPIO_MCU_SEL_Msk) >> esp.IO_MUX_GPIO_MCU_SEL_Pos
		drv := (mux & esp.IO_MUX_GPIO_FUN_DRV_Msk) >> esp.IO_MUX_GPIO_FUN_DRV_Pos
		oe := outputEnabled(ps.gpio)
		good := sel == ps.signal && mcuSel == ioMuxFuncGPIO && oe
		if !good {
			ok = false
		}
		logf("GPIO%-2d %-6s OUT_SEL=%3d (want %3d) MCU_SEL=%d DRV=%d OE=%v %s\n",
			ps.gpio, ps.name, sel, ps.signal, mcuSel, drv, oe, okStr(good))
	}
	return ok
}
