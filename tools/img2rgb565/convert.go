package main

import (
	"image"
	"image/color"

	"golang.org/x/image/draw"
)

// Fit selects how an image with a different aspect ratio is fitted.
type Fit int

const (
	FitCover   Fit = iota // fill the screen, crop the overflow (centred)
	FitContain            // show the whole image, black bars
)

// Convert scales img to w x h and encodes it as big endian RGB565
// (2 bytes per pixel, row-major), the format of framebuf.DrawRGBBitmap8 and
// pixel.RGB565BE.
//
// With dither, the quantisation error to 5/6/5 bits is diffused
// (Floyd-Steinberg), which removes the banding of smooth gradients.
func Convert(img image.Image, w, h int, fit Fit, dither bool) []byte {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.Black), image.Point{}, draw.Src)
	draw.CatmullRom.Scale(dst, targetRect(img.Bounds(), w, h, fit), img, sourceRect(img.Bounds(), w, h, fit), draw.Over, nil)
	return encode(dst, dither)
}

// sourceRect is the part of the source that is shown (cropped for cover).
func sourceRect(b image.Rectangle, w, h int, fit Fit) image.Rectangle {
	if fit != FitCover {
		return b
	}
	sw, sh := b.Dx(), b.Dy()
	if sw*h > sh*w { // source is wider: crop left and right
		cw := sh * w / h
		x := b.Min.X + (sw-cw)/2
		return image.Rect(x, b.Min.Y, x+cw, b.Max.Y)
	}
	ch := sw * h / w // source is taller: crop top and bottom
	y := b.Min.Y + (sh-ch)/2
	return image.Rect(b.Min.X, y, b.Max.X, y+ch)
}

// targetRect is where the image goes on the screen (centred for contain).
func targetRect(b image.Rectangle, w, h int, fit Fit) image.Rectangle {
	if fit != FitContain {
		return image.Rect(0, 0, w, h)
	}
	sw, sh := b.Dx(), b.Dy()
	if sw*h > sh*w {
		th := sh * w / sw
		y := (h - th) / 2
		return image.Rect(0, y, w, y+th)
	}
	tw := sw * h / sh
	x := (w - tw) / 2
	return image.Rect(x, 0, x+tw, h)
}

// quant rounds an 8-bit value to a level of the given bit count and
// returns the level and the 8-bit value it is displayed as (the top bits
// copied into the low bits, like framebuf.ToRGBA).
func quant(v int, bits uint) (level int, shown int) {
	max := 1<<bits - 1
	if v < 0 {
		v = 0
	}
	if v > 255 {
		v = 255
	}
	level = (v*max + 127) / 255
	shown = level<<(8-bits) | level>>(2*bits-8)
	return level, shown
}

func encode(img *image.RGBA, dither bool) []byte {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	out := make([]byte, 0, w*h*2)
	// Error buffers for the current and the next row, 3 channels, with one
	// pixel of padding on each side.
	cur := make([]int, (w+2)*3)
	next := make([]int, (w+2)*3)
	bits := [3]uint{5, 6, 5}
	for y := 0; y < h; y++ {
		for i := range next {
			next[i] = 0
		}
		for x := 0; x < w; x++ {
			o := img.PixOffset(x, y)
			var lv [3]int
			for c := 0; c < 3; c++ {
				v := int(img.Pix[o+c])
				if dither {
					v += cur[(x+1)*3+c] / 16
				}
				l, shown := quant(v, bits[c])
				lv[c] = l
				if dither {
					e := v - shown
					if e > 255 {
						e = 255
					} else if e < -255 {
						e = -255
					}
					cur[(x+2)*3+c] += e * 7
					next[(x)*3+c] += e * 3
					next[(x+1)*3+c] += e * 5
					next[(x+2)*3+c] += e * 1
				}
			}
			v := uint16(lv[0])<<11 | uint16(lv[1])<<5 | uint16(lv[2])
			out = append(out, byte(v>>8), byte(v))
		}
		cur, next = next, cur
	}
	return out
}
