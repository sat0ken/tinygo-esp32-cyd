// 03_tinydraw draws the application screen (tinydraw shapes and tinyfont
// text) once and prints how long drawing took. It builds for the board and
// for the browser.
//
//	tinygo flash -target=./targets/esp32-4827s043.json -monitor ./examples/03_tinydraw
//	tinygo build -target=wasm -o web/app.wasm ./examples/03_tinydraw
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/app"
	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/platform"
)

func main() {
	time.Sleep(time.Second)
	d, _, err := platform.Init()
	if err != nil {
		println("platform.Init:", err.Error())
		platform.Halt()
	}

	start := time.Now()
	app.New(d, hal.NoTouch{}).Draw()
	d.Display()
	println("03_tinydraw: full screen drawn in", time.Since(start).Microseconds(), "us")

	// Measure a full-screen fill (the fastest possible redraw).
	start = time.Now()
	for i := 0; i < 10; i++ {
		d.FillRectangle(0, 200, 200, 72, app.Black)
	}
	println("FillRectangle 200x72 x10:", time.Since(start).Microseconds(), "us")
	app.New(d, hal.NoTouch{}).Draw()
	d.Display()

	for i := 0; ; i++ {
		time.Sleep(time.Second)
		println("tick", i)
	}
}
