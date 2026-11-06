package futsal

import "github.com/sat0ken/tinygo-cyd/picopad"

// rect is a part of the screen to restore from the pitch.
type rect struct{ x, y, w, h int }

// restore draws the pitch back over r.
func (g *Game) restore(r rect) {
	x0, y0 := max(r.x, 0), max(r.y, hudH)
	x1, y1 := min(r.x+r.w, picopad.Width), min(r.y+r.h, picopad.Height)
	if x0 >= x1 || y0 >= y1 {
		return
	}
	g.DrawImg4Pal(pitchImg, pitchImgPal, x0, y0, x0, y0, x1-x0, y1-y0, picopad.Width)
}

// hudH is the height of the score board at the top.
const hudH = 30

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

// box draws a panel.
func (g *Game) box(x, y, w, h int) {
	g.DrawRect(x, y, w, h, picopad.RGB(16, 22, 48))
	g.DrawFrame(x, y, w, h, picopad.RGB(150, 160, 200))
}

// num draws n by characters, without building a string, and returns the
// x after it.
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

// scoreBoard draws the names, the score and the clock when they changed.
func (g *Game) scoreBoard() {
	m := &g.m
	now := [3]int{m.score[0], m.score[1], m.minute()}
	if now == g.hud {
		return
	}
	g.hud = now
	navy := picopad.RGB(16, 22, 48)
	g.DrawRect(0, 0, picopad.Width, hudH-1, navy)
	g.SelFont8x16()
	for i := 0; i < 2; i++ {
		t := m.t[i]
		x, nx, sx := 4, 16, 76
		if i == 1 {
			x, nx, sx = 310, 268, 232
		}
		g.DrawRect(x, 7, 6, 16, t.color)
		g.DrawText(t.short, nx, 7, picopad.ColWhite)
		n := m.score[i]
		if n >= 10 && i == 1 {
			sx -= 8
		}
		g.num(n, sx, 7, picopad.ColYellow)
	}
	g.SelFont8x8()
	if m.half == 1 {
		g.DrawText("1ST", 148, 3, picopad.RGB(150, 170, 220))
	} else {
		g.DrawText("2ND", 148, 3, picopad.RGB(150, 170, 220))
	}
	s := now[2]
	x := 140
	g.SelFont8x16()
	g.DrawChar(byte('0'+s/600), x, 12, picopad.ColWhite)
	g.DrawChar(byte('0'+s/60%10), x+8, 12, picopad.ColWhite)
	g.DrawChar(':', x+16, 12, picopad.ColWhite)
	g.DrawChar(byte('0'+s%60/10), x+24, 12, picopad.ColWhite)
	g.DrawChar(byte('0'+s%10), x+32, 12, picopad.ColWhite)
}

// message shows a word over the pitch.
func (g *Game) message(text string, col picopad.Color) {
	g.SelFont8x16()
	w := len(text) * 16
	x := (picopad.Width - w) / 2
	g.DrawRect(x-6, 50, w+12, 36, picopad.RGB(16, 22, 48))
	g.DrawText2(text, x, 52, col)
	g.mark(x-6, 50, w+12, 36)
}

// frameOf returns the sprite of player p: the direction it faces and the
// step of its run.
func frameOf(p *player) int {
	dir := 0
	if abs32(p.face.x) > abs32(p.face.y) {
		dir = 2
		if p.face.x < 0 {
			dir = 3
		}
	} else if p.face.y < 0 {
		dir = 1
	}
	step := 0
	if p.step > 0 {
		step = 1 + int(p.step)%2
	}
	return dir*3 + step
}

// scene draws the players, the ball and the marks.
func (g *Game) scene() {
	m := &g.m
	g.clean()
	g.scoreBoard()
	// back to front: the players and the ball by the height of their feet
	var order [11]int
	n := 0
	for i := 0; i <= len(m.p); i++ {
		y := m.b.pos.y
		if i < len(m.p) {
			y = m.p[i].pos.y
		}
		k := n
		for k > 0 && g.ySort(order[k-1]) > y {
			order[k] = order[k-1]
			k--
		}
		order[k] = i
		n++
	}
	for _, i := range order[:n] {
		if i == len(m.p) {
			g.ball()
			continue
		}
		p := &m.p[i]
		pal := g.pals[p.team][0]
		if p.gk {
			pal = g.pals[p.team][1]
		}
		x, y := int(p.pos.x)-playerW/2, int(p.pos.y)-playerH+1
		g.DrawBlitPal(playerImg, pal, frameOf(p)*playerW, 0, x, y, playerW, playerH, playerW*12, picopad.ColMagenta)
		g.mark(x, y, playerW, playerH)
	}
	if m.human[0] && m.state != stGoal {
		p := &m.p[m.ctl[0]]
		x, y := int(p.pos.x), int(p.pos.y)-playerH-5
		yel := picopad.RGB(250, 220, 40)
		g.DrawRect(x-2, y, 5, 1, yel)
		g.DrawRect(x-1, y+1, 3, 1, yel)
		g.DrawPoint(x, y+2, yel)
		g.mark(x-2, y, 5, 3)
		if g.charging {
			g.DrawRect(x-7, y-4, 14, 3, picopad.RGB(16, 22, 48))
			g.DrawRect(x-6, y-3, 1+int(11*min(g.charge/0.6, 1)), 1, picopad.RGB(250, 80, 60))
			g.mark(x-7, y-4, 14, 3)
		}
	}
	switch m.state {
	case stKickoff:
		if m.stateT < 0.8 {
			g.message("KICK OFF", picopad.ColWhite)
		}
	case stGoal:
		g.message("GOAL!", picopad.ColYellow)
		if s := m.scorer; s >= 0 {
			p := &m.p[s]
			name := p.d.name
			g.SelFont8x8()
			w := len(name) * 8
			x := (picopad.Width - w) / 2
			g.DrawRect(x-4, 86, w+8, 12, picopad.RGB(16, 22, 48))
			g.DrawText(name, x, 88, m.t[p.team].color)
			g.mark(x-4, 86, w+8, 12)
		}
	}
}

// ySort is the height of the feet of item i of the scene (10: the ball).
func (g *Game) ySort(i int) float32 {
	if i == len(g.m.p) {
		return g.m.b.pos.y
	}
	return g.m.p[i].pos.y
}

// ball draws the ball and its shadow.
func (g *Game) ball() {
	x, y := int(g.m.b.pos.x), int(g.m.b.pos.y)
	white := picopad.RGB(250, 250, 250)
	g.DrawRect(x-1, y-1, 4, 2, picopad.RGB(30, 80, 36)) // the shadow
	g.DrawRect(x-2, y-3, 4, 2, white)
	g.DrawRect(x-1, y-4, 2, 4, white)
	g.DrawPoint(x, y-2, picopad.RGB(40, 40, 50))
	g.mark(x-2, y-4, 5, 5)
}
