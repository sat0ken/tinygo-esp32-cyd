package main

// The slides are made from images/ by tools/img2rgb565 into one slidepack
// file. To use your own pictures, put PNG/JPEG files in images/ and run
// `make slides` (which runs this go:generate), then `make flash-slides`
// for the board.
//
// Where the pack comes from depends on the target:
//   - board (source_flash.go): a flash region written separately with
//     `make flash-slides` and memory-mapped at run time. Embedding it in
//     the program does not work: the ESP32-S3 ROM loader refuses a
//     program image with a ~1MB rodata segment.
//   - browser and host tests (source_embed.go): go:embed.
//
//go:generate go run ../../tools/img2rgb565 -pack slides.pack images
