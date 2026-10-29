// Package zoom is ZOOM of PicoLibSDK (PicoPad/GAME/ZOOM, by Miroslav
// Nemecek, Unlicense), renamed Gate Runner, ported from C to package
// picopad. It follows the C source file by file (src/*.cpp); the C globals
// are the fields of Game.
//
// A ship flies low over a planet through the gates of pylons and shoots
// the saucers; each sector has four parts (the last one in space, with a
// mother ship). LEFT/RIGHT steer, UP/DOWN change the speed, A shoots, Y
// ends the game (and on the title Run).
//
// Differences from the C source:
//   - The C game is a version of a 1984 arcade game of another company,
//     and its title and pictures come from that game. All the pictures but
//     the grass (gradients) are new pixel art (artgen, art.go), and the
//     game has a new name; the title shows two lines of help in place of
//     the copyright of that company.
//   - A frame takes at least 20 ms, as in C, but the time waited for it
//     is not counted again in the next frame (C adds it to the next
//     frame: where a frame is drawn in less than 20 ms, the game went
//     twice as fast; on the PicoPad a frame takes longer).
//   - The title screen draws at most every 30 ms (the C loop is as fast as
//     the PicoPad can draw, and the stars move a step a loop).
//   - There is no sound.
package zoom

import (
	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/picopad"
)

//go:generate go run ../../tools/picoimg -pkg zoom -root $PICOLIBSDK -o img.go PicoPad/GAME/ZOOM/img/grass1.cpp PicoPad/GAME/ZOOM/img/grass2.cpp PicoPad/GAME/ZOOM/img/grass3.cpp PicoPad/GAME/ZOOM/img/grass4.cpp
//go:generate go run ./artgen

const (
	width  = picopad.Width
	height = picopad.Height

	frac    = 6  // fraction bits of the coordinates and the fuel
	horizon = 56 // Y of the horizon

	openFrame = 30 // the shortest loop of the title [ms]
)

// Game is the game on a console.
type Game struct {
	*picopad.Console

	level    int // sector - 1
	subLevel int // part of the sector (0..3)

	// ship (ship.go)
	shipX         int16 // with frac
	speed         int16 // with frac
	shipInCrash   int16 // > 0: the ship explodes
	shipCrashTime uint32
	shipNewTime   uint32 // a new ship is protected for a while

	// shots
	shootSpeed   int
	shots        [shotsMax]shot
	lastShotTime uint32

	// gates
	gates    [gatesMax]gate
	gateNext int // distance to the next gate

	// enemies (enemy.go)
	timeLastDisk uint32
	timeLastBig  uint32
	bossIsGen    bool
	bossLives    int
	timeLastBomb uint32
	obj          [objMax]obj

	// ground, stars, score, fuel, goals, lives (hud.go)
	grassPhase   int16
	perspTab     [distMax + 1 - distMin]int16
	stars        [starsMax]star
	lastStarTime uint32
	score        int // with frac
	maxScore     int
	lastScore    int // the score of the last new life
	fuel         int // with frac
	goals        int // goals left in this part
	goalTime     uint32
	lives        int

	// the buttons in the game for the tests (nil: the console's)
	keyNext func() uint8
	keyHeld func(key uint8) bool
}

// New returns the game on d (at least 320x240) with the touch panel t;
// seed starts the random numbers. Set Pad to use a game controller.
func New(d hal.Display, t hal.Touch, seed uint32) *Game {
	g := &Game{Console: picopad.New(d, t, seed)}
	g.SelFont8x8() // config.h: FontBold8x8
	g.calcPerspTab()
	return g
}

// keyGet is KeyGet (in the tests keyNext).
func (g *Game) keyGet() uint8 {
	if g.keyNext != nil {
		return g.keyNext()
	}
	return g.KeyGet()
}

// keyPressed is KeyPressed (in the tests keyHeld).
func (g *Game) keyPressed(key uint8) bool {
	if g.keyHeld != nil {
		return g.keyHeld(key)
	}
	return g.KeyPressed(key)
}

// delay waits ms and moves the times of the game on by as much.
func (g *Game) delay(ms int) {
	g.WaitMs(ms)
	t := uint32(ms * 1000)
	g.lastStarTime += t
	g.timeLastDisk += t
	g.timeLastBig += t
	g.shipNewTime += t
	g.shipCrashTime += t
	g.lastShotTime += t
	g.goalTime += t
	for i := range g.obj {
		o := &g.obj[i]
		o.time += t
		o.startTime += t
		o.nextBomb += t
	}
}

// newGame starts a game.
func (g *Game) newGame() {
	g.shootSpeed = 8
	g.lives = 3
	g.level = 0
	g.subLevel = 0
	g.score = 0
	g.lastScore = 0
	g.speed = shipSpeedDef
	g.lastShotTime = g.Time() - shotTime
}

// enter shows the number of the sector for 3.6 s.
func (g *Game) enter() {
	g.DrawClear()
	g.DrawText("ENTERING SECTOR", (width-15*8)/2, 100, picopad.ColWhite)
	g.DrawText(itoa(g.level+1)+"."+itoa(g.subLevel), (width-3*8)/2, 112, picopad.ColWhite)
	// PLAYSOUND(EnteringSnd)
	g.DispUpdate()
	g.WaitMs(3600)
	g.KeyFlush()
}

// dispAll draws the screen.
func (g *Game) dispAll() {
	if g.subLevel == 3 { // in space
		g.DrawClear()
		g.dispStars()
	} else {
		g.DrawImgPal(skylineImg, skylineImgPal, 0, 0, 0, 0, skylineW, skylineH, skylineW)
		switch { // the city grows
		case g.level == 1:
			g.DrawBlitPal(city1Img, city1ImgPal, 0, 0, (width-city1W)/2, horizon-city1H, city1W, city1H, city1W, picopad.ColMagenta)
		case g.level >= 2:
			g.DrawBlitPal(city2Img, city2ImgPal, 0, 0, (width-city2W)/2, horizon-city2H, city2W, city2H, city2W, picopad.ColMagenta)
		}
		g.dispGround()
	}
	g.dispLives()
	g.dispFuel()
	g.dispGoals()
	g.dispScore()
	g.dispObj()
	g.dispGate()
	g.dispShots()
	g.dispShip()
	g.DispUpdate()
}

// initLevel starts a part of a sector.
func (g *Game) initLevel() {
	if g.subLevel == 0 {
		g.fuel = fuelMax << frac
	}
	g.enter()

	g.shipX = shipDefX
	g.shipInCrash = 0
	g.shipNewTime = g.Time()
	g.initStars()
	g.resetShots()
	g.resetGates()
	g.initObj()
	g.timeLastDisk = g.Time() - 5000000
	g.timeLastBig = g.Time()
	g.timeLastBomb = g.Time()
	g.bossIsGen = false
	g.bossLives = enemyBossLives
	g.goals = goalsInit[g.subLevel]
	g.dispAll()
}

// game plays a game until the last ship or Y.
func (g *Game) game() {
	g.newGame()
	g.initLevel()
	t := g.Time()
	for {
		g.dispAll()
		if g.subLevel == 3 {
			g.shiftStars()
			g.newStars()
		}

		// the time of the frame: 20 to 150 ms (C adds the time it waits
		// to the next frame as well)
		dt := g.Time() - t
		if dt < 20000 {
			g.WaitUs(int(20000 - dt))
			dt = 20000
		}
		t += dt
		if dt > 150000 {
			dt = 150000
		}
		if !g.step(dt) {
			return
		}
	}
}

// step moves the game by dt µs; it returns false at the end of the game.
func (g *Game) step(dt uint32) bool {
	// the distance flown, with frac
	var dz int
	if g.subLevel == 3 {
		dz = int(dt * uint32(shipSpeedDef>>frac) >> (21 - frac))
	} else {
		dz = int(dt * uint32(int(g.speed)>>frac) >> (21 - frac))
	}
	g.addScore(dz)
	g.subFuel(int(dt >> 13))
	g.grassPhase = int16(int(g.grassPhase) - dz)
	for g.grassPhase < 0 {
		g.grassPhase += groundDY << frac
	}

	g.doObj()
	g.shipEnemyCrash()
	g.doShip(dt, uint32(dz))
	g.shooting(int(dt))
	g.gating(dz)

	// the part is done a second after the last goal
	if g.goals <= 0 && int32(g.Time()-g.goalTime) > 1000000 {
		g.subLevel++
		if g.subLevel > 3 {
			g.subLevel = 0
			g.level++
		}
		g.initLevel()
	}

	if g.shipInCrash > shipCrashWait+2 && g.lives == 0 {
		return false
	}
	return g.keyGet() != picopad.KeyY
}

// open is the title; it returns false to quit (Y).
func (g *Game) open() bool {
	newMax := false
	if g.score > g.maxScore {
		g.maxScore = g.score
		newMax = true
	}
	col := picopad.ColWhite
	if newMax {
		col = picopad.ColGreen
	}
	score, high := itoa(g.score>>frac), itoa(g.maxScore>>frac)
	g.initStars()
	last := g.Time()
	for {
		g.DrawClear()
		g.dispStars()
		if g.maxScore > 0 {
			g.DrawText("SCORE:", 3*8, 0, picopad.ColWhite)
			g.DrawText(score, (3+6)*8, 0, col)
			g.DrawText("HIGH:", 26*8, 0, picopad.ColWhite)
			g.DrawText(high, (26+5)*8, 0, col)
		}
		g.DrawBlitPal(titleImg, titleImgPal, 0, 0, (width-titleW)/2, 10, titleW, titleH, titleW, picopad.ColMagenta)
		// C: the copyright of the arcade game
		g.DrawText("FLY THROUGH THE GATES", (width-21*8)/2, height-38, picopad.ColWhite)
		g.DrawText("AND SHOOT THE SAUCERS", (width-21*8)/2, height-30, picopad.ColWhite)
		g.DrawText("Version for Raspberry Pico", (width-26*8)/2, height-16, picopad.ColYellow)
		g.DrawText("(c) 2023 Miroslav Nemecek", (width-25*8)/2, height-8, picopad.ColYellow)
		g.DispUpdate()

		switch g.KeyGet() {
		case picopad.KeyY:
			return false
		case picopad.KeyA:
			return true
		}
		g.shiftStars()
		g.newStars()
		if d := g.Time() - last; d < openFrame*1000 {
			g.WaitUs(int(openFrame*1000 - d))
		}
		last = g.Time()
	}
}

// Run plays until Y on the title (main of the C source, which then resets
// to the boot loader).
func (g *Game) Run() {
	for g.open() {
		g.game()
	}
}

// itoa returns n in decimal (DecNum).
func itoa(n int) string {
	if n < 0 {
		return "-" + itoa(-n)
	}
	var buf [12]byte
	i := len(buf)
	for {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
		if n == 0 {
			return string(buf[i:])
		}
	}
}
