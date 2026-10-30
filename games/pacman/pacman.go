// Package pacman is the PacMan of PicoLibSDK (PicoPad/GAME/PACMAN, by
// Miroslav Nemecek, Unlicense), ported from C to package picopad. It
// follows the C source function by function (src/game.cpp, src/open.cpp,
// src/main.cpp); the C globals are the fields of Game.
//
// The open screen shows the ghosts, then the game starts: the arrows
// steer, A pauses (a key goes on), Y goes back to the open screen; Y on the
// open screen ends Run (on the PicoPad: back to the boot loader). There is
// no sound yet: the PLAYSOUND calls of the C source are comments, and
// where the C source waits for a sound to end, it does not wait.
package pacman

import (
	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/picopad"
)

//go:generate go run ../../tools/picoimg -pkg pacman -root $PICOLIBSDK -o img.go PicoPad/GAME/PACMAN/img/logo.cpp PicoPad/GAME/PACMAN/img/sprites.cpp PicoPad/GAME/PACMAN/img/status.cpp PicoPad/GAME/PACMAN/img/tiles.cpp

const (
	width  = picopad.Width
	height = picopad.Height

	logoW      = 144
	logoH      = 34
	spriteW    = 20
	spriteH    = 20
	spriteImgW = 1360
	statusW    = 11
	statusH    = 11
	statusImgW = 99
	tileW      = 10
	tileH      = 10
	tileSize   = 10
	tilesImgW  = 430

	// game board
	mapW      = 28                     // width of game board (tiles)
	mapH      = 23                     // height of game board (tiles)
	mapSize   = mapW * mapH            // board size (644)
	mapWidth  = mapW * tileW           // width of game board in pixels (280)
	mapHeight = mapH * tileH           // height of game board in pixels (230)
	mapY      = 0                      // map Y coordinate
	mapX      = (width - mapWidth) / 2 // map X coordinate

	// tiles
	tileEmpty     = 0  // empty
	tileGate      = 29 // passable gate
	tileDot       = 30 // pac-dot
	tilePellet    = 31 // power pellet
	tilePelletOff = 34 // power pellet OFF
	tileReady1    = 35 // "Ready!", 6 tiles

	// sprites
	spritePacFull = 1
	spritePacL    = 2
	spritePacL2   = 3
	spritePacD    = 4
	spritePacD2   = 5
	spritePacR    = 6
	spritePacR2   = 7
	spritePacU    = 8
	spritePacU2   = 9
	spriteDie     = 8 // pacman die 1st sprite
	spriteDieNum  = 8
	spriteBlinky  = 16
	spritePinky   = 24
	spriteInky    = 32
	spriteClyde   = 40
	spriteEye     = 48
	spriteFrigh   = 52
	spriteFrozen  = 54
	spriteFruit   = 56
	sprite200     = 64
	sprite400     = 65
	sprite800     = 66
	sprite1600    = 67

	phaseNum = 4 // number of animation phases

	// directions
	dirL   = 0
	dirD   = 1
	dirR   = 2
	dirU   = 3
	dirNum = 4

	// sprite characters
	chPacman   = 0
	chBlinky   = 1
	chPinky    = 2
	chInky     = 3
	chClyde    = 4
	chFrozen   = 5
	chFrighten = 6
	chEyes     = 7
	chDie      = 8
	charNum    = 9

	// ghost states
	sScatter  = 0
	sChase    = 1
	sFrighten = 2
	sEaten    = 3

	// ghost home
	homeXMin = 11
	homeXMax = 16
	homeYMin = 10
	homeYMax = 11
	homeXMid = 13 // middle home X (gate)

	fruitTabMax    = 19
	frightenTabMax = 19

	speed      = 100                          // delay of 1 animation phase [ms]
	charSteps  = 2                            // animation phases per tile
	charSpeed  = tileSize / charSteps         // pixels per animation phase
	frighSpeed = tileSize * 3 / 4 / charSteps // frightened speed

	dotNum    = 210          // total number of dots (including 4 pellets)
	dotFruit1 = dotNum - 60  // fruit 1
	dotFruit2 = dotNum - 150 // fruit 2
)

// mapToX and mapToY convert map coordinates to sprite coordinates.
func mapToX(x int) int { return mapX + x*tileW + tileW/2 - spriteW/2 }
func mapToY(y int) int { return mapY + y*tileH + tileH/2 - spriteH/2 }

// xToMap and yToMap convert sprite coordinates to map coordinates.
func xToMap(x int) int { return (x - mapX + spriteW/2) / tileW }
func yToMap(y int) int { return (y - mapY + spriteH/2) / tileH }

// char is a character (sChar).
type char struct {
	x, y       int   // graphics coordinates on the screen
	sleepTimer int   // sleep timer
	dx, dy     int   // move vector (0: not moving)
	steps      int   // step counter to the next tile
	dir        uint8 // current direction dir*
	state      uint8 // ghost state s*, pacman: 0 normal or a die sprite
	anim       int   // sprite animation base index
}

// Game is PacMan on a console.
type Game struct {
	*picopad.Console

	board          [mapSize]uint8
	phase          uint8 // animation phase 0..3
	nextDir        uint8 // pacman next direction
	score          int
	maxScore       int
	level          int // current level 0...
	lives          int // remaining lives
	frightenCnt    int
	eaten          int // next eaten points
	scatterCnt     int
	chaseCnt       int
	scatterTabNext int // next entry in the scatter-chase table
	dotCnt         int // dots to eat
	fruitCnt       int
	dispNextTime   uint32
	fruitSpriteImg int // -1: not visible
	fruitSpriteX   int
	fruitSpriteY   int
	blink          uint8 // pellet blink counter
	chars          [5]char

	lastTime uint32 // the open screen
	quit     bool   // Y on the open screen
}

// New returns the game on d (at least 320x240) with the touch panel t;
// seed starts the random numbers. Set Pad to use a game controller.
func New(d hal.Display, t hal.Touch, seed uint32) *Game {
	return &Game{Console: picopad.New(d, t, seed)}
}

func (g *Game) putTile(x, y int, tile uint8) { g.board[x+y*mapW] = tile }

func (g *Game) getTile(x, y int) uint8 { return g.board[x+y*mapW] }

// drawNum draws n centred at x (the C source: DecUNum and DrawText at
// x - n*4), by characters: a string would allocate in every frame.
func (g *Game) drawNum(n, x, y int, col picopad.Color) {
	var buf [10]byte
	i := len(buf)
	for {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
		if n == 0 {
			break
		}
	}
	x -= (len(buf) - i) * 4
	for ; i < len(buf); i++ {
		g.DrawChar(buf[i], x, y, col)
		x += 8
	}
}

// dispAll draws everything and waits wait ms from the last time.
func (g *Game) dispAll(wait uint32) {
	// blink the pellets
	b := g.blink + 1
	if b == 3 {
		b = 0
		for y := 0; y < mapH; y++ {
			for x := 0; x < mapW; x++ {
				switch g.getTile(x, y) {
				case tilePellet:
					g.putTile(x, y, tilePelletOff)
				case tilePelletOff:
					g.putTile(x, y, tilePellet)
				}
			}
		}
	}
	g.blink = b

	g.DrawClear()

	// the board
	for y := 0; y < mapH; y++ {
		for x := 0; x < mapW; x++ {
			g.DrawImg4Pal(tilesImg, tilesImgPal, int(g.board[x+y*mapW])*tileW, 0, x*tileW+mapX, y*tileH+mapY, tileW, tileH, tilesImgW)
		}
	}

	// the fruit
	if g.fruitSpriteImg >= 0 {
		g.DrawBlit4Pal(spritesImg, spritesImgPal, g.fruitSpriteImg*spriteW, 0, g.fruitSpriteX,
			g.fruitSpriteY, spriteW, spriteH, spriteImgW, picopad.ColBlack)
	}

	// the score and the max. score
	g.drawNum(g.score, 115, height-9, picopad.ColWhite)
	col := picopad.ColWhite
	if g.maxScore == g.score && g.maxScore > 0 {
		col = picopad.ColGreen
	}
	g.drawNum(g.maxScore, 185, height-9, col)

	// the remaining lives
	for i := 0; i < 5; i++ {
		if i < g.lives {
			g.DrawImg4Pal(statusImg, statusImgPal, 0, 0, i*statusW+mapX, height-statusH, statusW, statusH, statusImgW)
		} else {
			g.DrawRect(i*statusW+mapX, height-statusH, statusW, statusH, picopad.ColBlack)
		}
	}

	// the level: its fruit and the ones before
	j, k := 0, g.level
	if k > 6 {
		j = k - 6
	}
	for i := 0; i < 7; i++ {
		if j > fruitTabMax-1 {
			j = fruitTabMax - 1
		}
		if j <= k {
			g.DrawImg4Pal(statusImg, statusImgPal, (int(fruitTab[j])+1)*statusW, 0, mapX+mapWidth-(i+1)*statusW,
				height-statusH, statusW, statusH, statusImgW)
		} else {
			g.DrawRect(mapX+mapWidth-(i+1)*statusW, height-statusH, statusW, statusH, picopad.ColBlack)
		}
		j++
	}

	// the sprites
	for i := 0; i < 5; i++ {
		ch := &g.chars[i]
		moving := ch.dx != 0 || ch.dy != 0
		k := i
		if ch.state == sFrighten {
			k = chFrighten
			if g.frightenCnt < 3000/speed && g.frightenCnt&4 == 0 {
				k = chFrozen
			}
		} else if ch.state == sEaten {
			k = chEyes
		}
		if i == chPacman && ch.state != 0 { // pacman dead
			k = int(ch.state)
		} else {
			ph := 1
			if moving {
				ph = int(g.phase)
			}
			k = int(spriteInx[k*4*4+int(ch.dir)*4+ph])
		}
		g.DrawBlit4Pal(spritesImg, spritesImgPal, k*spriteW, 0, ch.x, ch.y, spriteW, spriteH, spriteImgW, picopad.ColBlack)
	}

	g.DispUpdate()

	// wait
	wait *= 1000
	var t uint32
	for {
		t = g.Time()
		if t-g.dispNextTime >= wait {
			break
		}
	}
	g.dispNextTime = t
}

// shiftX shifts x in the direction (wrapped).
func shiftX(x, dir int) int {
	x += dirX[dir&3]
	if x < 0 {
		x += mapW
	}
	if x >= mapW {
		x -= mapW
	}
	return x
}

// shiftY shifts y in the direction (wrapped).
func shiftY(y, dir int) int {
	y += dirY[dir&3]
	if y < 0 {
		y += mapH
	}
	if y >= mapH {
		y -= mapH
	}
	return y
}

// getBoardDir returns the tile 1 step in the (wrapped) direction.
func (g *Game) getBoardDir(x, y, dir int) uint8 {
	return g.getTile(shiftX(x, dir), shiftY(y, dir))
}

// checkBoard returns whether the tile in the direction is free (gate: the
// gate is passable).
func (g *Game) checkBoard(x, y, dir int, gate bool) bool {
	b := g.getBoardDir(x, y, dir)
	return b == tileEmpty || b == tileDot || b == tilePellet || b == tilePelletOff || (b == tileGate && gate)
}

// addScore adds to the score; an extra pacman at 10000.
func (g *Game) addScore(score int) {
	g.score += score
	if g.score > g.maxScore {
		g.maxScore = g.score
	}
	if g.score >= 10000 && g.score-score < 10000 {
		g.lives++
		g.dispAll(1)
		// PLAYSOUND(ExtraPacSnd); while (PlayingSound()) {}
	}
}

// eatDot counts an eaten dot; the fruit appears at two counts.
func (g *Game) eatDot() {
	g.dotCnt--
	if g.dotCnt == dotFruit1 || g.dotCnt == dotFruit2 {
		i := g.level
		if i > fruitTabMax-1 {
			i = fruitTabMax - 1
		}
		g.fruitSpriteImg = int(fruitTab[i]) + spriteFruit
		g.fruitSpriteX = (width - spriteW) / 2
		g.fruitSpriteY = mapToY(13)
		g.fruitCnt = 10 * 1000 / speed // 10 seconds
	}
}

// move moves the characters; it returns false on death.
func (g *Game) move() bool {
	xpac := xToMap(g.chars[chPacman].x)
	ypac := yToMap(g.chars[chPacman].y)
	dirpac := int(g.chars[chPacman].dir)
	var targx, targy int // C: kept from the ghost before when not set

	for i := 0; i < 5; i++ {
		ch := &g.chars[i]

		// move
		ch.x += ch.dx
		if ch.x >= mapX+mapWidth-spriteW/2 {
			ch.x -= mapWidth
		}
		if ch.x < mapX-spriteW/2 {
			ch.x += mapWidth
		}
		ch.y += ch.dy
		if ch.y >= mapY+mapHeight-spriteH/2 {
			ch.y -= mapHeight
		}
		if ch.y < mapY-spriteH/2 {
			ch.y += mapHeight
		}
		ch.steps--
		if ch.steps > 0 {
			continue
		}

		// end of the step
		if i == chPacman {
			// correct the coordinates if the step is not precise
			xpac = xToMap(ch.x)
			ypac = yToMap(ch.y)
			ch.x = mapToX(xpac)
			ch.y = mapToY(ypac)

			// collision with a ghost
			for j := 1; j < 5; j++ {
				ch2 := &g.chars[j]
				if (ch2.x-ch.x)*(ch2.x-ch.x)+(ch2.y-ch.y)*(ch2.y-ch.y) < (spriteW-3)*(spriteH-3) {
					if ch2.state == sFrighten {
						// eat the ghost
						ch2.state = sEaten
						// PLAYSOUND(EatGhostSnd)
						g.addScore(g.eaten)
						g.eaten *= 2
					} else if ch2.state != sEaten {
						// pacman's death
						// PLAYSOUND(DeathSnd)
						g.lives--
						for n := 1; n < 5; n++ { // hide the ghosts
							g.chars[n].y = height + 100
						}
						for n := spriteDie; n < spriteDie+spriteDieNum; n++ {
							ch.state = uint8(n)
							g.dispAll(300)
						}
						ch.state = 0
						ch.y = height + 100
						g.dispAll(2000)
						return false
					}
				}
			}

			// collect a dot
			tile := g.getTile(xpac, ypac)
			if tile == tileDot {
				// SetNextSound(WakkaSnd)
				g.putTile(xpac, ypac, tileEmpty)
				g.addScore(10)
				g.eatDot()
			} else if tile == tilePellet || tile == tilePelletOff {
				// collect a pellet: frighten the ghosts
				// SetNextSound(FrightenSnd)
				if g.level < frightenTabMax {
					g.frightenCnt = int(frightenTab[g.level]) * 1000 / speed
				}
				g.eaten = 200
				g.putTile(xpac, ypac, tileEmpty)
				g.addScore(50)
				for j := 1; j < 5; j++ {
					ch2 := &g.chars[j]
					if ch2.state != sEaten {
						if g.frightenCnt > 0 {
							ch2.state = sFrighten
						}
						ch2.dir = (ch2.dir + 2) & 3 // turn around
					}
				}
				g.eatDot()
			} else if g.fruitCnt > 0 &&
				(ch.x-g.fruitSpriteX)*(ch.x-g.fruitSpriteX)+(ch.y-g.fruitSpriteY)*(ch.y-g.fruitSpriteY) < spriteW*spriteH {
				// collect the bonus fruit
				j := g.level
				if j > fruitTabMax-1 {
					j = fruitTabMax - 1
				}
				g.addScore(bonusTab[fruitTab[j]])
				// PLAYSOUND(EatFruitSnd)
				g.fruitSpriteImg = -1
				g.fruitCnt = 0
			}

			// change the direction
			if g.checkBoard(xpac, ypac, int(g.nextDir), false) {
				dirpac = int(g.nextDir)
				g.chars[chPacman].dir = uint8(dirpac)
			}

			if g.checkBoard(xpac, ypac, dirpac, false) {
				// do a step
				dx := shiftX(xpac, dirpac) - xpac
				if dx < -mapW/2 {
					dx += mapW
				}
				if dx > mapW/2 {
					dx -= mapW
				}
				ch.dx = dx * charSpeed
				dy := shiftY(ypac, dirpac) - ypac
				if dy < -mapH/2 {
					dy += mapH
				}
				if dy > mapH/2 {
					dy -= mapH
				}
				ch.dy = dy * charSpeed
				ch.steps = charSteps
			} else {
				// stop
				ch.dx, ch.dy = 0, 0
				ch.steps = 1
			}
			continue
		}

		// ghosts
		x0 := xToMap(ch.x)
		y0 := yToMap(ch.y)
		dir := int(ch.dir)

		// correct the coordinates (the step may not be precise when eaten)
		ch.y = mapToY(y0)
		ch.x = mapToX(x0)

		athome := x0 >= homeXMin && x0 <= homeXMax && y0 >= homeYMin && y0 <= homeYMax
		atgate := (dir == dirU || dir == dirD) && x0 == homeXMid && y0 == 9

		if athome || atgate {
			if atgate {
				ch.x = (width - spriteW) / 2
			}

			// sleeping at home
			if ch.sleepTimer > 0 {
				if y0 == homeYMin {
					ch.dy = charSpeed
					ch.dir = dirD
				}
				if y0 == homeYMax {
					ch.dy = -charSpeed
					ch.dir = dirU
				}
				ch.steps = charSteps
				continue
			}

			// eaten: go home
			if ch.state == sEaten {
				targx, targy = homeXMid, homeYMax
			}

			// respawn at home: go out of the house
			if athome {
				if ch.state == sEaten {
					if g.scatterCnt > 0 {
						ch.state = sScatter
					} else {
						ch.state = sChase
					}
				}
				targx, targy = homeXMid, 8
			}
		} else {
			// the target tile
			switch ch.state {
			case sScatter: // the corners
				switch i {
				case chBlinky: // right up
					targx, targy = mapW-3, -3
				case chPinky: // left up
					targx, targy = 3, -3
				case chInky: // right down
					targx, targy = mapW-1, mapH+2
				case chClyde: // left down
					targx, targy = 0, mapH+2
				}

			case sChase: // follow pacman
				switch i {
				case chBlinky: // the tile of pacman
					targx, targy = xpac, ypac
				case chPinky: // 4 steps before pacman (up: and 4 to the left)
					targx = xpac + 4*dirX[dirpac]
					targy = ypac + 4*dirY[dirpac]
					if dirpac == dirU {
						targx -= 4
					}
				case chInky: // 2 steps before pacman, then the inverse vector to Blinky
					targx = xpac + 2*dirX[dirpac]
					targy = ypac + 2*dirY[dirpac]
					if dirpac == dirU {
						targx -= 2
					}
					targx += targx - xToMap(g.chars[chBlinky].x)
					targy += targy - yToMap(g.chars[chBlinky].y)
				case chClyde: // pacman when further than 8 tiles, else the left down corner
					targx = xpac - x0
					targy = ypac - y0
					if targx*targx+targy*targy >= 8*8 {
						targx, targy = xpac, ypac
					} else {
						targx, targy = 0, mapH+2
					}
				}

			case sEaten: // go home
				targx, targy = homeXMid, homeYMax

			default: // random
				targx = x0 + int(g.RandS8MinMax(-10, +10))
				targy = y0 + int(g.RandS8MinMax(-10, +10))
			}
		}

		// the best next tile
		bestdist := 100 * 100
		bestdir, bestx, besty := dir, x0, y0
		last := dir + 1
		if athome {
			last = dir + 2
		}
		for dir2 := dir - 1; dir2 <= last; dir2++ {
			if g.checkBoard(x0, y0, dir2, athome || ch.state == sEaten) {
				x2 := shiftX(x0, dir2)
				y2 := shiftY(y0, dir2)
				if dist := (targx-x2)*(targx-x2) + (targy-y2)*(targy-y2); dist < bestdist {
					bestdist = dist
					bestx, besty = x2, y2
					bestdir = dir2
				}
			}
		}

		if bestdist == 100*100 {
			// not found
			ch.dx, ch.dy = 0, 0
			ch.steps = 1
			continue
		}

		// start moving to the next tile
		ch.dir = uint8(bestdir & 3)
		dx := bestx - x0
		if dx < mapW/2 {
			dx += mapW
		}
		if dx > mapW/2 {
			dx -= mapW
		}
		dy := besty - y0
		if dy < mapH/2 {
			dy += mapH
		}
		if dy > mapH/2 {
			dy -= mapH
		}
		switch ch.state {
		case sFrighten:
			ch.dx = dx * (charSpeed / 2)
			ch.dy = dy * (charSpeed / 2)
			ch.steps = charSteps * 2
		case sEaten:
			ch.dx = dx * charSpeed * 2
			ch.dy = dy * charSpeed * 2
			ch.steps = charSteps / 2
		default:
			ch.dx = dx * charSpeed
			ch.dy = dy * charSpeed
			ch.steps = charSteps
		}
	}
	return true
}

// newGame starts a level (cont: continue the level after a death).
func (g *Game) newGame(cont bool) {
	g.dispNextTime = g.Time()

	// hide the sprites
	for i := range g.chars {
		g.chars[i] = char{steps: charSteps / 2, state: sScatter, anim: i * 4 * 4, x: width, y: height}
	}

	if !cont {
		g.frightenCnt = 0
		g.dotCnt = dotNum
		g.fruitSpriteImg = -1
		g.fruitCnt = 0
		g.board = boardTemp
	} else {
		for i := 11; i <= 16; i++ { // READY
			g.putTile(i, 13, uint8(tileReady1+i-11))
		}
	}

	// PLAYSOUND(StartSnd)
	g.dispAll(2000)

	g.phase = 0
	g.nextDir = dirL
	g.blink = 0

	// the sprites
	c := &g.chars
	c[chPacman].x, c[chPacman].y = (width-spriteW)/2, mapToY(17)
	c[chPacman].dx, c[chPacman].dir = charSpeed, dirL
	c[chBlinky].x, c[chBlinky].y = (width-spriteW)/2, mapToY(8)
	c[chBlinky].dx, c[chBlinky].dir = charSpeed, dirL
	c[chPinky].x, c[chPinky].y = (width-spriteW)/2, mapToY(10)
	c[chPinky].dy, c[chPinky].dir = charSpeed, dirD
	c[chInky].x, c[chInky].y = (width-spriteW)/2-spriteW, mapToY(10)
	c[chInky].dy, c[chInky].dir = -charSpeed, dirU
	c[chInky].sleepTimer = 5000 / speed // 5 seconds
	c[chClyde].x, c[chClyde].y = (width-spriteW)/2+spriteW, mapToY(10)
	c[chClyde].dy, c[chClyde].dir = -charSpeed, dirU
	c[chClyde].sleepTimer = 15000 / speed // 15 seconds

	g.dispAll(2000) // while (PlayingSound()) {}

	// clear READY
	for i := 11; i <= 16; i++ {
		g.putTile(i, 13, tileEmpty)
	}

	// the scatter timer
	i := g.level
	if i > 4 {
		i = 4
	}
	g.scatterCnt = int(scatterTab[i*4*2]) * 1000 / speed
	g.scatterTabNext = 1
	g.chaseCnt = 0
}

// wakeUp counts the sleep of the ghosts.
func (g *Game) wakeUp() {
	for i := 1; i < 5; i++ {
		if g.chars[i].sleepTimer > 0 {
			g.chars[i].sleepTimer--
		}
	}
}

// setGhosts changes the ghosts in state from to state to, turning them
// around if turn.
func (g *Game) setGhosts(from, to uint8, turn bool) {
	for i := 1; i < 5; i++ {
		if g.chars[i].state == from {
			g.chars[i].state = to
			if turn {
				g.chars[i].dir = (g.chars[i].dir + 2) & 3
			}
		}
	}
}

// unfrighten counts the frighten, scatter and chase times.
func (g *Game) unfrighten() {
	i := g.level
	if i > 4 {
		i = 4
	}
	switch {
	case g.frightenCnt > 0:
		g.frightenCnt--
		if g.frightenCnt == 0 {
			if g.scatterCnt > 0 {
				g.setGhosts(sFrighten, sScatter, false)
			} else {
				g.setGhosts(sFrighten, sChase, false)
			}
		}
	case g.scatterCnt > 0:
		g.scatterCnt--
		if g.scatterCnt == 0 {
			g.setGhosts(sScatter, sChase, true)
			g.chaseCnt = int(scatterTab[i*4*2+g.scatterTabNext]) * 1000 / speed
			g.scatterTabNext++
		}
	case g.chaseCnt > 0:
		g.chaseCnt--
		if g.chaseCnt == 0 {
			g.setGhosts(sChase, sScatter, true)
			g.scatterCnt = int(scatterTab[i*4*2+g.scatterTabNext]) * 1000 / speed
			g.scatterTabNext++
		}
	}
}

// openGame plays a game until the last life or Y.
func (g *Game) openGame() {
	g.score = 0
	g.level = 0
	g.lives = 4
	g.newGame(false)

	for {
		switch c := g.KeyGet(); c {
		case picopad.KeyLeft:
			g.nextDir = dirL
		case picopad.KeyRight:
			g.nextDir = dirR
		case picopad.KeyUp:
			g.nextDir = dirU
		case picopad.KeyDown:
			g.nextDir = dirD
		case picopad.KeyA: // pause
			g.KeyFlush()
			g.dispAll(1)
			for g.KeyGet() == picopad.NoKey {
			}
		case picopad.KeyY:
			return
		case picopad.NoKey:
		default:
			g.KeyFlush()
		}

		if !g.step() {
			return
		}
	}
}

// step is a frame of the game; it returns false after the last life.
func (g *Game) step() bool {
	g.phase = (g.phase + 1) & 3
	g.wakeUp()
	g.unfrighten()

	// hide the fruit
	if g.fruitCnt > 0 {
		g.fruitCnt--
		if g.fruitCnt == 0 {
			g.fruitSpriteImg = -1
		}
	}

	if !g.move() {
		// death
		if g.lives < 0 {
			return false
		}
		g.newGame(true)
	}

	g.dispAll(speed)

	// the next level
	if g.dotCnt <= 0 {
		// StopSound()
		g.WaitMs(2000)
		g.level++
		g.newGame(false)
	}
	return true
}

// Run plays until Y on the open screen (main of the C source, which then
// resets to the boot loader). With OneGame it returns after a game (the
// last life, or Y).
func (g *Game) Run() {
	g.lastTime = g.Time()
	g.maxScore = 0
	g.quit = false
	for {
		g.openScreen()
		if g.quit {
			return
		}
		g.openGame()
		if g.OneGame {
			return
		}
	}
}
