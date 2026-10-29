// Package maze is the Maze of PicoLibSDK (PicoPad/GAME/MAZE, by Miroslav
// Nemecek, Unlicense), ported from C to package picopad. It follows the
// C source function by function; the C globals are the fields of Game.
//
// Find the way out: the arrows move the face, and only the places next to
// the paths it has gone are shown. A shows the door for a moment. At the
// door the whole maze is shown, and any button starts a new one. Y ends
// the game and Run returns (on the PicoPad: back to the boot loader).
//
// As in the C source, the main loop does not wait: it reads the buttons,
// and every 100 turns it draws the whole board again (the face winks
// only when a move draws it between the 90th and the 100th turn).
package maze

import (
	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/picopad"
)

//go:generate go run ../../tools/picoimg -pkg maze -root $PICOLIBSDK -o tiles.go PicoPad/GAME/MAZE/img/tiles.cpp

// The "wave" maze generator of the C source:
// https://www.itnetwork.cz/navrh/algoritmy/algoritmy-bludiste/algoritmus-tvorba-nahodneho-bludiste

const (
	tileSize = 8                         // tile size in pixels
	mapW     = picopad.Width / tileSize  // board width (40)
	mapH     = picopad.Height / tileSize // board height (30)
	mapSize  = mapW * mapH               // board size (1200)

	mazeW = (mapW+1)/2*2 - 1 // maze width (39, must be odd)
	mazeH = (mapH+1)/2*2 - 1 // maze height (29, must be odd)

	speed = 1 // step speed (delay in [ms])
)

// The tiles of tilesImg.
const (
	sBlack     = iota // black tile
	sEmptyHide        // hidden empty field (grass)
	sWallHide         // hidden wall
	sDoorHide         // hidden door
	sEmpty            // empty field (grass)
	sFace             // face
	sWink             // winking face
	sRight1           // face right 1 (left half)
	sRight2           // face right 2 (right half)
	sLeft1            // face left 1 (left half)
	sLeft2            // face left 2 (right half)
	sDown1            // face down 1 (top half)
	sDown2            // face down 2 (bottom half)
	sUp1              // face up 1 (bottom half)
	sUp2              // face up 2 (top half)
	sWall             // wall
	sDoor             // door
)

// The directions of the generator.
const (
	dirL = iota // left
	dirU        // up
	dirR        // right
	dirD        // down
)

// Game is the maze on a console.
type Game struct {
	*picopad.Console

	board        [mapSize]uint8 // the game board
	stopX, stopY int            // stop position (the door)
	posX, posY   int            // current position
	anim         uint8          // animation phase
}

// New returns the game on d (at least 320x240) with the touch panel t;
// seed starts the random mazes. Set Pad to use a game controller.
func New(d hal.Display, t hal.Touch, seed uint32) *Game {
	return &Game{Console: picopad.New(d, t, seed)}
}

// at returns the board place x, y (BoardAddr).
func (g *Game) at(x, y int) *uint8 { return &g.board[x+y*mapW] }

// drawBoard draws the board and updates the display.
func (g *Game) drawBoard() {
	for i := 0; i < mapH; i++ {
		for j := 0; j < mapW; j++ {
			b := int(g.board[i*mapW+j])
			x := j*tileSize + tileSize/2
			y := i*tileSize + tileSize/2
			g.DrawImg4Pal(tilesImg, tilesImgPal, 0, b*tileSize, x, y, tileSize, tileSize, tileSize)
		}
	}
	g.DispUpdate()
}

// gener generates a new maze.
func (g *Game) gener() {
	// clear the board with black (black tiles stay on the right and at the
	// bottom)
	g.board = [mapSize]uint8{}

	// clear the maze: empty tiles
	for y := 0; y < mazeH; y++ {
		for x := 0; x < mazeW; x++ {
			*g.at(x, y) = sEmptyHide
		}
	}

	// add stones, only at odd coordinates
	for y := 1; y < mazeH; y += 2 {
		for x := 1; x < mazeW; x += 2 {
			*g.at(x, y) = sDoorHide
		}
	}

	for {
		// count the stones left
		s := int16(0)
		for _, b := range g.board {
			if b == sDoorHide {
				s++
			}
		}
		if s == 0 {
			break // no stones left: done
		}

		// select a random stone
		s = g.RandS16Max(s - 1)

		// find it (only odd coordinates)
		x, y := 1, 1
	find:
		for y = 1; y < mazeH; y += 2 {
			for x = 1; x < mazeW; x += 2 {
				if *g.at(x, y) == sDoorHide {
					s--
					if s < 0 {
						break find
					}
				}
			}
		}

		// select a random direction
		i := g.RandU8Max(3)

		// build a wall until it meets another one
		for {
			*g.at(x, y) = sWallHide
			switch i {
			case dirL:
				x--
			case dirU:
				y--
			case dirR:
				x++
			default:
				y++
			}
			if x < 0 || x >= mazeW || y < 0 || y >= mazeH {
				break // out of the maze
			}
			if *g.at(x, y) == sWallHide {
				break
			}
		}
	}
}

// getTile returns a tile as shown (a wall out of the maze).
func (g *Game) getTile(x, y int) uint8 {
	if x < 0 || x >= mazeW || y < 0 || y >= mazeH {
		return sWall
	}
	if x == g.posX && y == g.posY {
		return sFace
	}
	if x == g.stopX && y == g.stopY {
		return sDoor
	}
	switch tile := *g.at(x, y); tile {
	case sWallHide:
		return sWall
	case sEmptyHide:
		return sEmpty
	case sDoorHide:
		return sDoor
	default:
		return tile
	}
}

// hide hides the face.
func (g *Game) hide() { *g.at(g.posX, g.posY) = sEmpty }

// show shows the face.
func (g *Game) show() { *g.at(g.posX, g.posY) = sFace }

// isEmpty reports whether the face can go to x, y.
func (g *Game) isEmpty(x, y int) bool {
	t := g.getTile(x, y)
	return t == sEmpty || t == sDoor
}

// showAll shows the whole maze, a column at a time.
func (g *Game) showAll() {
	for x := 0; x < mazeW; x++ {
		for y := 0; y < mazeH; y++ {
			*g.at(x, y) = g.getTile(x, y)
		}
		g.drawBoard()
		g.WaitMs(40)
	}
}

// look shows the places around the face.
func (g *Game) look() {
	if g.posX > 0 {
		*g.at(g.posX-1, g.posY) = g.getTile(g.posX-1, g.posY)
	}
	if g.posX < mazeW-1 {
		*g.at(g.posX+1, g.posY) = g.getTile(g.posX+1, g.posY)
	}
	if g.posY > 0 {
		*g.at(g.posX, g.posY-1) = g.getTile(g.posX, g.posY-1)
	}
	if g.posY < mazeH-1 {
		*g.at(g.posX, g.posY+1) = g.getTile(g.posX, g.posY+1)
	}
}

// newScene makes a new maze and draws it (Clear in the C source).
func (g *Game) newScene() {
	g.gener()

	// the door
	for {
		g.stopX = int(g.RandU8Max(mazeW - 1))
		g.stopY = int(g.RandU8Max(mazeH - 1))
		if *g.at(g.stopX, g.stopY) == sEmptyHide {
			break
		}
	}
	*g.at(g.stopX, g.stopY) = sDoorHide

	// the start
	for {
		g.posX = int(g.RandU8Max(mazeW - 1))
		g.posY = int(g.RandU8Max(mazeH - 1))
		if *g.at(g.posX, g.posY) == sEmptyHide && abs(g.posX-g.stopX) > 4 && abs(g.posY-g.stopY) > 4 {
			break
		}
	}
	g.show()
	g.look()
	g.anim = 0
	g.drawBoard()
}

// step moves the face from x0, y0 to x1, y1 (next to it) with the two
// pairs of half faces of the animation.
func (g *Game) step(x0, y0, x1, y1 int, a0, a1, b0, b1 uint8) {
	*g.at(x0, y0), *g.at(x1, y1) = a0, a1
	g.drawBoard()
	g.WaitMs(speed)
	*g.at(x0, y0), *g.at(x1, y1) = b0, b1
	g.drawBoard()
	g.WaitMs(speed)
	g.hide()
	g.posX, g.posY = x1, y1
	g.show()
	g.look()
	g.drawBoard()
}

// Run plays until Y (main of the C source, which then resets to the boot
// loader).
func (g *Game) Run() {
	g.DrawClear()
	g.newScene()

	for {
		// face animation
		g.anim++
		if g.anim == 90 {
			*g.at(g.posX, g.posY) = sWink
		}
		if g.anim == 100 {
			*g.at(g.posX, g.posY) = sFace
			g.anim = 0
			g.drawBoard()
		}

		switch g.KeyGet() {
		case picopad.KeyA: // show the door
			t := *g.at(g.stopX, g.stopY)
			for k := 3; k > 0; k-- {
				*g.at(g.stopX, g.stopY) = sDoor
				g.drawBoard()
				g.WaitMs(100)
				*g.at(g.stopX, g.stopY) = sDoorHide
				g.drawBoard()
				g.WaitMs(50)
			}
			*g.at(g.stopX, g.stopY) = t
			g.drawBoard()
			g.KeyFlush()

		case picopad.KeyLeft:
			if g.posX > 0 && g.isEmpty(g.posX-1, g.posY) {
				g.step(g.posX, g.posY, g.posX-1, g.posY, sLeft2, sLeft1, sRight2, sRight1)
			}
			g.KeyFlush()

		case picopad.KeyRight:
			if g.posX < mazeW-1 && g.isEmpty(g.posX+1, g.posY) {
				g.step(g.posX, g.posY, g.posX+1, g.posY, sRight1, sRight2, sLeft1, sLeft2)
			}
			g.KeyFlush()

		case picopad.KeyUp:
			if g.posY > 0 && g.isEmpty(g.posX, g.posY-1) {
				g.step(g.posX, g.posY, g.posX, g.posY-1, sUp1, sUp2, sDown2, sDown1)
			}
			g.KeyFlush()

		case picopad.KeyDown:
			if g.posY < mazeH-1 && g.isEmpty(g.posX, g.posY+1) {
				g.step(g.posX, g.posY, g.posX, g.posY+1, sDown1, sDown2, sUp2, sUp1)
			}
			g.KeyFlush()

		case picopad.KeyY:
			return

		case picopad.NoKey:

		default:
			g.KeyFlush()
		}

		// solved
		if g.posX == g.stopX && g.posY == g.stopY {
			g.showAll()

			// wink
			for i := 0; i < 5; i++ {
				*g.at(g.posX, g.posY) = sWink
				g.drawBoard()
				g.WaitMs(100)
				*g.at(g.posX, g.posY) = sFace
				g.drawBoard()
				g.WaitMs(100)
			}
			*g.at(g.stopX, g.stopY) = sDoor
			g.drawBoard()

			// wait for a key
			g.KeyFlush()
			for g.KeyGet() == picopad.NoKey {
			}

			g.newScene()
			g.KeyFlush()
		}
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
