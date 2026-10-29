package starguard

import (
	"image/color"
	"strconv"
	"unsafe"

	"github.com/sat0ken/tinygo-cyd/romfont/freesans"
	"github.com/sat0ken/tinygo-cyd/romfont/proggy"
	"tinygo.org/x/tinyfont"
)

var (
	panelColor = color.RGBA{38, 42, 54, 255}
	edgeColor  = color.RGBA{95, 102, 128, 255}
	labelColor = color.RGBA{150, 160, 185, 255}
	textColor  = color.RGBA{255, 255, 255, 255}
	gaugeBack  = color.RGBA{15, 18, 26, 255}
	boxColor   = color.RGBA{10, 14, 30, 255}
	overColor  = color.RGBA{255, 80, 60, 255}
)

const radarRange = 600 // world units to each side shown on the radar

func (g *Game) fill(r rect, c color.RGBA) { g.d.FillRectangle(r.x, r.y, r.w, r.h, c) }

func (g *Game) centered(r rect, font tinyfont.Fonter, s string, c color.RGBA, base int16) {
	_, w := tinyfont.LineWidth(font, s)
	tinyfont.WriteLine(g.d, font, r.x+(r.w-int16(w))/2, r.y+base, s, c)
}

// drawPanel draws the dashboard without its values.
func (g *Game) drawPanel() {
	l := &g.l
	g.fill(l.panel, panelColor)
	g.fill(rect{0, l.panel.y, g.w, 2}, edgeColor)
	tinyfont.WriteLine(g.d, &proggy.TinySZ8pt7b, l.shield.x, l.labelsBase, "SHIELD", labelColor)
	tinyfont.WriteLine(g.d, &proggy.TinySZ8pt7b, l.scoreX, l.labelsBase, "SCORE", labelColor)
	r := l.shield
	g.fill(rect{r.x - 2, r.y - 2, r.w + 4, r.h + 4}, edgeColor)
	g.shownScore, g.shownShield = -1, -1
}

// drawGauges updates the shield gauge and the score when they change, and
// the radar every frame.
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
	plot := func(x, y int, c uint16) {
		for dy := 0; dy < 2; dy++ {
			for dx := 0; dx < 2; dx++ {
				if px, py := x+dx, y+dy; px >= 0 && py >= 0 && px < w && py < h {
					buf[py*w+px] = c
				}
			}
		}
	}
	for i := range g.enemies {
		e := &g.enemies[i]
		if !e.alive {
			continue
		}
		x := w/2 + int(e.x*float32(w/2)/radarRange)
		y := h - 1 - int(e.z*float32(h-1)/spawnFar)
		c := c565(200, 190, 170)
		if e.kind == robot {
			c = c565(255, 60, 60)
		}
		plot(x-1, y-1, c)
	}
	plot(w/2-1, h-2, c565(80, 220, 255)) // the ship
	g.bd.DrawRGBBitmap8(r.x, r.y, unsafe.Slice((*byte)(unsafe.Pointer(&buf[0])), len(buf)*2), r.w, r.h)
}

// drawGameOver writes the result over the view.
func (g *Game) drawGameOver() {
	b := rect{g.w/2 - 120, g.l.viewH/2 - 46, 240, 92}
	g.fill(b, boxColor)
	g.centered(b, &freesans.Bold18pt7b, "GAME OVER", overColor, 34)
	g.centered(b, &freesans.Bold12pt7b, "SCORE "+strconv.Itoa(g.score), textColor, 62)
	g.centered(b, &proggy.TinySZ8pt7b, "A OR TAP TO PLAY AGAIN", labelColor, 82)
	g.d.Display()
}
