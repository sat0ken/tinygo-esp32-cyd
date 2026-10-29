// Package starguard is a space shooting game seen from the cockpit (a
// first prototype): rocks and robot ships come at the ship from far away,
// growing as they come; aim with the stick (or touch) and fire before they
// hit the shield.
//
// The view is drawn like games/kart: a band of rows at a time into one of
// two buffers while the other goes out (with DMA on the ESP32). The
// dashboard below the view is drawn once, its gauges only when they
// change.
//
// It depends only on hal: examples/14_starguard runs it.
package starguard

import (
	"image/color"
	"math"
	"time"
	"unsafe"

	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/pad"
)

type rect struct{ x, y, w, h int16 }

func (r rect) has(x, y int16) bool { return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h }

// layout is where everything is on a screen size.
type layout struct {
	viewH      int16   // the view is rows 0 to viewH
	focal      float32 // pixels; how wide the view is
	aimSpeed   float32 // pixels per second of the crosshair with the stick
	panel      rect    // the dashboard
	shield     rect    // the shield gauge
	scoreX     int16
	scoreBase  int16
	radar      rect
	labelsBase int16 // "SHIELD", "SCORE"
}

// layoutFor returns the layout for a w x h screen: 480x272 and larger, or
// 320x240 (ESP32-2432S028R).
func layoutFor(w, h int16) layout {
	if w >= 480 && h >= 272 {
		return layout{
			viewH: 200, focal: 300, aimSpeed: 300,
			panel: rect{0, 200, w, h - 200}, shield: rect{14, 232, 170, 16},
			scoreX: 200, scoreBase: 250, radar: rect{w - 84, 204, 78, 64}, labelsBase: 220,
		}
	}
	return layout{
		viewH: 176, focal: 200, aimSpeed: 220,
		panel: rect{0, 176, w, h - 176}, shield: rect{10, 204, 116, 14},
		scoreX: 138, scoreBase: 220, radar: rect{w - 66, 180, 60, 56}, labelsBase: 194,
	}
}

type bitmapDrawer interface {
	DrawRGBBitmap8(x, y int16, data []uint8, w, h int16) error
}

// asyncDrawer is spilcd.Device (and viewport.Display over one).
type asyncDrawer interface {
	DrawRGBBitmap8Async(x, y int16, data []uint8, w, h int16) error
	Wait()
}

// Game is the space shooting game.
type Game struct {
	d    hal.Display
	bd   bitmapDrawer
	ad   asyncDrawer // nil: draw at once
	t    hal.Touch
	w, h int16
	l    layout

	// Pad, if set, is the game controller: the stick moves the crosshair,
	// A fires (held: again and again). Without it, touching the view aims
	// there and fires.
	Pad hal.Pad
	in  pad.Input

	rng                 uint32
	stars               [numStars]star
	enemies             [maxEnemies]enemy
	aimX, aimY          float32
	shield, score       int
	spawnT              float32
	fireT, laser, flash float32
	over                bool

	rock, robot *sprite
	boom        [3]*sprite
	dots        [numStars]dot
	shown       [maxEnemies]shown

	bufs                                          [2][]uint16 // bandRows rows of RGB565, stored so that the bytes are big endian
	bytes                                         [2][]byte   // bufs as bytes, for DrawRGBBitmap8
	next                                          int         // the buffer to draw into: the other may be going out
	starColors                                    [3]uint16
	crossColor, lockColor, laserColor, flashColor uint16
	radarBuf                                      []uint16
	shownScore, shownShield                       int

	// Timing of the last second, for the log.
	frames              int
	frameTime, sendTime time.Duration
	secStart            time.Time
	stats               Stats
}

// Stats is the timing of the last second.
type Stats struct {
	FPS         int
	Frame, Send time.Duration // per frame: all, waiting for the panel
	Enemies     int
	Score       int
}

// NewGame makes a game; seed makes it repeatable. (The sprites and
// colours are made here, not by package initializers: see
// games/invaders/sprites.go.)
func NewGame(d hal.Display, t hal.Touch, seed uint32) *Game {
	w, h := d.Size()
	bd, _ := d.(bitmapDrawer)
	ad, _ := d.(asyncDrawer)
	if seed == 0 {
		seed = 1
	}
	g := &Game{d: d, bd: bd, ad: ad, t: t, w: w, h: h, l: layoutFor(w, h), rng: seed}
	g.rock = parseSprite(rockArt[:])
	g.robot = parseSprite(robotArt[:])
	for i := range g.boom {
		g.boom[i] = parseSprite(boomArt[i][:])
	}
	for k := range g.bufs {
		g.bufs[k] = make([]uint16, int(w)*bandRows)
		g.bytes[k] = unsafe.Slice((*byte)(unsafe.Pointer(&g.bufs[k][0])), len(g.bufs[k])*2)
	}
	g.starColors = [3]uint16{c565(90, 90, 120), c565(170, 170, 200), c565(255, 255, 255)}
	g.crossColor, g.lockColor = c565(80, 255, 120), c565(255, 60, 60)
	g.laserColor, g.flashColor = c565(80, 220, 255), c565(220, 30, 30)
	g.radarBuf = make([]uint16, int(g.l.radar.w)*int(g.l.radar.h))
	g.reset()
	return g
}

// reset starts again: full shield, no enemies, the stars all over.
func (g *Game) reset() {
	for i := range g.stars {
		g.newStar(&g.stars[i], g.randf(20, starFar))
	}
	g.enemies = [maxEnemies]enemy{}
	g.aimX, g.aimY = float32(g.w)/2, float32(g.l.viewH)/2
	g.shield, g.score = shield0, 0
	g.spawnT, g.fireT, g.laser, g.flash = spawnEvery-0.3, 0, 0, 0
	g.over = false
	g.shownScore, g.shownShield = -1, -1
}

// DrawAll draws the dashboard and the view.
func (g *Game) DrawAll() {
	g.drawPanel()
	g.render()
	g.drawGauges()
	g.d.Display()
}

// Update reads the controls and runs one frame.
func (g *Game) Update(dt float32) {
	start := time.Now()
	if g.secStart.IsZero() {
		g.secStart = start
	}
	g.in.Update(g.Pad, dt)
	x, y, pressed := g.t.ReadTouch()
	if g.over {
		if g.in.Pressed(hal.PadA|hal.PadOK) || (pressed && y < g.l.viewH) {
			g.reset()
			g.DrawAll()
		}
		return
	}
	trigger := g.aim(dt, x, y, pressed)
	g.fireT -= dt
	if trigger && g.fireT <= 0 {
		g.fire()
		g.fireT = fireEvery
	}
	g.moveWorld(dt)
	if g.laser > 0 {
		g.laser -= dt
	}
	if g.flash > 0 {
		g.flash -= dt
	}
	g.render()
	g.drawGauges()
	if g.shield <= 0 {
		g.shield = 0
		g.over = true
		g.drawGauges()
		g.drawGameOver()
	}
	g.d.Display()

	g.frames++
	g.frameTime += time.Since(start)
	if s := time.Since(g.secStart); s >= time.Second {
		n := time.Duration(g.frames)
		alive := 0
		for i := range g.enemies {
			if g.enemies[i].alive {
				alive++
			}
		}
		g.stats = Stats{
			FPS: int(int64(g.frames) * int64(time.Second) / int64(s)), Frame: g.frameTime / n, Send: g.sendTime / n,
			Enemies: alive, Score: g.score,
		}
		g.frames, g.frameTime, g.sendTime, g.secStart = 0, 0, 0, time.Now()
	}
}

// Stats returns the timing of the last full second.
func (g *Game) Stats() Stats { return g.stats }

// aim moves the crosshair and returns whether the trigger is pulled.
func (g *Game) aim(dt float32, x, y int16, pressed bool) bool {
	if pressed && y < g.l.viewH {
		g.aimX, g.aimY = float32(x), float32(y)
		return true
	}
	if !g.in.Used {
		return false
	}
	v := g.l.aimSpeed * dt
	switch {
	case g.in.Held(hal.PadLeft):
		g.aimX -= v
	case g.in.Held(hal.PadRight):
		g.aimX += v
	}
	switch {
	case g.in.Held(hal.PadUp):
		g.aimY -= v
	case g.in.Held(hal.PadDown):
		g.aimY += v
	}
	g.aimX = clamp(g.aimX, 0, float32(g.w-1))
	g.aimY = clamp(g.aimY, 0, float32(g.l.viewH-1))
	return g.in.Held(hal.PadA | hal.PadOK)
}

func sinf(a float32) float32 { return float32(math.Sin(float64(a))) }

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

// c565 is an RGB colour stored like bufs.
func c565(r, g, b uint8) uint16 { return bigEndian(rgb565(color.RGBA{r, g, b, 255})) }
