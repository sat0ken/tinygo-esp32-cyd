package kart

import (
	"image/color"
	"strconv"
	"unsafe"

	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/pad"
	"github.com/sat0ken/tinygo-cyd/romfont/freesans"
	"github.com/sat0ken/tinygo-cyd/romfont/proggy"
	"tinygo.org/x/tinyfont"
)

var (
	skyColors   = [...]color.RGBA{{90, 160, 230, 255}, {120, 185, 240, 255}, {160, 210, 245, 255}}
	hillColor   = color.RGBA{60, 120, 90, 255}
	textColor   = color.RGBA{255, 255, 255, 255}
	shadowColor = color.RGBA{20, 40, 80, 255}
	countColor  = color.RGBA{255, 220, 40, 255}
	panelColor  = color.RGBA{20, 28, 48, 255}
	dimText     = color.RGBA{170, 180, 200, 255}
	btnColor    = color.RGBA{70, 78, 96, 255}
	focusColor  = color.RGBA{255, 210, 40, 255}
	goldColor   = color.RGBA{255, 210, 40, 255}
	levelColors = [len(levels)]color.RGBA{{40, 140, 70, 255}, {190, 140, 30, 255}, {190, 50, 50, 255}}

	miniGrass = color.RGBA{25, 70, 35, 255}
	miniRoad  = color.RGBA{170, 170, 170, 255}
	miniStart = color.RGBA{255, 255, 255, 255}
)

func (g *Game) fill(r rect, c color.RGBA) { g.d.FillRectangle(r.x, r.y, r.w, r.h, c) }

func (g *Game) centered(r rect, font tinyfont.Fonter, s string, c color.RGBA, base int16) {
	_, w := tinyfont.LineWidth(font, s)
	tinyfont.WriteLine(g.d, font, r.x+(r.w-int16(w))/2, r.y+base, s, c)
}

// ---- the sky ----

// hillHeight is the height of the hills at column x (in steps of 4).
func hillHeight(x int16) int16 {
	k := (x / 4) % 24
	if k > 12 {
		k = 24 - k
	}
	return 4 + k + (x/4)%5
}

// drawSky draws the part of the sky and the hills inside r.
func (g *Game) drawSky(r rect) {
	hz := g.l.horizon
	band := hz / int16(len(skyColors))
	for i := range skyColors {
		b := rect{0, int16(i) * band, g.w, band}
		if i == len(skyColors)-1 {
			b.h = hz - b.y
		}
		if c, ok := clip(b, r); ok {
			g.fill(c, skyColors[i])
		}
	}
	for x := r.x &^ 3; x < r.x+r.w && x < g.w; x += 4 {
		hh := hillHeight(x)
		if c, ok := clip(rect{x, hz - hh, 4, hh}, r); ok {
			g.fill(c, hillColor)
		}
	}
}

// clip returns the part of a inside b.
func clip(a, b rect) (rect, bool) {
	x0, y0 := max(a.x, b.x), max(a.y, b.y)
	x1, y1 := min(a.x+a.w, b.x+b.w), min(a.y+a.h, b.y+b.h)
	return rect{x0, y0, x1 - x0, y1 - y0}, x1 > x0 && y1 > y0
}

// ---- the menu ----

// DrawMenu shows the menu over the karts on the grid: the CPU level and
// "< GAMES".
func (g *Game) DrawMenu() {
	g.screen = menuScreen
	g.newRace()
	if g.focus > len(levels) || (g.focus == len(levels) && g.OnExit == nil) {
		g.focus = g.level
	}
	g.drawSky(rect{0, 0, g.w, g.l.horizon})
	g.render()
	l := &g.l
	title := rect{0, 0, g.w, 0}
	g.centered(rect{2, 2, g.w, 0}, &freesans.Bold18pt7b, "TINYGO KART", shadowColor, l.titleBase)
	g.centered(title, &freesans.Bold18pt7b, "TINYGO KART", textColor, l.titleBase)
	g.fill(l.panel, panelColor)
	g.centered(l.panel, &proggy.TinySZ8pt7b, "CPU LEVEL - 3 LAPS", dimText, 12)
	hint := "HOLD THE GROUND TO DRIVE, TAP THE MAP TO QUIT"
	if g.Pad != nil {
		hint = "STICK: STEER  A: GAS  B: BRAKE  Y: QUIT"
	}
	g.centered(l.panel, &proggy.TinySZ8pt7b, hint, dimText, l.panel.h-6)
	for k := range g.menuBoxes() {
		g.drawMenuItem(k)
	}
	g.d.Display()
}

// menuBoxes are the buttons of the menu: the levels and "< GAMES".
func (g *Game) menuBoxes() []pad.Box {
	var b [len(levels) + 1]pad.Box
	for k := range levels {
		b[k] = g.l.levelButton(k).box()
	}
	if g.OnExit == nil {
		return b[:len(levels)]
	}
	b[len(levels)] = backButton().box()
	return b[:]
}

// drawMenuItem draws button k of the menu (see menuBoxes).
func (g *Game) drawMenuItem(k int) {
	var r rect
	if k == len(levels) {
		r = backButton()
		g.fill(r, btnColor)
		g.centered(r, &proggy.TinySZ8pt7b, "< GAMES", textColor, 18)
	} else {
		r = g.l.levelButton(k)
		g.fill(r, levelColors[k])
		font, base := tinyfont.Fonter(&freesans.Bold9pt7b), (r.h+12)/2
		if r.w >= 120 {
			font, base = &freesans.Bold12pt7b, (r.h+17)/2
		}
		g.centered(r, font, levels[k].Name, textColor, base)
	}
	if g.in.Used && k == g.focus {
		pad.Frame(g.d, r.x, r.y, r.w, r.h, 3, focusColor)
	}
}

func (g *Game) updateMenu(tap bool, x, y int16) {
	boxes := g.menuBoxes()
	if g.in.Used {
		if g.firstUse {
			g.drawMenuItem(g.focus)
		}
		if d := g.in.Direction(); d != 0 {
			if f := pad.Move(boxes, g.focus, d); f != g.focus {
				old := g.focus
				g.focus = f
				g.drawMenuItem(old)
				g.drawMenuItem(f)
			}
		}
		switch {
		case g.in.Pressed(hal.PadB) && g.OnExit != nil:
			g.OnExit()
			return
		case g.in.Pressed(hal.PadA | hal.PadOK):
			if g.focus == len(levels) {
				g.OnExit()
				return
			}
			g.level = g.focus
			g.start()
			return
		}
	}
	if !tap {
		return
	}
	if g.OnExit != nil && backButton().has(x, y) {
		g.OnExit()
		return
	}
	for k := range levels {
		if g.l.levelButton(k).has(x, y) {
			g.level, g.focus = k, k
			g.start()
			return
		}
	}
}

// ---- the race ----

// drawRaceScreen draws the whole screen for the start of a race.
func (g *Game) drawRaceScreen() {
	g.drawSky(rect{0, 0, g.w, g.l.horizon})
	m := g.l.minimap
	g.bd.DrawRGBBitmap8(m.x, m.y, u16bytes(g.mini), m.w, m.h)
	g.shown = [4]string{}
	for i := range g.karts {
		g.karts[i].mapX, g.karts[i].mapY = -1, -1
	}
	g.render()
	g.drawHUD()
	g.drawCountdown()
}

// hudText writes s in r (clearing it with the sky) if it is not shown yet.
func (g *Game) hudText(k int, r rect, font tinyfont.Fonter, s string, c color.RGBA, base int16, center bool) {
	if g.shown[k] == s {
		return
	}
	g.drawSky(r)
	if center {
		g.centered(r, font, s, shadowColor, base+2)
		g.centered(rect{r.x - 2, r.y, r.w, r.h}, font, s, c, base)
	} else {
		tinyfont.WriteLine(g.d, font, r.x+1, r.y+base+1, s, shadowColor)
		tinyfont.WriteLine(g.d, font, r.x, r.y+base, s, c)
	}
	g.shown[k] = s
}

// drawHUD writes the lap, the place and the time, and moves the dots of
// the minimap.
func (g *Game) drawHUD() {
	l := &g.l
	p := g.player()
	lap := p.lap + 1
	if lap < 1 {
		lap = 1
	}
	if lap > laps {
		lap = laps
	}
	g.hudText(0, rect{l.hudX, l.lapBase - 19, 112, 23}, &freesans.Bold12pt7b,
		"LAP "+strconv.Itoa(lap)+"/"+strconv.Itoa(laps), textColor, 19, false)
	g.hudText(1, rect{l.hudX, l.posBase - 19, 112, 23}, &freesans.Bold12pt7b,
		ordinal(g.placeNow())+" / 4", textColor, 19, false)
	g.hudText(2, rect{l.hudX, l.timeBase - 8, 112, 11}, &proggy.TinySZ8pt7b,
		"TIME "+clock(g.raceTime), textColor, 8, false)
	if g.screen == racing {
		text := ""
		if g.goT > 0 {
			text = "GO!"
		}
		g.hudText(3, g.countRect(), &freesans.Bold24pt7b, text, countColor, 36, true)
	}
	g.drawDots()
}

func (g *Game) countRect() rect {
	return rect{g.w/2 - 60, g.l.countBase - 38, 120, 46}
}

// drawCountdown writes 3, 2, 1 in the sky.
func (g *Game) drawCountdown() {
	if g.screen != countdown {
		return
	}
	n := int(g.countT) + 1
	if n > countFrom {
		n = countFrom
	}
	g.hudText(3, g.countRect(), &freesans.Bold24pt7b, strconv.Itoa(n), countColor, 36, true)
}

// makeMinimap draws the course into g.mini: grey road on dark grass,
// the start line white.
func (g *Game) makeMinimap() {
	m := g.l.minimap
	g.mini = make([]uint16, int(m.w)*int(m.h))
	for py := int16(0); py < m.h; py++ {
		for px := int16(0); px < m.w; px++ {
			cx, cy := int(px)*mapSize/int(m.w), int(py)*mapSize/int(m.h)
			c := miniGrass
			switch tile(g.track.cells[cy*mapSize+cx] >> 1) {
			case road, curb, boost:
				c = miniRoad
			case startLine:
				c = miniStart
			}
			g.mini[int(py)*int(m.w)+int(px)] = bigEndian(rgb565(c))
		}
	}
}

// drawDots moves the karts' dots on the minimap: the old ones are drawn
// over with the minimap, then all are drawn again (the player's last).
func (g *Game) drawDots() {
	m := g.l.minimap
	moved := false
	for i := range g.karts {
		k := &g.karts[i]
		x := int16(k.x*float32(m.w)/worldSize) - 1
		y := int16(k.y*float32(m.h)/worldSize) - 1
		x, y = min(max(x, 0), m.w-3), min(max(y, 0), m.h-3)
		if x == k.mapX && y == k.mapY {
			continue
		}
		moved = true
		if k.mapX >= 0 {
			for r := int16(0); r < 3; r++ {
				copy(g.dot[r*3:r*3+3], g.mini[int(k.mapY+r)*int(m.w)+int(k.mapX):])
			}
			g.bd.DrawRGBBitmap8(m.x+k.mapX, m.y+k.mapY, u16bytes(g.dot[:]), 3, 3)
		}
		k.mapX, k.mapY = x, y
	}
	if !moved {
		return
	}
	for i := len(g.karts) - 1; i >= 0; i-- {
		k := &g.karts[i]
		g.d.FillRectangle(m.x+k.mapX, m.y+k.mapY, 3, 3, k.dot)
	}
}

// ---- the results ----

// drawResults shows the place and the times over the last frame.
func (g *Game) drawResults() {
	b := g.l.box
	g.fill(b, panelColor)
	pad.Frame(g.d, b.x, b.y, b.w, b.h, 3, goldColor)
	g.centered(b, &freesans.Bold18pt7b, "FINISH!", goldColor, 32)
	c := textColor
	if g.place == 1 {
		c = goldColor
	}
	g.centered(b, &freesans.Bold12pt7b, ordinal(g.place)+" PLACE", c, 58)
	g.centered(b, &proggy.TinySZ8pt7b, "TIME "+clock(g.raceTime)+"   BEST LAP "+clock(g.bestLap), dimText, 78)
	g.drawResultButtons()
	g.d.Display()
}

func (g *Game) drawResultButtons() {
	for k, r := range [2]rect{g.l.again, g.l.menu} {
		g.fill(r, btnColor)
		g.centered(r, &freesans.Bold9pt7b, [2]string{"AGAIN", "MENU"}[k], textColor, (r.h+12)/2)
		if g.in.Used && k == g.focus {
			pad.Frame(g.d, r.x, r.y, r.w, r.h, 2, focusColor)
		}
	}
}

func (g *Game) updateResults(tap bool, x, y int16) {
	if g.in.Used {
		if g.firstUse {
			g.drawResultButtons()
		}
		if d := g.in.Direction(); d != 0 {
			boxes := [2]pad.Box{g.l.again.box(), g.l.menu.box()}
			if f := pad.Move(boxes[:], g.focus, d); f != g.focus {
				g.focus = f
				g.drawResultButtons()
			}
		}
		switch {
		case g.in.Pressed(hal.PadB):
			g.focus = g.level
			g.DrawMenu()
			return
		case g.in.Pressed(hal.PadA | hal.PadOK):
			if g.focus == 0 {
				g.start()
			} else {
				g.focus = g.level
				g.DrawMenu()
			}
			return
		}
	}
	switch {
	case tap && g.l.again.has(x, y):
		g.start()
	case tap && g.l.menu.has(x, y):
		g.focus = g.level
		g.DrawMenu()
	}
}

// ---- text ----

func ordinal(n int) string {
	switch n {
	case 1:
		return "1ST"
	case 2:
		return "2ND"
	case 3:
		return "3RD"
	}
	return strconv.Itoa(n) + "TH"
}

// clock formats seconds as m:ss.t.
func clock(sec float32) string {
	t := int(sec * 10)
	m, s, d := t/600, t/10%60, t%10
	ss := strconv.Itoa(s)
	if s < 10 {
		ss = "0" + ss
	}
	return strconv.Itoa(m) + ":" + ss + "." + strconv.Itoa(d)
}

// u16bytes views RGB565 pixels stored like bufs as bytes.
func u16bytes(p []uint16) []byte {
	return unsafe.Slice((*byte)(unsafe.Pointer(&p[0])), len(p)*2)
}
