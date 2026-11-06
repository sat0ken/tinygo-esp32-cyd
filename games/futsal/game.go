// Package futsal is a five-a-side football game (futsal) on one screen,
// seen from above: SKY HAWKS against RED FOXES, a goalkeeper and four
// outfield players each. It runs on package picopad (320x240, the
// PicoPad's buttons or the touch panel).
//
// The player controls the outfield player under the yellow mark (the one
// with the ball, or the one nearest to it when the other team wins it).
// The arrows run; with the ball A passes (to the teammate the arrows
// point at) and B shoots: hold B longer for a harder shot, UP or DOWN as
// it is let go aim high or low in the goal. Without the ball B tackles and
// A changes to the player nearest the ball. The computer plays the others.
//
// Boards stand round the pitch: the ball bounces off them and stays in
// play. Two halves (1, 2 or 3 minutes each, shown as 20); the teams change
// ends at half time. Y quits the game.
package futsal

import (
	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/picopad"
)

//go:generate go run ./artgen

// dt is the time of a frame (s).
const (
	frameMs = 33
	dt      = 1.0 / 30
)

// Game is the futsal game on a console.
type Game struct {
	*picopad.Console

	m       match
	halfLen int  // seconds: the setting of the title screen
	watch   bool // the computer against the computer

	pals     [2][2]picopad.Palette // each team's outfield players and keeper
	drawn    [24]rect
	nDrawn   int
	charging bool // B held for a shot
	charge   float32
	hud      [3]int // what the score board shows (the scores, the clock)
	last     uint32 // the time of the last frame (µs)
	quit     bool
}

// New returns the game on d (at least 320x240) with the touch panel t;
// seed starts the random numbers. Set Pad to use a game controller.
func New(d hal.Display, t hal.Touch, seed uint32) *Game {
	g := &Game{Console: picopad.New(d, t, seed), halfLen: 120}
	for i := range teams {
		t := &teams[i]
		g.pals[i] = [2]picopad.Palette{recolor(t.color, t.dark), recolor(t.gkCol, t.gkDark)}
	}
	return g
}

// recolor returns the players' palette with the stand-in shirt colours
// replaced by col and dark.
func recolor(col, dark picopad.Color) picopad.Palette {
	b := []byte(playerImgPal)
	for i := 0; i+1 < len(b); i += 2 {
		c := picopad.Color(b[i]) | picopad.Color(b[i+1])<<8
		switch c {
		case standInTeam:
			c = col
		case standInDark:
			c = dark
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

// frame shows the frame and waits for the next.
func (g *Game) frame() {
	g.DispUpdate()
	if d := g.Time() - g.last; d < frameMs*1000 {
		g.WaitMs(int(frameMs - d/1000))
	}
	g.last = g.Time()
}

// showPitch draws the whole pitch and the score board.
func (g *Game) showPitch() {
	g.DrawImg4Pal(pitchImg, pitchImgPal, 0, 0, 0, 0, picopad.Width, picopad.Height, picopad.Width)
	g.nDrawn = 0
	g.hud = [3]int{-1, -1, -1}
	g.scoreBoard()
}

// The lengths of a half to choose from (s).
var halfChoices = [...]int{60, 120, 180}

// title is the title screen; it returns false on Y.
func (g *Game) title() bool {
	g.KeyFlush()
	redraw := true
	for {
		if redraw {
			g.DrawImg4Pal(pitchImg, pitchImgPal, 0, 0, 0, 0, picopad.Width, picopad.Height, picopad.Width)
			g.SelFont8x16()
			g.box(56, 6, 208, 52)
			g.DrawText2("FUTSAL", 112, 10, picopad.ColYellow)
			g.SelFont8x8()
			g.DrawText("5-A-SIDE SOCCER", 100, 45, picopad.ColWhite)
			g.DrawRect(40, 66, 240, 14, picopad.RGB(16, 22, 48))
			g.DrawText(teams[0].name, 48, 69, teams[0].color)
			g.DrawText("vs", 152, 69, picopad.ColYellow)
			g.DrawText(teams[1].name, 200, 69, teams[1].color)

			g.box(70, 88, 180, 74)
			g.SelFont8x16()
			modes := [2]string{"1P vs CPU", "CPU vs CPU"}
			for i, s := range modes {
				col := picopad.RGB(140, 150, 180)
				if g.watch == (i == 1) {
					col = picopad.ColYellow
					g.DrawText(">", 82, 94+i*18, col)
				}
				g.DrawText(s, 98, 94+i*18, col)
			}
			g.DrawText("HALF", 98, 136, picopad.ColWhite)
			g.DrawText("<", 146, 136, picopad.ColWhite)
			g.num(g.halfLen/60, 162, 136, picopad.ColYellow)
			g.DrawText("MIN >", 178, 136, picopad.ColWhite)

			g.SelFont8x8()
			g.box(2, 172, 316, 66)
			g.DrawText("arrows run   A pass   B shoot (hold)", 8, 177, picopad.ColWhite)
			g.DrawText("no ball: B tackle, A change player", 8, 188, picopad.ColWhite)
			g.DrawText("UP/DOWN as B is let go: high/low", 8, 199, picopad.RGB(150, 170, 220))
			g.DrawText("A: KICK OFF   Y: QUIT", 8, 213, picopad.ColYellow)
			g.DrawText("UP/DOWN mode  LEFT/RIGHT half", 8, 225, picopad.RGB(150, 170, 220))
			g.DispUpdate()
			redraw = false
		}
		switch g.KeyGet() {
		case picopad.KeyUp, picopad.KeyDown:
			g.watch = !g.watch
			redraw = true
		case picopad.KeyLeft, picopad.KeyRight:
			i := 0
			for k, n := range halfChoices {
				if n == g.halfLen {
					i = k
				}
			}
			if g.KeyPressed(picopad.KeyLeft) {
				i = (i + len(halfChoices) - 1) % len(halfChoices)
			} else {
				i = (i + 1) % len(halfChoices)
			}
			g.halfLen = halfChoices[i]
			redraw = true
		case picopad.KeyA:
			return true
		case picopad.KeyY:
			return false
		}
		g.WaitMs(20)
	}
}

// play plays a match.
func (g *Game) play() {
	m := &g.m
	m.start(g, float32(g.halfLen), !g.watch, false)
	g.quit, g.charging = false, false
	g.KeyFlush()
	g.showPitch()
	g.last = g.Time()
	for {
		g.input()
		if g.quit {
			return
		}
		m.step(dt)
		switch m.state {
		case stHalfTime:
			g.scene()
			g.frame()
			g.WaitMs(600)
			g.board("HALF TIME")
			m.secondHalf()
			g.showPitch()
			continue
		case stOver:
			g.scene()
			g.frame()
			g.WaitMs(600)
			switch {
			case !m.human[0]:
				g.board("FULL TIME")
			case m.score[0] > m.score[1]:
				g.board("YOU WIN!")
			case m.score[0] < m.score[1]:
				g.board("YOU LOSE")
			default:
				g.board("DRAW")
			}
			return
		}
		g.scene()
		g.frame()
	}
}

// input reads the buttons into the human team's input.
func (g *Game) input() {
	m := &g.m
	in := &m.in[0]
	has := m.b.owner == m.ctl[0]
	for k := g.KeyGet(); k != picopad.NoKey; k = g.KeyGet() {
		switch {
		case k == picopad.KeyY:
			if g.askQuit() {
				return
			}
		case !m.human[0]:
		case k == picopad.KeyA:
			in.pass = true
		case k == picopad.KeyB && has:
			g.charging, g.charge = true, 0
		case k == picopad.KeyB:
			in.tackle = true
		}
	}
	if !m.human[0] {
		return
	}
	var mv vec
	if g.KeyPressed(picopad.KeyLeft) {
		mv.x--
	}
	if g.KeyPressed(picopad.KeyRight) {
		mv.x++
	}
	if g.KeyPressed(picopad.KeyUp) {
		mv.y--
	}
	if g.KeyPressed(picopad.KeyDown) {
		mv.y++
	}
	in.move = mv.norm()
	if g.charging {
		switch {
		case !has || m.state != stPlay:
			g.charging = false
		case !g.KeyPressed(picopad.KeyB) || g.charge >= 0.6:
			in.shoot, in.power = true, clamp(g.charge/0.6, 0.2, 1)
			in.move = mv // the aim, not normalised: UP/DOWN alone is -1/1
			g.charging = false
		default:
			g.charge += dt
		}
	}
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
			g.last = g.Time()
			return false
		}
		g.WaitMs(20)
	}
}

// board shows the score under a title, until A or a while.
func (g *Game) board(title string) {
	m := &g.m
	g.KeyFlush()
	g.SelFont8x16()
	g.box(40, 50, 240, 140)
	g.DrawText2(title, (picopad.Width-len(title)*16)/2, 58, picopad.ColYellow)
	for i := 0; i < 2; i++ {
		y := 100 + i*24
		t := m.t[i]
		g.DrawRect(56, y+2, 6, 12, t.color)
		g.DrawText(t.name, 70, y, picopad.ColWhite)
		g.num(m.score[i], 236, y, picopad.ColYellow)
	}
	g.SelFont8x8()
	g.DrawText("SHOTS", 70, 150, picopad.RGB(150, 170, 220))
	x := g.num(m.shots[0], 140, 150, picopad.ColWhite)
	g.DrawText("-", x+4, 150, picopad.ColWhite)
	g.num(m.shots[1], x+16, 150, picopad.ColWhite)
	g.DrawText("A: GO ON", 128, 172, picopad.RGB(150, 170, 220))
	g.DispUpdate()
	t0 := g.Time()
	for g.Time()-t0 < 5e6 {
		if g.KeyGet() == picopad.KeyA {
			break
		}
		g.WaitMs(20)
	}
	g.KeyFlush()
	g.last = g.Time()
}
