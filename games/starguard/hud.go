package starguard

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
	panelColor  = color.RGBA{38, 42, 54, 255}
	edgeColor   = color.RGBA{95, 102, 128, 255}
	labelColor  = color.RGBA{150, 160, 185, 255}
	textColor   = color.RGBA{255, 255, 255, 255}
	gaugeBack   = color.RGBA{15, 18, 26, 255}
	boxColor    = color.RGBA{10, 14, 30, 255}
	overColor   = color.RGBA{255, 80, 60, 255}
	winColor    = color.RGBA{255, 210, 40, 255}
	titleColor  = color.RGBA{120, 220, 255, 255}
	shadowColor = color.RGBA{20, 40, 90, 255}
	btnColor    = color.RGBA{70, 78, 96, 255}
	focusColor  = color.RGBA{255, 210, 40, 255}
	levelColors = [len(levels)]color.RGBA{{40, 140, 70, 255}, {190, 140, 30, 255}, {190, 50, 50, 255}}
)

const radarRange = 600 // world units to each side shown on the radar

func (g *Game) fill(r rect, c color.RGBA) { g.d.FillRectangle(r.x, r.y, r.w, r.h, c) }

func (g *Game) centered(r rect, font tinyfont.Fonter, s string, c color.RGBA, base int16) {
	_, w := tinyfont.LineWidth(font, s)
	tinyfont.WriteLine(g.d, font, r.x+(r.w-int16(w))/2, r.y+base, s, c)
}

func stageName(n int) string { return "STAGE " + strconv.Itoa(n+1) }

// ---- the title screen ----

// DrawMenu shows the title screen over the stars: the level buttons and
// "< GAMES".
func (g *Game) DrawMenu() {
	g.screen = menuScreen
	g.reset()
	if g.focus > len(levels) || (g.focus == len(levels) && g.OnExit == nil) {
		g.focus = g.level
	}
	g.render()
	g.drawPanel()
	g.drawGauges()
	l := &g.l
	view := rect{0, 0, g.w, l.viewH}
	g.centered(rect{2, 2, g.w, 0}, &freesans.Bold18pt7b, "STAR GUARD", shadowColor, l.titleBase)
	g.centered(view, &freesans.Bold18pt7b, "STAR GUARD", titleColor, l.titleBase)
	g.centered(view, &proggy.TinySZ8pt7b, "3 STAGES, THEN THE MOTHERSHIP", labelColor, l.subBase)
	hint := "TOUCH TO AIM AND FIRE, TAP THE RADAR TO QUIT"
	if g.Pad != nil {
		hint = "STICK: AIM  A: FIRE  Y: QUIT"
	}
	g.centered(view, &proggy.TinySZ8pt7b, hint, labelColor, l.hintBase)
	for k := range g.menuBoxes() {
		g.drawMenuItem(k)
	}
	g.d.Display()
}

// menuBoxes are the buttons of the title screen: the levels and
// "< GAMES".
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

// drawMenuItem draws button k of the title screen (see menuBoxes).
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
	if g.in.Used {
		if g.firstUse {
			g.drawMenuItem(g.focus)
		}
		if d := g.in.Direction(); d != 0 {
			if f := pad.Move(g.menuBoxes(), g.focus, d); f != g.focus {
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

// ---- the dashboard ----

// drawPanel draws the dashboard without its values.
func (g *Game) drawPanel() {
	l := &g.l
	g.fill(l.panel, panelColor)
	g.fill(rect{0, l.panel.y, g.w, 2}, edgeColor)
	tinyfont.WriteLine(g.d, &proggy.TinySZ8pt7b, l.shield.x, l.labelsBase, "SHIELD", labelColor)
	tinyfont.WriteLine(g.d, &proggy.TinySZ8pt7b, l.scoreX, l.labelsBase, "SCORE", labelColor)
	r := l.shield
	g.fill(rect{r.x - 2, r.y - 2, r.w + 4, r.h + 4}, edgeColor)
	g.shownScore, g.shownShield, g.shownStage = -1, -1, -1
}

// drawGauges updates the shield gauge, the score and the stage when they
// change, and the radar every frame.
func (g *Game) drawGauges() {
	l := &g.l
	if g.shield != g.shownShield {
		r := l.shield
		g.fill(r, gaugeBack)
		c := color.RGBA{60, 220, 90, 255}
		switch {
		case g.shield <= shield0/4:
			c = color.RGBA{230, 50, 40, 255}
		case g.shield <= shield0/2:
			c = color.RGBA{240, 200, 40, 255}
		}
		g.fill(rect{r.x, r.y, r.w * int16(g.shield) / shield0, r.h}, c)
		g.shownShield = g.shield
	}
	if g.score != g.shownScore {
		g.fill(rect{l.scoreX, l.scoreBase - 18, 100, 22}, panelColor)
		tinyfont.WriteLine(g.d, &freesans.Bold12pt7b, l.scoreX, l.scoreBase, strconv.Itoa(g.score), textColor)
		g.shownScore = g.score
	}
	if s := g.stage*1000 + g.killed; s != g.shownStage {
		text := "MOTHERSHIP"
		if g.stage < bossStage {
			text = stageName(g.stage) + "   " + strconv.Itoa(g.killed) + "/" + strconv.Itoa(stages[g.stage].count)
		}
		g.fill(rect{l.shield.x, l.stageBase - 8, l.scoreX - l.shield.x - 4, 10}, panelColor)
		tinyfont.WriteLine(g.d, &proggy.TinySZ8pt7b, l.shield.x, l.stageBase, text, labelColor)
		g.shownStage = s
	}
	g.drawRadar()
}

// drawRadar shows the enemies from above: the ship at the bottom in the
// middle, far ones at the top.
func (g *Game) drawRadar() {
	r := g.l.radar
	if g.bd == nil {
		return
	}
	buf := g.radarBuf
	bg, line := c565(8, 30, 18), c565(30, 80, 50)
	for i := range buf {
		buf[i] = bg
	}
	w, h := int(r.w), int(r.h)
	for y := 0; y < h; y++ {
		buf[y*w+w/2] = line // straight ahead
	}
	for x := 0; x < w; x++ {
		buf[(h/2)*w+x] = line // half way
	}
	if b := &g.boss; b.active {
		x, y := g.radarAt(b.x, b.z)
		g.radarDot(x-3, y-1, 6, c565(255, 120, 40))
	}
	for i := range g.enemies {
		e := &g.enemies[i]
		if !e.alive {
			continue
		}
		x, y := g.radarAt(e.x, e.z)
		c := c565(200, 190, 170)
		switch e.kind {
		case robot, drone:
			c = c565(255, 60, 60)
		case capsule:
			c = c565(70, 235, 165)
		}
		g.radarDot(x-1, y-1, 2, c)
	}
	g.radarDot(w/2-1, h-2, 2, c565(80, 220, 255)) // the ship
	g.bd.DrawRGBBitmap8(r.x, r.y, unsafe.Slice((*byte)(unsafe.Pointer(&buf[0])), len(buf)*2), r.w, r.h)
}

// radarAt returns where world point (x, z) is on the radar.
func (g *Game) radarAt(x, z float32) (int, int) {
	w, h := int(g.l.radar.w), int(g.l.radar.h)
	return w/2 + int(x*float32(w/2)/radarRange), h - 1 - int(z*float32(h-1)/spawnFar)
}

// radarDot draws a size by size square at (x, y) on the radar.
// (A method, not a closure: a closure over buf would be allocated every
// frame.)
func (g *Game) radarDot(x, y, size int, c uint16) {
	w, h := int(g.l.radar.w), int(g.l.radar.h)
	for dy := 0; dy < size; dy++ {
		for dx := 0; dx < size; dx++ {
			if px, py := x+dx, y+dy; px >= 0 && py >= 0 && px < w && py < h {
				g.radarBuf[py*w+px] = c
			}
		}
	}
}

// drawNotice writes a title and a line in a box in the middle of the view
// (over a frozen view).
func (g *Game) drawNotice(title, line string) {
	b := rect{g.w/2 - 110, g.l.viewH/2 - 32, 220, 64}
	g.fill(b, boxColor)
	pad.Frame(g.d, b.x, b.y, b.w, b.h, 2, titleColor)
	g.centered(b, &freesans.Bold18pt7b, title, titleColor, 36)
	g.centered(b, &proggy.TinySZ8pt7b, line, labelColor, 54)
	g.d.Display()
}

// ---- the results ----

// showResults shows the end of the game over the last frame.
func (g *Game) showResults(win bool) {
	g.screen, g.complete, g.focus = results, win, 0
	g.drawResults()
}

func (g *Game) drawResults() {
	b := g.l.box
	c, title := overColor, "GAME OVER"
	if g.complete {
		c, title = winColor, "MISSION COMPLETE!"
	}
	g.fill(b, boxColor)
	pad.Frame(g.d, b.x, b.y, b.w, b.h, 3, c)
	var font tinyfont.Fonter = &freesans.Bold18pt7b
	if _, w := tinyfont.LineWidth(font, title); int16(w) > b.w-12 {
		font = &freesans.Bold12pt7b
	}
	g.centered(b, font, title, c, 34)
	g.centered(b, &freesans.Bold12pt7b, "SCORE "+strconv.Itoa(g.score), textColor, 62)
	reached := stageName(g.stage)
	if g.stage == bossStage {
		reached = "MOTHERSHIP"
	}
	g.centered(b, &proggy.TinySZ8pt7b, levels[g.level].Name+" - "+reached, labelColor, 80)
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
