// 05_slideshow shows the images in slides/ one after another with wipe and
// blinds transitions.
//
//   - Every 5 seconds the next slide is shown.
//   - Touch the right third of the screen for the next slide, the left third
//     for the previous one, the middle to pause / resume.
//   - A caption with the slide number and name is shown for 2 seconds.
//
// The slides are embedded in the program (flash on the board). Add your own
// pictures: put PNG/JPEG files in examples/05_slideshow/images and run
// `make slides`.
//
// It uses package platform, so it runs on the board and in the browser:
//
//	make flash PKG=./examples/05_slideshow
//	make wasm PKG=./examples/05_slideshow && make serve
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/platform"
)

func main() {
	time.Sleep(time.Second) // time to open the serial monitor
	d, t, err := platform.Init()
	if err != nil {
		println("platform.Init:", err.Error())
		select {}
	}
	s, err := NewShow(d, t, slidesFS, "slides")
	if err != nil {
		println(err.Error())
		select {}
	}
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
