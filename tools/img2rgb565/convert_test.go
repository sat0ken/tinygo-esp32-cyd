package main

import (
	"image"
	"image/color"
	"testing"

	"github.com/sat0ken/tinygo-cyd/framebuf"
)

func solid(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
	}
	return img
}

func TestConvertSolidColors(t *testing.T) {
	// Colours that RGB565 represents exactly must come out exactly, with
	// or without dithering, in big endian order.
	tests := []struct {
		c    color.RGBA
		want uint16
	}{
		{color.RGBA{255, 0, 0, 255}, 0xF800},
		{color.RGBA{0, 255, 0, 255}, 0x07E0},
		{color.RGBA{0, 0, 255, 255}, 0x001F},
		{color.RGBA{255, 255, 255, 255}, 0xFFFF},
		{color.RGBA{0, 0, 0, 255}, 0x0000},
	}
	for _, tc := range tests {
		for _, dither := range []bool{false, true} {
			data := Convert(solid(64, 40, tc.c), 32, 20, FitCover, dither)
			if len(data) != 32*20*2 {
				t.Fatalf("len %d", len(data))
			}
			for i := 0; i < len(data); i += 2 {
				if v := uint16(data[i])<<8 | uint16(data[i+1]); v != tc.want {
					t.Fatalf("%v dither=%v: pixel %d = %04x, want %04x", tc.c, dither, i/2, v, tc.want)
				}
			}
		}
	}
}

func TestDitherKeepsAverage(t *testing.T) {
	// A grey between two RGB565 levels: without dithering every pixel is the
	// same level; with dithering the average must be close to the input.
	const grey = 100
	img := solid(64, 64, color.RGBA{grey, grey, grey, 255})
	for _, dither := range []bool{false, true} {
		data := Convert(img, 64, 64, FitCover, dither)
		sum := 0
		for i := 0; i < len(data); i += 2 {
			sum += int(framebuf.ToRGBA(uint16(data[i])<<8 | uint16(data[i+1])).R)
		}
		avg := float64(sum) / float64(len(data)/2)
		t.Logf("dither=%v average red %.2f (input %d)", dither, avg, grey)
		if dither && (avg < grey-0.5 || avg > grey+0.5) {
			t.Errorf("dithered average %.2f, want ~%d", avg, grey)
		}
	}
}

func TestFitRects(t *testing.T) {
	// 4:3 source on a 480x272 screen.
	b := image.Rect(0, 0, 800, 600)
	if r := sourceRect(b, 480, 272, FitCover); r != image.Rect(0, 73, 800, 526) {
		t.Errorf("cover source %v", r)
	}
	if r := targetRect(b, 480, 272, FitContain); r != image.Rect(59, 0, 421, 272) {
		t.Errorf("contain target %v", r)
	}
	// Wide source.
	b = image.Rect(0, 0, 1000, 272)
	if r := sourceRect(b, 480, 272, FitCover); r != image.Rect(260, 0, 740, 272) {
		t.Errorf("cover wide source %v", r)
	}
}
