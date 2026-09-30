// Package app is the application. It depends only on the hal interfaces,
// so the same code runs on the board, in the browser and in host tests.
//
// Screen (480x272):
//
//	+--------------------------------------------------+
//	| title bar                                        |
//	+-------------------+------------------------------+
//	| tinydraw/tinyfont | paint area                   |
//	| demo              |                              |
//	|                   +------------------------------+
//	|                   | palette ............ [CLEAR] |
//	+-------------------+------------------------------+
package app

import (
	"image/color"

	"github.com/sat0ken/tinygo-cyd/hal"
	"tinygo.org/x/tinydraw"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/freesans"
	"tinygo.org/x/tinyfont/proggy"
)

var (
	Black   = color.RGBA{0, 0, 0, 255}
	White   = color.RGBA{255, 255, 255, 255}
	Red     = color.RGBA{255, 0, 0, 255}
	Green   = color.RGBA{0, 255, 0, 255}
	Blue    = color.RGBA{0, 0, 255, 255}
	Yellow  = color.RGBA{255, 255, 0, 255}
	Cyan    = color.RGBA{0, 255, 255, 255}
	Magenta = color.RGBA{255, 0, 255, 255}

	bgColor     = color.RGBA{16, 24, 48, 255}
	titleColor  = color.RGBA{0, 90, 170, 255}
	panelColor  = color.RGBA{32, 40, 72, 255}
	paperColor  = color.RGBA{248, 248, 240, 255}
	borderColor = color.RGBA{160, 170, 200, 255}
	buttonColor = color.RGBA{200, 60, 60, 255}
)

// Rect is a screen rectangle.
type Rect struct{ X, Y, W, H int16 }

// Contains reports whether (x, y) is inside r.
func (r Rect) Contains(x, y int16) bool {
	return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

// Layout of the screen.
var (
	titleBar  = Rect{0, 0, 480, 32}
	demoPanel = Rect{0, 32, 200, 240}
	paintArea = Rect{204, 36, 272, 196}
	clearBtn  = Rect{412, 238, 64, 30}
	palette   = []color.RGBA{Black, Red, Green, Blue, Yellow, Magenta}
)

func swatch(i int) Rect {
	return Rect{204 + int16(i)*34, 238, 30, 30}
}

// App is the paint application.
type App struct {
	d hal.Display
	t hal.Touch

	pen      int  // selected palette index
	down     bool // pen is on the paint area
	lastX    int16
	lastY    int16
	wasTouch bool
}

// New creates the app. Call Draw once, then Step periodically.
func New(d hal.Display, t hal.Touch) *App {
	return &App{d: d, t: t}
}

// Draw draws the whole screen.
func (a *App) Draw() {
	d := a.d
	w, h := d.Size()
	d.FillRectangle(0, 0, w, h, bgColor)

	// Title bar.
	d.FillRectangle(titleBar.X, titleBar.Y, titleBar.W, titleBar.H, titleColor)
	tinyfont.WriteLine(d, &freesans.Bold12pt7b, 8, 23, "ESP32-4827S043 + TinyGo", White)

	DrawDemo(d, demoPanel)

	// Paint area and palette.
	tinydraw.Rectangle(d, paintArea.X-1, paintArea.Y-1, paintArea.W+2, paintArea.H+2, borderColor)
	a.clearPaint()
	for i := range palette {
		a.drawSwatch(i)
	}
	d.FillRectangle(clearBtn.X, clearBtn.Y, clearBtn.W, clearBtn.H, buttonColor)
	tinyfont.WriteLine(d, &proggy.TinySZ8pt7b, clearBtn.X+14, clearBtn.Y+19, "CLEAR", White)
}

// DrawDemo draws tinydraw shapes and tinyfont text inside r.
func DrawDemo(d hal.Display, r Rect) {
	d.FillRectangle(r.X, r.Y, r.W, r.H, panelColor)
	x, y := r.X, r.Y
	tinyfont.WriteLine(d, &freesans.Regular9pt7b, x+8, y+22, "tinydraw", Cyan)

	tinydraw.Rectangle(d, x+10, y+34, 50, 36, Yellow)
	tinydraw.FilledRectangle(d, x+70, y+34, 50, 36, Green)
	tinydraw.Circle(d, x+155, y+52, 18, Magenta)

	tinydraw.FilledCircle(d, x+35, y+104, 22, Red)
	tinydraw.Triangle(d, x+70, y+126, x+95, y+82, x+120, y+126, White)
	tinydraw.FilledTriangle(d, x+132, y+126, x+155, y+82, x+178, y+126, Blue)

	for i := int16(0); i < 8; i++ {
		tinydraw.Line(d, x+10, y+140+i*4, x+189, y+168-i*4, palette[1+int(i)%5])
	}

	tinyfont.WriteLine(d, &freesans.Regular9pt7b, x+8, y+194, "tinyfont", Cyan)
	tinyfont.WriteLine(d, &proggy.TinySZ8pt7b, x+8, y+212, "480x272 RGB565", White)
	tinyfont.WriteLine(d, &proggy.TinySZ8pt7b, x+8, y+228, "LCD_CAM + GDMA", White)
}

// DrawColorBars draws 8 vertical colour bars (white, yellow, cyan, green,
// magenta, red, blue, black) and a 1-pixel white frame on all edges.
func DrawColorBars(d hal.Display) {
	bars := []color.RGBA{White, Yellow, Cyan, Green, Magenta, Red, Blue, Black}
	w, h := d.Size()
	for i, c := range bars {
		x0 := int16(i) * w / int16(len(bars))
		x1 := int16(i+1) * w / int16(len(bars))
		d.FillRectangle(x0, 0, x1-x0, h, c)
	}
	d.FillRectangle(0, 0, w, 1, White)
	d.FillRectangle(0, h-1, w, 1, White)
	d.FillRectangle(0, 0, 1, h, White)
	d.FillRectangle(w-1, 0, 1, h, White)
}

func (a *App) clearPaint() {
	a.d.FillRectangle(paintArea.X, paintArea.Y, paintArea.W, paintArea.H, paperColor)
}

func (a *App) drawSwatch(i int) {
	s := swatch(i)
	frame := bgColor
	if i == a.pen {
		frame = White
	}
	a.d.FillRectangle(s.X, s.Y, s.W, s.H, frame)
	a.d.FillRectangle(s.X+3, s.Y+3, s.W-6, s.H-6, palette[i])
}

// Step reads the touch panel and updates the screen. It returns true if
// anything was drawn.
func (a *App) Step() bool {
	x, y, pressed := a.t.ReadTouch()
	defer func() { a.wasTouch = pressed }()
	if !pressed {
		a.down = false
		return false
	}
	newPress := !a.wasTouch

	if paintArea.Contains(x, y) {
		if a.down {
			a.stroke(a.lastX, a.lastY, x, y)
		} else {
			a.dot(x, y)
		}
		a.down = true
		a.lastX, a.lastY = x, y
		return true
	}
	a.down = false
	if !newPress {
		return false
	}
	if clearBtn.Contains(x, y) {
		a.clearPaint()
		return true
	}
	for i := range palette {
		if swatch(i).Contains(x, y) && i != a.pen {
			old := a.pen
			a.pen = i
			a.drawSwatch(old)
			a.drawSwatch(i)
			return true
		}
	}
	return false
}

// dot draws a 3x3 pen dot clipped to the paint area.
func (a *App) dot(x, y int16) {
	x0, y0, x1, y1 := x-1, y-1, x+2, y+2
	if x0 < paintArea.X {
		x0 = paintArea.X
	}
	if y0 < paintArea.Y {
		y0 = paintArea.Y
	}
	if x1 > paintArea.X+paintArea.W {
		x1 = paintArea.X + paintArea.W
	}
	if y1 > paintArea.Y+paintArea.H {
		y1 = paintArea.Y + paintArea.H
	}
	if x0 < x1 && y0 < y1 {
		a.d.FillRectangle(x0, y0, x1-x0, y1-y0, palette[a.pen])
	}
}

// stroke draws dots along the line from (x0, y0) to (x1, y1) (Bresenham).
func (a *App) stroke(x0, y0, x1, y1 int16) {
	dx, sx := abs(x1-x0), int16(1)
	if x0 > x1 {
		sx = -1
	}
	dy, sy := -abs(y1-y0), int16(1)
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	for {
		a.dot(x0, y0)
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func abs(v int16) int16 {
	if v < 0 {
		return -v
	}
	return v
}
