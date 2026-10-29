// Package life is the Life cell simulator of PicoLibSDK (PicoPad/GAME/LIFE,
// by Miroslav Nemecek, Unlicense), ported from C to package picopad. It
// follows the C source function by function; the C globals are the fields
// of Game.
//
// The world is 50 x 50 cells (wrapping around at the edges); the screen
// shows the middle 20 x 15. The arrows move the cursor, A flips the cell
// under it, B runs life until a button (then the world is as before the
// run), X goes to the next of the 10 slots (the first ones hold known
// patterns: Life, penta-decathlon, R-pentomino, Diehard, Acorn, glider,
// spaceship, pulsar, Gosper and Simkin glider guns), Y ends the game and
// Run returns (on the PicoPad: back to the boot loader). A cell with 3
// neighbours is born, one with 2 stays, the others die.
package life

import (
	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/picopad"
)

//go:generate go run ../../tools/picoimg -pkg life -root $PICOLIBSDK -o tiles.go PicoPad/GAME/LIFE/img/tiles16.cpp

const (
	tileSize = 16                        // tile size in pixels
	mapW0    = 50                        // hidden width
	mapH0    = 50                        // hidden height
	mapW     = picopad.Width / tileSize  // shown width (20)
	mapH     = picopad.Height / tileSize // shown height (15)
	mapSize  = mapW0 * mapH0             // board size
	slotNum  = 10                        // number of slots
	mapX     = (mapW0 - mapW) / 2        // first column shown
	mapY     = (mapH0 - mapH) / 2        // first row shown

	speed = 200 // sleep between generations [ms]

	tileEmpty    = 0 // empty tile
	tileFull     = 1 // full tile
	tileSelEmpty = 2 // selected empty tile
	tileSelFull  = 3 // selected full tile
)

// samples are the patterns of the slots: the width, the height, then the
// cells.
var samples = [...]uint8{
	// 0: Life
	5, 5,
	1, 1, 1, 1, 1,
	1, 1, 1, 1, 1,
	1, 1, 1, 1, 1,
	1, 1, 1, 1, 1,
	1, 1, 1, 1, 1,

	// 1: penta-decathlon
	3, 12,
	1, 1, 1,
	0, 1, 0,
	0, 1, 0,
	1, 1, 1,
	0, 0, 0,
	1, 1, 1,
	1, 1, 1,
	0, 0, 0,
	1, 1, 1,
	0, 1, 0,
	0, 1, 0,
	1, 1, 1,

	// 2: R-pentomino
	3, 3,
	0, 1, 1,
	1, 1, 0,
	0, 1, 0,

	// 3: Diehard
	8, 3,
	0, 0, 0, 0, 0, 0, 1, 0,
	1, 1, 0, 0, 0, 0, 0, 0,
	0, 1, 0, 0, 0, 1, 1, 1,

	// 4. Acorn
	7, 3,
	0, 1, 0, 0, 0, 0, 0,
	0, 0, 0, 1, 0, 0, 0,
	1, 1, 0, 0, 1, 1, 1,

	// 5: Glider
	3, 3,
	0, 0, 1,
	1, 0, 1,
	0, 1, 1,

	// 6: spaceship
	5, 4,
	1, 0, 0, 1, 0,
	0, 0, 0, 0, 1,
	1, 0, 0, 0, 1,
	0, 1, 1, 1, 1,

	// 7: pulsar
	15, 15,
	0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0,
	0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0,
	0, 0, 0, 0, 1, 1, 0, 0, 0, 1, 1, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	1, 1, 1, 0, 0, 1, 1, 0, 1, 1, 0, 0, 1, 1, 1,
	0, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 0,
	0, 0, 0, 0, 1, 1, 0, 0, 0, 1, 1, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 1, 1, 0, 0, 0, 1, 1, 0, 0, 0, 0,
	0, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 0,
	1, 1, 1, 0, 0, 1, 1, 0, 1, 1, 0, 0, 1, 1, 1,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 1, 1, 0, 0, 0, 1, 1, 0, 0, 0, 0,
	0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0,
	0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0,

	// 8: Gosper glider gun
	38, 9,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 0,
	0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 0,
	0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 1, 0, 1, 1, 0, 0, 0, 0, 1, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,

	// 9: Simkin glider gun
	33, 21,
	1, 1, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	1, 1, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 0, 1, 1, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 1, 0, 0, 1, 1,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 1, 0, 0, 0, 1, 1,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0,
}

// Game is Life on a console. (The C source also has a clipboard the size
// of the board that it only clears; it is left out.)
type Game struct {
	*picopad.Console

	board      [mapSize]uint8           // the world
	neigh      [mapSize]uint8           // the neighbours of each cell
	slot       [mapSize * slotNum]uint8 // the slots
	selSlot    int                      // the current slot
	curX, curY int                      // the cursor
}

// New returns the game on d (at least 320x240) with the touch panel t. Set
// Pad to use a game controller.
func New(d hal.Display, t hal.Touch) *Game {
	return &Game{Console: picopad.New(d, t, 0)}
}

// drawBoard draws the part of the world shown and updates the display.
func (g *Game) drawBoard() {
	for i := 0; i < mapH; i++ {
		for j := 0; j < mapW; j++ {
			b := int(g.board[(i+mapY)*mapW0+j+mapX])
			g.DrawImg4Pal(tiles16Img, tiles16ImgPal, 0, b*tileSize, j*tileSize, i*tileSize, tileSize, tileSize, tileSize)
		}
	}
	g.DispUpdate()
}

// cur returns the cell under the cursor.
func (g *Game) cur() *uint8 { return &g.board[g.curX+g.curY*mapW0] }

// curOn shows the cursor.
func (g *Game) curOn() {
	switch d := g.cur(); *d {
	case tileFull:
		*d = tileSelFull
	case tileEmpty:
		*d = tileSelEmpty
	}
}

// curOff hides the cursor.
func (g *Game) curOff() {
	switch d := g.cur(); *d {
	case tileSelFull:
		*d = tileFull
	case tileSelEmpty:
		*d = tileEmpty
	}
}

// curFlip flips the cell under the cursor.
func (g *Game) curFlip() {
	switch d := g.cur(); *d {
	case tileSelFull:
		*d = tileSelEmpty
	case tileSelEmpty:
		*d = tileSelFull
	}
}

// cell returns 1 if cell x, y is alive (the world wraps around).
func (g *Game) cell(x, y int) uint8 {
	for x < 0 {
		x += mapW0
	}
	for x >= mapW0 {
		x -= mapW0
	}
	for y < 0 {
		y += mapH0
	}
	for y >= mapH0 {
		y -= mapH0
	}
	if g.board[x+y*mapW0] == tileFull {
		return 1
	}
	return 0
}

// saveSlot saves the world to the current slot.
func (g *Game) saveSlot() { copy(g.slot[mapSize*g.selSlot:], g.board[:]) }

// loadSlot loads the world from the current slot.
func (g *Game) loadSlot() { copy(g.board[:], g.slot[mapSize*g.selSlot:]) }

// generation computes the next generation (the loop of Run in the C
// source).
func (g *Game) generation() {
	// count the neighbours
	d := 0
	for y := 0; y < mapH0; y++ {
		for x := 0; x < mapW0; x++ {
			g.neigh[d] = g.cell(x-1, y-1) + g.cell(x, y-1) + g.cell(x+1, y-1) +
				g.cell(x-1, y) + g.cell(x+1, y) +
				g.cell(x-1, y+1) + g.cell(x, y+1) + g.cell(x+1, y+1)
			d++
		}
	}

	// update the cells
	for i, b := range g.neigh {
		switch b {
		case 3: // a new cell
			g.board[i] = tileFull
		case 2: // unchanged
		default: // dead
			g.board[i] = tileEmpty
		}
	}
}

// runLife runs life until a key (Run in the C source).
func (g *Game) runLife() {
	g.KeyFlush()
	for g.KeyGet() == picopad.NoKey {
		g.generation()
		g.drawBoard()
		g.WaitMs(speed)
	}
}

// initSlot puts the samples in the middle of the slots.
func (g *Game) initSlot() {
	s := 0
	for slot := 0; slot < slotNum; slot++ {
		w, h := int(samples[s]), int(samples[s+1])
		s += 2
		d := mapSize*slot + (mapW0-w)/2 + (mapH0-h)/2*mapW0
		for ; h > 0; h-- {
			copy(g.slot[d:d+w], samples[s:s+w])
			d += mapW0
			s += w
		}
	}
}

// Run plays until Y (main of the C source, which then resets to the boot
// loader).
func (g *Game) Run() {
	g.board = [mapSize]uint8{}
	g.slot = [mapSize * slotNum]uint8{}
	g.curX = mapW0 / 2
	g.curY = mapH0 / 2
	g.selSlot = 1

	g.initSlot()
	g.loadSlot()
	g.curOn()
	g.drawBoard()

	for {
		switch g.KeyGet() {
		case picopad.KeyRight:
			g.curOff()
			g.curX++
			if g.curX >= mapX+mapW {
				g.curX = mapX
			}
			g.curOn()
			g.drawBoard()

		case picopad.KeyLeft:
			g.curOff()
			g.curX--
			if g.curX < mapX {
				g.curX = mapX + mapW - 1
			}
			g.curOn()
			g.drawBoard()

		case picopad.KeyDown:
			g.curOff()
			g.curY++
			if g.curY >= mapY+mapH {
				g.curY = mapY
			}
			g.curOn()
			g.drawBoard()

		case picopad.KeyUp:
			g.curOff()
			g.curY--
			if g.curY < mapY {
				g.curY = mapY + mapH - 1
			}
			g.curOn()
			g.drawBoard()

		case picopad.KeyA: // flip the cell
			g.curFlip()
			g.drawBoard()

		case picopad.KeyB: // run
			g.curOff()
			g.saveSlot()
			g.runLife()
			g.loadSlot()
			g.curOn()
			g.drawBoard()

		case picopad.KeyX: // the next slot
			g.curOff()
			g.saveSlot()
			g.selSlot++
			if g.selSlot >= slotNum {
				g.selSlot = 0
			}
			g.loadSlot()
			g.curOn()
			g.drawBoard()

		case picopad.KeyY:
			return
		}
	}
}
