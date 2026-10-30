// Package tetris is the Tetris of PicoLibSDK (PicoPad/GAME/TETRIS, by
// Miroslav Nemecek, Unlicense), ported from C to package picopad. It
// follows the C source function by function; the C globals are the fields
// of Game.
//
// LEFT and RIGHT move the brick, UP or X turns it, DOWN or A drops it, B
// pauses and Y ends the game. A new game starts paused: any button plays,
// and after 10 seconds the demo plays instead. Any button ends the demo
// and starts a game; Y ends it and Run returns (on the PicoPad: back to
// the boot loader). There is no sound yet: the PLAYSOUND calls of the C
// source are comments.
package tetris

import (
	"strconv"

	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/picopad"
)

//go:generate go run ../../tools/picoimg -pkg tetris -root $PICOLIBSDK -o tiles.go PicoPad/GAME/TETRIS/img/tiles.cpp

const (
	tileSize  = 12                                         // tile size in pixels
	mapW      = (picopad.Width + tileSize - 1) / tileSize  // board width (27)
	mapH      = (picopad.Height + tileSize - 1) / tileSize // board height (20)
	mapSize   = mapW * mapH                                // board size
	tileEmpty = 8                                          // empty game tile
	tilesImgW = 228                                        // width of tilesImg
	scoreX    = tileSize + 3*tileSize                      // middle of the numbers on the left
	brickX0   = mapW/2 - 2                                 // where a new brick starts

	baseBrickNum  = 7  // number of base bricks
	brickTempNum  = 19 // number of brick templates
	brickTempSize = 16 // size of one brick template
	levelNum      = 10 // number of levels

	timeDelay = 10        // base time delay [ms]
	speedFall = timeDelay // falling speed
)

// boardTemp is the board of a new game: tiles of tilesImg.
var boardTemp = [mapSize]uint8{
	9, 9, 9, 9, 9, 9, 9, 10, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 12, 0, 0, 0, 0, 0, 0, 0, 0,
	9, 0, 0, 0, 0, 0, 0, 13, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 14, 0, 1, 1, 1, 1, 0, 0, 0,
	9, 0, 0, 0, 0, 0, 0, 13, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 14, 0, 0, 0, 0, 0, 0, 0, 0,
	9, 0, 0, 0, 0, 0, 0, 13, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 14, 0, 0, 2, 2, 0, 0, 0, 0,
	9, 0, 0, 0, 0, 0, 0, 13, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 14, 0, 0, 2, 2, 0, 0, 0, 0,
	9, 0, 0, 0, 0, 0, 0, 13, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 14, 0, 0, 0, 0, 0, 0, 0, 0,
	9, 9, 9, 9, 9, 9, 9, 13, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 14, 0, 3, 3, 3, 0, 0, 0, 0,
	9, 9, 9, 9, 9, 9, 9, 13, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 14, 0, 3, 0, 0, 0, 0, 0, 0,
	9, 0, 0, 0, 0, 0, 0, 13, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 14, 0, 0, 0, 0, 0, 0, 0, 0,
	9, 0, 0, 0, 0, 0, 0, 13, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 14, 0, 4, 4, 4, 0, 0, 0, 0,
	9, 0, 0, 0, 0, 0, 0, 13, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 14, 0, 0, 0, 4, 0, 0, 0, 0,
	9, 9, 9, 9, 9, 9, 9, 13, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 14, 0, 0, 0, 0, 0, 0, 0, 0,
	9, 0, 0, 0, 0, 0, 0, 13, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 14, 0, 5, 5, 0, 0, 0, 0, 0,
	9, 0, 0, 0, 0, 0, 0, 13, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 14, 0, 0, 5, 5, 0, 0, 0, 0,
	9, 0, 0, 0, 0, 0, 0, 13, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 14, 0, 0, 0, 0, 0, 0, 0, 0,
	9, 9, 9, 9, 9, 9, 9, 13, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 14, 0, 0, 6, 6, 0, 0, 0, 0,
	9, 0, 0, 0, 0, 0, 0, 13, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 14, 0, 6, 6, 0, 0, 0, 0, 0,
	9, 0, 0, 0, 0, 0, 0, 13, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 14, 0, 0, 0, 0, 0, 0, 0, 0,
	9, 0, 0, 0, 0, 0, 0, 13, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 14, 0, 7, 7, 7, 0, 0, 0, 0,
	9, 9, 9, 9, 9, 9, 9, 15, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 17, 0, 0, 7, 0, 0, 0, 0, 0,
}

// brickTemp are the bricks, 4x4 each (rotations are counterclockwise).
var brickTemp = [brickTempNum * brickTempSize]uint8{
	// 0 (7)
	0, 0, 0, 0,
	1, 1, 1, 1,
	0, 0, 0, 0,
	0, 0, 0, 0,

	// 1
	0, 0, 0, 0,
	0, 2, 2, 0,
	0, 2, 2, 0,
	0, 0, 0, 0,

	// 2 (8) (9) (10)
	0, 0, 0, 0,
	3, 3, 3, 0,
	3, 0, 0, 0,
	0, 0, 0, 0,

	// 3 (11) (12) (13)
	0, 0, 0, 0,
	4, 4, 4, 0,
	0, 0, 4, 0,
	0, 0, 0, 0,

	// 4 (14)
	0, 0, 0, 0,
	5, 5, 0, 0,
	0, 5, 5, 0,
	0, 0, 0, 0,

	// 5 (15)
	0, 0, 0, 0,
	0, 6, 6, 0,
	6, 6, 0, 0,
	0, 0, 0, 0,

	// 6 (16) (17) (18)
	0, 0, 0, 0,
	7, 7, 7, 0,
	0, 7, 0, 0,
	0, 0, 0, 0,

	// (0) 7
	0, 1, 0, 0,
	0, 1, 0, 0,
	0, 1, 0, 0,
	0, 1, 0, 0,

	// (2) 8 (9) (10)
	0, 3, 0, 0,
	0, 3, 0, 0,
	0, 3, 3, 0,
	0, 0, 0, 0,

	// (2) (8) 9 (10)
	0, 0, 0, 0,
	0, 0, 3, 0,
	3, 3, 3, 0,
	0, 0, 0, 0,

	// (2) (8) (9) 10
	3, 3, 0, 0,
	0, 3, 0, 0,
	0, 3, 0, 0,
	0, 0, 0, 0,

	// (3) 11 (12) (13)
	0, 4, 4, 0,
	0, 4, 0, 0,
	0, 4, 0, 0,
	0, 0, 0, 0,

	// (3) (11) 12 (13)
	0, 0, 0, 0,
	4, 0, 0, 0,
	4, 4, 4, 0,
	0, 0, 0, 0,

	// (3) (11) (12) 13
	0, 4, 0, 0,
	0, 4, 0, 0,
	4, 4, 0, 0,
	0, 0, 0, 0,

	// (4) 14
	0, 5, 0, 0,
	5, 5, 0, 0,
	5, 0, 0, 0,
	0, 0, 0, 0,

	// (5) 15
	0, 6, 0, 0,
	0, 6, 6, 0,
	0, 0, 6, 0,
	0, 0, 0, 0,

	// (6) 16 (17) (18)
	0, 7, 0, 0,
	0, 7, 7, 0,
	0, 7, 0, 0,
	0, 0, 0, 0,

	// (6) (16) 17 (18)
	0, 0, 0, 0,
	0, 7, 0, 0,
	7, 7, 7, 0,
	0, 0, 0, 0,

	// (6) (16) (17) 18
	0, 7, 0, 0,
	7, 7, 0, 0,
	0, 7, 0, 0,
	0, 0, 0, 0,
}

// brickRot is the brick after turning each one.
var brickRot = [brickTempNum]uint8{
	// 0  1  2   3   4   5   6  7  8   9  10  11  12 13 14 15  16  17 18
	7, 1, 8, 11, 14, 15, 16, 0, 9, 10, 2, 12, 13, 3, 4, 5, 17, 18, 6,
}

// levScore is the score of a brick at each level.
var levScore = [levelNum]int{10, 20, 50, 100, 200, 500, 1000, 2000, 5000, 10000}

// nextLevelTab is the count of lines to the next level.
var nextLevelTab = [levelNum]int{10, 20, 50, 100, 150, 250, 400, 500, 700, 1000000000}

// levSpeed is the step delay at each level [ms] (a multiple of timeDelay).
var levSpeed = [levelNum]int{1000, 700, 500, 350, 250, 200, 180, 150, 120, 100}

// Game is Tetris on a console.
type Game struct {
	*picopad.Console

	level, levelBest    uint8 // current level (1..), best level
	brick, nextBrick    uint8 // current and next brick (0..18)
	brickX, brickY      int   // brick coordinates
	falling             bool  // the brick is falling
	brickScore          int   // score of the current brick
	score, scoreBest    uint32
	stepSpeed, speedCnt int // step speed, speed counter
	lines, linesBest    int
	nextLevel           int  // lines to the next level
	demo                bool // demo mode

	stat  [baseBrickNum]int // bricks of each kind
	board [mapSize]uint8    // the game board
}

// New returns the game on d (at least 320x240) with the touch panel t;
// seed starts the random bricks. Set Pad to use a game controller.
func New(d hal.Display, t hal.Touch, seed uint32) *Game {
	g := &Game{Console: picopad.New(d, t, seed), levelBest: 1}
	g.SelFont8x8() // FONT in config.h
	return g
}

// dispTile displays one tile of the board.
func (g *Game) dispTile(x, y int) {
	tile := int(g.board[x+y*mapW])
	g.DrawImgPal(tilesImg, tilesImgPal, tile*tileSize, 0, x*tileSize, y*tileSize, tileSize, tileSize, tilesImgW)
}

// dispBoard displays all tiles.
func (g *Game) dispBoard() {
	for y := 0; y < mapH; y++ {
		for x := 0; x < mapW; x++ {
			g.dispTile(x, y)
		}
	}
}

// putTile puts a tile on the board.
func (g *Game) putTile(x, y int, tile uint8) {
	g.board[x+y*mapW] = tile
	g.dispTile(x, y)
}

// getTile returns a tile of the board.
func (g *Game) getTile(x, y int) uint8 { return g.board[x+y*mapW] }

// dispTileNum displays a number (x is its middle).
func (g *Game) dispTileNum(x, y, num int, col picopad.Color) {
	s := strconv.Itoa(num)
	g.DrawTextBg(s, x-len(s)*4, y, col, picopad.ColBlack)
}

// dispTitle displays the titles.
func (g *Game) dispTitle() {
	g.DrawText("STATISTICS", 234, 0, picopad.ColGreen)
	g.DrawText("NEXT", tileSize+(6*tileSize-4*8)/2, 1*tileSize+2, picopad.ColGreen)
	g.DrawText("SCORE", tileSize+(6*tileSize-5*8)/2, 8*tileSize+2, picopad.ColGreen)
	g.DrawText("LINE", tileSize+(6*tileSize-4*8)/2, 12*tileSize+2, picopad.ColGreen)
	g.DrawText("LEVEL", tileSize+(6*tileSize-5*8)/2, 16*tileSize+2, picopad.ColGreen)
}

// dispScore displays the score, lines and level, with the best ones.
func (g *Game) dispScore() {
	g.dispTileNum(scoreX, 8*tileSize+3+10, int(g.score), picopad.ColWhite)
	g.dispTileNum(scoreX, 8*tileSize+3+20, int(g.scoreBest), picopad.ColYellow)

	g.dispTileNum(scoreX, 12*tileSize+3+10, g.lines, picopad.ColWhite)
	g.dispTileNum(scoreX, 12*tileSize+3+20, g.linesBest, picopad.ColYellow)

	g.dispTileNum(scoreX, 16*tileSize+3+10, int(g.level), picopad.ColWhite)
	g.dispTileNum(scoreX, 16*tileSize+3+20, int(g.levelBest), picopad.ColYellow)
}

// dispStat displays the statistics.
func (g *Game) dispStat() {
	g.dispTileNum(25*tileSize, 1*tileSize, g.stat[0], picopad.ColWhite)
	for i := 1; i < baseBrickNum; i++ {
		g.dispTileNum(25*tileSize, i*3*tileSize+tileSize/2, g.stat[i], picopad.ColWhite)
	}
}

// dispNext displays the next brick.
func (g *Game) dispNext() {
	s := brickTemp[int(g.nextBrick)*brickTempSize:]
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			g.putTile(2+j, 2+i, s[i*4+j])
		}
	}
}

// dispAll displays everything and updates the display.
func (g *Game) dispAll() {
	g.dispBoard()
	g.dispTitle()
	g.dispScore()
	g.dispStat()
	g.dispNext()
	g.DispUpdate()
}

// dispBrick displays a brick.
func (g *Game) dispBrick(x, y int, brick uint8) {
	s := brickTemp[int(brick)*brickTempSize:]
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			if b := s[i*4+j]; b != 0 {
				g.putTile(x+j, y+i, b)
			}
		}
	}
}

// clearBrick clears a brick.
func (g *Game) clearBrick(x, y int, brick uint8) {
	s := brickTemp[int(brick)*brickTempSize:]
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			if s[i*4+j] != 0 {
				g.putTile(x+j, y+i, tileEmpty)
			}
		}
	}
}

// checkBrick reports whether the brick fits at x, y.
func (g *Game) checkBrick(x, y int, brick uint8) bool {
	s := brickTemp[int(brick)*brickTempSize:]
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			if s[i*4+j] != 0 {
				if y < 1 || y >= mapH-1 || g.getTile(x+j, y) != tileEmpty {
					return false
				}
			}
		}
		y++
	}
	return true
}

// scoreBrick is the score of a brick at x, y for the demo (the sum of the
// squared Y of its stones, less the empty places next to them).
func (g *Game) scoreBrick(x, y int, brick uint8) int {
	s := brickTemp[int(brick)*brickTempSize:]
	score := 0
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			k := i*4 + j
			if s[k] != 0 {
				score += y * y
				if y > 2 {
					// minus score on empty neighbors
					if y < mapH-2 && g.getTile(x+j, y+1) == tileEmpty && // below
						(i == 3 || s[k+4] == 0) {
						score -= 10
					}
					if g.getTile(x+j-1, y) == tileEmpty && // left
						(j == 0 || s[k-1] == 0) {
						score -= 2
					}
					if g.getTile(x+j+1, y) == tileEmpty && // right
						(j == 3 || s[k+1] == 0) {
						score -= 2
					}
				}
			}
		}
		y++
	}
	return score
}

// delRow deletes a full row.
func (g *Game) delRow(y int) {
	// PLAYSOUND(LineSnd)

	g.lines++
	g.dispScore()

	// shift the full row out
	for i := 10; i > 0; i-- {
		j := 0
		for ; j < 9; j++ {
			g.putTile(8+j, y, g.getTile(8+j+1, y))
		}
		g.putTile(8+j, y, tileEmpty)
		g.DispUpdate()
		g.WaitMs(50)
	}

	// move the stones above down
	for ; y > 1; y-- {
		for j := 0; j < 10; j++ {
			g.putTile(8+j, y, g.getTile(8+j, y-1))
		}
	}

	// empty the top row
	for j := 0; j < 10; j++ {
		g.putTile(8+j, 1, tileEmpty)
	}
}

// delAllRows deletes the full rows.
func (g *Game) delAllRows() {
	for y := mapH - 2; y > 1; y-- {
		ok := true
		for i := 0; i < 10; i++ {
			if g.getTile(8+i, y) == tileEmpty {
				ok = false
				break
			}
		}
		if ok {
			g.delRow(y)
			y++ // stay at this row
			g.nextLevel--
		}
	}

	// next level
	if g.nextLevel <= 0 && g.level < levelNum && !g.demo {
		g.level++
		g.dispScore()
		g.nextLevel = nextLevelTab[g.level-1]

		// PLAYSOUND(LevelUpSnd)

		g.DrawTextBg(" LEVEL UP ", (picopad.Width-10*8)/2, picopad.Height/2, picopad.RGB(0, 0, 128), picopad.ColYellow)
		g.DispUpdate()
		g.WaitMs(1000)

		// restore the board
		g.dispAll()

		g.KeyFlush()
		g.WaitMs(200)
	}
}

// genBrick takes the next brick and makes a new next one.
func (g *Game) genBrick() {
	g.brick = g.nextBrick
	g.stat[g.brick]++
	g.dispStat()

	g.nextBrick = g.RandU8Max(baseBrickNum - 1)
	g.dispNext()

	g.brickX = brickX0
	g.brickY = 0
	g.brickScore = levScore[g.level-1]
	g.speedCnt = g.stepSpeed
	g.falling = false
}

// checkBrickSave checks the brick and saves the new position if it fits.
func (g *Game) checkBrickSave(x, y int, brick uint8) bool {
	res := g.checkBrick(x, y, brick)
	if res {
		g.brickX = x
		g.brickY = y
		g.brick = brick
	}
	return res
}

// checkBrickX checks the brick, also moved to the left and right.
func (g *Game) checkBrickX(x, y int, brick uint8) bool {
	if g.checkBrickSave(x, y, brick) {
		return true
	}
	if g.checkBrickSave(x-1, y, brick) {
		return true
	}
	if g.checkBrickSave(x+1, y, brick) {
		return true
	}
	if g.checkBrickSave(x-2, y, brick) {
		return true
	}
	return g.checkBrickSave(x+2, y, brick)
}

// checkBrickY checks the brick, also moved down.
func (g *Game) checkBrickY(x, y int, brick uint8) bool {
	if g.checkBrickX(x, y, brick) {
		return true
	}
	if g.checkBrickX(x, y+1, brick) {
		return true
	}
	return g.checkBrickX(x, y+2, brick)
}

// newGame starts a new game.
func (g *Game) newGame() {
	g.board = boardTemp
	g.dispBoard()
	g.dispTitle()

	g.nextBrick = 0
	g.level = 1
	g.nextLevel = nextLevelTab[g.level-1]
	g.score = 0
	g.lines = 0
	g.stat = [baseBrickNum]int{}
	g.stepSpeed = levSpeed[g.level-1]

	// this brick
	g.genBrick()
	g.stat[0]--

	// the next brick (and the statistics)
	g.genBrick()

	g.dispScore()
	g.dispBrick(g.brickX, g.brickY, g.brick)
	g.DispUpdate()
}

// checkEnd checks the end of the game (it returns true to go on).
func (g *Game) checkEnd() bool {
	// the new brick does not fit at the start
	if g.checkBrick(g.brickX, g.brickY, g.brick) {
		return true
	}
	if !g.demo {
		g.levelBest = max(g.levelBest, g.level)
		g.scoreBest = max(g.scoreBest, g.score)
		g.linesBest = max(g.linesBest, g.lines)
		g.dispScore()
	}

	// PLAYSOUND(GameOverSnd)

	// blink for 2 seconds
	for i := 10; i > 0; i-- {
		g.DrawTextBg(" GAME OVER ", (picopad.Width-11*8)/2, picopad.Height/2, picopad.ColRed, picopad.ColWhite)
		g.DispUpdate()
		g.WaitMs(100)
		g.dispAll()
		g.WaitMs(100)
	}
	if g.demo {
		return false
	}

	g.DrawTextBg(" GAME OVER ", (picopad.Width-11*8)/2, picopad.Height/2, picopad.ColRed, picopad.ColWhite)
	g.DispUpdate()

	// wait for a key
	g.KeyFlush()
	for g.KeyGet() == picopad.NoKey {
	}

	g.dispAll()
	return false
}

// pause pauses the game until a key (it returns false after 10 seconds
// with timeout).
func (g *Game) pause(timeout bool) bool {
	g.DrawTextBg(" PAUSE ", (picopad.Width-7*8)/2, picopad.Height/2, picopad.ColRed, picopad.ColWhite)
	g.DispUpdate()

	g.KeyFlush()
	i := 1000000000
	if timeout {
		i = 100
	}
	for ; i > 0; i-- {
		if g.KeyGet() != picopad.NoKey {
			break
		}
		g.WaitMs(100)
	}

	g.dispAll()
	return i > 0
}

// play is the game (Game in the C source): it returns on Y, or when the
// pause at the start of a game times out (false); with OneGame also at the
// end of the game.
func (g *Game) play() bool {
	g.newGame()
	if !g.pause(true) {
		return false
	}

	for {
		// step counter
		g.DispUpdate()
		g.WaitMs(timeDelay)
		g.speedCnt -= timeDelay
		if g.speedCnt <= 0 {
			g.clearBrick(g.brickX, g.brickY, g.brick)

			if !g.checkBrick(g.brickX, g.brickY+1, g.brick) {
				// the brick has landed
				g.dispBrick(g.brickX, g.brickY, g.brick)
				g.score += uint32(g.brickScore)
				g.dispScore()

				// PLAYSOUND(BrickSnd)

				g.delAllRows()
				g.DispUpdate()
				g.WaitMs(50)

				g.genBrick()
				if !g.checkEnd() {
					if g.OneGame {
						return true
					}
					g.newGame()
					if !g.pause(true) {
						return false
					}
				} else {
					g.dispBrick(g.brickX, g.brickY, g.brick)
				}
			} else {
				// move the brick down
				g.brickY++
				g.dispBrick(g.brickX, g.brickY, g.brick)
				g.speedCnt = g.stepSpeed
				if g.falling {
					g.speedCnt = speedFall
				}
				if !g.falling && g.brickScore > 0 {
					g.brickScore--
				}
			}
		}

		// keys, if not falling
		if g.falling {
			continue
		}
		switch g.KeyGet() {
		case picopad.KeyRight:
			g.clearBrick(g.brickX, g.brickY, g.brick)
			if g.checkBrick(g.brickX+1, g.brickY, g.brick) {
				g.brickX++
			}
			g.dispBrick(g.brickX, g.brickY, g.brick)

		case picopad.KeyLeft:
			g.clearBrick(g.brickX, g.brickY, g.brick)
			if g.checkBrick(g.brickX-1, g.brickY, g.brick) {
				g.brickX--
			}
			g.dispBrick(g.brickX, g.brickY, g.brick)

		case picopad.KeyDown, picopad.KeyA: // fall
			g.falling = true
			g.speedCnt = min(g.speedCnt, speedFall)

		case picopad.KeyUp, picopad.KeyX: // turn
			g.clearBrick(g.brickX, g.brickY, g.brick)
			// try to turn the brick, at the nearest free place
			g.checkBrickY(g.brickX, g.brickY, brickRot[g.brick])
			g.dispBrick(g.brickX, g.brickY, g.brick)

		case picopad.KeyB:
			g.pause(false)

		case picopad.KeyY:
			return true

		case picopad.NoKey:

		default:
			g.KeyFlush()
		}
	}
}

// findDemo finds the move of the demo and puts the brick there.
func (g *Game) findDemo() {
	bestScore, bestX, bestY, bestBrick := 0, g.brickX, g.brickY, g.brick

	g.clearBrick(g.brickX, g.brickY, g.brick)

	for rot := 4; rot > 0; rot-- {
		// try to turn the brick, at the nearest free place
		g.checkBrickY(g.brickX, g.brickY, brickRot[g.brick])

		// to the left
		for g.checkBrick(g.brickX-1, g.brickY, g.brick) {
			g.brickX--
		}

		// every position
		for {
			saveX, saveY, saveBrick := g.brickX, g.brickY, g.brick

			// down
			for g.checkBrick(g.brickX, g.brickY+1, g.brick) {
				g.brickY++
			}
			score := g.scoreBrick(g.brickX, g.brickY, g.brick)

			g.brick, g.brickX, g.brickY = saveBrick, saveX, saveY

			if score > bestScore || score == bestScore && g.RandU8() < 0x50 {
				bestScore, bestX, bestY, bestBrick = score, saveX, saveY, saveBrick
			}

			// the next position to the right
			if !g.checkBrick(g.brickX+1, g.brickY, g.brick) {
				break
			}
			g.brickX++
		}
	}

	g.brickX, g.brickY, g.brick = bestX, bestY, bestBrick
	g.dispBrick(g.brickX, g.brickY, g.brick)
}

// playDemo plays the demo until a key.
func (g *Game) playDemo() {
	g.demo = true
	g.newGame()

	for {
		if g.KeyGet() != picopad.NoKey {
			break
		}

		g.findDemo()
		g.DispUpdate()
		g.WaitMs(200)

		if g.KeyGet() != picopad.NoKey {
			break
		}

		// move the brick down
		for {
			g.DispUpdate()
			g.WaitMs(5 * timeDelay)
			g.clearBrick(g.brickX, g.brickY, g.brick)
			if !g.checkBrick(g.brickX, g.brickY+1, g.brick) {
				break
			}
			g.brickY++
			g.dispBrick(g.brickX, g.brickY, g.brick)
		}
		g.dispBrick(g.brickX, g.brickY, g.brick)

		g.score += uint32(g.brickScore)
		g.dispScore()

		// PLAYSOUND(BrickSnd)

		g.delAllRows()
		g.DispUpdate()
		g.WaitMs(200)

		if g.KeyGet() != picopad.NoKey {
			break
		}

		g.genBrick()
		if !g.checkEnd() {
			break
		}
		g.dispBrick(g.brickX, g.brickY, g.brick)
		g.DispUpdate()
		g.WaitMs(200)
	}

	g.demo = false
}

// Run plays: a game, then the demo, and again, until Y ends the demo
// (main of the C source, which then resets to the boot loader). With
// OneGame it returns after a game (game over or Y).
func (g *Game) Run() {
	for {
		if g.play() && g.OneGame {
			return
		}
		g.playDemo()
		if g.KeyPressed(picopad.KeyY) {
			return
		}
	}
}
