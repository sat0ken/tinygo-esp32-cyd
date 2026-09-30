package main

import (
	"image/color"
	"strconv"

	"github.com/sat0ken/tinygo-cyd/hal"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/proggy"
)

// Layout for 480x272.
const (
	hudH = 20

	cols   = 11
	rows   = 5
	cellW  = 32 // horizontal spacing of the formation
	cellH  = 24 // vertical spacing
	invW   = 24 // widest invader (12 bits x 2)
	invH   = 16
	formX0 = 64 // left edge of the first column
	formY0 = 44 // top of the first row at wave 1
	stepX  = 4  // horizontal move of one invader
	dropY  = 8  // move down at the edges
	edge   = 4  // margin to the screen edge

	ufoY = 24

	bunkers    = 4
	bunkerY    = 190
	bunkerCell = 4 // pixels per bunker cell

	bulletSpeed = 360 // pixels per second
	bombSpeed   = 120
	maxBombs    = 3
	ufoSpeed    = 70

	lives0        = 3
	movesPerSec   = 60 // invaders moved per second (one at a time)
	dyingTime     = 1.0
	clearTime     = 1.5
	explosionTime = 0.25
)

var (
	bgColor     = color.RGBA{0, 0, 0, 255}
	hudColor    = color.RGBA{30, 30, 60, 255}
	textColor   = color.RGBA{255, 255, 255, 255}
	msgColor    = color.RGBA{255, 220, 60, 255}
	groundColor = color.RGBA{60, 220, 60, 255}
	cannonColor = color.RGBA{60, 255, 60, 255}
	bunkerColor = color.RGBA{60, 220, 60, 255}
	shotColor   = color.RGBA{255, 255, 255, 255}
	ufoColor    = color.RGBA{255, 60, 60, 255}
	explColor   = color.RGBA{255, 255, 255, 255}
	kindColors  = [3]color.RGBA{
		{255, 120, 220, 255}, // squid
		{80, 220, 255, 255},  // crab
		{160, 255, 100, 255}, // octopus
	}
	kindPoints = [3]int{30, 20, 10}
	rowKinds   = [rows]int{0, 1, 1, 2, 2}
	ufoPoints  = [4]int{50, 100, 150, 300}
)

type state int

const (
	ready    state = iota // waiting for a tap
	playing               //
	dying                 // the cannon was hit
	cleared               // wave cleared, next one starts shortly
	gameOver              // tap to play again
)

type rect struct{ x, y, w, h int16 }

func (a rect) overlaps(b rect) bool {
	return a.x < b.x+b.w && a.x+a.w > b.x && a.y < b.y+b.h && a.y+a.h > b.y
}

func union(a, b rect) rect {
	x0, y0 := min16(a.x, b.x), min16(a.y, b.y)
	x1, y1 := max16(a.x+a.w, b.x+b.w), max16(a.y+a.h, b.y+b.h)
	return rect{x0, y0, x1 - x0, y1 - y0}
}

type invader struct {
	alive bool
	kind  int
	x, y  int16 // top-left of the invW x invH cell
	frame int
}

type shot struct {
	active bool
	x, y   float32
	drawn  rect
	shown  bool
}

type explosion struct {
	r     rect
	until float32
}

// bitmapDrawer is implemented by every backend (framebuf.Buffer).
type bitmapDrawer interface {
	DrawRGBBitmap8(x, y int16, data []uint8, w, h int16) error
}

// Game is a space invaders game. It depends only on hal, so it runs on the
// board, in the browser and in host tests. Update both advances the game
// and redraws what changed.
type Game struct {
	d    hal.Display
	bd   bitmapDrawer
	t    hal.Touch
	w, h int16

	state  state
	stateT float32 // clock when the state was entered
	now    float32 // game clock in seconds

	inv      [rows][cols]invader
	alive    int
	dx       int16
	hitEdge  bool
	dropping bool
	moveIdx  int
	moveAcc  float32

	cannonX     int16
	drawnCannon int16 // -1: not on screen

	bullet shot
	bombs  [maxBombs]shot
	bunker [bunkers][8][11]bool

	ufoOn    bool
	ufoX     float32
	ufoDir   float32
	ufoDrawn rect
	ufoShown bool
	nextUFO  float32

	expl []explosion

	score, hi, lives, wave int

	hudDirty bool
	msgShown string
	wasTouch bool
	rng      uint32
	buf      []byte
}

// NewGame creates a game. seed makes the invaders' shots repeatable.
func NewGame(d hal.Display, t hal.Touch, seed uint32) *Game {
	bd, _ := d.(bitmapDrawer)
	w, h := d.Size()
	if seed == 0 {
		seed = 1
	}
	g := &Game{d: d, bd: bd, t: t, w: w, h: h, rng: seed, buf: make([]byte, 64*32*2)}
	g.reset()
	return g
}

func (g *Game) rand() uint32 {
	// xorshift32
	g.rng ^= g.rng << 13
	g.rng ^= g.rng >> 17
	g.rng ^= g.rng << 5
	return g.rng
}

func (g *Game) groundY() int16 { return g.h - 4 }
func (g *Game) cannonY() int16 { return g.groundY() - 2 - cannonSprite.ph() }

func (g *Game) reset() {
	g.score, g.lives, g.wave = 0, lives0, 1
	g.newWave()
	g.setState(ready)
}

func (g *Game) setState(s state) {
	g.state = s
	g.stateT = g.now
}

func (g *Game) newWave() {
	down := int16(g.wave-1) * dropY
	if down > 4*dropY {
		down = 4 * dropY
	}
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			g.inv[r][c] = invader{
				alive: true,
				kind:  rowKinds[r],
				x:     formX0 + int16(c)*cellW,
				y:     formY0 + down + int16(r)*cellH,
			}
		}
	}
	g.alive = rows * cols
	g.dx, g.hitEdge, g.dropping, g.moveIdx, g.moveAcc = stepX, false, false, 0, 0
	for i := range g.bunker {
		for y, row := range bunkerShape {
			for x, c := range row {
				g.bunker[i][y][x] = c == '#'
			}
		}
	}
	g.bullet = shot{}
	g.bombs = [maxBombs]shot{}
	g.ufoOn, g.ufoShown = false, false
	g.nextUFO = g.now + 15
	g.expl = g.expl[:0]
	g.cannonX = (g.w - cannonSprite.pw()) / 2
}

// ---- drawing helpers ----

// blit draws area with the background colour and sp (if not nil) at
// (sx, sy), in one transfer, so moving sprites do not flicker.
func (g *Game) blit(sp *sprite, sx, sy int16, area rect, fg color.RGBA) {
	// Clip to the screen.
	if area.x < 0 {
		area.w += area.x
		area.x = 0
	}
	if area.y < 0 {
		area.h += area.y
		area.y = 0
	}
	if area.x+area.w > g.w {
		area.w = g.w - area.x
	}
	if area.y+area.h > g.h {
		area.h = g.h - area.y
	}
	if area.w <= 0 || area.h <= 0 {
		return
	}
	if g.bd == nil || int(area.w)*int(area.h)*2 > len(g.buf) {
		// Too large for one transfer: clear, then draw the sprite alone.
		g.d.FillRectangle(area.x, area.y, area.w, area.h, bgColor)
		if sp != nil {
			sr := rect{sx, sy, sp.pw(), sp.ph()}
			if int(sr.w)*int(sr.h)*2 <= len(g.buf) && g.bd != nil {
				g.blit(sp, sx, sy, sr, fg)
			}
		}
		return
	}
	bg := rgb565(bgColor)
	fc := rgb565(fg)
	buf := g.buf[:int(area.w)*int(area.h)*2]
	i := 0
	for y := area.y; y < area.y+area.h; y++ {
		for x := area.x; x < area.x+area.w; x++ {
			v := bg
			if sp != nil {
				bx, by := (x-sx)/scale, (y-sy)/scale
				if x >= sx && y >= sy && bx < sp.w && by < sp.h && sp.on(bx, by) {
					v = fc
				}
			}
			buf[i], buf[i+1] = byte(v>>8), byte(v)
			i += 2
		}
	}
	g.bd.DrawRGBBitmap8(area.x, area.y, buf, area.w, area.h)
}

func rgb565(c color.RGBA) uint16 {
	return uint16(c.R&0xF8)<<8 | uint16(c.G&0xFC)<<3 | uint16(c.B)>>3
}

func (g *Game) invSprite(v *invader) *sprite { return &invaderSprites[v.kind][v.frame] }

// spriteRect is where the sprite of v is (centred in its cell).
func (g *Game) spriteRect(v *invader) rect {
	sp := g.invSprite(v)
	return rect{v.x + (invW-sp.pw())/2, v.y, sp.pw(), sp.ph()}
}

func (g *Game) drawInvader(v *invader, area rect) {
	sr := g.spriteRect(v)
	g.blit(g.invSprite(v), sr.x, sr.y, area, kindColors[v.kind])
}

// redrawInvadersIn redraws the living invaders that overlap r (after r
// was cleared).
func (g *Game) redrawInvadersIn(r rect) {
	for i := range g.inv {
		for j := range g.inv[i] {
			v := &g.inv[i][j]
			if v.alive {
				if sr := g.spriteRect(v); sr.overlaps(r) {
					g.drawInvader(v, sr)
				}
			}
		}
	}
}

func (g *Game) bunkerX(i int) int16 {
	return g.w*int16(i+1)/(bunkers+1) - int16(len(bunkerShape[0]))*bunkerCell/2
}

func (g *Game) cellRect(i, y, x int) rect {
	return rect{g.bunkerX(i) + int16(x)*bunkerCell, bunkerY + int16(y)*bunkerCell, bunkerCell, bunkerCell}
}

// DrawAll draws the whole screen.
func (g *Game) DrawAll() {
	g.d.FillRectangle(0, 0, g.w, g.h, bgColor)
	g.d.FillRectangle(0, g.groundY(), g.w, 2, groundColor)
	for i := range g.bunker {
		for y := range g.bunker[i] {
			for x, on := range g.bunker[i][y] {
				if on {
					c := g.cellRect(i, y, x)
					g.d.FillRectangle(c.x, c.y, c.w, c.h, bunkerColor)
				}
			}
		}
	}
	for i := range g.inv {
		for j := range g.inv[i] {
			if v := &g.inv[i][j]; v.alive {
				g.drawInvader(v, g.spriteRect(v))
			}
		}
	}
	g.drawnCannon = -1
	g.bullet.shown = false
	for i := range g.bombs {
		g.bombs[i].shown = false
	}
	g.ufoShown = false
	g.hudDirty = true
	g.msgShown = ""
	g.drawDynamic()
	g.d.Display()
}

// ---- game logic ----

// Update reads the touch panel, advances the game by dt seconds and
// redraws what changed.
func (g *Game) Update(dt float32) {
	g.now += dt
	x, _, pressed := g.t.ReadTouch()
	tap := pressed && !g.wasTouch
	g.wasTouch = pressed

	if pressed && (g.state == ready || g.state == playing) {
		cx := x - cannonSprite.pw()/2
		if cx < 0 {
			cx = 0
		}
		if max := g.w - cannonSprite.pw(); cx > max {
			cx = max
		}
		g.cannonX = cx
	}

	switch g.state {
	case ready:
		if tap {
			g.setState(playing)
		}
	case playing:
		if pressed && !g.bullet.active {
			g.fire()
		}
		g.moveBullet(dt)
		g.dropBomb()
		g.moveBombs(dt)
		g.moveUFO(dt)
		g.moveAcc += dt * movesPerSec
		for g.moveAcc >= 1 && g.state == playing {
			g.moveAcc--
			g.stepInvader()
		}
	case dying:
		if g.now-g.stateT >= dyingTime {
			g.d.FillRectangle(g.cannonX, g.cannonY(), cannonSprite.pw(), cannonSprite.ph(), bgColor)
			g.drawnCannon = -1
			if g.lives == 0 {
				g.setState(gameOver)
			} else {
				g.setState(playing)
			}
		}
	case cleared:
		if g.now-g.stateT >= clearTime {
			g.wave++
			g.newWave()
			g.setState(playing)
			g.DrawAll()
		}
	case gameOver:
		if tap {
			g.reset()
			g.DrawAll()
		}
	}
	g.updateExplosions()
	if g.score > g.hi {
		g.hi = g.score
	}
	g.drawDynamic()
	g.d.Display()
}

func (g *Game) fire() {
	g.bullet.active = true
	g.bullet.x = float32(g.cannonX + cannonSprite.pw()/2 - 1)
	g.bullet.y = float32(g.cannonY() - 8)
}

func bulletRect(s *shot) rect { return rect{int16(s.x), int16(s.y), 2, 8} }

// moveBullet moves the cannon's shot up in steps of 4 pixels, so it
// cannot skip over anything.
func (g *Game) moveBullet(dt float32) {
	b := &g.bullet
	for dist := bulletSpeed * dt; b.active && dist > 0; dist -= 4 {
		step := float32(4)
		if dist < step {
			step = dist
		}
		b.y -= step
		if b.y < hudH+2 {
			b.active = false
			return
		}
		br := bulletRect(b)
		if g.ufoOn {
			ur := rect{int16(g.ufoX), ufoY, ufoSprite.pw(), ufoSprite.ph()}
			if br.overlaps(ur) {
				b.active = false
				g.ufoOn = false
				g.score += ufoPoints[g.rand()%4]
				g.hudDirty = true
				g.explode(ur)
				g.ufoShown = false
				g.nextUFO = g.now + 20 + float32(g.rand()%10)
				return
			}
		}
		for i := range g.inv {
			for j := range g.inv[i] {
				v := &g.inv[i][j]
				if v.alive && br.overlaps(g.spriteRect(v)) {
					b.active = false
					g.kill(v)
					return
				}
			}
		}
		if g.hitBunker(br) {
			b.active = false
			return
		}
		for k := range g.bombs {
			if bm := &g.bombs[k]; bm.active && br.overlaps(bulletRect(bm)) {
				b.active, bm.active = false, false
				return
			}
		}
	}
}

func (g *Game) kill(v *invader) {
	v.alive = false
	g.alive--
	g.score += kindPoints[v.kind]
	g.hudDirty = true
	g.explode(rect{v.x, v.y, invW, invH})
	if g.alive == 0 {
		g.setState(cleared)
	}
}

// explode clears r and shows the explosion sprite centred in it.
func (g *Game) explode(r rect) {
	ex := r.x + (r.w-explosionSprite.pw())/2
	ey := r.y + (r.h-explosionSprite.ph())/2
	area := union(r, rect{ex, ey, explosionSprite.pw(), explosionSprite.ph()})
	g.blit(&explosionSprite, ex, ey, area, explColor)
	g.expl = append(g.expl, explosion{r: area, until: g.now + explosionTime})
}

func (g *Game) updateExplosions() {
	n := 0
	for _, e := range g.expl {
		if g.now < e.until {
			g.expl[n] = e
			n++
			continue
		}
		g.d.FillRectangle(e.r.x, e.r.y, e.r.w, e.r.h, bgColor)
		g.redrawInvadersIn(e.r)
	}
	g.expl = g.expl[:n]
}

// hitBunker removes the first bunker cell r touches.
func (g *Game) hitBunker(r rect) bool {
	if r.y+r.h <= bunkerY || r.y >= bunkerY+8*bunkerCell {
		return false
	}
	for i := range g.bunker {
		for y := range g.bunker[i] {
			for x, on := range g.bunker[i][y] {
				if on {
					if c := g.cellRect(i, y, x); c.overlaps(r) {
						g.bunker[i][y][x] = false
						g.d.FillRectangle(c.x, c.y, c.w, c.h, bgColor)
						return true
					}
				}
			}
		}
	}
	return false
}

// erodeBunkers removes every cell an invader overlaps.
func (g *Game) erodeBunkers(r rect) {
	for g.hitBunker(r) {
	}
}

func (g *Game) dropBomb() {
	n := 0
	for i := range g.bombs {
		if g.bombs[i].active {
			n++
		}
	}
	if n >= maxBombs || g.rand()%30 != 0 {
		return
	}
	// A random column that still has invaders; its lowest one shoots.
	start := int(g.rand() % cols)
	for k := 0; k < cols; k++ {
		c := (start + k) % cols
		for r := rows - 1; r >= 0; r-- {
			v := &g.inv[r][c]
			if !v.alive {
				continue
			}
			for i := range g.bombs {
				if !g.bombs[i].active {
					g.bombs[i] = shot{active: true, x: float32(v.x + invW/2 - 1), y: float32(v.y + invH),
						drawn: g.bombs[i].drawn, shown: g.bombs[i].shown}
					return
				}
			}
			return
		}
	}
}

func (g *Game) moveBombs(dt float32) {
	cr := rect{g.cannonX, g.cannonY(), cannonSprite.pw(), cannonSprite.ph()}
	for i := range g.bombs {
		b := &g.bombs[i]
		for dist := bombSpeed * dt; b.active && dist > 0; dist -= 4 {
			step := float32(4)
			if dist < step {
				step = dist
			}
			b.y += step
			br := bulletRect(b)
			if br.y+br.h >= g.groundY() {
				b.active = false
			} else if g.hitBunker(br) {
				b.active = false
			} else if br.overlaps(cr) && g.state == playing {
				b.active = false
				g.cannonHit()
			}
		}
	}
}

func (g *Game) cannonHit() {
	g.lives--
	g.hudDirty = true
	for i := range g.bombs {
		g.bombs[i].active = false
	}
	g.bullet.active = false
	g.blit(&explosionSprite, g.cannonX, g.cannonY(),
		rect{g.cannonX, g.cannonY(), cannonSprite.pw(), cannonSprite.ph()}, cannonColor)
	g.drawnCannon = -2 // the explosion is there now
	g.setState(dying)
}

func (g *Game) moveUFO(dt float32) {
	if !g.ufoOn {
		if g.now >= g.nextUFO {
			g.ufoOn = true
			if g.rand()%2 == 0 {
				g.ufoDir, g.ufoX = 1, -float32(ufoSprite.pw())
			} else {
				g.ufoDir, g.ufoX = -1, float32(g.w)
			}
		}
		return
	}
	g.ufoX += g.ufoDir * ufoSpeed * dt
	if g.ufoX > float32(g.w) || g.ufoX < -float32(ufoSprite.pw()) {
		g.ufoOn = false
		g.nextUFO = g.now + 20 + float32(g.rand()%10)
	}
}

// stepInvader moves one invader, like the original game: the formation
// moves one invader per frame, from the bottom left, so it speeds up as
// invaders are destroyed.
func (g *Game) stepInvader() {
	const total = rows * cols
	if g.moveIdx == 0 && g.hitEdge {
		g.hitEdge = false
		g.dropping = true
		g.dx = -g.dx
	}
	for g.moveIdx < total && !g.invAt(g.moveIdx).alive {
		g.moveIdx++
	}
	if g.moveIdx >= total {
		g.moveIdx, g.dropping = 0, false
		return
	}
	v := g.invAt(g.moveIdx)
	old := g.spriteRect(v)
	if g.dropping {
		v.y += dropY
	} else {
		v.x += g.dx
	}
	v.frame ^= 1
	nr := g.spriteRect(v)
	g.drawInvader(v, union(old, nr))

	if !g.dropping && ((g.dx > 0 && v.x+invW+g.dx > g.w-edge) || (g.dx < 0 && v.x+g.dx < edge)) {
		g.hitEdge = true
	}
	g.erodeBunkers(nr)
	if v.y+invH >= g.cannonY() {
		// Invaded: game over at once.
		g.lives = 0
		g.hudDirty = true
		g.setState(gameOver)
	}
	g.moveIdx++
	if g.moveIdx >= total {
		g.moveIdx, g.dropping = 0, false
	}
}

// invAt maps the move order (bottom row first, left to right) to an invader.
func (g *Game) invAt(i int) *invader { return &g.inv[rows-1-i/cols][i%cols] }

// ---- per-frame drawing ----

func (g *Game) drawDynamic() {
	g.drawCannon()
	g.drawShot(&g.bullet, shotColor)
	for i := range g.bombs {
		g.drawShot(&g.bombs[i], shotColor)
	}
	g.drawUFO()
	if g.hudDirty {
		g.hudDirty = false
		g.d.FillRectangle(0, 0, g.w, hudH, hudColor)
		text := "SCORE " + pad(g.score) + "   HI " + pad(g.hi) +
			"   LIVES " + strconv.Itoa(g.lives) + "   WAVE " + strconv.Itoa(g.wave)
		tinyfont.WriteLine(g.d, &proggy.TinySZ8pt7b, 8, 14, text, textColor)
	}
	g.drawMessage()
}

func pad(n int) string {
	s := strconv.Itoa(n)
	for len(s) < 5 {
		s = "0" + s
	}
	return s
}

func (g *Game) drawCannon() {
	if g.state == dying || g.state == gameOver {
		return
	}
	x := g.cannonX
	if x == g.drawnCannon {
		return
	}
	y := g.cannonY()
	nr := rect{x, y, cannonSprite.pw(), cannonSprite.ph()}
	if g.drawnCannon >= 0 {
		old := rect{g.drawnCannon, y, nr.w, nr.h}
		if u := union(old, nr); int(u.w)*int(u.h)*2 <= len(g.buf) {
			g.blit(&cannonSprite, x, y, u, cannonColor)
			g.drawnCannon = x
			return
		}
		g.d.FillRectangle(old.x, old.y, old.w, old.h, bgColor)
	}
	g.blit(&cannonSprite, x, y, nr, cannonColor)
	g.drawnCannon = x
}

func (g *Game) drawShot(s *shot, c color.RGBA) {
	nr := bulletRect(s)
	if s.shown && (!s.active || nr != s.drawn) {
		o := s.drawn
		g.d.FillRectangle(o.x, o.y, o.w, o.h, bgColor)
		s.shown = false
	}
	if s.active && !s.shown {
		g.d.FillRectangle(nr.x, nr.y, nr.w, nr.h, c)
		s.drawn, s.shown = nr, true
	}
}

func (g *Game) drawUFO() {
	nr := rect{int16(g.ufoX), ufoY, ufoSprite.pw(), ufoSprite.ph()}
	if !g.ufoOn {
		if g.ufoShown {
			g.d.FillRectangle(g.ufoDrawn.x, g.ufoDrawn.y, g.ufoDrawn.w, g.ufoDrawn.h, bgColor)
			g.ufoShown = false
		}
		return
	}
	if g.ufoShown && nr == g.ufoDrawn {
		return
	}
	area := nr
	if g.ufoShown {
		area = union(g.ufoDrawn, nr)
	}
	g.blit(&ufoSprite, nr.x, nr.y, area, ufoColor)
	g.ufoDrawn, g.ufoShown = nr, true
}

func (g *Game) msgRect() rect { return rect{80, 164, g.w - 160, 20} }

func (g *Game) drawMessage() {
	var msg string
	switch g.state {
	case ready:
		msg = "TAP TO START - SLIDE TO MOVE, HOLD TO FIRE"
	case cleared:
		msg = "WAVE " + strconv.Itoa(g.wave) + " CLEAR"
	case gameOver:
		msg = "GAME OVER - TAP TO PLAY AGAIN"
	}
	if msg == g.msgShown {
		return
	}
	r := g.msgRect()
	g.d.FillRectangle(r.x, r.y, r.w, r.h, bgColor)
	if msg != "" {
		_, tw := tinyfont.LineWidth(&proggy.TinySZ8pt7b, msg)
		tinyfont.WriteLine(g.d, &proggy.TinySZ8pt7b, r.x+(r.w-int16(tw))/2, r.y+14, msg, msgColor)
	} else {
		g.redrawInvadersIn(r)
	}
	g.msgShown = msg
}

func min16(a, b int16) int16 {
	if a < b {
		return a
	}
	return b
}

func max16(a, b int16) int16 {
	if a > b {
		return a
	}
	return b
}
