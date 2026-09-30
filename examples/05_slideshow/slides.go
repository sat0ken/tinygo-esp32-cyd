package main

import "embed"

// The slides are raw RGB565 files made from images/ by tools/img2rgb565.
// To use your own pictures, put PNG/JPEG files in images/ and run
// `go generate ./examples/05_slideshow` (or `make slides`).
//
// go:embed data is read-only, so it stays in flash (DROM) on the board and
// costs no RAM: 255KB per slide.
//
//go:generate go run ../../tools/img2rgb565 -o slides images
//go:embed slides/*.rgb565
var slidesFS embed.FS
