// Package baseball is a baseball game in the style of the Japanese console
// baseball games: the duel of pitcher and batter seen from behind the
// catcher, then the field from above when the ball is in play. It runs on
// package picopad (320x240, the PicoPad's buttons or the touch panel).
//
// Batting: the arrows move the cursor over the strike zone, A swings when
// the ball reaches the plate, B switches between a meet swing (a large
// cursor) and a power swing (a small cursor, harder hits). Where the ball
// meets the cursor decides the ball: the middle of the cursor hits it
// hard, its top half beats it into the ground, its bottom half lifts it; an
// early swing pulls it, a late one goes the other way.
//
// Pitching: an arrow chooses a breaking ball (right a slider, left a shoot,
// down a fork, up a curve, those the pitcher has), B the straight ball, A
// takes the pitch; then the arrows aim and A throws (B to choose again). The ball does not go
// exactly where it is aimed: less so for a pitcher with good control.
//
// The ball in play is a flight with air resistance, bounces and rolls;
// the fielders run for it, throw to the bases, turn double plays, and the
// runners take as many bases as they can. Y quits the game.
package baseball

import (
	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/picopad"
)

//go:generate go run ./artgen

// teamPals are the sprites' palettes in a team's colours.
type teamPals struct {
	batter, pitcher, catcher, fielder picopad.Palette
}

// Game is the baseball game on a console.
type Game struct {
	*picopad.Console

	m       match
	innings int // the setting of the title screen
	watch   bool

	bg     string // the background being shown
	bgPal  picopad.Palette
	pals   [2]teamPals
	drawn  [32]rect
	nDrawn int

	cursor vec  // the batter's cursor
	power  bool // the power swing
	aim    vec  // the pitcher's aim

	hits, atBats [2][9]int8
	lastSpeed    float32
	lastKind     int
	sit          situation // before the ball in play
	last         uint32    // the time of the last frame (µs)
	quit         bool
}

// New returns the game on d (at least 320x240) with the touch panel t;
// seed starts the random numbers. Set Pad to use a game controller.
func New(d hal.Display, t hal.Touch, seed uint32) *Game {
	g := &Game{Console: picopad.New(d, t, seed), innings: 3}
	for i := range teams {
		t := &teams[i]
		g.pals[i] = teamPals{recolor(batterImgPal, t), recolor(pitcherImgPal, t), recolor(catcherImgPal, t), recolor(fielderImgPal, t)}
	}
	return g
}

// recolor returns pal with the stand-in colours of the team replaced by
// the team's.
func recolor(pal string, t *team) picopad.Palette {
	b := []byte(pal)
	for i := 0; i+1 < len(b); i += 2 {
		c := picopad.Color(b[i]) | picopad.Color(b[i+1])<<8
		switch c {
		case standInTeam:
			c = t.color
		case standInDark:
			c = t.dark
		default:
			continue
		}
		b[i], b[i+1] = byte(c), byte(c>>8)
	}
	return picopad.Palette(b)
}

// Run shows the title and plays until Y there (with OneGame, one game).
func (g *Game) Run() {
	for g.title() {
		g.play()
		if g.OneGame {
			return
		}
	}
}

const frameMs = 33

// frame shows the frame and waits for the next.
func (g *Game) frame() {
	g.DispUpdate()
	if d := g.Time() - g.last; d < frameMs*1000 {
		g.WaitMs(int(frameMs - d/1000))
	}
	g.last = g.Time()
}

// seconds is the time since t0.
func (g *Game) seconds(t0 uint32) float32 { return float32(g.Time()-t0) / 1e6 }

// showBg draws the whole background b.
func (g *Game) showBg(img string, pal picopad.Palette) {
	g.bg, g.bgPal = img, pal
	g.DrawImgPal(img, pal, 0, 0, 0, 0, picopad.Width, picopad.Height, picopad.Width)
	g.nDrawn = 0
}

// The innings to choose from.
var inningChoices = [...]int{1, 3, 6, 9}

// title is the title screen; it returns false on Y.
func (g *Game) title() bool {
	g.KeyFlush()
	redraw := true
	for {
		if redraw {
			g.showBg(batBgImg, batBgImgPal)
			g.SelFont8x16()
			g.DrawRect(40, 26, 240, 44, picopad.RGB(16, 22, 48))
			g.DrawFrame(40, 26, 240, 44, picopad.RGB(250, 210, 40))
			g.DrawText2("BASEBALL", 96, 32, picopad.ColYellow)
			g.SelFont8x8()
			g.DrawText(teams[0].name, 40, 80, picopad.ColWhite)
			g.DrawText("vs", 152, 80, picopad.ColYellow)
			g.DrawText(teams[1].name, 200, 80, picopad.ColWhite)

			g.box(70, 96, 180, 74)
			g.SelFont8x16()
			modes := [2]string{"1P vs CPU", "CPU vs CPU"}
			for i, s := range modes {
				col := picopad.RGB(140, 150, 180)
				if g.watch == (i == 1) {
					col = picopad.ColYellow
					g.DrawText(">", 82, 102+i*18, col)
				}
				g.DrawText(s, 98, 102+i*18, col)
			}
			g.DrawText("INNINGS", 98, 144, picopad.ColWhite)
			g.DrawText("<", 172, 144, picopad.ColWhite)
			g.num(g.innings, 188, 144, picopad.ColYellow)
			g.DrawText(">", 204, 144, picopad.ColWhite)

			g.SelFont8x8()
			g.box(2, 184, 316, 54)
			g.DrawText("BAT  : arrows aim, A swing, B power", 8, 189, picopad.ColWhite)
			g.DrawText("PITCH: arrow+A pitch, arrows aim, A", 8, 200, picopad.ColWhite)
			g.DrawText("A: PLAY BALL   Y: QUIT", 8, 216, picopad.ColYellow)
			g.DrawText("UP/DOWN mode  LEFT/RIGHT innings", 8, 227, picopad.RGB(150, 170, 220))
			g.DispUpdate()
			redraw = false
		}
		switch g.KeyGet() {
		case picopad.KeyUp, picopad.KeyDown:
			g.watch = !g.watch
			redraw = true
		case picopad.KeyLeft, picopad.KeyRight:
			i := 0
			for k, n := range inningChoices {
				if n == g.innings {
					i = k
				}
			}
			if g.KeyPressed(picopad.KeyLeft) {
				i = (i + len(inningChoices) - 1) % len(inningChoices)
			} else {
				i = (i + 1) % len(inningChoices)
			}
			g.innings = inningChoices[i]
			redraw = true
		case picopad.KeyA:
			return true
		case picopad.KeyY:
			return false
		}
		g.WaitMs(20)
	}
}

// play plays a game.
func (g *Game) play() {
	m := &g.m
	m.start(g.innings, !g.watch, false)
	g.hits, g.atBats = [2][9]int8{}, [2][9]int8{}
	g.lastSpeed = 0
	g.cursor, g.power = vec{}, false
	g.quit = false
	g.board("PLAY BALL!")
	for !m.over && !g.quit {
		inning, top := m.inning, m.top
		g.pitchOnce()
		if g.quit {
			return
		}
		if !m.over && (m.inning != inning || m.top != top) {
			g.board("CHANGE")
		}
	}
	switch {
	case m.score[0] == m.score[1]:
		g.board("DRAW")
	case m.human[0] && m.score[0] > m.score[1]:
		g.board("YOU WIN!")
	case m.human[0]:
		g.board("YOU LOSE")
	default:
		g.board("GAME SET")
	}
}

// board shows the line score under a title, until A or a while.
func (g *Game) board(title string) {
	m := &g.m
	g.DrawClear()
	g.SelFont8x16()
	g.DrawText2(title, (picopad.Width-len(title)*16)/2, 16, picopad.ColYellow)
	g.SelFont8x8()
	n := max(m.innings, m.inning)
	cw := min(20, 200/n)
	x0 := 60
	g.DrawRect(4, 80, 312, 64, picopad.RGB(16, 22, 48))
	for i := 0; i < n; i++ {
		g.num(i+1, x0+i*cw+4, 86, picopad.RGB(150, 170, 220))
	}
	g.DrawText("R", x0+n*cw+8, 86, picopad.ColYellow)
	g.DrawText("H", x0+n*cw+28, 86, picopad.ColYellow)
	for t := 0; t < 2; t++ {
		y := 104 + t*18
		g.DrawRect(8, y, 4, 8, m.t[t].color)
		g.DrawText(m.t[t].short, 16, y, picopad.ColWhite)
		for i := 0; i < n; i++ {
			played := i < m.inning-1 || i == m.inning-1 && (t == 0 || !m.top)
			if m.over && i == m.inning-1 && t == 1 && m.top {
				played = false // the home team did not need to bat
			}
			if played && i < maxInnings {
				g.num(int(m.line[t][i]), x0+i*cw+4, y, picopad.ColWhite)
			}
		}
		g.num(m.score[t], x0+n*cw+8, y, picopad.ColYellow)
		g.num(m.hits[t], x0+n*cw+28, y, picopad.ColWhite)
	}
	g.DrawText("A: GO ON", 128, 200, picopad.RGB(150, 170, 220))
	g.DispUpdate()
	g.KeyFlush()
	t0 := g.Time()
	for g.seconds(t0) < 3 {
		if k := g.KeyGet(); k == picopad.KeyA {
			break
		}
		g.WaitMs(20)
	}
}

// batView draws the batting view: the background, the HUD and the players.
func (g *Game) batView() {
	g.showBg(batBgImg, batBgImgPal)
	g.hud()
	g.batterBox()
	g.pitcherBox()
}

// players draws the pitcher in phase pp, the batter in phase bp and the
// catcher.
func (g *Game) players(pp, bp int) {
	m := &g.m
	f, b := m.fielding(), m.batting()
	g.sprite(pitcherImg, g.pals[f].pitcher, pp, pitcherW, pitcherH, 3, pitcherX, pitcherY)
	x, n := batterXR, bp
	if m.curBatter().left {
		x, n = batterXL, bp+3
	}
	g.sprite(batterImg, g.pals[b].batter, n, batterW, batterH, 6, x, batterY)
}

func (g *Game) catcherSprite() {
	g.sprite(catcherImg, g.pals[g.m.fielding()].catcher, 0, catcherW, catcherH, 1, catcherX, catcherY)
}

// cursorFrame draws the batter's cursor.
func (g *Game) cursorFrame() {
	cs := cursorSize(g.m.curBatter(), g.power)
	x, y := zoneToScreen(g.cursor)
	w, h := int(cs.x*zoneHW), int(cs.y*zoneHH)
	col := picopad.RGB(250, 220, 60)
	if g.power {
		col = picopad.RGB(250, 80, 70)
	}
	g.DrawFrame(x-w, y-h, 2*w+1, 2*h+1, col)
	g.DrawFrame(x-w+1, y-h+1, 2*w-1, 2*h-1, col)
	g.DrawRect(x-1, y-1, 3, 3, col)
	g.mark(x-w, y-h, 2*w+1, 2*h+1)
}

// moveCursor moves p with the arrows by dt seconds, within lim.
func (g *Game) moveCursor(p *vec, dt float32, lim vec) {
	const speed = 2.6 // zone units a second
	if g.KeyPressed(picopad.KeyLeft) {
		p.x -= speed * dt
	}
	if g.KeyPressed(picopad.KeyRight) {
		p.x += speed * dt
	}
	if g.KeyPressed(picopad.KeyUp) {
		p.y += speed * dt
	}
	if g.KeyPressed(picopad.KeyDown) {
		p.y -= speed * dt
	}
	p.x = clamp(p.x, -lim.x, lim.x)
	p.y = clamp(p.y, -lim.y, lim.y)
}

// askQuit asks whether to end the game (Y again: yes).
func (g *Game) askQuit() bool {
	g.message("QUIT? Y/A", picopad.ColWhite)
	g.DispUpdate()
	g.KeyFlush()
	for {
		switch g.KeyGet() {
		case picopad.KeyY:
			g.quit = true
			return true
		case picopad.KeyA, picopad.KeyB:
			g.clean()
			return false
		}
		g.WaitMs(20)
	}
}

// The pitch selection: the arrow for each pitch.
var pitchKey = [numPitches]uint8{ptFast: picopad.KeyA, ptSlider: picopad.KeyRight, ptCurve: picopad.KeyUp, ptShoot: picopad.KeyLeft, ptFork: picopad.KeyDown}

var pitchArrow = [numPitches]string{ptFast: "A", ptSlider: ">", ptCurve: "^", ptShoot: "<", ptFork: "v"}

// choosePitch lets the player choose the pitch and aim; it returns false
// on quit.
func (g *Game) choosePitch() (kind, level int, ok bool) {
	pt := g.m.curPitcher()
	kind, level = ptFast, 4
	aiming := false
	g.aim = vec{}
	g.KeyFlush()
	last := g.Time()
	panel := func() {
		g.SelFont8x8()
		g.box(238, 60, 80, 74)
		y := 64
		show := func(k, lv int) {
			col := picopad.RGB(150, 170, 220)
			if k == kind {
				col = picopad.ColYellow
			}
			g.DrawText(pitchArrow[k], 242, y, col)
			g.DrawText(pitchInfo[k].name, 252, y, col)
			y += 10
		}
		show(ptFast, 4)
		for _, p := range pt.pitches {
			if p.level > 0 {
				show(int(p.kind), int(p.level))
			}
		}
		if aiming {
			g.DrawText("AIM, A", 242, 118, picopad.ColWhite)
		} else {
			g.DrawText("PITCH?", 242, 118, picopad.ColWhite)
		}
	}
	panel()
	for {
		dt := g.seconds(last)
		last = g.Time()
		for k := g.KeyGet(); k != picopad.NoKey; k = g.KeyGet() {
			switch {
			case k == picopad.KeyY:
				if g.askQuit() {
					return 0, 0, false
				}
				g.batView()
				panel()
			case !aiming && k == picopad.KeyA:
				aiming = true
				panel()
			case !aiming:
				for _, p := range pt.pitches {
					if p.level > 0 && pitchKey[p.kind] == k {
						kind, level = int(p.kind), int(p.level)
						panel()
					}
				}
				if k == picopad.KeyB {
					kind, level = ptFast, 4
					panel()
				}
			case k == picopad.KeyA:
				return kind, level, true
			case k == picopad.KeyB:
				aiming = false
				panel()
			}
		}
		if aiming {
			g.moveCursor(&g.aim, dt, vec{1.7, 1.7})
		}
		g.clean()
		g.players(0, 0)
		if aiming {
			x, y := zoneToScreen(g.aim)
			c := picopad.RGB(250, 140, 40)
			g.DrawFrame(x-4, y-4, 9, 9, c)
			g.DrawRect(x, y-6, 1, 13, c)
			g.DrawRect(x-6, y, 13, 1, c)
			g.mark(x-6, y-6, 13, 13)
		}
		g.catcherSprite()
		g.frame()
	}
}

// ready is the moment before the computer's pitch, the batter's cursor
// moving; it returns false on quit.
func (g *Game) ready(d float32, human bool) bool {
	g.KeyFlush()
	t0, last := g.Time(), g.Time()
	for g.seconds(t0) < d {
		dt := g.seconds(last)
		last = g.Time()
		for k := g.KeyGet(); k != picopad.NoKey; k = g.KeyGet() {
			switch k {
			case picopad.KeyY:
				if g.askQuit() {
					return false
				}
				g.batView()
			case picopad.KeyB:
				if human {
					g.power = !g.power
					g.batterBox()
				}
			}
		}
		g.clean()
		g.players(0, 0)
		if human {
			g.moveCursor(&g.cursor, dt, vec{1.6, 1.5})
			g.cursorFrame()
		}
		g.catcherSprite()
		g.frame()
	}
	return true
}

// pitchOnce plays a pitch.
func (g *Game) pitchOnce() {
	m := &g.m
	humanBat, humanPitch := m.human[m.batting()], m.human[m.fielding()]
	pt, b := m.curPitcher(), m.curBatter()
	g.batView()

	var kind, level int
	var aim vec
	if humanPitch {
		var ok bool
		if kind, level, ok = g.choosePitch(); !ok {
			return
		}
		aim = g.aim
		g.batView()
	} else {
		kind, level, aim = cpuPitch(g, pt, m.cnt)
		if !g.ready(0.9, humanBat) {
			return
		}
	}

	// the wind-up
	t0, last := g.Time(), g.Time()
	for g.seconds(t0) < 0.45 {
		dt := g.seconds(last)
		last = g.Time()
		g.clean()
		g.players(1, 0)
		if humanBat {
			g.moveCursor(&g.cursor, dt, vec{1.6, 1.5})
			g.cursorFrame()
		}
		g.catcherSprite()
		g.frame()
	}

	p := throwPitch(g, pt, kind, level, aim)
	var sw swing
	if !humanBat {
		sw = cpuSwing(g, &p, b, m.cnt)
	}
	g.KeyFlush()
	wantQuit := false // Y during the pitch: asked after it
	t0, last = g.Time(), g.Time()
	for {
		t := g.seconds(t0)
		dt := g.seconds(last)
		last = g.Time()
		for k := g.KeyGet(); k != picopad.NoKey; k = g.KeyGet() {
			switch {
			case k == picopad.KeyY:
				wantQuit = true
			case !humanBat:
			case k == picopad.KeyA && !sw.swung:
				sw = swing{swung: true, t: t, at: g.cursor, power: g.power}
			case k == picopad.KeyB && !sw.swung:
				g.power = !g.power
				g.batterBox()
			}
		}
		if humanBat {
			if !sw.swung {
				g.moveCursor(&g.cursor, dt, vec{1.6, 1.5})
			}
		}
		if t > p.flight+0.35 {
			break
		}
		g.clean()
		bp := 0
		if sw.swung && t >= sw.t {
			bp = 1
			if t > sw.t+0.12 {
				bp = 2
			}
		}
		g.players(2, bp)
		if humanBat {
			g.cursorFrame()
		}
		if f := t / p.flight; f < 1.04 {
			w := f * sqrt32(f) // the perspective: faster as it comes near
			to := p.at(min(f, 1))
			x1, y1 := zoneToScreen(to)
			x := releaseX + int(float32(x1-releaseX)*w)
			y := releaseY + int(float32(y1-releaseY)*w)
			g.ball(x, y, 1+int(3.5*w))
		}
		g.catcherSprite()
		g.frame()
	}
	g.lastSpeed, g.lastKind = p.speed, kind

	res, hb := result(g, &p, sw, b)
	i, k := m.batting(), m.batter[m.batting()]
	if res == pitchInPlay {
		g.hitAway(hb)
		g.sit = m.situation()
		m.pl.resolve(hb, g.sit, &m.def)
		g.fieldPlay()
		if m.pl.result != resSacFly {
			g.atBats[i][k]++
		}
		if isHit(m.pl.result) {
			g.hits[i][k]++
		}
		m.playDone()
		if wantQuit && !m.over {
			g.batView()
			g.askQuit()
		}
		return
	}
	ev := m.pitchDone(res)
	if ev == evStrikeOut {
		g.atBats[i][k]++
	}
	g.clean()
	g.players(0, 0)
	g.catcherSprite()
	switch ev {
	case evBall:
		g.message("BALL", picopad.RGB(60, 220, 90))
	case evStrike:
		g.message("STRIKE!", picopad.RGB(250, 210, 40))
	case evFoul:
		g.message("FOUL", picopad.ColWhite)
	case evStrikeOut:
		g.message("STRIKE OUT", picopad.RGB(240, 60, 60))
	case evWalk:
		g.message("BALL FOUR", picopad.RGB(60, 220, 90))
	}
	g.hud()
	g.pitcherBox()
	g.DispUpdate()
	g.WaitMs(900)
	if wantQuit && !m.over {
		g.clean()
		g.askQuit()
	}
}

// hitAway shows the ball leaving the bat, toward the field.
func (g *Game) hitAway(hb hitBall) {
	t0 := g.Time()
	dx := hb.dir / 45 * 150
	up := 60 + hb.launch*2
	for {
		t := g.seconds(t0)
		if t > 0.5 {
			return
		}
		g.clean()
		g.players(2, 2)
		g.catcherSprite()
		f := t / 0.5
		x := zoneCX + int(dx*f)
		y := zoneCY - int(up*f) + int(80*f*f*(1-f))
		g.ball(x, y, 4-int(3*f))
		g.frame()
	}
}

// fieldPlay shows the ball in play from above.
func (g *Game) fieldPlay() {
	m := &g.m
	pl := &m.pl
	g.showBg(fieldBgImg, fieldBgImgPal)
	g.hud()
	end := max(pl.end+0.8, 2.5)
	if pl.fl.hr {
		end = max(end, float32(pl.fl.n)*sampleT)
	}
	t0 := g.Time()
	for {
		t := g.seconds(t0) * 1.2 // a little faster than life
		g.clean()
		g.fielders(t)
		g.runners(t)
		g.fieldBall(t)
		if t > pl.end || pl.fl.hr && t > float32(pl.fl.hrAt)*sampleT {
			g.resultText(pl.result)
		}
		g.frame()
		if t > end {
			break
		}
		if g.KeyGet() == picopad.KeyA && t > 1 {
			break
		}
	}
	g.resultText(pl.result)
	g.DispUpdate()
	g.WaitMs(700)
}

// resultText writes the result of the play at the top of the field view.
func (g *Game) resultText(r int) {
	text := resultText[r]
	col := picopad.ColWhite
	if isHit(r) {
		col = picopad.ColYellow
	}
	g.SelFont8x16()
	x := (picopad.Width - len(text)*8) / 2
	g.DrawRect(x-4, 40, len(text)*8+8, 20, picopad.RGB(16, 22, 48))
	g.DrawText(text, x, 42, col)
	g.mark(x-4, 40, len(text)*8+8, 20)
}

// fielders draws the fielders at time t.
func (g *Game) fielders(t float32) {
	m := &g.m
	pl := &m.pl
	pal := g.pals[m.fielding()].fielder
	for pos := 0; pos < numPos; pos++ {
		p := spot[pos]
		target := pl.cover[pos]
		if pos == pl.who {
			if span := pl.fieldT - reaction(pos); t > reaction(pos) && span > 0 {
				p = lerp(spot[pos], pl.fieldP, clamp((t-reaction(pos))/span, 0, 1))
			}
		} else if d := dist(spot[pos], target); d > 0 && t > 0.3 {
			p = lerp(spot[pos], target, clamp((t-0.3)*m.def[pos].speed/d, 0, 1))
		}
		x, y := fieldToScreen(p)
		g.sprite(fielderImg, pal, 0, fielderW, fielderH, 2, x-5, y-5)
	}
}

// runners draws the runners at time t.
func (g *Game) runners(t float32) {
	m := &g.m
	pl := &m.pl
	pal := g.pals[m.batting()].fielder
	moving := [5]bool{} // the bases whose runners move (0: the batter)
	for i := 0; i < pl.nRuns; i++ {
		r := &pl.runs[i]
		moving[r.from] = true
		if r.out && t > r.t1+0.6 {
			continue
		}
		s := float32(r.from)
		if t > r.t0 {
			s += float32(r.to-r.from) * clamp((t-r.t0)/(r.t1-r.t0), 0, 1)
		}
		if s >= 4 && t > r.t1+0.5 {
			continue // home
		}
		b := min(int(s), 3)
		p := lerp(basePos[b], basePos[(b+1)%4], s-float32(b))
		x, y := fieldToScreen(p)
		g.sprite(fielderImg, pal, 1, fielderW, fielderH, 2, x-5, y-5)
	}
	for b := 0; b < 3; b++ {
		if g.sit.bases[b].on && !moving[b+1] {
			x, y := fieldToScreen(basePos[b+1])
			g.sprite(fielderImg, pal, 1, fielderW, fielderH, 2, x-5, y-5)
		}
	}
}

// fieldBall draws the ball (and its shadow) at time t.
func (g *Game) fieldBall(t float32) {
	pl := &g.m.pl
	fl := &pl.fl
	if t < pl.fieldT || pl.who < 0 || pl.fl.hr {
		k := min(int(t/sampleT), fl.n-1)
		s := fl.s[k]
		x, y := fieldToScreen(s.p)
		g.DrawRect(x-1, y, 3, 2, picopad.RGB(30, 70, 40))
		g.mark(x-1, y, 3, 2)
		g.ball(x, y-int(s.z*0.35), 1+min(int(s.z/40), 2))
		return
	}
	for i := 0; i < pl.nThrows; i++ {
		th := &pl.throws[i]
		if t >= th.t0 && t < th.t1 {
			p := lerp(th.from, th.to, (t-th.t0)/(th.t1-th.t0))
			x, y := fieldToScreen(p)
			g.ball(x, y-3, 1)
			return
		}
	}
}
