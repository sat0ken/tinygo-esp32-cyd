package baseball

import "github.com/sat0ken/tinygo-cyd/picopad"

// The batting view: the strike zone and where things stand.
const (
	zoneCX, zoneCY = 160, 150 // the middle of the strike zone
	zoneHW, zoneHH = 26, 30   // its half width and height (pixels)
	releaseX       = 152      // where the pitcher lets the ball go
	releaseY       = 78
	pitcherX       = 148 // the pitcher sprite
	pitcherY       = 72
	batterY        = 150
	batterXR       = 84  // a right-handed batter (left of the plate)
	batterXL       = 188 // a left-handed one
	catcherX       = 140
	catcherY       = 206
)

// zoneToScreen converts a point of the plate (zone units) to the screen.
func zoneToScreen(p vec) (int, int) {
	return zoneCX + int(p.x*zoneHW), zoneCY - int(p.y*zoneHH)
}

// fieldToScreen converts a point of the field (feet) to the fielding view.
func fieldToScreen(p vec) (int, int) {
	return 160 + int(p.x/1.75), 232 - int(p.y/1.75)
}

// rect is a part of the screen to restore from the background.
type rect struct{ x, y, w, h int }

// restore draws the background back over r.
func (g *Game) restore(r rect) {
	x0, y0 := max(r.x, 0), max(r.y, 0)
	x1, y1 := min(r.x+r.w, picopad.Width), min(r.y+r.h, picopad.Height)
	if x0 >= x1 || y0 >= y1 {
		return
	}
	g.DrawImgPal(g.bg, g.bgPal, x0, y0, x0, y0, x1-x0, y1-y0, picopad.Width)
}

// mark notes a rectangle drawn this frame, to restore in the next.
func (g *Game) mark(x, y, w, h int) {
	if g.nDrawn < len(g.drawn) {
		g.drawn[g.nDrawn] = rect{x, y, w, h}
		g.nDrawn++
	}
}

// clean restores what the last frame drew.
func (g *Game) clean() {
	for i := 0; i < g.nDrawn; i++ {
		g.restore(g.drawn[i])
	}
	g.nDrawn = 0
}

// sprite draws frame n of a sprite sheet (frames side by side) and marks it.
func (g *Game) sprite(img string, pal picopad.Palette, n, w, h, frames, x, y int) {
	g.DrawBlitPal(img, pal, n*w, 0, x, y, w, h, w*frames, picopad.ColMagenta)
	g.mark(x, y, w, h)
}

// ball draws a ball of radius r at x, y (a shaded disc) and marks it.
func (g *Game) ball(x, y, r int) {
	white := picopad.RGB(250, 250, 250)
	if r <= 1 {
		g.DrawRect(x-1, y-1, 2, 2, white)
		g.mark(x-1, y-1, 2, 2)
		return
	}
	for dy := -r; dy <= r; dy++ {
		w := isqrt(r*r - dy*dy)
		g.DrawRect(x-w, y+dy, 2*w+1, 1, white)
	}
	g.DrawRect(x-r/2, y+r/2, r, 1, picopad.RGB(200, 200, 200)) // the shade below
	g.DrawPoint(x+r/2, y-r/2+1, picopad.RGB(220, 40, 40))      // the seam
	g.mark(x-r, y-r, 2*r+1, 2*r+1)
}

func isqrt(v int) int {
	r := 0
	for (r+1)*(r+1) <= v {
		r++
	}
	return r
}

// box draws a panel of the HUD.
func (g *Game) box(x, y, w, h int) {
	g.DrawRect(x, y, w, h, picopad.RGB(16, 22, 48))
	g.DrawFrame(x, y, w, h, picopad.RGB(150, 160, 200))
}

// num draws n (at least digits digits, without leading zeros) by
// characters, without building a string.
func (g *Game) num(n, x, y int, col picopad.Color) int {
	var buf [6]byte
	i := len(buf)
	for {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
		if n == 0 || i == 0 {
			break
		}
	}
	for k := i; k < len(buf); k++ {
		g.DrawChar(buf[k], x, y, col)
		x += g.DrawFontWidth()
	}
	return x
}

// lights draws n of max lamps of col (the count).
func (g *Game) lights(x, y, n, max int, col picopad.Color) {
	for i := 0; i < max; i++ {
		c := picopad.RGB(50, 56, 80)
		if i < n {
			c = col
		}
		g.DrawRect(x+i*9, y, 7, 7, c)
	}
}

// diamond draws the bases with the runners on them.
func (g *Game) diamond(x, y int) {
	on := picopad.RGB(250, 210, 40)
	off := picopad.RGB(60, 66, 96)
	pos := [3][2]int{{x + 12, y + 6}, {x + 6, y}, {x, y + 6}}
	for b, p := range pos {
		c := off
		if g.m.bases[b].on {
			c = on
		}
		g.DrawRect(p[0], p[1], 6, 6, c)
	}
}

// hud draws the count, the inning and the score.
func (g *Game) hud() {
	m := &g.m
	g.SelFont8x8()
	g.box(2, 2, 104, 32)
	if m.top {
		g.DrawText("TOP", 6, 6, picopad.ColWhite)
	} else {
		g.DrawText("BOT", 6, 6, picopad.ColWhite)
	}
	g.num(m.inning, 34, 6, picopad.ColWhite)
	g.diamond(82, 8)
	g.DrawText("B", 6, 16, picopad.ColWhite)
	g.lights(16, 16, m.cnt.balls, 3, picopad.RGB(60, 220, 90))
	g.DrawText("S", 6, 25, picopad.ColWhite)
	g.lights(16, 25, m.cnt.strikes, 2, picopad.RGB(250, 210, 40))
	g.DrawText("O", 58, 25, picopad.ColWhite)
	g.lights(68, 25, m.cnt.outs, 2, picopad.RGB(240, 60, 60))

	g.box(250, 2, 68, 32)
	for i := 0; i < 2; i++ {
		t := m.t[i]
		y := 7 + i*12
		g.DrawRect(254, y, 4, 8, t.color)
		g.DrawText(t.short, 261, y, picopad.ColWhite)
		g.num(m.score[i], 295, y, picopad.ColYellow)
	}
}

// batterBox draws the batter's name and abilities.
func (g *Game) batterBox() {
	m := &g.m
	b := m.curBatter()
	g.SelFont8x8()
	g.box(2, 182, 80, 56)
	g.num(m.batter[m.batting()]+1, 6, 186, picopad.ColYellow)
	g.DrawText(b.name, 18, 186, picopad.ColWhite)
	grade := func(label string, v int8, x, y int) {
		g.DrawText(label, x, y, picopad.RGB(150, 170, 220))
		g.DrawChar(gradeLetters[min(max(int(v), 1), 8)-1], x+20, y, picopad.ColWhite)
	}
	grade("MT", b.contact, 6, 197)
	grade("PW", b.power, 44, 197)
	grade("SP", b.speed, 6, 207)
	i := m.batting()
	k := m.batter[i]
	x := g.num(int(g.hits[i][k]), 46, 207, picopad.ColWhite)
	g.DrawChar('/', x, 207, picopad.RGB(150, 170, 220))
	g.num(int(g.atBats[i][k]), x+8, 207, picopad.ColWhite)
	if m.human[i] {
		if g.power {
			g.DrawText("POWER", 6, 220, picopad.RGB(250, 90, 80))
		} else {
			g.DrawText("MEET ", 6, 220, picopad.RGB(250, 220, 60))
		}
		g.DrawText("B", 54, 220, picopad.RGB(150, 170, 220))
	}
}

// pitcherBox draws the pitcher and the last pitch.
func (g *Game) pitcherBox() {
	m := &g.m
	g.SelFont8x8()
	g.box(238, 182, 80, 56)
	g.DrawText(m.fielderAt(posP).name, 242, 186, picopad.ColWhite)
	if g.lastSpeed > 0 {
		x := g.num(int(g.lastSpeed+0.5), 242, 200, picopad.ColYellow)
		g.DrawText("km/h", x+2, 200, picopad.ColWhite)
		g.DrawText(pitchInfo[g.lastKind].name, 242, 212, picopad.RGB(150, 170, 220))
	}
}

// message shows a word in the middle of the batting view.
func (g *Game) message(text string, col picopad.Color) {
	g.SelFont8x16()
	w := len(text) * 16
	x := (picopad.Width - w) / 2
	g.DrawRect(x-6, 40, w+12, 36, picopad.RGB(16, 22, 48))
	g.DrawText2(text, x, 42, col)
	g.mark(x-6, 40, w+12, 36)
}
