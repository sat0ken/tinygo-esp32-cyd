package main

import (
	"image/color"
	"math"
	"strconv"

	"github.com/sat0ken/tinygo-cyd/hal"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/proggy"
)

// Layout (480x272).
const (
	hudH = 20 // score bar at the top

	brickCols = 10
	brickRows = 5
	brickW    = 44
	brickH    = 14
	brickGap  = 2
	brickTop  = 36

	paddleW   = 72
	paddleH   = 8
	paddleGap = 18 // paddle top = screen height - paddleGap

	ballSize = 6

	lives0     = 3
	baseSpeed  = 220 // pixels per second at level 1
	levelBoost = 0.1 // +10% speed per level
	maxAngle   = 60  // degrees from vertical when the ball leaves the paddle
)

var (
	bgColor     = color.RGBA{10, 16, 40, 255}
	hudColor    = color.RGBA{0, 70, 140, 255}
	textColor   = color.RGBA{255, 255, 255, 255}
	paddleColor = color.RGBA{230, 230, 230, 255}
	ballColor   = color.RGBA{255, 255, 255, 255}
	msgColor    = color.RGBA{255, 220, 60, 255}
	rowColors   = [brickRows]color.RGBA{
		{230, 60, 60, 255},  // red
		{240, 140, 40, 255}, // orange
		{230, 210, 50, 255}, // yellow
		{70, 190, 80, 255},  // green
		{60, 130, 230, 255}, // blue
	}
	rowPoints = [brickRows]int{50, 40, 30, 20, 10}
)

type state int

const (
	ready    state = iota // ball on the paddle, waiting for a tap
	playing               // ball moving
	gameOver              // no lives left, tap to restart
	cleared               // all bricks gone, tap for the next level
)

type rect struct{ x, y, w, h int16 }

// Game is a breakout game. It depends only on hal, so it runs on the
// board, in the browser and in host tests.
type Game struct {
	d    hal.Display
	t    hal.Touch
	w, h int16

	state  state
	bricks [brickRows][brickCols]bool
	left   int // bricks left
	score  int
	lives  int
	level  int

	paddleX float32 // left edge
	ballX   float32 // top-left corner
	ballY   float32
	vx, vy  float32 // pixels per second

	wasTouch bool

	// What is on screen, to redraw only what changed.
	drawnPaddle int16
	drawnBall   rect
	ballShown   bool
	erased      []rect // bricks removed since the last Draw
	hudDirty    bool
	msgShown    string
}

// NewGame creates a game at level 1.
func NewGame(d hal.Display, t hal.Touch) *Game {
	w, h := d.Size()
	g := &Game{d: d, t: t, w: w, h: h}
	g.reset()
	return g
}

func (g *Game) reset() {
	g.score, g.lives, g.level = 0, lives0, 1
	g.newLevel()
}

func (g *Game) newLevel() {
	for r := range g.bricks {
		for c := range g.bricks[r] {
			g.bricks[r][c] = true
		}
	}
	g.left = brickRows * brickCols
	g.paddleX = float32(g.w-paddleW) / 2
	g.serve()
}

// serve puts the ball on the paddle.
func (g *Game) serve() {
	g.state = ready
	g.ballX = g.paddleX + (paddleW-ballSize)/2
	g.ballY = float32(g.paddleTop() - ballSize)
	g.vx, g.vy = 0, 0
}

func (g *Game) paddleTop() int16 { return g.h - paddleGap }

func (g *Game) speed() float32 {
	return baseSpeed * (1 + levelBoost*float32(g.level-1))
}

func (g *Game) brickRect(r, c int) rect {
	left := (g.w - (brickCols*brickW + (brickCols-1)*brickGap)) / 2
	return rect{
		x: left + int16(c)*(brickW+brickGap),
		y: brickTop + int16(r)*(brickH+brickGap),
		w: brickW, h: brickH,
	}
}

func overlap(ax, ay float32, aw, ah int16, b rect) bool {
	return ax < float32(b.x+b.w) && ax+float32(aw) > float32(b.x) &&
		ay < float32(b.y+b.h) && ay+float32(ah) > float32(b.y)
}

// Step reads the touch panel and advances the game by dt seconds.
func (g *Game) Step(dt float32) {
	x, _, pressed := g.t.ReadTouch()
	tap := pressed && !g.wasTouch
	g.wasTouch = pressed

	if pressed {
		// The paddle follows the finger.
		px := float32(x) - paddleW/2
		if px < 0 {
			px = 0
		}
		if max := float32(g.w - paddleW); px > max {
			px = max
		}
		g.paddleX = px
	}

	switch g.state {
	case ready:
		g.ballX = g.paddleX + (paddleW-ballSize)/2
		if tap {
			g.launch()
		}
	case playing:
		// Move in steps of at most 2 pixels so the ball cannot jump over
		// a brick or the paddle.
		dist := float32(math.Hypot(float64(g.vx), float64(g.vy))) * dt
		n := int(dist/2) + 1
		for i := 0; i < n && g.state == playing; i++ {
			g.move(dt / float32(n))
		}
	case gameOver:
		if tap {
			g.reset()
			g.hudDirty = true
			g.redrawBricks()
		}
	case cleared:
		if tap {
			g.level++
			g.newLevel()
			g.hudDirty = true
			g.redrawBricks()
		}
	}
}

func (g *Game) launch() {
	g.state = playing
	s := g.speed()
	a := 30 * math.Pi / 180 // up and to the right
	g.vx = s * float32(math.Sin(a))
	g.vy = -s * float32(math.Cos(a))
}

// move moves the ball one small step, one axis at a time, bouncing off
// walls, bricks and the paddle.
func (g *Game) move(dt float32) {
	// Horizontal.
	g.ballX += g.vx * dt
	if g.ballX < 0 {
		g.ballX, g.vx = -g.ballX, -g.vx
	}
	if max := float32(g.w - ballSize); g.ballX > max {
		g.ballX, g.vx = 2*max-g.ballX, -g.vx
	}
	if g.hitBrick() {
		g.ballX -= g.vx * dt
		g.vx = -g.vx
	}

	// Vertical.
	g.ballY += g.vy * dt
	if g.ballY < hudH {
		g.ballY, g.vy = 2*hudH-g.ballY, -g.vy
	}
	if g.hitBrick() {
		g.ballY -= g.vy * dt
		g.vy = -g.vy
	}

	// Paddle: only when falling onto its top.
	pt := g.paddleTop()
	if g.vy > 0 && overlap(g.ballX, g.ballY, ballSize, ballSize, rect{int16(g.paddleX), pt, paddleW, paddleH}) {
		// The angle depends on where the ball hits: centre = straight up.
		off := (g.ballX + ballSize/2 - (g.paddleX + paddleW/2)) / (paddleW / 2)
		if off < -1 {
			off = -1
		}
		if off > 1 {
			off = 1
		}
		a := float64(off) * maxAngle * math.Pi / 180
		s := g.speed()
		g.vx = s * float32(math.Sin(a))
		g.vy = -s * float32(math.Cos(a))
		g.ballY = float32(pt - ballSize)
	}

	// Missed.
	if g.ballY > float32(g.h) {
		g.lives--
		g.hudDirty = true
		if g.lives == 0 {
			g.state = gameOver
		} else {
			g.serve()
		}
	}
}

// hitBrick removes the first brick the ball overlaps and reports whether
// there was one.
func (g *Game) hitBrick() bool {
	for r := range g.bricks {
		for c := range g.bricks[r] {
			if !g.bricks[r][c] {
				continue
			}
			br := g.brickRect(r, c)
			if overlap(g.ballX, g.ballY, ballSize, ballSize, br) {
				g.bricks[r][c] = false
				g.left--
				g.score += rowPoints[r]
				g.erased = append(g.erased, br)
				g.hudDirty = true
				if g.left == 0 {
					g.state = cleared
				}
				return true
			}
		}
	}
	return false
}

// DrawAll draws the whole screen.
func (g *Game) DrawAll() {
	g.d.FillRectangle(0, 0, g.w, g.h, bgColor)
	g.redrawBricks()
	g.hudDirty = true
	g.ballShown = false
	g.drawnPaddle = -1
	g.msgShown = ""
	g.Draw()
}

func (g *Game) redrawBricks() {
	for r := range g.bricks {
		for c := range g.bricks[r] {
			br := g.brickRect(r, c)
			col := bgColor
			if g.bricks[r][c] {
				col = rowColors[r]
			}
			g.d.FillRectangle(br.x, br.y, br.w, br.h, col)
		}
	}
	g.erased = g.erased[:0]
}

// Draw updates only what changed since the last call.
func (g *Game) Draw() {
	d := g.d
	for _, e := range g.erased {
		d.FillRectangle(e.x, e.y, e.w, e.h, bgColor)
	}
	g.erased = g.erased[:0]

	if g.hudDirty {
		g.hudDirty = false
		d.FillRectangle(0, 0, g.w, hudH, hudColor)
		text := "SCORE " + strconv.Itoa(g.score) + "   LIVES " + strconv.Itoa(g.lives) + "   LEVEL " + strconv.Itoa(g.level)
		tinyfont.WriteLine(d, &proggy.TinySZ8pt7b, 8, 14, text, textColor)
	}

	g.drawPaddle()
	g.drawBall()
	g.drawMessage()
	d.Display()
}

// drawPaddle erases only the part the paddle left, so it does not flicker.
func (g *Game) drawPaddle() {
	x := int16(g.paddleX)
	if x == g.drawnPaddle {
		return
	}
	y := g.paddleTop()
	old := g.drawnPaddle
	if old >= 0 {
		if x > old {
			g.d.FillRectangle(old, y, min16(x-old, paddleW), paddleH, bgColor)
		} else {
			start := x + paddleW
			if start < old {
				start = old
			}
			g.d.FillRectangle(start, y, old+paddleW-start, paddleH, bgColor)
		}
	}
	g.d.FillRectangle(x, y, paddleW, paddleH, paddleColor)
	g.drawnPaddle = x
}

func (g *Game) drawBall() {
	show := g.state == ready || g.state == playing
	nb := rect{int16(g.ballX), int16(g.ballY), ballSize, ballSize}
	if g.ballShown && (!show || nb != g.drawnBall) {
		o := g.drawnBall
		g.d.FillRectangle(o.x, o.y, o.w, o.h, bgColor)
		// The old ball may have covered part of the paddle.
		if o.y+o.h > g.paddleTop() {
			g.drawnPaddle = -2 // force a full paddle redraw
			g.drawPaddle()
		}
		g.ballShown = false
	}
	if show && !g.ballShown && nb.y < g.h {
		g.d.FillRectangle(nb.x, nb.y, nb.w, nb.h, ballColor)
		g.drawnBall = nb
		g.ballShown = true
	}
}

// Message area between the bricks and the paddle.
func (g *Game) msgRect() rect { return rect{90, 170, g.w - 180, 30} }

func (g *Game) drawMessage() {
	var msg string
	switch g.state {
	case ready:
		msg = "TAP TO START"
	case gameOver:
		msg = "GAME OVER - TAP TO RETRY"
	case cleared:
		msg = "CLEAR! - TAP FOR LEVEL " + strconv.Itoa(g.level+1)
	}
	if msg == g.msgShown {
		return
	}
	r := g.msgRect()
	g.d.FillRectangle(r.x, r.y, r.w, r.h, bgColor)
	if msg != "" {
		_, tw := tinyfont.LineWidth(&proggy.TinySZ8pt7b, msg)
		tinyfont.WriteLine(g.d, &proggy.TinySZ8pt7b, r.x+(r.w-int16(tw))/2, r.y+19, msg, msgColor)
	}
	g.msgShown = msg
}

func min16(a, b int16) int16 {
	if a < b {
		return a
	}
	return b
}
