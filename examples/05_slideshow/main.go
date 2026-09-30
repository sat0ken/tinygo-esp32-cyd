// 05_slideshow shows the images in slides/ one after another with wipe and
// blinds transitions.
//
//   - Every 5 seconds the next slide is shown.
//   - Touch the right third of the screen for the next slide, the left third
//     for the previous one, the middle to pause / resume.
//   - A caption with the slide number and name is shown for 2 seconds.
//
// The slides are one slidepack file, slides.pack (see slides.go). Add your
// own pictures: put PNG/JPEG files in examples/05_slideshow/images and run
// `make slides`.
//
// It uses package platform, so it runs on the board and in the browser:
//
//	make flash-slides                               # board: write slides.pack to flash
//	make flash-noerase PKG=./examples/05_slideshow  # program only, keeps the slides
//	make monitor
//	make wasm PKG=./examples/05_slideshow && make serve
package main

import (
	"image/color"
	"time"

	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/platform"
	"github.com/sat0ken/tinygo-cyd/slideshow"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/proggy"
)

func main() {
	time.Sleep(time.Second) // time to open the serial monitor
	d, t, err := platform.Init()
	if err != nil {
		println("platform.Init:", err.Error())
		platform.Halt()
	}
	pack, err := openPack()
	if err == nil {
		var s *slideshow.Show
		if s, err = slideshow.NewShow(d, t, slideshow.PackSource{Pack: pack}); err == nil {
			run(s)
		}
	}
	println("slideshow:", err.Error())
	println(packHelp)
	showError(d, err.Error(), packHelp)
	platform.Halt()
}

func run(s *slideshow.Show) {
	if err := s.Start(); err != nil {
		println("draw:", err.Error())
	}
	for {
		if err := s.Step(); err != nil {
			println("slideshow:", err.Error())
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// showError puts the error on the screen, so a missing pack is obvious
// without the serial log.
func showError(d hal.Display, msg, help string) {
	w, h := d.Size()
	d.FillRectangle(0, 0, w, h, color.RGBA{64, 0, 0, 255})
	white := color.RGBA{255, 255, 255, 255}
	tinyfont.WriteLine(d, &proggy.TinySZ8pt7b, 10, 30, "05_slideshow: no slides", white)
	tinyfont.WriteLine(d, &proggy.TinySZ8pt7b, 10, 60, msg, white)
	tinyfont.WriteLine(d, &proggy.TinySZ8pt7b, 10, 90, help, white)
	d.Display()
}
