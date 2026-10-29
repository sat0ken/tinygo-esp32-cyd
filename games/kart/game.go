// Package kart is a kart racing game: three laps of a course against
// three CPU karts, seen from behind the kart. The ground is drawn like the
// old "mode 7" racers: every row of the screen below the horizon is a line
// across the flat map of the course, further away the higher it is.
//
// There is no frame buffer on the SPI panels, so the ground is drawn
// again every frame, a band of rows at a time. A band is sent in the
// background (DrawRGBBitmap8Async, with DMA on the ESP32) while the next
// one is drawn into the other buffer. The sky is drawn once; the HUD in it
// only where it changes.
//
// It depends only on hal: examples/13_kart runs it alone, cmd/games from
// a launcher.
package kart

import (
	"image/color"
	"math"
	"time"
	"unsafe"

	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/pad"
)

var (
	outsideColor = color.RGBA{40, 110, 40, 255}

	// The two colours of each tile (see track.cells).
	tileColors = [numTiles][2]color.RGBA{
		grass:     {{70, 160, 60, 255}, {60, 145, 52, 255}},
		road:      {{105, 105, 110, 255}, {95, 95, 100, 255}},
		curb:      {{220, 40, 40, 255}, {245, 245, 245, 255}},
		startLine: {{20, 20, 20, 255}, {245, 245, 245, 255}},
		boost:     {{255, 140, 0, 255}, {255, 225, 60, 255}},
	}
)

type rect struct{ x, y, w, h int16 }

func (r rect) has(x, y int16) bool { return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h }

func (r rect) box() pad.Box { return pad.Box{X: r.x, Y: r.y, W: r.w, H: r.h} }

// layout is where everything is on a screen size.
type layout struct {
	horizon int16   // the first row of the ground
	focal   float32 // pixels; smaller is wider
	scale   int16   // pixels per pixel of the player's kart

	hudX                       int16 // the left of the HUD texts
	lapBase, posBase, timeBase int16 // their baselines
	minimap                    rect
	countBase                  int16 // the countdown

	titleBase int16
	panel     rect // behind the level buttons
	level0    rect // the first level button
	levelStep int16
	box       rect // the results
	again     rect // in the results: race again, or back to the menu
	menu      rect
}

// layoutFor returns the layout for a w x h screen: 480x272 and larger, or
// 320x240 (ESP32-2432S028R). The kart's wheels are on the row horizon +
// camHeight*focal/camBack, about 90% down the ground.
func layoutFor(w, h int16) layout {
	if w >= 480 && h >= 272 {
		return layout{
			horizon: 100, focal: 136, scale: 2,
			hudX: 10, lapBase: 24, posBase: 50, timeBase: 68,
			minimap: rect{w - 70, 4, 64, 64}, countBase: 72,
			titleBase: 66,
			panel:     rect{40, 156, 400, 100}, level0: rect{52, 178, 120, 50}, levelStep: 128,
			box:   rect{100, 108, 280, 140},
			again: rect{112, 212, 124, 28}, menu: rect{244, 212, 124, 28},
		}
	}
	return layout{
		horizon: 88, focal: 120, scale: 2,
		hudX: 8, lapBase: 22, posBase: 46, timeBase: 62,
		minimap: rect{w - 66, 2, 64, 64}, countBase: 62,
		titleBase: 62, // clears the "< GAMES" button
		panel:     rect{6, 136, 308, 92}, level0: rect{14, 154, 92, 46}, levelStep: 100,
		box:   rect{30, 96, 260, 130},
		again: rect{40, 194, 116, 26}, menu: rect{164, 194, 116, 26},
	}
}

func (l *layout) levelButton(k int) rect {
	r := l.level0
	r.x += int16(k) * l.levelStep
	return r
}

func backButton() rect { return rect{8, 8, 84, 28} }

type bitmapDrawer interface {
	DrawRGBBitmap8(x, y int16, data []uint8, w, h int16) error
}

// asyncDrawer is spilcd.Device (and viewport.Display over one).
type asyncDrawer interface {
	DrawRGBBitmap8Async(x, y int16, data []uint8, w, h int16) error
	Wait()
}

type screen int

const (
	menuScreen screen = iota
	countdown         // 3, 2, 1: nobody moves
	racing
	finished // the results over the last frame
)

const countFrom = 3 // seconds of countdown

// Game is the kart game.
type Game struct {
	d    hal.Display
	bd   bitmapDrawer
	ad   asyncDrawer // nil: draw at once
	t    hal.Touch
	w, h int16
	l    layout

	// OnExit, if set, shows a "< GAMES" button on the menu; tapping it
	// (or B) calls OnExit (used by the cmd/games launcher).
	OnExit func()

	// Pad, if set, is the game controller: the stick steers, A
	// accelerates, B brakes, Y gives up the race; on the menu and the
	// results the stick moves the focus and A presses. Without it,
	// touching the ground accelerates and the touch steers towards its
	// side of the screen; tapping the minimap gives up.
	Pad      hal.Pad
	in       pad.Input
	firstUse bool // the controller was used for the first time in this frame

	screen   screen
	level    int
	focus    int // menu: a level or 3 for "< GAMES"; results: 0 again, 1 menu
	wasTouch bool

	track *track
	karts [4]kart // the player's first

	countT                               float32
	goT                                  float32 // "GO!" stays a moment
	raceTime, lapStart, lastLap, bestLap float32
	place                                int // at the finish

	bufs    [2][]uint16 // bandRows rows of RGB565, stored so that the bytes are big endian
	bytes   [2][]byte   // bufs as bytes, for DrawRGBBitmap8
	next    int         // the buffer to draw into: the other may be going out
	pal     [16]uint16
	outside uint16
	mini    []uint16 // the minimap without the dots, stored like bufs
	dot     [9]uint16

	shown [4]string // HUD texts on the screen: lap, place, time, countdown

	// Timing of the last second, for the log.
	frames              int
	frameTime, sendTime time.Duration
	secStart            time.Time
	stats               Stats
}

// Stats is the timing of the last second.
type Stats struct {
	FPS         int
	Frame, Send time.Duration // per frame: all, sending to the panel (with DMA: waiting for it)
	Speed, X, Y int
	OnRoad      bool
}

// NewGame makes the course and the karts; call DrawMenu to start. (The
// course, the sprites and the colours are made here, not by package
// initializers: see games/invaders/sprites.go.)
func NewGame(d hal.Display, t hal.Touch) *Game {
	w, h := d.Size()
	bd, _ := d.(bitmapDrawer)
	ad, _ := d.(asyncDrawer)
	g := &Game{d: d, bd: bd, ad: ad, t: t, w: w, h: h, l: layoutFor(w, h), level: 1, focus: 1}
	g.track = newTrack()
	for i := range g.karts {
		g.karts[i].sprite = parseSprite(kartArt[:], kartColors[i])
	}
	for k := range g.bufs {
		g.bufs[k] = make([]uint16, int(w)*bandRows)
		g.bytes[k] = unsafe.Slice((*byte)(unsafe.Pointer(&g.bufs[k][0])), len(g.bufs[k])*2)
	}
	for k, c := range tileColors {
		g.pal[2*k], g.pal[2*k+1] = bigEndian(rgb565(c[0])), bigEndian(rgb565(c[1]))
	}
	g.outside = bigEndian(rgb565(outsideColor))
	g.makeMinimap()
	g.newRace()
	return g
}

// Update reads the controls and runs one frame.
func (g *Game) Update(dt float32) {
	start := time.Now()
	if g.secStart.IsZero() {
		g.secStart = start
	}
	used := g.in.Used
	g.in.Update(g.Pad, dt)
	g.firstUse = g.in.Used && !used
	x, y, pressed := g.t.ReadTouch()
	tap := pressed && !g.wasTouch
	g.wasTouch = pressed

	switch g.screen {
	case menuScreen:
		g.updateMenu(tap, x, y)
	case countdown:
		g.countT -= dt
		if g.countT <= 0 {
			g.screen, g.goT = racing, 0.8
		}
		g.drawCountdown()
	case racing:
		g.race(dt, x, y, pressed, tap)
	case finished:
		g.updateResults(tap, x, y)
	}
	g.d.Display()

	g.frames++
	g.frameTime += time.Since(start)
	if s := time.Since(g.secStart); s >= time.Second {
		n := time.Duration(g.frames)
		p := g.player()
		g.stats = Stats{
			FPS: int(int64(g.frames) * int64(time.Second) / int64(s)), Frame: g.frameTime / n, Send: g.sendTime / n,
			Speed: int(p.speed), X: int(p.x), Y: int(p.y), OnRoad: g.track.at(p.x, p.y) != grass,
		}
		g.frames, g.frameTime, g.sendTime, g.secStart = 0, 0, 0, time.Now()
	}
}

// Stats returns the timing of the last full second.
func (g *Game) Stats() Stats { return g.stats }

// race runs a frame of the race.
func (g *Game) race(dt float32, x, y int16, pressed, tap bool) {
	if g.in.Pressed(hal.PadY) || (tap && g.l.minimap.has(x, y)) {
		g.DrawMenu() // give up
		return
	}
	g.raceTime += dt
	if g.goT > 0 {
		g.goT -= dt
	}
	gas, brakes, steer := g.controls(x, y, pressed)
	p := g.player()
	g.drive(p, dt, gas, brakes, steer)
	for i := 1; i < len(g.karts); i++ {
		g.driveCPU(&g.karts[i], dt)
	}
	g.bump()
	done := g.follow(p)
	g.render()
	g.drawHUD()
	if done {
		g.screen = finished
		g.place = g.placeNow()
		g.focus = 0
		g.drawResults()
	}
}

// controls returns the gas and brake (on or off) and the steering (-1 is
// full left, 1 full right).
func (g *Game) controls(x, y int16, pressed bool) (gas, brakes bool, steer float32) {
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
	if !pressed || y < g.l.horizon {
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

// start begins a race at the current level: the karts on the grid, the
// countdown.
func (g *Game) start() {
	g.newRace()
	g.screen, g.countT = countdown, countFrom
	g.drawRaceScreen()
}

func sincos(a float32) (sin, cos float32) {
	s, c := math.Sincos(float64(a))
	return float32(s), float32(c)
}

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
