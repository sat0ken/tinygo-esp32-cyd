// Package kart is a kart racing game (a first prototype): drive round a
// course seen from behind the kart. The ground is drawn like the old
// "mode 7" racers: every row of the screen below the horizon is a line
// across the flat map of the course, further away the higher it is.
//
// There is no frame buffer on the SPI panels, so the ground is drawn
// again every frame, a band of rows at a time. A band is sent in the
// background (DrawRGBBitmap8Async, with DMA on the ESP32) while the next
// one is drawn into the other buffer. The sky is drawn once.
//
// It depends only on hal: examples/13_kart runs it.
package kart

import (
	"image/color"
	"math"
	"strconv"
	"time"
	"unsafe"

	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/pad"
	"github.com/sat0ken/tinygo-cyd/romfont/proggy"
	"tinygo.org/x/tinyfont"
)

// The camera, in world units: how high it is and how far behind the kart.
const (
	camHeight = 32
	camBack   = 28

	bandRows = 8 // rows drawn and sent at a time
)

// Driving, in world units and seconds.
const (
	maxSpeed   = 140
	grassSpeed = 45 // on the grass the kart slows down to this
	accel      = 70
	brake      = 140
	drag       = 20
	grassDrag  = 120
	turnRate   = 1.9 // radians per second at full lock
)

var (
	skyColors    = [...]color.RGBA{{90, 160, 230, 255}, {120, 185, 240, 255}, {160, 210, 245, 255}}
	hillColor    = color.RGBA{60, 120, 90, 255}
	hudColor     = color.RGBA{255, 255, 255, 255}
	outsideColor = color.RGBA{40, 110, 40, 255}

	// The two colours of each tile (see track.cells).
	tileColors = [numTiles][2]color.RGBA{
		grass:     {{70, 160, 60, 255}, {60, 145, 52, 255}},
		road:      {{105, 105, 110, 255}, {95, 95, 100, 255}},
		curb:      {{220, 40, 40, 255}, {245, 245, 245, 255}},
		startLine: {{20, 20, 20, 255}, {245, 245, 245, 255}},
	}
)

// layout is where the horizon is and how wide the view is.
type layout struct {
	horizon int16   // the first row of the ground
	focal   float32 // pixels; smaller is wider
	scale   int16   // pixels per bit of the kart sprite
}

// layoutFor returns the layout for a w x h screen. The kart's wheels are
// on the row horizon + camHeight*focal/camBack, about 90% down the ground.
func layoutFor(w, h int16) layout {
	if w >= 480 && h >= 272 {
		return layout{horizon: 100, focal: 136, scale: 2}
	}
	return layout{horizon: 88, focal: 120, scale: 2}
}

type bitmapDrawer interface {
	DrawRGBBitmap8(x, y int16, data []uint8, w, h int16) error
}

// asyncDrawer is spilcd.Device (and viewport.Display over one).
type asyncDrawer interface {
	DrawRGBBitmap8Async(x, y int16, data []uint8, w, h int16) error
	Wait()
}

// Game is the kart game.
type Game struct {
	d    hal.Display
	bd   bitmapDrawer
	ad   asyncDrawer // nil: draw at once
	t    hal.Touch
	w, h int16
	l    layout

	// Pad, if set, is the game controller: the stick steers, A
	// accelerates, B brakes. Without it, touching accelerates and the
	// touch steers towards its side of the screen.
	Pad hal.Pad
	in  pad.Input

	track   *track
	kart    *sprite
	bufs    [2][]uint16 // bandRows rows of RGB565, stored so that the bytes are big endian
	bytes   [2][]byte   // bufs as bytes, for DrawRGBBitmap8
	next    int         // the buffer to draw into: the other may be going out
	pal     [numTiles * 2]uint16
	outside uint16

	x, y, angle float32 // the kart, in world units; angle 0 is east (+x), y grows south
	speed       float32

	// Timing of the last second, for the log and the HUD.
	frames              int
	frameTime, sendTime time.Duration
	secStart            time.Time
	fps                 int
	stats               Stats
	hudShown            string
}

// Stats is the timing of the last second.
type Stats struct {
	FPS         int
	Frame, Send time.Duration // per frame: all, sending to the panel (with DMA: waiting for it)
	Speed, X, Y int
	OnRoad      bool
}

// NewGame makes the course and puts the kart on the start line. (The
// course, the sprite and the colours are made here, not by package
// initializers: see games/invaders/sprites.go.)
func NewGame(d hal.Display, t hal.Touch) *Game {
	w, h := d.Size()
	bd, _ := d.(bitmapDrawer)
	ad, _ := d.(asyncDrawer)
	g := &Game{d: d, bd: bd, ad: ad, t: t, w: w, h: h, l: layoutFor(w, h)}
	g.track = newTrack()
	g.kart = parseSprite(kartArt[:])
	for k := range g.bufs {
		g.bufs[k] = make([]uint16, int(w)*bandRows)
		g.bytes[k] = unsafe.Slice((*byte)(unsafe.Pointer(&g.bufs[k][0])), len(g.bufs[k])*2)
	}
	for k, c := range tileColors {
		g.pal[2*k], g.pal[2*k+1] = bigEndian(rgb565(c[0])), bigEndian(rgb565(c[1]))
	}
	g.outside = bigEndian(rgb565(outsideColor))
	g.reset()
	return g
}

// reset puts the kart on the start line, facing along the course.
func (g *Game) reset() {
	p, q := g.track.path[0], g.track.path[2]
	g.x, g.y = p[0]*cellSize, p[1]*cellSize
	g.angle = float32(math.Atan2(float64(q[1]-p[1]), float64(q[0]-p[0])))
	g.speed = 0
}

// DrawAll draws the sky and the ground.
func (g *Game) DrawAll() {
	hz := g.l.horizon
	band := hz / int16(len(skyColors))
	for i, c := range skyColors {
		y := int16(i) * band
		hh := band
		if i == len(skyColors)-1 {
			hh = hz - y
		}
		g.d.FillRectangle(0, y, g.w, hh, c)
	}
	// Far hills: a row of bumps along the horizon.
	for x := int16(0); x < g.w; x += 4 {
		k := (x / 4) % 24
		if k > 12 {
			k = 24 - k
		}
		hh := 4 + k + (x/4)%5
		g.d.FillRectangle(x, hz-hh, 4, hh, hillColor)
	}
	g.hudShown = ""
	g.render()
	g.d.Display()
}

// Update reads the controls, moves the kart by dt seconds and draws the
// ground.
func (g *Game) Update(dt float32) {
	start := time.Now()
	if g.secStart.IsZero() {
		g.secStart = start
	}
	g.in.Update(g.Pad, dt)
	gas, brakes, steer := g.controls()
	g.drive(dt, gas, brakes, steer)
	g.render()
	g.drawHUD()
	g.d.Display()

	g.frames++
	g.frameTime += time.Since(start)
	if s := time.Since(g.secStart); s >= time.Second {
		n := time.Duration(g.frames)
		g.stats = Stats{
			FPS: int(int64(g.frames) * int64(time.Second) / int64(s)), Frame: g.frameTime / n, Send: g.sendTime / n,
			Speed: int(g.speed), X: int(g.x), Y: int(g.y), OnRoad: g.track.at(g.x, g.y) != grass,
		}
		g.fps = g.stats.FPS
		g.frames, g.frameTime, g.sendTime, g.secStart = 0, 0, 0, time.Now()
	}
}

// Stats returns the timing of the last full second.
func (g *Game) Stats() Stats { return g.stats }

// controls returns the gas and brake (on or off) and the steering (-1 is
// full left, 1 full right).
func (g *Game) controls() (gas, brakes bool, steer float32) {
	if g.in.Used {
		gas = g.in.Held(hal.PadA | hal.PadOK)
		brakes = g.in.Held(hal.PadB)
		switch {
		case g.in.Held(hal.PadLeft):
			steer = -1
		case g.in.Held(hal.PadRight):
			steer = 1
		}
		return
	}
	x, _, pressed := g.t.ReadTouch()
	if !pressed {
		return
	}
	steer = float32(x-g.w/2) / float32(g.w/3)
	if steer > 1 {
		steer = 1
	} else if steer < -1 {
		steer = -1
	}
	if steer > -0.15 && steer < 0.15 {
		steer = 0 // a little dead zone in the middle
	}
	return true, false, steer
}

// drive moves the kart: speed, steering (more grip at speed), the grass.
func (g *Game) drive(dt float32, gas, brakes bool, steer float32) {
	onGrass := g.track.at(g.x, g.y) == grass
	switch {
	case brakes:
		g.speed -= brake * dt
	case gas:
		g.speed += accel * dt
	default:
		g.speed -= drag * dt
	}
	top := float32(maxSpeed)
	if onGrass {
		top = grassSpeed
	}
	if g.speed > top {
		g.speed -= grassDrag * dt
		if g.speed < top {
			g.speed = top
		}
	}
	if g.speed < 0 {
		g.speed = 0
	}
	grip := g.speed / 40
	if grip > 1 {
		grip = 1
	}
	g.angle += steer * turnRate * grip * dt
	sin, cos := sincos(g.angle)
	g.x += cos * g.speed * dt
	g.y += sin * g.speed * dt
	const margin = 2 * cellSize
	g.x = clamp(g.x, margin, worldSize-margin)
	g.y = clamp(g.y, margin, worldSize-margin)
}

// render draws the ground below the horizon, bandRows rows at a time,
// with the kart on it.
func (g *Game) render() {
	l := &g.l
	sin, cos := sincos(g.angle)
	fx, fy := cos, sin  // forward
	rx, ry := -sin, cos // right
	camX, camY := g.x-fx*camBack, g.y-fy*camBack

	// The kart sprite: its wheels touch the ground on the kart's row.
	kartRow := l.horizon + int16(camHeight*l.focal/camBack)
	sw, sh := g.kart.w*l.scale, g.kart.h*l.scale
	sx, sy := (g.w-sw)/2, kartRow+4-sh

	half := float32(g.w) / 2
	for y0 := l.horizon; y0 < g.h; y0 += bandRows {
		n := g.h - y0
		if n > bandRows {
			n = bandRows
		}
		buf, bytes := g.bufs[g.next], g.bytes[g.next]
		i := 0
		for y := y0; y < y0+n; y++ {
			// The row is a line across the map dist units ahead.
			dist := camHeight * l.focal / float32(y-l.horizon+1)
			step := dist / l.focal
			wx := camX + fx*dist - rx*step*half
			wy := camY + fy*dist - ry*step*half
			mask := uint8(0xFF)
			if dist > 300 {
				mask = 0xFE // plain colours: the pattern would only flicker
			}
			row := buf[i : i+int(g.w)]
			g.groundRow(row, fixed(wx), fixed(wy), fixed(rx*step), fixed(ry*step), mask, dist > halfDist)
			if y >= sy && y < sy+sh {
				g.kartRow(row, (y-sy)/l.scale, sx)
			}
			i += int(g.w)
		}
		t := time.Now()
		switch {
		case g.ad != nil:
			g.ad.DrawRGBBitmap8Async(0, y0, bytes[:i*2], g.w, n) // waits for the other buffer
		case g.bd != nil:
			g.bd.DrawRGBBitmap8(0, y0, bytes[:i*2], g.w, n)
		}
		g.sendTime += time.Since(t)
		g.next ^= 1
	}
}

// Rows further than halfDist are drawn at half the width (every pixel
// twice): there is no detail to lose up there.
const halfDist = 120

// groundRow fills row with the ground: u, v is the map position of the
// first pixel (16.16 fixed point world units), du, dv the step to the
// next; mask clears the shade bit of the cells far away. With half, each
// computed pixel is written twice.
func (g *Game) groundRow(row []uint16, u, v, du, dv int32, mask uint8, half bool) {
	cells := &g.track.cells
	pal := &g.pal
	step := 1
	if half {
		step = 2
		du, dv = du*2, dv*2
	}
	for k := 0; k+step <= len(row); k += step {
		c := g.outside
		cu, cv := uint32(u)>>(16+cellBits), uint32(v)>>(16+cellBits)
		if cu|cv < mapSize {
			c = pal[cells[cv*mapSize+cu]&mask&uint8(numTiles*2-1)]
		}
		row[k] = c
		if half {
			row[k+1] = c
		}
		u += du
		v += dv
	}
}

// kartRow draws row r of the kart sprite over row, starting at pixel x.
func (g *Game) kartRow(row []uint16, r int16, x int16) {
	s := g.kart
	for b := int16(0); b < s.w; b++ {
		c, on := s.at(b, r)
		if !on {
			continue
		}
		for k := int16(0); k < g.l.scale; k++ {
			row[x+b*g.l.scale+k] = c
		}
	}
}

// drawHUD writes the speed and the frames per second in the sky.
func (g *Game) drawHUD() {
	text := "SPEED " + strconv.Itoa(int(g.speed)) + "   " + strconv.Itoa(g.fps) + " FPS"
	if text == g.hudShown {
		return
	}
	g.d.FillRectangle(4, 4, 150, 12, skyColors[0])
	tinyfont.WriteLine(g.d, &proggy.TinySZ8pt7b, 6, 13, text, hudColor)
	g.hudShown = text
}

func sincos(a float32) (sin, cos float32) {
	s, c := math.Sincos(float64(a))
	return float32(s), float32(c)
}

func fixed(v float32) int32 { return int32(v * 65536) }

func clamp(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// bigEndian swaps the bytes of an RGB565 colour, so that storing it in
// little endian memory (ESP32, WebAssembly) gives the panel's byte order.
func bigEndian(c uint16) uint16 { return c<<8 | c>>8 }

func rgb565(c color.RGBA) uint16 {
	return uint16(c.R&0xF8)<<8 | uint16(c.G&0xFC)<<3 | uint16(c.B)>>3
}
