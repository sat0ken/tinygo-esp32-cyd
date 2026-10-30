// Package train is the Train of PicoLibSDK (PicoPad/GAME/TRAIN, by
// Miroslav Nemecek, Unlicense), ported from C to package picopad. It
// follows the C source function by function (src/game.cpp, src/main.cpp);
// the C globals are the fields of Game.
//
// A logic game: the train collects the things on the board, each one a
// new wagon, and leaves through the gate, which opens when everything is
// collected; it must not run into a wall or itself. The arrows start and
// steer the train, A enters a password (arrows, A: OK, B: cancel), X
// shows the solution of the level, Y starts the level again (before the
// train starts: back to the demo). A key ends the demo; Y there ends Run
// (on the PicoPad: back to the boot loader). There is no sound yet: the
// PLAYSOUND calls of the C source are comments.
package train

import (
	"strconv"

	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/picopad"
)

//go:generate go run ../../tools/picoimg -pkg train -root $PICOLIBSDK -o tiles.go PicoPad/GAME/TRAIN/img/tiles.cpp

const (
	levFirst  = 1   // first level
	levNum    = 50  // number of levels (without the demo scene)
	gameSpeed = 166 // delay between game steps [ms] (6 steps per second)

	width     = picopad.Width
	height    = picopad.Height
	tilesImgW = 320 // TILESIMGW
	tilesNumX = 20  // number of tiles in X direction

	mapW    = 20          // width of game board
	mapH    = 12          // height of game board
	mapSize = mapW * mapH // size of game board
	mapY    = 16          // board Y coordinate

	boxW = 16 * 8                  // width of info box
	boxH = 5 * 16                  // height of info box
	boxX = (width - boxW) / 2      // coordinate X of info box
	boxY = (height - 8 - boxH) / 2 // coordinate Y of info box

	tileSize = 16 // tile size in number of pixels
	pswLen   = 5  // password length

	itemMin  = 0*tilesNumX + 0  // ITEMS(0,0): items
	itemMax  = 2*tilesNumX + 19 // ITEMS(2,19)
	wagonMin = 3*tilesNumX + 0  // ITEMS(3,0): wagons
	locoR    = 7*tilesNumX + 2  // locomotive right
	locoU    = 7*tilesNumX + 1  // locomotive up
	locoL    = 7*tilesNumX + 0  // locomotive left
	locoD    = 7*tilesNumX + 3  // locomotive down
	locoMin  = 7*tilesNumX + 0
	locoMax  = 7*tilesNumX + 11
	gate     = 7*tilesNumX + 12
	gateMin  = 7*tilesNumX + 12
	gateMax  = 7*tilesNumX + 17
	wall     = 7*tilesNumX + 18 // wall tile
	empty    = 7*tilesNumX + 19 // empty tile
	crash    = 8*tilesNumX + 0  // crash
	crashMax = 8*tilesNumX + 9

	dirR = 0 // right
	dirU = 1 // up
	dirL = 2 // left
	dirD = 3 // down

	sWait   = 0 // wait for start
	sGo     = 1 // game progress
	sCrash  = 2 // crash
	sFinish = 3 // game finish
	sHelp   = 4 // help mode

	levelTxt = "LEVEL"
	pswTxt   = "password"
	levelHlp = "KEY-A:PSW"
	okHlp    = "KEY-A:OK"
	escHlp   = "KEY-B:CANCEL"
	congTxt  = "CONGRATULATIONS!"
	cong2Txt = "You became absolute"
	cong3Txt = "winner of the game!"
	scoreHlp = "SCORE"
)

// Game is the Train on a console.
type Game struct {
	*picopad.Console

	board [mapSize]uint8 // game board
	dir   [mapSize]uint8 // direction map dir*

	level        uint8  // current level
	headX, headY int    // head (loco) coordinates
	gateX, gateY int    // gate coordinates
	length       int    // length of the train (including loco)
	itemNum      int    // number of items on game board
	state        uint8  // game state s*
	phase        uint8  // current animation phase (0, 1, 2)
	curDir       uint8  // current direction dir*
	score        int    // current score
	lastTime     uint32 // time of last step

	keyBuf1, keyBuf2 uint8 // key buffers for 2 keys (NoKey: no key)
}

// New returns the game on d (at least 320x240) with the touch panel t;
// seed starts the random numbers (the levels of the demo). Set Pad to use
// a game controller.
func New(d hal.Display, t hal.Touch, seed uint32) *Game {
	return &Game{Console: picopad.New(d, t, seed)}
}

// tile maps a character of levels to its tile.
func tile(c byte) uint8 {
	switch c {
	case '.':
		return empty
	case '#':
		return wall
	case '>':
		return locoR
	case '+':
		return gate
	}
	return itemMin + c - 'a'
}

// wagonDir returns the offset of the wagon tiles of a direction (a row of
// tiles a direction: left, up, right, down).
func wagonDir(d uint8) uint8 {
	switch d {
	case dirU:
		return 20
	case dirR:
		return 2 * 20
	case dirD:
		return 3 * 20
	}
	return 0
}

// waitStep updates the display and waits for the next game step.
func (g *Game) waitStep() {
	g.DispUpdate()
	var t uint32
	for {
		t = g.Time()
		if t-g.lastTime >= gameSpeed*1000 {
			break
		}
	}
	g.lastTime = t
}

// infoClear clears the info row.
func (g *Game) infoClear() {
	g.DrawRect(0, mapY+mapH*tileSize, width, 16, picopad.ColBlack)
}

// infoDispTextBg displays text in the info row (the font is FontBold8x16,
// the default, which the game does not change).
func (g *Game) infoDispTextBg(x int, txt string, col, bgcol picopad.Color) {
	g.DrawTextBg(txt, x, mapY+mapH*tileSize, col, bgcol)
}

// infoDispText displays text in the info row.
func (g *Game) infoDispText(x int, txt string, col picopad.Color) {
	g.infoDispTextBg(x, txt, col, picopad.ColBlack)
}

// infoRow displays the info row: the score and the level.
func (g *Game) infoRow() {
	g.infoClear()
	g.infoDispText(0, scoreHlp, picopad.ColGreen)
	g.infoDispText(6*8, strconv.Itoa(g.score), picopad.ColYellow)
	g.infoDispText(16*8, levelHlp, picopad.ColRed)
	g.infoDispText(width-8*8, levelTxt, picopad.ColGreen)
	g.infoDispText(width-2*8, strconv.Itoa(int(g.level)), picopad.ColYellow)
}

// psw enters a password.
func (g *Game) psw() {
	g.infoClear()
	g.infoDispText(0, okHlp, picopad.ColGreen)
	g.infoDispText(width-len(escHlp)*8, escHlp, picopad.ColGreen)

	// load the password of the level
	var buf [pswLen]byte
	copy(buf[:], levelPsw[int(g.level)*pswLen:])

	x := (width-(len(pswTxt)+1+5)*8)/2 - 16
	g.infoDispText(x, pswTxt, picopad.ColRed)
	x += (len(pswTxt) + 1) * 8

	pos := 0
	for {
		// display the password (by characters: a string of buf would
		// allocate in the loop)
		for k, ch := range buf {
			if k == pos {
				g.DrawCharBg(ch, x+k*8, mapY+mapH*tileSize, picopad.ColBlack, picopad.ColYellow)
			} else {
				g.DrawCharBg(ch, x+k*8, mapY+mapH*tileSize, picopad.ColYellow, picopad.ColBlack)
			}
		}
		g.DispUpdate()

		switch g.KeyGet() {
		case picopad.KeyB, picopad.KeyX, picopad.KeyY: // Esc
			g.infoRow()
			return
		case picopad.KeyA: // OK: search the password
			for i := 1; i <= levNum; i++ {
				if levelPsw[i*pswLen:(i+1)*pswLen] == string(buf[:]) {
					g.level = uint8(i)
					break
				}
			}
			// restart this scene
			g.infoRow()
			g.openLevel()
			g.initLevel()
			return
		case picopad.KeyLeft:
			pos--
			if pos < 0 {
				pos = pswLen - 1
			}
		case picopad.KeyRight:
			pos++
			if pos >= pswLen {
				pos = 0
			}
		case picopad.KeyUp:
			buf[pos]++
			if buf[pos] > 'Z' {
				buf[pos] = 'A'
			}
		case picopad.KeyDown:
			buf[pos]--
			if buf[pos] < 'A' {
				buf[pos] = 'Z'
			}
		}
	}
}

// dispTile displays one tile of the board.
func (g *Game) dispTile(x, y int) {
	t := int(g.board[x+y*mapW])
	row := t / tilesNumX
	col := t - row*tilesNumX
	g.DrawImg4Pal(tilesImg, tilesImgPal, col*tileSize, row*tileSize,
		x*tileSize, y*tileSize+mapY, tileSize, tileSize, tilesImgW)
}

// dispBoard displays all the tiles.
func (g *Game) dispBoard() {
	for y := 0; y < mapH; y++ {
		for x := 0; x < mapW; x++ {
			g.dispTile(x, y)
		}
	}
}

// putTile puts a tile on the board.
func (g *Game) putTile(x, y int, t uint8) {
	g.board[x+y*mapW] = t
	g.dispTile(x, y)
}

// getTile returns a tile of the board.
func (g *Game) getTile(x, y int) uint8 { return g.board[x+y*mapW] }

// clearLevel fills the board with a tile, column by column.
func (g *Game) clearLevel(t uint8) {
	for x := 0; x < mapW; x++ {
		for y := 0; y < mapH; y++ {
			g.putTile(x, y, t)
		}
		g.DispUpdate()
		g.WaitMs(20)
	}
}

// congLevel congratulates after the last level.
func (g *Game) congLevel() {
	g.clearLevel(wall)
	g.DrawRect(boxX-3*8, boxY, boxW+6*8, boxH, picopad.ColBlack)
	g.DrawFrame(boxX-1-3*8, boxY-1, boxW+2+6*8, boxH+2, picopad.ColWhite)
	g.DrawText(congTxt, (width-len(congTxt)*8)/2, boxY+16, picopad.ColYellow)
	g.DrawText(cong2Txt, (width-len(cong2Txt)*8)/2, boxY+2*16, picopad.ColGreen)
	g.DrawText(cong3Txt, (width-len(cong3Txt)*8)/2, boxY+3*16, picopad.ColGreen)
	g.infoRow()
	g.DispUpdate()

	g.WaitMs(250)
	g.KeyFlush()
	for g.KeyGet() == picopad.NoKey {
	}
	g.clearLevel(empty)
}

// openLevel shows the number and the password of the level.
func (g *Game) openLevel() {
	g.clearLevel(wall)
	g.DrawRect(boxX, boxY, boxW, boxH, picopad.ColBlack)
	g.DrawFrame(boxX-1, boxY-1, boxW+2, boxH+2, picopad.ColWhite)

	// the level ("LEVEL 12")
	buf := strconv.Itoa(int(g.level))
	x := (width - (len(levelTxt)+1+len(buf))*8) / 2
	g.DrawText(levelTxt, x, boxY+16, picopad.ColGreen)
	x += (len(levelTxt) + 1) * 8
	g.DrawText(buf, x, boxY+16, picopad.ColYellow)

	// the password
	x = (width - (len(pswTxt)+1+pswLen)*8) / 2
	g.DrawText(pswTxt, x, boxY+3*16, picopad.ColGreen)
	x += (len(pswTxt) + 1) * 8
	g.DrawText(levelPsw[int(g.level)*pswLen:int(g.level+1)*pswLen], x, boxY+3*16, picopad.ColYellow)

	g.infoRow()
	g.DispUpdate()

	g.WaitMs(250)
	g.KeyFlush()
	for g.KeyGet() == picopad.NoKey {
	}
	g.clearLevel(empty)
}

// initLevel initializes the current level.
func (g *Game) initLevel() {
	// copy the items to the board
	lev := levels[int(g.level)*mapSize:]
	g.itemNum = 0
	for y := 0; y < mapH; y++ {
		for x := 0; x < mapW; x++ {
			b := tile(lev[x+y*mapW])
			g.board[x+y*mapW] = b
			if b >= locoMin && b <= locoMax {
				g.headX, g.headY = x, y
			} else if b == gate {
				g.gateX, g.gateY = x, y
			} else if b >= itemMin && b <= itemMax {
				g.itemNum++
			}
		}
	}

	g.state = sWait
	g.length = 1
	g.curDir = dirR
	g.phase = 0
	g.keyBuf1 = picopad.NoKey
	g.keyBuf2 = picopad.NoKey

	if g.level == 0 {
		g.infoClear()
	} else {
		g.infoRow()
	}
	g.dispBoard()
	g.lastTime = g.Time()
}

// stepLevel moves the train one step.
func (g *Game) stepLevel() {
	// next position
	x, y := g.headX, g.headY
	d := g.curDir
	switch d {
	case dirL:
		x--
	case dirU:
		y--
	case dirR:
		x++
	case dirD:
		y++
	}

	// crash: out of the board, or not an empty field, an open gate or an
	// item
	var b uint8
	if x >= 0 && x < mapW && y >= 0 && y < mapH {
		b = g.board[x+y*mapW]
	}
	if x < 0 || x >= mapW || y < 0 || y >= mapH ||
		(b != empty && (b <= gateMin || b > gateMax) && (b < itemMin || b > itemMax)) {
		g.state = sCrash
		b = crash
		// PLAYSOUND(CrashSnd)
		for {
			g.putTile(g.headX, g.headY, b)
			g.waitStep()
			b++
			if b > crashMax {
				b = crashMax - 2
			}
			if g.KeyGet() != picopad.NoKey {
				break
			}
		}
		g.initLevel()
		return
	}

	// set the loco to the new position
	xold, yold := g.headX, g.headY
	var b2 uint8
	switch d {
	case dirL:
		b2 = locoL
	case dirU:
		b2 = locoU
	case dirR:
		b2 = locoR
	default:
		b2 = locoD
	}
	g.putTile(x, y, b2)
	g.dir[x+y*mapW] = d
	g.headX, g.headY = x, y

	// move the wagons
	for i := g.length - 1; i > 0; i-- {
		x, y = xold, yold

		// position of the previous wagon
		d = g.dir[x+y*mapW]
		switch d {
		case dirL:
			x++
		case dirU:
			y++
		case dirR:
			x--
		case dirD:
			y--
		}

		// change the direction of the previous wagon
		b2 = g.getTile(x, y)
		for b2 >= wagonMin+20 {
			b2 -= 20
		}
		b2 += wagonDir(d)
		g.putTile(xold, yold, b2)

		xold, yold = x, y
	}

	if b >= itemMin && b <= itemMax {
		// collect the item: a new wagon
		// PLAYSOUND(CollectSnd)
		g.length++
		b += wagonMin - itemMin
		for b >= wagonMin+20 {
			b -= 20
		}
		b += wagonDir(g.dir[xold+yold*mapW])
		g.putTile(xold, yold, b)

		// start opening the gate
		g.itemNum--
		if g.itemNum == 0 {
			g.putTile(g.gateX, g.gateY, gateMin+1)
		}

		if g.state == sGo {
			g.score += 10
			g.infoRow()
		}
	} else {
		// PLAYSOUND(StepSnd)
		g.putTile(xold, yold, empty)
	}

	// the gate
	if g.headX == g.gateX && g.headY == g.gateY {
		s := g.state
		g.state = sFinish
		g.DispUpdate()
		g.WaitMs(500)
		g.KeyFlush()
		if s == sHelp {
			return
		}

		// PLAYSOUND(SuccessSnd)
		g.WaitMs(1000)

		// open the next level
		if g.level < levNum {
			g.level++
			g.openLevel()
		} else {
			g.congLevel()
		}
		g.initLevel()
	}
}

// animLevel animates the level, called every game step.
func (g *Game) animLevel() {
	for y := 0; y < mapH; y++ {
		for x := 0; x < mapW; x++ {
			b := g.getTile(x, y)
			switch {
			case b >= itemMin && b <= itemMax: // items
				for b >= itemMin+20 {
					b -= 20
				}
				g.putTile(x, y, b+g.phase*20)
			case b >= locoMin && b <= locoMax: // locomotive
				for b >= locoMin+4 {
					b -= 4
				}
				g.putTile(x, y, b+g.phase*4)
			case b > gateMin && b < gateMax: // opening gate
				g.putTile(x, y, b+1)
			}
		}
	}

	g.phase++
	if g.phase >= 3 {
		g.phase = 0

		// change the direction by keyboard (and potentially start the game)
		c := g.keyBuf1
		g.keyBuf1 = g.keyBuf2
		g.keyBuf2 = picopad.NoKey
		if c != picopad.NoKey {
			if g.state == sWait {
				g.state = sGo
			}
			switch c {
			case picopad.KeyDown:
				g.curDir = dirD
			case picopad.KeyRight:
				g.curDir = dirR
			case picopad.KeyLeft:
				g.curDir = dirL
			case picopad.KeyUp:
				g.curDir = dirU
			}
		}

		// one step
		if g.state == sGo || g.state == sHelp {
			g.stepLevel()
		}
	}
}

// helpLevel plays the solution of the level; it returns true on a key.
func (g *Game) helpLevel() bool {
	g.initLevel()
	g.state = sHelp
	s := solution[g.level]
	for g.state == sHelp {
		// next key from the solution
		if len(s) == 0 {
			g.state = sFinish
			// wait 1.6 seconds at the end of the scene
			for b := 10; b > 0; b-- {
				if g.KeyGet() != picopad.NoKey {
					return true
				}
				g.animLevel()
				g.waitStep()
			}
			return false
		}
		switch s[0] {
		case 'R':
			g.curDir = dirR
		case 'U':
			g.curDir = dirU
		case 'L':
			g.curDir = dirL
		case 'D':
			g.curDir = dirD
		}
		s = s[1:]

		if g.KeyGet() != picopad.NoKey {
			return true
		}
		g.animLevel()
		g.waitStep()
		if g.KeyGet() != picopad.NoKey {
			return true
		}
		g.animLevel()
		g.waitStep()
		if g.KeyGet() != picopad.NoKey {
			return true
		}
		g.animLevel()

		// the title on the demo screen, behind the train
		if g.level == 0 {
			if g.headY == 9 {
				g.DrawImg4Pal(tilesImg, tilesImgPal, 0, 9*16, 0, 8*16+mapY, g.headX*tileSize, 32, tilesImgW)
			} else if g.headY == 10 && g.headX >= 13 {
				g.DrawImg4Pal(tilesImg, tilesImgPal, 11*16+(g.headX-13)*16, 8*16,
					(g.headX+1)*16, 10*16+mapY, 6*16-(g.headX-13), 16, tilesImgW)
				g.DrawImg4Pal(tilesImg, tilesImgPal, 0, 9*16, 0, 8*16+mapY, tilesImgW, 32, tilesImgW)
			}
		}

		g.waitStep()
		if g.KeyGet() != picopad.NoKey {
			return true
		}
	}
	return false
}

// gameLoop is called every game step; it returns true on Y before the
// train starts.
func (g *Game) gameLoop() bool {
	g.animLevel()

	c := g.KeyGet()
	switch c {
	case picopad.NoKey:
	case picopad.KeyA: // enter password
		g.psw()
	case picopad.KeyY:
		if g.state != sGo {
			return true
		}
		g.initLevel() // restart the scene
	case picopad.KeyX: // help
		g.helpLevel()
		g.initLevel()
	case picopad.KeyRight, picopad.KeyUp, picopad.KeyLeft, picopad.KeyDown:
		// save the key into the key buffer
		if g.keyBuf1 == picopad.NoKey {
			g.keyBuf1 = c
		} else if g.keyBuf2 == picopad.NoKey {
			g.keyBuf2 = c
		} else {
			g.keyBuf1 = g.keyBuf2
			g.keyBuf2 = c
		}
	default:
		g.KeyFlush()
	}
	return false
}

// Run plays until Y in the demo (main of the C source, which then resets
// to the boot loader). With OneGame, Y in the game returns.
func (g *Game) Run() {
	g.DrawClear()
	g.level = levFirst
	g.score = 0
	for {
		// the demo: scene 0 and a random level, until a key
		i := g.level
		for {
			g.level = 0
			if g.helpLevel() {
				break
			}
			g.level = g.RandU8MinMax(1, levNum)
			if g.helpLevel() {
				break
			}
		}
		if g.KeyPressed(picopad.KeyY) {
			return
		}

		g.level = i
		g.initLevel()
		for !g.gameLoop() {
			g.waitStep()
		}
		if g.OneGame { // Y: the game has no end
			return
		}
	}
}
