//go:build esp32s3

// 02_colorbars brings up the RGB panel step by step (phases 1-5 of the plan)
// and shows 8 colour bars with a 1-pixel white frame on all four edges
// (app.DrawColorBars, the same pattern as testdata/golden/colorbars.png).
//
// Every step prints the registers it set, read back from the hardware, with
// OK/NG against the expected value.
//
// Serial commands (115200 baud): c = colour bars, r/g/b/w/k = solid
// red/green/blue/white/black, d = dump registers again.
//
// Flash: tinygo flash -target=./targets/esp32-4827s043.json -monitor ./examples/02_colorbars
package main

import (
	"image/color"
	"machine"
	"strconv"
	"time"
	"unsafe"

	"github.com/sat0ken/tinygo-cyd/app"
	"github.com/sat0ken/tinygo-cyd/platform"
	"github.com/sat0ken/tinygo-cyd/rgblcd"
)

// regsOK is the result of the last register check (Dump).
var regsOK bool

func main() {
	// Opening the serial monitor resets the board (DTR/RTS), and the monitor
	// needs about 2 seconds to connect: wait so the dump below is not lost.
	time.Sleep(3 * time.Second)
	println("\n=== 02_colorbars ===")

	println("--- phase 1: register access ---")
	selfOK := rgblcd.SelfTest(0)
	println("self test:", okStr(selfOK))
	fb := platform.FrameBuffer()
	println("frame buffer at", hex(uint32(uintptr(unsafe.Pointer(&fb[0])))), "bytes", len(fb)*2)

	println("--- phases 2-4: GPIO matrix, LCD_CAM, GDMA ---")
	d, err := rgblcd.New(platform.RGBConfig(), fb)
	if err != nil {
		println("rgblcd.New:", err.Error())
		for {
			time.Sleep(time.Second)
		}
	}
	d.Dump()

	println("--- phase 5: start ---")
	app.DrawColorBars(d)
	d.Start()
	platform.SetBacklight(100)
	regsOK = d.Dump()

	uart := machine.Serial
	for i := 0; ; i++ {
		// Measure the frame rate from the VSYNC flag for one second.
		frames := 0
		start := time.Now()
		for time.Since(start) < time.Second {
			if d.WaitVSync(100 * time.Millisecond) {
				frames++
			}
			for uart.Buffered() > 0 {
				b, _ := uart.ReadByte()
				command(d, b)
			}
		}
		// Sample the DMA position 6 times, 3ms apart, over one frame
		// (~17.6ms). The descriptor index (0..67) should advance with
		// the scan and wrap around, which shows the DMA is running.
		dscr, state := d.DMAStatus()
		idx := ""
		for s := 0; s < 6; s++ {
			a, _ := d.DMAStatus()
			idx += " " + strconv.Itoa(d.DescriptorIndex(a))
			time.Sleep(3 * time.Millisecond)
		}
		// The result of the self test and register check is repeated on
		// every line, so it is visible even if the boot log was missed.
		println("tick", i, "fps", frames, "selftest", okStr(selfOK), "regs", okStr(regsOK),
			"OUT_DSCR", hex(dscr), "OUT_STATE", hex(state), "desc:"+idx)
	}
}

func command(d *rgblcd.Device, b byte) {
	switch b {
	case 'c':
		app.DrawColorBars(d)
	case 'r':
		d.FillScreen(color.RGBA{255, 0, 0, 255})
	case 'g':
		d.FillScreen(color.RGBA{0, 255, 0, 255})
	case 'b':
		d.FillScreen(color.RGBA{0, 0, 255, 255})
	case 'w':
		d.FillScreen(color.RGBA{255, 255, 255, 255})
	case 'k':
		d.FillScreen(color.RGBA{0, 0, 0, 255})
	case 'd':
		regsOK = d.Dump()
	default:
		return
	}
	println("command:", string(rune(b)))
}

func okStr(ok bool) string {
	if ok {
		return "OK"
	}
	return "NG"
}

func hex(v uint32) string {
	const digits = "0123456789abcdef"
	var buf [8]byte
	for i := 7; i >= 0; i-- {
		buf[i] = digits[v&0xF]
		v >>= 4
	}
	return string(buf[:])
}
