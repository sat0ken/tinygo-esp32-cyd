package app_test

import (
	"testing"

	"github.com/sat0ken/tinygo-cyd/app"
	"github.com/sat0ken/tinygo-cyd/board"
	"github.com/sat0ken/tinygo-cyd/internal/goldentest"
	"github.com/sat0ken/tinygo-cyd/memlcd"
)

func checkGolden(t *testing.T, name string, d *memlcd.Display) {
	t.Helper()
	goldentest.Check(t, name, d.Image())
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
