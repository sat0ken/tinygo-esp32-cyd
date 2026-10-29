// Package fifteen is the Fifteen puzzle of PicoLibSDK (PicoPad/GAME/FIFTEEN,
// by Miroslav Nemecek, Unlicense), ported from C to package picopad. It
// follows the C source function by function; the C globals are the fields
// of Game.
//
// Sort the tiles from 1 to 15: the arrows move a tile into the hole (RIGHT
// moves the tile left of the hole to the right). A shuffles a new game, X
// lets the solver play (any button stops it), Y ends the game and Run
// returns (on the PicoPad: back to the boot loader). There is no sound
// yet: the PLAYSOUND calls of the C source are comments.
package fifteen

import (
	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/picopad"
)

//go:generate go run ../../tools/picoimg -pkg fifteen -root $PICOLIBSDK -o tiles.go PicoPad/GAME/FIFTEEN/img/tiles.cpp

const (
	tileW     = 48              // tile width
	tileH     = 48              // tile height
	tilesX    = 4               // game tiles in X direction (192 pixels)
	tilesY    = 4               // game tiles in Y direction (192 pixels)
	tilesNum  = tilesX * tilesY // number of tiles (16)
	tileEmpty = 15              // empty tile (the hole)
	tileWait  = 16              // wait tile
	tileWait2 = 17              // wait tile 2

	boardW = tilesX * tileW                // board width
	boardH = tilesY * tileH                // board height
	boardX = (picopad.Width - boardW) / 2  // board X coordinate
	boardY = (picopad.Height - boardH) / 2 // board Y coordinate

	preDepthMax = 14 // max. depth of the pre-solver (<= depthMax)
	depthMax    = 17 // max. depth of the solver
)

// solverTab are the groups of tiles the solver puts in place one after the
// other: the number of tiles (0 ends), then the tiles (1..).
var solverTab = [...]uint8{
	1, 1,
	1, 2,
	1, 5,
	1, 6,
	2, 3, 4,
	2, 7, 8,
	2, 9, 13,
	2, 10, 14,
	3, 11, 12, 15,
	0,
}

// Game is the Fifteen puzzle on a console.
type Game struct {
	*picopad.Console

	board  [tilesNum]uint8 // visible board
	pos    [tilesNum]uint8 // positions of the tiles (with the hole)
	posErr int             // tiles of the solved groups not in place
	locked [tilesNum]bool  // solved groups: locked tiles
	solved [tilesNum]bool  // solved groups: tiles to solve
	dist   int             // distance of the tiles from their places (0 = done)

	moves     [depthMax]int8 // moves
	movesNum  int            // number of moves
	depthMaxN int            // current max. depth (DepthMax)

	bestMoves    [depthMax]int8 // best moves
	bestMovesNum int            // number of best moves
	bestDist     int            // distance of the best moves
}

// New returns the game on d (at least 320x240) with the touch panel t;
// seed starts the random shuffles. Set Pad to use a game controller.
func New(d hal.Display, t hal.Touch, seed uint32) *Game {
	return &Game{Console: picopad.New(d, t, seed)}
}

// hole returns the position of the hole.
func (g *Game) hole() int { return int(g.pos[tileEmpty]) }

// holeX returns the X coordinate of the hole.
func (g *Game) holeX() int { return g.hole() & 3 }

// holeY returns the Y coordinate of the hole.
func (g *Game) holeY() int { return g.hole() >> 2 }

// check reports whether the solved groups are in place.
func (g *Game) check() bool { return g.posErr == 0 }

// drawBoard draws the board and updates the display.
func (g *Game) drawBoard() {
	for i := 0; i < tilesY; i++ {
		for j := 0; j < tilesX; j++ {
			b := int(g.board[i*tilesX+j])
			g.DrawImg4Pal(tilesImg, tilesImgPal, 0, b*tileH, j*tileW+boardX, i*tileH+boardY, tileW, tileH, tileW)
		}
	}
	g.DispUpdate()
}

// drawHole draws tile in the hole and updates the display (WaitOn,
// WaitOn2, WaitOff).
func (g *Game) drawHole(tile int) {
	x := g.holeX()*tileW + boardX
	y := g.holeY()*tileH + boardY
	g.DrawImg4Pal(tilesImg, tilesImgPal, 0, tile*tileH, x, y, tileW, tileH, tileW)
	g.DispUpdate()
}

// distance returns the distance of tile b at place i (4 times as far for
// the tiles to solve).
func (g *Game) distance(b uint8, i int) int {
	x := int(b&3) - i&3
	y := int(b>>2) - i>>2
	if g.solved[b] {
		x *= 4
		y *= 4
	}
	return x*x + y*y
}

// calcPos calculates the positions, the tiles out of place and the
// distance.
func (g *Game) calcPos() {
	err, dist := 0, 0
	for i := 0; i < tilesNum; i++ {
		b := g.board[i]
		if b != tileEmpty {
			if g.solved[b] && int(b) != i {
				err++
			}
			dist += g.distance(b, i)
		}
		g.pos[b] = uint8(i)
	}
	g.posErr = err
	g.dist = dist
}

// shift moves the hole by -4, -1, +1 or +4 (the coordinates are not
// checked).
func (g *Game) shift(sh int) {
	oldInx := g.hole()    // the tile's new place
	newInx := oldInx + sh // the tile's old place

	b := g.board[newInx]
	if int(b) == newInx && g.solved[b] {
		g.posErr++
	}
	g.dist -= g.distance(b, newInx)

	// move the tile
	g.board[oldInx] = b
	if int(b) == oldInx && g.solved[b] {
		g.posErr--
	}
	g.dist += g.distance(b, oldInx)
	g.pos[b] = uint8(oldInx)

	// the hole
	g.board[newInx] = tileEmpty
	g.pos[tileEmpty] = uint8(newInx)
}

// newGame starts a new game, shuffled or in order.
func (g *Game) newGame(shuffle bool) {
	g.DrawClear()

	for i := 0; i < tilesNum; i++ {
		g.locked[i] = false
		g.solved[i] = true
		g.board[i] = uint8(i)
		g.pos[i] = uint8(i)
	}
	g.dist = 0
	g.posErr = 0

	if shuffle {
		// PLAYSOUNDREP(ShufflingSnd)

		// shuffle the board (2 seconds)
		for i := 4000; i > 0; i-- {
			r := g.RandU8()
			if r >= 0x80 { // in X direction
				if r&1 == 0 {
					if g.holeX() > 0 {
						g.shift(-1)
					}
				} else if g.holeX() < tilesX-1 {
					g.shift(1)
				}
			} else { // in Y direction
				if r&1 == 0 {
					if g.holeY() > 0 {
						g.shift(-tilesX)
					}
				} else if g.holeY() < tilesY-1 {
					g.shift(tilesX)
				}
			}

			if i&0x7F == 0 {
				g.drawBoard()
				g.WaitMs(20)
			}
		}

		// StopSound()

		g.KeyFlush()
	}

	g.drawBoard()
}

// try tries moving the hole by sh if ok (it stays on the board) and the
// tile there is not locked: it shifts, notes the move at depth n, goes a
// level deeper (pre: in the pre-solver) and shifts back. It returns what
// the solver returned. (The four moves of each level are written out in
// the C source; a function value for the level would allocate in TinyGo.)
func (g *Game) try(n, sh int, ok, pre bool) bool {
	if !ok || g.locked[g.board[g.hole()+sh]] {
		return false
	}
	g.shift(sh)
	g.moves[n] = int8(sh)
	res := false
	if pre {
		g.preSolve1()
	} else {
		res = g.solve1()
	}
	g.shift(-sh)
	return res
}

// preSolve1 is one level of the pre-solver: it looks for the moves that
// bring the tiles nearest to their places.
func (g *Game) preSolve1() {
	if g.dist == 0 {
		return // completed
	}

	// note a better solution
	if g.dist < g.bestDist || g.dist == g.bestDist && g.movesNum < g.bestMovesNum {
		copy(g.bestMoves[:], g.moves[:g.movesNum])
		g.bestMovesNum = g.movesNum
		g.bestDist = g.dist
	}

	n := g.movesNum
	if n >= g.depthMaxN {
		return
	}
	g.movesNum = n + 1

	g.try(n, -1, g.holeX() > 0, true)            // hole left
	g.try(n, 1, g.holeX() < tilesX-1, true)      // hole right
	g.try(n, -tilesX, g.holeY() > 0, true)       // hole up
	g.try(n, tilesX, g.holeY() < tilesY-1, true) // hole down

	g.movesNum = n
}

// solve1 is one level of the solver: it returns true when the moves put
// the solved groups in place.
func (g *Game) solve1() bool {
	if g.posErr == 0 {
		return true // completed
	}

	n := g.movesNum
	if n >= g.depthMaxN {
		return false
	}
	g.movesNum = n + 1

	if g.try(n, -1, g.holeX() > 0, false) || // hole left
		g.try(n, 1, g.holeX() < tilesX-1, false) || // hole right
		g.try(n, -tilesX, g.holeY() > 0, false) || // hole up
		g.try(n, tilesX, g.holeY() < tilesY-1, false) { // hole down
		return true
	}

	g.movesNum = n
	return false
}

// play plays moves, 200ms each; it returns false if a key stops it.
func (g *Game) play(moves []int8) bool {
	for _, m := range moves {
		g.shift(int(m))
		// PLAYSOUND(MoveSnd)
		g.drawBoard()

		g.WaitMs(100)
		if g.KeyGet() != picopad.NoKey {
			return false
		}
		g.WaitMs(100)
		if g.KeyGet() != picopad.NoKey {
			return false
		}
	}
	return true
}

// solver solves the puzzle, a group of tiles after the other (any key
// stops it).
func (g *Game) solver() {
	t := 0
	faster1 := 7  // speed-up of the pre-solver
	faster2 := 10 // speed-up of the solver

	for i := range g.locked {
		g.locked[i] = false
		g.solved[i] = false
	}

	for {
		// prepare the next group
		g.movesNum = 0
		for i := range g.solved {
			if g.solved[i] {
				g.locked[i] = true // lock the tiles solved already
			}
		}
		n := int(solverTab[t])
		t++
		if n == 0 {
			break // end of the table
		}
		for ; n > 0; n-- {
			g.solved[solverTab[t]-1] = true
			t++
		}
		g.calcPos()

		for { // PRE_AGAIN
			err := true

			// pre-solver
			g.drawHole(tileWait)
			g.bestMovesNum = 0
			g.bestDist = g.dist
			g.depthMaxN = preDepthMax - faster1
			g.preSolve1()

			if g.bestMovesNum > 0 {
				g.drawHole(tileEmpty)
				if !g.play(g.bestMoves[:g.bestMovesNum]) {
					return
				}
				g.drawHole(tileWait)
				err = false

				// found some solution: slower solver
				if faster2 < 10 {
					faster2++
				}
			}

			if g.KeyGet() != picopad.NoKey {
				return
			}

			// solver
			g.drawHole(tileWait2)
			ok := false
			g.depthMaxN = 0
			for {
				g.depthMaxN++
				ok = g.solve1()
				if ok || g.depthMaxN >= depthMax-faster2 {
					break
				}
			}

			// not found: slower everything and try again
			if !ok && !g.check() {
				if err && faster1 > 0 {
					faster1--
				}
				if err && faster2 > 0 {
					faster2--
				}
				continue // PRE_AGAIN
			}

			if g.movesNum > 0 {
				g.drawHole(tileEmpty)
				if !g.play(g.moves[:g.movesNum]) {
					return
				}
				g.drawHole(tileWait2)
				err = false

				// found some solution: slower pre-solver
				if faster1 < 7 {
					faster1++
				}
			}

			// no luck with either: slower everything
			if err && faster1 > 0 {
				faster1--
			}
			if err && faster2 > 0 {
				faster2--
			}
			break
		}
	}

	for i := range g.locked {
		g.locked[i] = false
		g.solved[i] = true
	}
	g.drawHole(tileEmpty)
	g.calcPos()
	g.drawBoard()
}

// Run plays until Y (main of the C source, which then resets to the boot
// loader).
func (g *Game) Run() {
	g.newGame(true)

	for {
		switch g.KeyGet() {
		case picopad.KeyRight:
			if g.holeX() > 0 {
				g.shift(-1)
				// PLAYSOUND(MoveSnd)
				g.drawBoard()
			} // else PLAYSOUND(BumpSnd)

		case picopad.KeyLeft:
			if g.holeX() < tilesX-1 {
				g.shift(1)
				g.drawBoard()
			}

		case picopad.KeyUp:
			if g.holeY() < tilesY-1 {
				g.shift(tilesX)
				g.drawBoard()
			}

		case picopad.KeyDown:
			if g.holeY() > 0 {
				g.shift(-tilesX)
				g.drawBoard()
			}

		case picopad.KeyA: // new game
			g.newGame(true)

		case picopad.KeyX: // solver
			g.solver()
			for i := range g.locked {
				g.locked[i] = false
				g.solved[i] = true
			}
			g.calcPos()

		case picopad.KeyY:
			return
		}

		// solved
		if g.check() {
			// PLAYSOUND(FanfareSnd)

			// blink
			for i := 4; i > 0; i-- {
				g.WaitMs(150)
				g.DrawClear()
				g.DispUpdate()
				g.WaitMs(150)
				g.drawBoard()
			}

			g.KeyFlush()
			for g.KeyGet() == picopad.NoKey {
			}

			g.newGame(true)
		}
	}
}
