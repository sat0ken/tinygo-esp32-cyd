// Command app is the application entry point, shared by the board and the
// browser:
//
//	make flash   # tinygo flash -target=./targets/esp32-4827s043.json ./cmd/app
//	make wasm    # tinygo build -target=wasm -o web/app.wasm ./cmd/app
//
// It has no build tags and does not import machine; package platform picks
// the display and touch implementation for the target.
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/app"
	"github.com/sat0ken/tinygo-cyd/platform"
)

func main() {
	d, t, err := platform.Init()
	if err != nil {
		println("platform.Init:", err.Error())
		platform.Halt()
	}
	a := app.New(d, t)
	a.Draw()
	d.Display()
	println("app: started")

	// Never return: in the browser, returning from main ends the program.
	for {
		if a.Step() {
			d.Display()
		}
		time.Sleep(10 * time.Millisecond)
	}
}
