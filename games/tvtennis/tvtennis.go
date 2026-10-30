// Package tvtennis is the TV Tennis of PicoLibSDK (PicoPad/GAME/TVTENNIS,
// by Miroslav Nemecek, Unlicense), ported from C to package picopad. It
// follows the C source function by function; the C globals are the fields
// of Game.
//
// The first screen chooses: LEFT plays with the left paddle against the
// computer, RIGHT with the right one, DOWN is two players, UP a demo, A
// one player with both paddles tied. In the game UP/DOWN move the left
// paddle, X/A the right one, LEFT, RIGHT or B serves, Y goes back to the
// first screen; Y there ends the game and Run returns (on the PicoPad:
// back to the boot loader). There is no sound yet: the PLAYSOUND calls of
// the C source are comments.
package tvtennis

import (
	"math"

	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/picopad"
)

const (
	segm     = 4                              // base segment size in pixels
	digW     = 4                              // digit width in segments
	digH     = 7                              // digit height in segments
	digSeg   = segm                           // size of a digit segment in pixels
	numW     = (digW + 2) * digSeg            // width of a digit in pixels (24)
	maxScore = 99                             // max. score
	netW     = 2 * segm                       // width of the net
	paddleW  = 2 * segm                       // paddle width
	paddleH  = 12 * segm                      // paddle height
	ballSize = 2 * segm                       // ball size
	paddleLX = segm                           // left paddle X coordinate
	paddleRX = picopad.Width - segm - paddleW // right paddle X coordinate
	width    = picopad.Width                  // WIDTH
	height   = picopad.Height                 // HEIGHT
	pi       = float32(math.Pi)               // PI

	paddleSpeed = 0.00025 // paddle speed [pixels per µs]
	ballSpeed   = 0.0003  // ball speed [pixels per µs]

	ballIn = 0 // the ball is in play
	ballL  = 1 // the left player serves
	ballR  = 2 // the right player serves
)

// digits are the patterns of the digits, 4 bits a row.
var digits = [10 * digH]uint8{
	// 0
	0b1111, 0b1001, 0b1001, 0b1001, 0b1001, 0b1001, 0b1111,
	// 1
	0b0001, 0b0001, 0b0001, 0b0001, 0b0001, 0b0001, 0b0001,
	// 2
	0b1111, 0b0001, 0b0001, 0b1111, 0b1000, 0b1000, 0b1111,
	// 3
	0b1111, 0b0001, 0b0001, 0b0111, 0b0001, 0b0001, 0b1111,
	// 4
	0b1001, 0b1001, 0b1001, 0b1111, 0b0001, 0b0001, 0b0001,
	// 5
	0b1111, 0b1000, 0b1000, 0b1111, 0b0001, 0b0001, 0b1111,
	// 6
	0b1111, 0b1000, 0b1000, 0b1111, 0b1001, 0b1001, 0b1111,
	// 7
	0b1111, 0b0001, 0b0001, 0b0001, 0b0001, 0b0001, 0b0001,
	// 8
	0b1111, 0b1001, 0b1001, 0b1111, 0b1001, 0b1001, 0b1111,
	// 9
	0b1111, 0b1001, 0b1001, 0b1111, 0b0001, 0b0001, 0b1111,
}

// Game is TV Tennis on a console.
type Game struct {
	*picopad.Console

	oldTime   uint32 // time of the last step
	timeDelta int    // time since the last step [µs]

	comp    [2]bool    // the computer plays the paddle
	compX   [2]float32 // X coordinate of the computer's delayed response
	compY   [2]float32 // Y coordinate of the computer's inaccuracy
	lastOut uint32     // last time out (to serve again)

	score       [2]uint8
	scoreUpdate bool
	tied        bool // the paddles are tied

	paddleY      [2]float32 // paddle Y coordinate (middle)
	paddleUpdate [2]bool
	oldPaddleY   [2]float32

	ballState             uint8 // ball*
	ballX, ballY, ballDir float32
	ballUpdate            bool
	oldBallX, oldBallY    float32
}

// New returns the game on d (at least 320x240) with the touch panel t;
// seed starts the random numbers. Set Pad to use a game controller.
func New(d hal.Display, t hal.Touch, seed uint32) *Game {
	return &Game{Console: picopad.New(d, t, seed)}
}

func cosf(x float32) float32 { return float32(math.Cos(float64(x))) }
func sinf(x float32) float32 { return float32(math.Sin(float64(x))) }

// newAuto prepares the computer's play.
func (g *Game) newAuto() {
	g.compX[0] = width * (0.25 + g.RandFloat()*0.5)
	g.compY[0] = 1.04 * (paddleH + ballSize) * (g.RandFloat() - 0.5)

	g.compX[1] = width * (0.25 + g.RandFloat()*0.5)
	g.compY[1] = 1.04 * (paddleH + ballSize) * (g.RandFloat() - 0.5)
}

// newGame starts a new game.
func (g *Game) newGame() {
	for i := 0; i < 2; i++ {
		g.score[i] = 0
		g.paddleY[i] = height / 2
	}
	g.ballState = ballL
	g.lastOut = g.Time()
	g.ballY = height / 2
	g.newAuto()
}

// dispNet displays the net.
func (g *Game) dispNet(h int) {
	g.DrawRect((width-netW)/2, 0, netW, h, picopad.RGB(0, 186, 255))
}

// dispScore displays the score of player inx.
func (g *Game) dispScore(inx int) {
	dig := [2]int{int(g.score[inx]) / 10, int(g.score[inx]) % 10}

	x := (width+netW)/2 + 2*digSeg
	if inx == 0 {
		x = (width-netW)/2 - 2*numW
	}

	for j := 0; j < 2; j++ {
		s := digits[dig[j]*digH:]
		for row := 0; row < digH; row++ {
			k := s[row]
			for col := 0; col < digW; col++ {
				c := picopad.ColBlack
				if k&0b1000 != 0 {
					c = picopad.ColGreen
				}
				g.DrawRect(x+col*digSeg, row*digSeg, digSeg, digSeg, c)
				k <<= 1
			}
		}
		x += numW
	}
}

// dispScores displays both scores.
func (g *Game) dispScores() {
	g.dispScore(0)
	g.dispScore(1)
}

// paddleX returns the X coordinate of paddle inx.
func paddleX(inx int) int {
	if inx == 0 {
		return paddleLX
	}
	return paddleRX
}

// clearPaddle clears paddle inx at its old place.
func (g *Game) clearPaddle(inx int) {
	g.DrawRect(paddleX(inx), int(g.oldPaddleY[inx])-paddleH/2, paddleW, paddleH, picopad.ColBlack)
}

// dispPaddle displays paddle inx.
func (g *Game) dispPaddle(inx int) {
	g.DrawRect(paddleX(inx), int(g.paddleY[inx])-paddleH/2, paddleW, paddleH, picopad.ColYellow)
}

// dispPaddles displays both paddles.
func (g *Game) dispPaddles() {
	g.dispPaddle(0)
	g.dispPaddle(1)
}

// clearBall clears the ball at its old place.
func (g *Game) clearBall() {
	g.DrawRect(int(g.oldBallX)-ballSize/2, int(g.oldBallY)-ballSize/2, ballSize, ballSize, picopad.ColBlack)
}

// dispBall displays the ball.
func (g *Game) dispBall() {
	if g.ballState == ballIn {
		g.DrawRect(int(g.ballX)-ballSize/2, int(g.ballY)-ballSize/2, ballSize, ballSize, picopad.ColRed)
	}
}

// dispAll displays everything.
func (g *Game) dispAll() {
	g.DrawClear()
	g.dispNet(height)
	g.dispScores()
	g.dispPaddles()
	g.dispBall()
	g.DispUpdate()
}

// gameControl moves the paddles: the keys and the computer.
func (g *Game) gameControl() {
	key0, key1 := picopad.NoKey, picopad.NoKey

	g.oldPaddleY = g.paddleY

	// the computer
	if g.comp[0] && g.ballX <= g.compX[0] {
		autoY := g.ballY + g.compY[0]
		if g.paddleY[0] < autoY {
			key0 = picopad.KeyDown
		}
		if g.paddleY[0] > autoY {
			key0 = picopad.KeyUp
		}
	}
	if g.comp[1] && g.ballX >= g.compX[1] {
		autoY := g.ballY + g.compY[1]
		if g.paddleY[1] < autoY {
			key1 = picopad.KeyDown
		}
		if g.paddleY[1] > autoY {
			key1 = picopad.KeyUp
		}
	}

	step := float32(g.timeDelta) * paddleSpeed

	// left player up
	if g.KeyPressed(picopad.KeyUp) && !g.comp[0] || (g.tied || g.comp[1]) && g.KeyPressed(picopad.KeyX) && !g.comp[0] || key0 == picopad.KeyUp {
		g.paddleY[0] = max(g.paddleY[0]-step, paddleH/2)
		g.paddleUpdate[0] = true
	}

	// left player down
	if g.KeyPressed(picopad.KeyDown) && !g.comp[0] || (g.tied || g.comp[1]) && g.KeyPressed(picopad.KeyA) && !g.comp[0] || key0 == picopad.KeyDown {
		g.paddleY[0] = min(g.paddleY[0]+step, height-paddleH/2)
		g.paddleUpdate[0] = true
	}

	// right player up
	if g.KeyPressed(picopad.KeyX) && !g.comp[1] || (g.tied || g.comp[0]) && g.KeyPressed(picopad.KeyUp) && !g.comp[1] || key1 == picopad.KeyUp {
		g.paddleY[1] = max(g.paddleY[1]-step, paddleH/2)
		g.paddleUpdate[1] = true
	}

	// right player down
	if g.KeyPressed(picopad.KeyA) && !g.comp[1] || (g.tied || g.comp[0]) && g.KeyPressed(picopad.KeyDown) && !g.comp[1] || key1 == picopad.KeyDown {
		g.paddleY[1] = min(g.paddleY[1]+step, height-paddleH/2)
		g.paddleUpdate[1] = true
	}
}

// gameBall moves the ball.
func (g *Game) gameBall() {
	if g.ballState != ballIn {
		return
	}
	d := float32(g.timeDelta) * ballSpeed

	g.oldBallX = g.ballX
	g.oldBallY = g.ballY

	ballX := g.ballX + d*cosf(g.ballDir)
	ballY := g.ballY - d*sinf(g.ballDir)

	// bounce off the top edge
	if ballY <= ballSize/2 {
		ballY = ballSize / 2
		g.ballDir = -g.ballDir
		// PLAYSOUND(BoundSnd)
	}

	// bounce off the bottom edge
	if ballY >= height-ballSize/2 {
		ballY = height - ballSize/2
		g.ballDir = -g.ballDir
		// PLAYSOUND(BoundSnd)
	}

	// bounce off the left paddle (the old X coordinate, in case the step
	// is large)
	if ballX-ballSize/2 <= paddleLX+paddleW &&
		g.ballX+ballSize/2 >= paddleLX &&
		ballY-ballSize/2 <= g.paddleY[0]+paddleH/2 &&
		ballY+ballSize/2 >= g.paddleY[0]-paddleH/2 {
		ballX = paddleLX + paddleW + ballSize/2
		g.ballDir = pi / 4 * (g.paddleY[0] - ballY) / (paddleH / 2)
		// PLAYSOUND(PaddleSnd)
		g.newAuto()
	}

	// bounce off the right paddle
	if ballX+ballSize/2 >= paddleRX &&
		g.ballX-ballSize/2 <= paddleRX+paddleW &&
		ballY-ballSize/2 <= g.paddleY[1]+paddleH/2 &&
		ballY+ballSize/2 >= g.paddleY[1]-paddleH/2 {
		ballX = paddleRX - ballSize/2
		g.ballDir = pi - pi/4*(g.paddleY[1]-ballY)/(paddleH/2)
		// PLAYSOUND(PaddleSnd)
		g.newAuto()
	}

	// out on the left
	if ballX < -ballSize {
		if g.score[1] < maxScore {
			g.score[1]++
		}
		g.scoreUpdate = true
		g.ballState = ballL
		// PLAYSOUND(OutSnd)
		g.lastOut = g.Time()
		ballY = height / 2
		g.newAuto()
	}

	// out on the right
	if ballX > width+ballSize {
		if g.score[0] < maxScore {
			g.score[0]++
		}
		g.scoreUpdate = true
		g.ballState = ballR
		// PLAYSOUND(OutSnd)
		g.lastOut = g.Time()
		ballY = height / 2
		g.newAuto()
	}

	g.ballX = ballX
	g.ballY = ballY
	g.ballUpdate = true
}

// gameUpdate draws what changed.
func (g *Game) gameUpdate() {
	if g.ballUpdate {
		switch {
		case g.ballY < digH*segm+ballSize &&
			g.ballX >= width/2-netW-2*numW-ballSize &&
			g.ballX <= width/2+netW+2*numW+ballSize: // near the score
			g.clearBall()
			g.dispScores()
			g.scoreUpdate = false
			g.dispNet(10 * segm)

		case g.ballX >= (width-netW)/2-2*ballSize &&
			g.ballX <= (width+netW)/2+2*ballSize: // near the net
			g.clearBall()
			g.dispNet(height)

		case g.ballX <= paddleLX+paddleW+ballSize &&
			g.ballY >= g.paddleY[0]-paddleH/2-ballSize &&
			g.ballY <= g.paddleY[0]+paddleH/2+ballSize: // near the left paddle
			g.clearBall()
			if g.paddleUpdate[0] {
				g.clearPaddle(0)
			}
			g.dispPaddle(0)
			g.paddleUpdate[0] = false

		case g.ballX >= paddleRX-ballSize &&
			g.ballY >= g.paddleY[1]-paddleH/2-ballSize &&
			g.ballY <= g.paddleY[1]+paddleH/2+ballSize: // near the right paddle
			g.clearBall()
			if g.paddleUpdate[1] {
				g.clearPaddle(1)
			}
			g.dispPaddle(1)
			g.paddleUpdate[1] = false

		default:
			g.clearBall()
		}
		g.dispBall()
		g.DispUpdate()
		g.ballUpdate = false
	}

	for inx := 0; inx < 2; inx++ {
		if g.paddleUpdate[inx] {
			g.clearPaddle(inx)
			g.dispPaddle(inx)
			g.paddleUpdate[inx] = false
			g.DispUpdate()
		}
	}

	if g.scoreUpdate {
		g.dispScores()
		g.scoreUpdate = false
		g.DispUpdate()
	}
}

// openDraw draws the first screen.
func (g *Game) openDraw() {
	g.SelFont8x16()
	g.DrawClear()
	g.DrawText2("TV Tennis", (width-9*16)/2, 5, picopad.ColYellow)

	const (
		menuX  = 10
		menuY  = 45
		menuDY = 16
		menuY2 = menuY + 7*menuDY
	)
	g.DrawText("Press key to start the game:", menuX, menuY+0*menuDY, picopad.ColGreen)
	g.DrawText("LEFT ...... Play with left paddle", menuX, menuY+1*menuDY, picopad.ColWhite)
	g.DrawText("RIGHT ..... Play with right paddle", menuX, menuY+2*menuDY, picopad.ColWhite)
	g.DrawText("DOWN ...... 2 players", menuX, menuY+3*menuDY, picopad.ColWhite)
	g.DrawText("UP ........ Demo", menuX, menuY+4*menuDY, picopad.ColWhite)
	g.DrawText("A ......... Play with tied paddles", menuX, menuY+5*menuDY, picopad.ColWhite)

	g.DrawText("Game control:", menuX, menuY2+0*menuDY, picopad.ColGreen)
	g.DrawText("LEFT or RIGHT ... Throw the ball", menuX, menuY2+1*menuDY, picopad.ColWhite)
	g.DrawText("UP/DOWN ......... Left paddle", menuX, menuY2+2*menuDY, picopad.ColWhite)
	g.DrawText("X/A ............. Right paddle", menuX, menuY2+3*menuDY, picopad.ColWhite)
	g.DrawText("Y ............... Quit the game", menuX, menuY2+4*menuDY, picopad.ColWhite)

	g.DispUpdate()
}

// game plays a game; comp1 and comp2 are the paddles the computer plays.
func (g *Game) game(comp1, comp2, tied bool) {
	g.comp = [2]bool{comp1, comp2}
	g.tied = tied

	g.newGame()

	// wait for the keys to be released
	g.dispAll()
	for g.KeyPressed(picopad.KeyLeft) || g.KeyPressed(picopad.KeyRight) ||
		g.KeyPressed(picopad.KeyUp) || g.KeyPressed(picopad.KeyDown) ||
		g.KeyPressed(picopad.KeyA) || g.KeyPressed(picopad.KeyX) {
	}
	g.KeyFlush()

	g.oldTime = g.Time()
	for {
		ch := g.KeyGet()

		// the computer serves
		if ch == picopad.NoKey && g.Time()-g.lastOut >= 1000000 {
			if g.ballState == ballL && g.comp[0] || g.ballState == ballR && g.comp[1] {
				ch = picopad.KeyLeft
			}
		}

		switch ch {
		case picopad.KeyY:
			g.KeyFlush()
			g.openDraw()
			return

		case picopad.KeyLeft, picopad.KeyRight, picopad.KeyB: // serve
			if g.ballState != ballIn {
				if g.ballState == ballL {
					g.ballX = paddleLX + paddleW + ballSize/2
					g.ballY = g.paddleY[0]
					g.ballDir = g.RandFloatMinMax(-pi/4, +pi/4)
				} else {
					g.ballX = paddleRX - ballSize/2
					g.ballY = g.paddleY[1]
					g.ballDir = g.RandFloatMinMax(pi*3/4, pi*5/4)
				}
				g.ballState = ballIn
				// PLAYSOUND(ServeSnd)
			}
			g.KeyFlush()

		case picopad.NoKey:

		default:
			g.KeyFlush()
		}

		t := g.Time()
		g.timeDelta = int(t - g.oldTime)
		g.oldTime = t

		g.gameControl()
		g.gameBall()
		g.gameUpdate()
	}
}

// Run shows the first screen and plays until Y there (Open in the C
// source, which then resets to the boot loader). With OneGame it returns
// after a game (Y: the game has no end).
func (g *Game) Run() {
	g.openDraw()
	for {
		played := true
		switch g.KeyGet() {
		case picopad.KeyLeft:
			g.game(false, true, false)
		case picopad.KeyRight:
			g.game(true, false, false)
		case picopad.KeyDown:
			g.game(false, false, false)
		case picopad.KeyUp:
			g.game(true, true, false)
		case picopad.KeyA:
			g.game(false, false, true)
		case picopad.KeyY:
			return
		default:
			played = false
		}
		if played && g.OneGame {
			return
		}
		g.WaitMs(10)
	}
}
