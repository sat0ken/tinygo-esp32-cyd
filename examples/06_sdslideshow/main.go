//go:build esp32s3

// 06_sdslideshow shows the pictures on a microSD card, with the same
// transitions, caption and touch control as 05_slideshow.
//
// Card: FAT32 (exFAT is not supported, cards larger than 32GB usually come
// formatted exFAT and must be reformatted as FAT32). Put the pictures in a
// folder named "slides" (or in the root folder), in either format:
//
//   - .bmp:    480x272, 24-bit or 32-bit, uncompressed (most image editors
//     can save this)
//   - .rgb565: made from PNG/JPEG with
//     go run ./tools/img2rgb565 -o <folder> <images...>
//
// They are shown in file name order. Other files are ignored. JPEG/PNG are
// not decoded on the board (not enough RAM); convert them on the PC first.
//
//	make flash-noerase PKG=./examples/06_sdslideshow && make monitor
package main

import (
	"image/color"
	"time"

	"github.com/sat0ken/tinygo-cyd/board"
	"github.com/sat0ken/tinygo-cyd/hal"
	_ "github.com/sat0ken/tinygo-cyd/internal/espmalloc" // malloc for fatfs (C)
	"github.com/sat0ken/tinygo-cyd/platform"
	"github.com/sat0ken/tinygo-cyd/slideshow"
	"tinygo.org/x/drivers/sdcard"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/proggy"
	"tinygo.org/x/tinyfs/fatfs"
)

// fatFS adapts tinyfs/fatfs to slideshow.FS.
type fatFS struct{ fs *fatfs.FATFS }

func (f fatFS) Open(path string) (slideshow.File, error) {
	file, err := f.fs.Open(path)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func main() {
	time.Sleep(time.Second) // time to open the serial monitor
	println("\n=== 06_sdslideshow ===")
	d, err := platform.InitLCD()
	if err != nil {
		println("InitLCD:", err.Error())
		platform.Halt()
	}
	showMessage(d, "reading the SD card...", "")

	bus := newSharedBus()
	touch := busTouch{bus: bus, t: platform.InitTouch()}

	bus.useSD()
	sd := sdcard.New(bus.spi, bus.sck, bus.mosi, bus.miso, bus.sdCS)
	if err := sd.Configure(); err != nil {
		fail(d, "SD card: "+err.Error(), "insert a FAT32 microSD card and reset")
	}
	bus.mode = 0 // the driver reconfigured the SPI bus at its own speed
	bus.useSD()
	println("SD card:", sd.Size()/1024/1024, "MB")

	fs := fatfs.New(&sd)
	fs.Configure(&fatfs.Config{SectorSize: 512})
	if err := fs.Mount(); err != nil {
		fail(d, "mount: "+err.Error(), "the card must be formatted FAT32 (not exFAT)")
	}

	var src *slideshow.DirSource
	for _, dir := range []string{"/slides", "/"} {
		src, err = slideshow.NewDirSource(fatFS{fs}, dir, board.Width, board.Height, bus.useSD)
		if err == nil && src.Len() > 0 {
			println("folder", dir+":", src.Len(), "pictures")
			for _, f := range src.Files() {
				println("  ", f)
			}
			break
		}
	}
	if src == nil || src.Len() == 0 {
		fail(d, "no pictures found in /slides or /",
			"480x272 .bmp (24/32-bit) or .rgb565 files are needed")
	}

	s, err := slideshow.NewShow(d, touch, src)
	if err != nil {
		fail(d, err.Error(), "")
	}
	// Transitions that read the file row by row, in order. wipe-right would
	// read every row 12 times (40-pixel strips), which is slow from a card.
	s.Transitions = []slideshow.Transition{slideshow.WipeDown, slideshow.Blinds, slideshow.Cut}
	// Reading a picture from the card already takes a moment, which paces
	// the transition by itself; do not add delays on top.
	s.Duration = 0
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

func fail(d hal.Display, msg, help string) {
	println("06_sdslideshow:", msg)
	if help != "" {
		println(help)
	}
	showMessage(d, msg, help)
	platform.Halt()
}

func showMessage(d hal.Display, msg, help string) {
	w, h := d.Size()
	d.FillRectangle(0, 0, w, h, color.RGBA{0, 32, 64, 255})
	white := color.RGBA{255, 255, 255, 255}
	tinyfont.WriteLine(d, &proggy.TinySZ8pt7b, 10, 30, "06_sdslideshow", white)
	tinyfont.WriteLine(d, &proggy.TinySZ8pt7b, 10, 60, msg, white)
	tinyfont.WriteLine(d, &proggy.TinySZ8pt7b, 10, 90, help, white)
	d.Display()
}
