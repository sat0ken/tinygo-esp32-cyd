// Package palquant makes a palette of a few colours for many (median cut),
// for the games whose scenes draw more colours than the frame buffer of
// package picopad holds (256): their tests make a palette for each scene
// from its images, the games keep it and set it with
// picopad.Console.SetPalette.
package palquant

import (
	"sort"

	"github.com/sat0ken/tinygo-cyd/picopad"
)

// Image is a paletted image of PicoLibSDK (as tools/picoimg writes them):
// a byte a pixel (Bits 8) or two (Bits 4, the high nibble first), the
// palette RGB565 little endian. Its pixels count Weight times; the
// pixels of colour Trans do not count when HasTrans.
type Image struct {
	Img      string
	Pal      string
	Bits     int
	Weight   int
	Trans    picopad.Color
	HasTrans bool
}

// Weighted is a colour and how much it counts.
type Weighted struct {
	Col picopad.Color
	W   uint64
}

// Count returns the colours of the images and how often they are drawn.
func Count(imgs []Image) []Weighted {
	var cs []Weighted
	for _, im := range imgs {
		var n [256]uint64
		for i := 0; i < len(im.Img); i++ {
			b := im.Img[i]
			if im.Bits == 4 {
				n[b>>4]++
				n[b&15]++
			} else {
				n[b]++
			}
		}
		for e := 0; e < len(im.Pal)/2 && e < 256; e++ {
			col := picopad.Color(im.Pal[2*e]) | picopad.Color(im.Pal[2*e+1])<<8
			if n[e] == 0 || (im.HasTrans && col == im.Trans) {
				continue
			}
			cs = append(cs, Weighted{col, n[e] * uint64(im.Weight)})
		}
	}
	return cs
}

// rgb returns the components of c on comparable scales: red and blue (5
// bits) twice, as the distance of package picopad.
func rgb(c picopad.Color) [3]int {
	return [3]int{int(c>>11) * 2, int(c >> 5 & 0x3F), int(c&0x1F) * 2}
}

// box is a part of the colours.
type box struct {
	cs   []Weighted
	w    uint64
	sse  float64 // the weighted squared deviation along axis
	axis int
}

func newBox(cs []Weighted) box {
	b := box{cs: cs}
	var sum [3]float64
	for _, c := range cs {
		b.w += c.W
		v := rgb(c.Col)
		for k := range sum {
			sum[k] += float64(v[k]) * float64(c.W)
		}
	}
	b.sse = -1
	for k := 0; k < 3; k++ {
		mean := sum[k] / float64(b.w)
		var s float64
		for _, c := range cs {
			d := float64(rgb(c.Col)[k]) - mean
			s += d * d * float64(c.W)
		}
		if s > b.sse {
			b.sse, b.axis = s, k
		}
	}
	return b
}

// Make returns at most n colours for cs: the box of colours with the
// largest spread (weighted squared deviation along its widest axis) is cut
// where the spread of both sides is least, until there are n boxes; each
// gives its weighted mean.
func Make(cs []Weighted, n int) []picopad.Color {
	// the same colours together
	sort.Slice(cs, func(i, j int) bool { return cs[i].Col < cs[j].Col })
	var u []Weighted
	for _, c := range cs {
		if c.W == 0 {
			continue
		}
		if len(u) > 0 && u[len(u)-1].Col == c.Col {
			u[len(u)-1].W += c.W
		} else {
			u = append(u, c)
		}
	}
	if len(u) == 0 || n <= 0 {
		return nil
	}

	boxes := []box{newBox(u)}
	for len(boxes) < n {
		bi := -1
		for i, b := range boxes {
			if len(b.cs) > 1 && b.sse > 0 && (bi < 0 || b.sse > boxes[bi].sse) {
				bi = i
			}
		}
		if bi < 0 {
			break // every colour has a box
		}
		b := boxes[bi]
		ax := b.axis
		sort.SliceStable(b.cs, func(i, j int) bool { return rgb(b.cs[i].Col)[ax] < rgb(b.cs[j].Col)[ax] })
		// the cut with the least squared deviation of both sides
		var tw, tx, txx float64
		for _, c := range b.cs {
			x, w := float64(rgb(c.Col)[ax]), float64(c.W)
			tw, tx, txx = tw+w, tx+w*x, txx+w*x*x
		}
		var lw, lx, lxx float64
		cut, best := 1, -1.0
		for i, c := range b.cs[:len(b.cs)-1] {
			x, w := float64(rgb(c.Col)[ax]), float64(c.W)
			lw, lx, lxx = lw+w, lx+w*x, lxx+w*x*x
			if rgb(b.cs[i+1].Col)[ax] == rgb(c.Col)[ax] {
				continue // the same value on both sides
			}
			rw, rx, rxx := tw-lw, tx-lx, txx-lxx
			sse := lxx - lx*lx/lw + rxx - rx*rx/rw
			if best < 0 || sse < best {
				cut, best = i+1, sse
			}
		}
		boxes[bi] = newBox(b.cs[:cut])
		boxes = append(boxes, newBox(b.cs[cut:]))
	}

	out := make([]picopad.Color, len(boxes))
	for i, b := range boxes {
		var r, g, bl uint64
		for _, c := range b.cs {
			r += uint64(c.Col>>11) * c.W
			g += uint64(c.Col>>5&0x3F) * c.W
			bl += uint64(c.Col&0x1F) * c.W
		}
		r, g, bl = (r+b.w/2)/b.w, (g+b.w/2)/b.w, (bl+b.w/2)/b.w
		out[i] = picopad.Color(r<<11 | g<<5 | bl)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// Palette returns fixed and the colours of the images made up to 256, as
// the bytes of a picopad.Palette.
func Palette(fixed []picopad.Color, imgs []Image) string {
	cols := append([]picopad.Color(nil), fixed...)
	cols = append(cols, Make(Count(imgs), 256-len(fixed))...)
	b := make([]byte, 0, 2*len(cols))
	for _, c := range cols {
		b = append(b, byte(c), byte(c>>8))
	}
	return string(b)
}
