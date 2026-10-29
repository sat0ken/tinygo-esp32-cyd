// Package starguard is a space shooting game seen from the cockpit: rocks
// and robot ships come at the ship from far away, growing as they come;
// aim with the stick (or touch) and fire before they hit the shield.
// Three stages, then the mothership. Shooting a capsule gives shield back.
//
// The view is drawn like games/kart: a band of rows at a time into one of
// two buffers while the other goes out (with DMA on the ESP32). The
// dashboard below the view is drawn once, its gauges only when they
// change.
//
// It depends only on hal: examples/14_starguard runs it alone, cmd/games
// from a launcher.
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

func (r rect) box() pad.Box { return pad.Box{X: r.x, Y: r.y, W: r.w, H: r.h} }

// layout is where everything is on a screen size.
type layout struct {
	viewH      int16   // the view is rows 0 to viewH
	focal      float32 // pixels; how wide the view is
	aimSpeed   float32 // pixels per second of the crosshair with the stick
	panel      rect    // the dashboard
	shield     rect    // the shield gauge
	scoreX     int16
	scoreBase  int16
	stageBase  int16 // the stage under the shield gauge
	radar      rect
	labelsBase int16 // "SHIELD", "SCORE"

	titleBase, subBase, hintBase int16 // the title screen
	level0                       rect  // the first level button
	levelStep                    int16
	box                          rect // the results
	again, menu                  rect // in the results
}

// layoutFor returns the layout for a w x h screen: 480x272 and larger, or
// 320x240 (ESP32-2432S028R).
func layoutFor(w, h int16) layout {
	if w >= 480 && h >= 272 {
		return layout{
			viewH: 200, focal: 300, aimSpeed: 300,
			panel: rect{0, 200, w, h - 200}, shield: rect{14, 228, 170, 16},
			scoreX: 200, scoreBase: 250, stageBase: 264, radar: rect{w - 84, 204, 78, 64}, labelsBase: 218,
			titleBase: 66, subBase: 88, hintBase: 186,
			level0: rect{52, 108, 120, 50}, levelStep: 128,
			box:   rect{110, 32, 260, 136},
			again: rect{122, 128, 116, 30}, menu: rect{242, 128, 116, 30},
		}
	}
	return layout{
		viewH: 176, focal: 200, aimSpeed: 220,
		panel: rect{0, 176, w, h - 176}, shield: rect{10, 200, 116, 14},
		scoreX: 138, scoreBase: 220, stageBase: 232, radar: rect{w - 66, 180, 60, 56}, labelsBase: 192,
		titleBase: 64, subBase: 82, hintBase: 164, // the title clears "< GAMES"
		level0: rect{14, 100, 92, 44}, levelStep: 100,
		box:   rect{30, 24, 260, 128},
		again: rect{40, 116, 116, 28}, menu: rect{164, 116, 116, 28},
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
	intro             // "STAGE 2": nothing moves
	playing
	cleared // "STAGE CLEAR": nothing moves
	results
)

const (
	introTime   = 1.6
	clearedTime = 2
)

// Game is the space shooting game.
type Game struct {
	d    hal.Display
	bd   bitmapDrawer
	ad   asyncDrawer // nil: draw at once
	t    hal.Touch
	w, h int16
	l    layout

	// OnExit, if set, shows a "< GAMES" button on the title screen;
	// tapping it (or B) calls OnExit (used by the cmd/games launcher).
	OnExit func()

	// Pad, if set, is the game controller: the stick moves the crosshair,
	// A fires (held: again and again), Y gives up; on the title and the
	// results the stick moves the focus and A presses. Without it,
	// touching the view aims there and fires; tapping the radar gives up.
	Pad      hal.Pad
	in       pad.Input
	firstUse bool // the controller was used for the first time in this frame

	screen   screen
	level    int
	focus    int // title: a level or 3 for "< GAMES"; results: 0 again, 1 menu
	wasTouch bool
	timer    float32
	complete bool // the results are a win

	rng                 uint32
	stars               [numStars]star
	enemies             [maxEnemies]enemy
	boss                mothership
	stage, killed       int
	aimX, aimY          float32
	shield, score       int
	spawnT              float32
	fireT, laser, flash float32

	kinds  [4]*sprite // by kind
	mother *sprite
	boom   [3]*sprite
	dots   [numStars]dot
	shown  [maxEnemies + 1]shown
	nShown int

	bufs                                          [2][]uint16 // bandRows rows of RGB565, stored so that the bytes are big endian
	bytes                                         [2][]byte   // bufs as bytes, for DrawRGBBitmap8
	next                                          int         // the buffer to draw into: the other may be going out
	starColors                                    [3]uint16
	crossColor, lockColor, laserColor, flashColor uint16
	gaugeEdge, gaugeFill                          uint16
	radarBuf                                      []uint16
	shownScore, shownShield                       int
	shownStage                                    int // stage*1000 + killed, -1: none

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

// NewGame makes a game; seed makes it repeatable. Call DrawMenu to start.
// (The sprites and colours are made here, not by package initializers:
// see games/invaders/sprites.go.)
func NewGame(d hal.Display, t hal.Touch, seed uint32) *Game {
	w, h := d.Size()
	bd, _ := d.(bitmapDrawer)
	ad, _ := d.(asyncDrawer)
	if seed == 0 {
		seed = 1
	}
	g := &Game{d: d, bd: bd, ad: ad, t: t, w: w, h: h, l: layoutFor(w, h), rng: seed, level: 1, focus: 1}
	g.kinds = [4]*sprite{rock: parseSprite(rockArt[:]), robot: parseSprite(robotArt[:]),
		drone: parseSprite(droneArt[:]), capsule: parseSprite(capsuleArt[:])}
	g.mother = parseSprite(motherArt[:])
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
	g.gaugeEdge, g.gaugeFill = c565(220, 220, 230), c565(230, 50, 60)
	g.radarBuf = make([]uint16, int(g.l.radar.w)*int(g.l.radar.h))
	g.reset()
	return g
}

// reset starts a new game: full shield, no enemies, the stars all over.
func (g *Game) reset() {
	for i := range g.stars {
		g.newStar(&g.stars[i], g.randf(20, starFar))
	}
	g.enemies = [maxEnemies]enemy{}
	g.boss = mothership{}
	g.aimX, g.aimY = float32(g.w)/2, float32(g.l.viewH)/2
	g.shield, g.score, g.stage, g.killed = shield0, 0, 0, 0
	g.spawnT, g.fireT, g.laser, g.flash = 0, 0, 0, 0
	g.shownScore, g.shownShield, g.shownStage = -1, -1, -1
}

// start begins a new game at the chosen level.
func (g *Game) start() {
	g.reset()
	g.drawPanel()
	g.startStage(0)
}

// startStage shows "STAGE n" and clears the field for stage n (the
// mothership after the last one).
func (g *Game) startStage(n int) {
	g.stage, g.killed = n, 0
	g.enemies = [maxEnemies]enemy{}
	g.spawnT = 0
	if n == bossStage {
		hp := levels[g.level].BossHP
		g.boss = mothership{active: true, z: spawnFar, hp: hp, max: hp}
	}
	g.screen, g.timer = intro, introTime
	g.laser, g.flash, g.fireT = 0, 0, 0
	g.render()
	g.drawGauges()
	if n == bossStage {
		g.drawNotice("MOTHERSHIP", "SHOOT IT DOWN!")
	} else {
		g.drawNotice(stageName(n), "GET READY")
	}
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
	case intro:
		if g.timer -= dt; g.timer <= 0 {
			g.screen = playing
		}
	case cleared:
		if g.timer -= dt; g.timer <= 0 {
			g.startStage(g.stage + 1)
		}
	case playing:
		g.play(dt, x, y, pressed, tap)
	case results:
		g.updateResults(tap, x, y)
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

// play runs a frame of a stage.
func (g *Game) play(dt float32, x, y int16, pressed, tap bool) {
	if g.in.Pressed(hal.PadY) || (tap && g.l.radar.has(x, y)) {
		g.DrawMenu() // give up
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
	switch {
	case g.shield <= 0:
		g.shield = 0
		g.drawGauges()
		g.showResults(false)
	case g.stage == bossStage && !g.boss.active:
		g.score += g.shield * 5 // what is left of the shield
		g.drawGauges()
		g.showResults(true)
	case g.stage < bossStage && g.killed >= stages[g.stage].count:
		g.score += 100 * (g.stage + 1)
		g.shield += clearBoost
		if g.shield > shield0 {
			g.shield = shield0
		}
		g.drawGauges()
		g.screen, g.timer = cleared, clearedTime
		g.drawNotice("STAGE CLEAR!", "SHIELD +20")
	}
}

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
