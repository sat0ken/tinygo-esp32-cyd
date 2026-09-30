package app_test

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/sat0ken/tinygo-cyd/app"
	"github.com/sat0ken/tinygo-cyd/board"
	"github.com/sat0ken/tinygo-cyd/memlcd"
)

var update = flag.Bool("update", false, "rewrite the golden images in testdata/golden")

const goldenDir = "../testdata/golden"

// checkGolden compares the display with testdata/golden/<name>.png pixel by
// pixel. On mismatch the actual image is written to
// testdata/golden/<name>.actual.png (ignored by git) for inspection.
func checkGolden(t *testing.T, name string, d *memlcd.Display) {
	t.Helper()
	path := filepath.Join(goldenDir, name+".png")
	actualPath := filepath.Join(goldenDir, name+".actual.png")
	if *update {
		if err := d.SavePNG(path); err != nil {
			t.Fatal(err)
		}
		os.Remove(actualPath)
		t.Logf("updated %s", path)
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run `go test ./app -update` to create it)", err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	got := d.Image()
	if diff := compare(img, got); diff != "" {
		d.SavePNG(actualPath)
		t.Fatalf("%s: %s (actual written to %s)", name, diff, actualPath)
	}
	os.Remove(actualPath)
}

func compare(want image.Image, got *image.RGBA) string {
	if want.Bounds() != got.Bounds() {
		return fmt.Sprintf("size %v, want %v", got.Bounds(), want.Bounds())
	}
	n := 0
	first := ""
	b := want.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			wr, wg, wb, _ := want.At(x, y).RGBA()
			gr, gg, gb, _ := got.At(x, y).RGBA()
			if wr != gr || wg != gg || wb != gb {
				if n == 0 {
					first = fmt.Sprintf("first at (%d,%d): got %02x%02x%02x want %02x%02x%02x",
						x, y, gr>>8, gg>>8, gb>>8, wr>>8, wg>>8, wb>>8)
				}
				n++
			}
		}
	}
	if n == 0 {
		return ""
	}
	return fmt.Sprintf("%d pixels differ, %s", n, first)
}

func TestColorBars(t *testing.T) {
	d := memlcd.New(board.Width, board.Height)
	app.DrawColorBars(d)
	// The frame must be on all four edges.
	for x := int16(0); x < board.Width; x++ {
		if d.Pixel(x, 0) != 0xFFFF || d.Pixel(x, board.Height-1) != 0xFFFF {
			t.Fatalf("top/bottom frame missing at x=%d", x)
		}
	}
	for y := int16(0); y < board.Height; y++ {
		if d.Pixel(0, y) != 0xFFFF || d.Pixel(board.Width-1, y) != 0xFFFF {
			t.Fatalf("left/right frame missing at y=%d", y)
		}
	}
	checkGolden(t, "colorbars", d)
}

func TestInitialScreen(t *testing.T) {
	d := memlcd.New(board.Width, board.Height)
	a := app.New(d, &memlcd.Touch{})
	a.Draw()
	checkGolden(t, "app_initial", d)
}

func TestTouchPaint(t *testing.T) {
	d := memlcd.New(board.Width, board.Height)
	touch := &memlcd.Touch{}
	a := app.New(d, touch)
	a.Draw()

	tap := func(x, y int16) {
		touch.X, touch.Y, touch.Pressed = x, y, true
		a.Step()
		touch.Pressed = false
		a.Step()
	}
	drag := func(pts ...[2]int16) {
		for _, p := range pts {
			touch.X, touch.Y, touch.Pressed = p[0], p[1], true
			a.Step()
		}
		touch.Pressed = false
		a.Step()
	}

	tap(204+34+15, 253) // red swatch
	drag([2]int16{230, 60}, [2]int16{300, 120}, [2]int16{450, 80})
	tap(204+3*34+15, 253) // blue swatch
	drag([2]int16{240, 200}, [2]int16{440, 210})
	checkGolden(t, "app_touch", d)

	tap(440, 250) // CLEAR
	d2 := memlcd.New(board.Width, board.Height)
	b := app.New(d2, &memlcd.Touch{})
	b.Draw()
	// After CLEAR only the palette selection (blue) differs from the start.
	for y := int16(36); y < 232; y++ {
		for x := int16(204); x < 476; x++ {
			if d.Pixel(x, y) != d2.Pixel(x, y) {
				t.Fatalf("paint area not cleared at (%d,%d)", x, y)
			}
		}
	}
}
