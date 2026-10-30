// Package eggs is the Eggs of PicoLibSDK (PicoPad/GAME/EGGS, by Miroslav
// Nemecek, Unlicense), ported from C to package picopad. It follows the C
// source function by function; the C globals are the fields of Game.
//
// A game like Reversi: a player puts a chicken next to a line of stones
// that ends with a chicken, and the stones on the line turn: for player 1
// hen -> chicken -> egg, for player 2 egg -> chicken -> hen. The first
// screen chooses: LEFT plays the eggs against the computer, RIGHT the
// hens, DOWN is two players, UP a demo; the arrows move the cursor, A puts
// a chicken, X shows the computer's choice, Y ends the game; Y on the
// first screen ends Run (on the PicoPad: back to the boot loader). There
// is no sound yet: the PLAYSOUND calls of the C source are comments.
package eggs

import (
	"strconv"

	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/picopad"
)

//go:generate go run ../../tools/picoimg -pkg eggs -root $PICOLIBSDK -o img.go PicoPad/GAME/EGGS/img/computer.cpp PicoPad/GAME/EGGS/img/frame.cpp PicoPad/GAME/EGGS/img/open.cpp PicoPad/GAME/EGGS/img/select.cpp PicoPad/GAME/EGGS/img/tiles.cpp

const (
	tileW     = 30 // tile width
	tileH     = 30 // tile height
	boardW    = 8  // game board width
	boardH    = 8  // game board height
	boardSize = boardW * boardH

	frameW = 80  // frame width
	frameH = 240 // frame height
	frameX = 240 // frame X coordinate
	frameY = 120 // player 2 offset

	compW = 12         // computer width
	compH = 10         // computer height
	compX = 8 + frameX // computer X offset
	compY = 11         // computer Y offset

	selectW = 80     // select width
	selectH = 50     // select height
	selectX = frameX // select X offset
	selectY = 0      // select Y offset

	scoreX  = 35 + frameX           // score X offset
	scoreY  = 64                    // score Y offset
	winX    = 35 + frameX           // win X offset
	winY    = 96                    // win Y offset
	scoreBg = picopad.Color(0x4A9F) // score background

	tileBlack    = 0
	tileEmpty    = 1
	tileDot      = 2 // a possible place
	tileEgg      = 3
	tileChicken  = 4
	tileHen      = 5
	tileEggWin   = 6
	tileHenWin   = 7
	tileEmpty2   = 8 // the tiles with the cursor frame
	tileHenWin2  = 14
	tileOffMin   = tileEmpty
	tileOffMax   = tileHenWin
	tileOnMin    = tileEmpty2
	tileOnMax    = tileHenWin2
	noPlayer     = 2
	winnerBlinks = 1000000000
)

// the directions
var (
	dirX = [8]int{1, 1, 0, -1, -1, -1, 0, 1}
	dirY = [8]int{0, 1, 1, 1, 0, -1, -1, -1}
)

// Game is Eggs on a console.
type Game struct {
	*picopad.Console

	board      [boardSize]uint8
	stones     [2]int  // stones (player 1: eggs, player 2: hens)
	wins       [2]int  // wins
	comp       [2]bool // the computer plays
	player     uint8   // player on turn (0 or 1, 2: none)
	curX, curY int     // cursor
	curBlink   uint8   // cursor blinking
	pos        int     // number of possible places
	free       int     // number of free places
}

// New returns the game on d (at least 320x240) with the touch panel t;
// seed starts the random numbers. Set Pad to use a game controller.
func New(d hal.Display, t hal.Touch, seed uint32) *Game {
	return &Game{Console: picopad.New(d, t, seed)}
}

// drawBoard draws the board and updates the display.
func (g *Game) drawBoard() {
	for i := 0; i < boardH; i++ {
		for j := 0; j < boardW; j++ {
			b := int(g.board[i*boardW+j])
			g.DrawImg4Pal(tilesImg, tilesImgPal, 0, b*tileH, j*tileW, i*tileH, tileW, tileH, tileW)
		}
	}
	g.DispUpdate()
}

// getBoard returns a tile (black out of the board).
func (g *Game) getBoard(x, y int) uint8 {
	if x < 0 || y < 0 || x >= boardW || y >= boardH {
		return tileBlack
	}
	return g.board[x+y*boardW]
}

// setBoard sets a tile.
func (g *Game) setBoard(x, y int, tile uint8) { g.board[x+y*boardW] = tile }

// curOn shows the cursor.
func (g *Game) curOn() {
	if c := g.getBoard(g.curX, g.curY); c >= tileOffMin && c <= tileOffMax {
		g.setBoard(g.curX, g.curY, c-tileOffMin+tileOnMin)
	}
	g.curBlink = 0
}

// curOff hides the cursor.
func (g *Game) curOff() {
	if c := g.getBoard(g.curX, g.curY); c >= tileOnMin && c <= tileOnMax {
		g.setBoard(g.curX, g.curY, c-tileOnMin+tileOffMin)
	}
}

// getStones1 returns the number of stones turned in one direction (the
// cursor must be off).
func (g *Game) getStones1(x, y, dx, dy int) int {
	steps, stones := 0, 0
	for {
		x += dx
		y += dy
		c := g.getBoard(x, y)
		if c == tileChicken {
			stones = steps // up to a chicken
		}
		steps++
		if c != tileEgg && c != tileChicken && c != tileHen {
			return stones
		}
	}
}

// getStones returns the number of stones turned in all directions.
func (g *Game) getStones(x, y int) int {
	stones := 0
	for i := range dirX {
		stones += g.getStones1(x, y, dirX[i], dirY[i])
	}
	return stones
}

// dispPos marks the possible places (the cursor must be off).
func (g *Game) dispPos() {
	g.pos = 0
	for y := 0; y < boardH; y++ {
		for x := 0; x < boardW; x++ {
			if c := g.getBoard(x, y); c == tileEmpty || c == tileDot {
				if g.getStones(x, y) > 0 {
					g.pos++
					g.setBoard(x, y, tileDot)
				} else {
					g.setBoard(x, y, tileEmpty)
				}
			}
		}
	}
}

// openScreen shows the first screen and returns the key pressed.
func (g *Game) openScreen() uint8 {
	g.DrawImg4Rle(openImg, openImgPal, 0, 0, picopad.Width, picopad.Height)
	g.DispUpdate()

	// PLAYSOUND(MappleSnd)

	for {
		switch ch := g.KeyGet(); ch {
		case picopad.KeyLeft, picopad.KeyRight, picopad.KeyUp, picopad.KeyDown, picopad.KeyY:
			// StopSound()
			return ch
		}
		g.WaitMs(10)
	}
}

// dispSel displays the players, with a frame on the one on turn.
func (g *Game) dispSel() {
	y := 0
	for play := 0; play < 2; play++ {
		g.DrawImg4Pal(frameImg, frameImgPal, 0, y, frameX, y, frameW, selectH, frameW)
		if g.comp[play] {
			g.DrawImg4Pal(computerImg, computerImgPal, 0, 0, compX, y+compY, compW, compH, compW)
		}
		if play == int(g.player) {
			g.DrawBlit4Pal(selectImg, selectImgPal, 0, 0, selectX, y+selectY, selectW, selectH, selectW, picopad.ColWhite)
		}
		y += frameY
	}
	g.DispUpdate()
}

// dispScore displays the stones.
func (g *Game) dispScore() {
	for play := 0; play < 2; play++ {
		g.DrawTextBg(strconv.Itoa(g.stones[play])+" ", scoreX, scoreY+play*frameY, picopad.ColWhite, scoreBg)
	}
	g.DispUpdate()
}

// dispWin displays the wins.
func (g *Game) dispWin() {
	for play := 0; play < 2; play++ {
		g.DrawTextBg(strconv.Itoa(g.wins[play])+"   ", winX, winY+play*frameY, picopad.ColWhite, scoreBg)
	}
	g.DispUpdate()
}

// openGame starts a game.
func (g *Game) openGame(comp1, comp2 bool) {
	for i := range g.board {
		g.board[i] = tileEmpty
	}
	g.board[3*8+3] = tileChicken
	g.board[3*8+4] = tileChicken
	g.board[4*8+3] = tileChicken
	g.board[4*8+4] = tileChicken
	g.drawBoard()

	g.DrawImg4Pal(frameImg, frameImgPal, 0, 0, frameX, 0, frameW, frameH, frameW)

	g.stones = [2]int{}
	g.comp = [2]bool{comp1, comp2}
	g.player = g.RandU8() & 1 // a random player starts
	g.curX, g.curY = 3, 3
	g.free = boardW*boardH - 4
	g.pos = 12

	g.dispSel()
	g.dispScore()
	g.dispWin()
}

// putStone1 turns the stones in one direction.
func (g *Game) putStone1(x, y, dx, dy int) {
	for stones := g.getStones1(x, y, dx, dy); stones > 0; stones-- {
		g.drawBoard()

		// PLAYSOUND(BeepSnd)
		g.WaitMs(100)

		x += dx
		y += dy
		c := g.getBoard(x, y)

		if g.player == 1 {
			// player 2: egg -> chicken -> hen
			switch c {
			case tileEgg:
				g.stones[0]--
				c = tileChicken
			case tileHen:
				g.stones[1]--
				g.stones[0]++
				c = tileEgg
			default:
				g.stones[1]++
				c = tileHen
			}
		} else {
			// player 1: hen -> chicken -> egg
			switch c {
			case tileEgg:
				g.stones[0]--
				g.stones[1]++
				c = tileHen
			case tileHen:
				g.stones[1]--
				c = tileChicken
			default:
				g.stones[0]++
				c = tileEgg
			}
		}

		g.setBoard(x, y, c)
		g.drawBoard()
		g.dispScore()
	}
}

// putStone puts a chicken at the cursor (the cursor must be off, on a
// dot).
func (g *Game) putStone() {
	// blink the cursor (leaving it off)
	for i := 5; i > 0; i-- {
		g.drawBoard()
		g.WaitMs(50)
		g.curOn()
		g.drawBoard()
		g.WaitMs(50)
		g.curOff()
	}
	g.drawBoard()

	g.setBoard(g.curX, g.curY, tileChicken)
	g.free--

	for i := range dirX {
		g.putStone1(g.curX, g.curY, dirX[i], dirY[i])
	}
}

// profit1 is the profit of a place in one direction.
func (g *Game) profit1(x, y, dx, dy int) int8 {
	profit := int8(0)
	for stones := g.getStones1(x, y, dx, dy); stones > 0; stones-- {
		x += dx
		y += dy
		c := g.getBoard(x, y)
		if g.player == 1 {
			// player 2 loses a hen, the other gains an egg
			if c == tileHen {
				profit -= 2
			} else {
				profit++
			}
		} else {
			// player 1 loses an egg, the other gains a hen
			if c == tileEgg {
				profit -= 2
			} else {
				profit++
			}
		}
	}
	return profit
}

// profit is the profit of a place.
func (g *Game) profit(x, y int) int8 {
	profit := int8(0)
	for i := range dirX {
		profit += g.profit1(x, y, dirX[i], dirY[i])
	}
	return profit
}

// findPos puts the cursor on the best place (the cursor must be off).
func (g *Game) findPos() {
	bestProfit := int8(-125)
	for y := 0; y < boardH; y++ {
		for x := 0; x < boardW; x++ {
			if g.getBoard(x, y) == tileDot {
				if p := g.profit(x, y); p > bestProfit || p == bestProfit && g.RandU8() < 80 {
					bestProfit = p
					g.curX, g.curY = x, y
				}
			}
		}
	}
}

// playGame plays a game; it returns when it ends (or Y).
func (g *Game) playGame(comp1, comp2 bool) {
	demo := comp1 && comp2

	g.openGame(comp1, comp2)
	g.dispPos()
	g.curOn()
	g.drawBoard()

	for g.pos > 0 && g.free > 0 {
		ch := g.KeyGet()

		if demo && ch != picopad.NoKey {
			return
		}

		// the computer
		if g.comp[g.player] {
			g.curOff()
			g.findPos()
			g.curOn()
			g.drawBoard()
			if !demo {
				g.KeyFlush()
			}
			ch = picopad.KeyA
		}

		switch ch {
		case picopad.KeyY:
			return

		case picopad.KeyRight:
			if g.curX < boardW-1 {
				g.curOff()
				g.curX++
				g.curOn()
				g.drawBoard()
			}

		case picopad.KeyUp:
			if g.curY > 0 {
				g.curOff()
				g.curY--
				g.curOn()
				g.drawBoard()
			}

		case picopad.KeyLeft:
			if g.curX > 0 {
				g.curOff()
				g.curX--
				g.curOn()
				g.drawBoard()
			}

		case picopad.KeyDown:
			if g.curY < boardH-1 {
				g.curOff()
				g.curY++
				g.curOn()
				g.drawBoard()
			}

		case picopad.KeyX: // help
			g.curOff()
			g.findPos()
			g.curOn()
			g.drawBoard()

		case picopad.KeyA: // put a chicken
			g.curOff()
			if g.getBoard(g.curX, g.curY) == tileDot {
				g.putStone()
				g.dispPos()
				g.player ^= 1
				g.dispSel()
				g.dispScore()
			}
			g.curOn()
			g.drawBoard()

		case picopad.NoKey:
			g.WaitMs(50)
		}

		// the blinking cursor
		g.curBlink++
		if g.curBlink == 5 {
			g.curOff()
			g.drawBoard()
		}
		if g.curBlink == 8 {
			g.curOn()
			g.drawBoard()
		}
	}

	g.curOff()
	g.player = noPlayer
	g.dispSel()

	blinks := winnerBlinks
	if demo {
		blinks = 10
	}

	switch {
	case g.stones[0] > g.stones[1]: // the eggs win
		g.showWinner(tileEgg, tileEggWin)
		g.wins[0]++
		g.dispWin()
		g.blinkPlayers(blinks, 0, noPlayer)

	case g.stones[1] > g.stones[0]: // the hens win
		g.showWinner(tileHen, tileHenWin)
		g.wins[1]++
		g.dispWin()
		g.blinkPlayers(blinks, 1, noPlayer)

	default: // a draw
		g.blinkPlayers(blinks, 0, 1)
	}
}

// showWinner turns the winner's stones into their winning tile, one
// after the other.
func (g *Game) showWinner(tile, win uint8) {
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			if g.getBoard(x, y) == tile {
				g.setBoard(x, y, win)
				// PLAYSOUND(Beep2Snd)
				g.drawBoard()
				g.WaitMs(200)
			}
		}
	}
}

// blinkPlayers frames player p1, then p2, n times or until a key.
func (g *Game) blinkPlayers(n int, p1, p2 uint8) {
	for ; n > 0; n-- {
		g.player = p1
		g.dispSel()
		g.drawBoard()
		g.WaitMs(200)
		if g.KeyGet() != picopad.NoKey {
			return
		}
		g.player = p2
		g.dispSel()
		g.drawBoard()
		g.WaitMs(200)
		if g.KeyGet() != picopad.NoKey {
			return
		}
	}
}

// Run plays until Y on the first screen (main of the C source, which then
// resets to the boot loader). With OneGame it returns after a game (the
// winner shown, or Y).
func (g *Game) Run() {
	g.wins = [2]int{}
	for {
		played := true
		switch g.openScreen() {
		case picopad.KeyLeft: // the eggs
			g.playGame(false, true)
		case picopad.KeyRight: // the hens
			g.playGame(true, false)
		case picopad.KeyUp: // demo
			g.playGame(true, true)
		case picopad.KeyDown: // 2 players
			g.playGame(false, false)
		case picopad.KeyY:
			return
		default:
			played = false
		}
		if played && g.OneGame { // the winner shown, or Y
			return
		}
	}
}
