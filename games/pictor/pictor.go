// Package pictor is the Pictor (Picopad Collector) of PicoLibSDK
// (PicoPad/GAME/PICTOR, by Miroslav Nemecek, Unlicense), a scrolling
// shooting game, ported from C to package picopad. It follows the C source
// file by file (src/*.cpp); the C globals are the fields of Game.
//
// Bara runs through 12 scenes and collects the parts of a PicoPad; in each
// scene a new helper joins, which she can transform into (B/X). The arrows
// move (UP jumps for the walking ones), A shoots, Y opens the game menu
// (Y continues, X ends the game, B changes the sound mode); X in the menu
// of the title ends Run (on the PicoPad: back to the boot loader).
//
// Differences from the C source:
//   - Five pictures are new pixel art (artgen, art.go): the C source has
//     characters of films there (R2-D2, the starship Enterprise, a Klingon
//     warship, the Death Star, Ant-Man), which cannot be distributed. The
//     helper of the spacecraft is a robot, the enemies of the galaxy are a
//     star cruiser, a warship and a space mine, the third enemy of the beach
//     a fire ant. The logo of a shop on the title is cleared.
//   - The scenes draw many more colours than the 256 of the frame buffer
//     of package picopad: each scene has a palette made from its pictures
//     (palettes.go, made by the test with -update) and sets it.
//   - A frame takes at least 86 ms (the C game draws as fast as the PicoPad
//     can, about 700 frames a minute: GAMELEN).
//   - The top list is kept in RAM (the C source writes PICTOR.TOP to the
//     SD card); the screenshot of the menu does nothing. There is no sound.
package pictor

import (
	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/picopad"
)

//go:generate sh -c "go run ../../tools/picoimg -pkg pictor -root $PICOLIBSDK -embed data -o img.go $(grep -o 'PicoPad/GAME/PICTOR/img/[0-9a-z_]*.cpp' imglist.txt)"
//go:generate go run ./artgen -root $PICOLIBSDK

const (
	width  = picopad.Width
	height = picopad.Height

	// game difficulty
	enemyGenLevel = 1.06  // more enemies a level (factor)
	gameLen       = 700   // frames of a scene (about 1 minute)
	enemyGenBeg   = 8000  // probability of an enemy at the beginning of a scene (of 65536)
	enemyGenEnd   = 14000 // probability of an enemy at the end of a scene
	enemyGenBul   = 1200  // probability of an enemy bullet

	bgNum       = 12        // scenes (backgrounds)
	actNum      = bgNum + 1 // actors, with Bara (13)
	enemyLevNum = 3         // enemies of a scene
	enemyNum    = bgNum * enemyLevNum
	enemyMax    = 20 // enemies on the screen
	frameTime   = 86 // the shortest frame [ms]

	// header (Y 0..19)
	titY        = 0
	titHeight   = 20
	headerCol   = picopad.Color(0x000F) // COL_DKBLUE
	headerFrame = picopad.Color(0x7BFF) // COL_LTBLUE

	// backgrounds
	bgSpeed1  = 1 // layer speeds
	bgSpeed2  = 2
	bgSpeed3  = 3
	bgYMin    = titHeight
	bgY       = bgYMin // layer 1 (Y 20..179)
	bgWidth   = 640
	bgHeight  = 160
	bg2Y      = bgY + bgHeight - 70 // layer 2 (110)
	bg2Width  = 640
	bg2Height = 80
	bg3Y      = bg2Y + bg2Height - 10 // layer 3 (180)
	bg3Width  = 640
	bg3Height = 40

	// foot (Y 220..239)
	footY      = bg3Y + bg3Height
	footHeight = 20
	bgYMax     = footY
	lifeInit   = 100
	lifeMax    = 500

	colAzure   = picopad.Color(0x03FF) // COLOR(0,127,255)
	colDkGreen = picopad.Color(0x03E0) // COLOR(0,127,0)
)

// Game is Pictor on a console.
type Game struct {
	*picopad.Console

	level        int // current scene - 1
	currentFrame int
	maxFrame     int
	lastTime     uint32 // the last frame

	score    int
	maxScore int
	life     int

	// background
	backInx                            int
	back                               backTemp
	backPhase1, backPhase2, backPhase3 int

	// actor
	actInx               int
	actor                actorTemp
	actorPhase           int
	actorX, actorY       int // middle
	actorMinX, actorMaxX int
	actorMinY, actorMaxY int
	actorJumpSpeed       int
	actorReload          int
	bloodTime            int
	bloodX, bloodY       int
	enemyNow             [enemyLevNum]*enemyTemp
	enemy                [enemyMax]enemy
	explosion            [explosionsMax]explosion
	bullet               [bulletMax]bullet
	missile              [missileMax]missile
	point                [pointNum]point
	bonusMask            int
	bonusInx             int
	bonus                *bonusTemp
	bonusX, bonusY       int
	bonusPhase           int
	enemyRandSeed        uint32
	soundMode            int
	top                  [topNum]topEntry
	playerName           [topNameLen]byte
	playerNameInx        int

	// the buttons in the game for the tests (nil: the console's)
	keyNext func() uint8
	keyHeld func(key uint8) bool
}

// New returns the game on d (at least 320x240) with the touch panel t;
// seed starts the random numbers. Set Pad to use a game controller.
func New(d hal.Display, t hal.Touch, seed uint32) *Game {
	g := &Game{Console: picopad.New(d, t, seed), soundMode: soundModeMusic}
	g.loadTop()
	for i := range g.playerName {
		g.playerName[i] = ' '
	}
	return g
}

// scenePalette sets the palette of the current scene (before the whole
// screen is drawn again).
func (g *Game) scenePalette() { g.SetPalette(scenePal[g.backInx]) }

// waitFrame waits for the end of the frame.
func (g *Game) waitFrame() {
	for g.Time()-g.lastTime < frameTime*1000 {
	}
	g.lastTime = g.Time()
}

// newGame starts scene level (0..); it returns false to break.
func (g *Game) newGame(level int) bool {
	g.initScene(level)
	if !g.gameStart() {
		return false
	}
	// MusicStart()
	return true
}

// initScene prepares scene level (NewGame without GameStart).
func (g *Game) initScene(level int) {
	g.level = level
	g.currentFrame = 0
	g.maxFrame = gameLen
	g.bloodTime = 0
	g.enemyRandSeed = uint32(g.level + 123456) // InitBonus uses it
	g.initBack()
	g.initActor()
	g.initMissile()
	g.initEnemy()
	g.initBullet()
	g.initBonus()
	g.initPoint()
}

// game plays until the last life or the end in the menu.
func (g *Game) game() {
	g.score = 0
	g.bonusMask = 0
	g.life = lifeInit
	g.actInx = 0
	if !g.newGame(0) {
		return
	}
	g.dispHeader()
	g.dispFoot()
	g.lastTime = g.Time()

	for g.step() {
		if g.currentFrame >= g.maxFrame {
			if !g.newGame(g.level + 1) {
				return
			}
			g.lastTime = g.Time()
		}
		g.waitFrame()

		if g.life <= 0 {
			return
		}
	}
}

// step plays a frame; it returns false to end the game (in the menu).
func (g *Game) step() bool {
	g.newEnemy()
	if !g.ctrlActor() {
		return false
	}

	// display
	g.dispBack()
	g.dispActor()
	g.enemyDisp()
	g.explosionDisp()
	g.bulletDisp()
	g.missileDisp()
	g.dispBlood()
	g.bonusDisp()
	g.pointDisp()
	g.dispHeader()
	g.dispFoot()
	g.DispUpdate()

	// move
	g.enemyShift()
	g.explosionShift()
	g.bulletShift()
	g.missileShift()
	g.enemyHit()
	g.shiftBack()
	g.shiftActor()
	g.shiftBlood()
	g.bonusShift()
	g.pointShift()
	g.hitActor()
	g.bonusCollect()

	g.currentFrame++
	return true
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

// titleScreen shows the title; it returns false to quit.
func (g *Game) titleScreen() bool {
	g.KeyFlush()
	for {
		g.SetPalette(titleImgPal)
		g.DrawImgPal(titleImg, titleImgPal, 0, 0, 0, 0, width, height, width)
		g.DrawText("max "+zeros(g.maxScore, 6), 4, 240-13, colDkGreen)
		g.DispUpdate()

		key := g.KeyGet()
		for key == picopad.NoKey {
			key = g.KeyGet()
		}
		switch key {
		case picopad.KeyA:
			return true
		case picopad.KeyY:
			if !g.gameMenu() {
				return false
			}
		}
	}
}

// Run plays until the end in the menu of the title (main of the C source,
// which then resets to the boot loader).
func (g *Game) Run() {
	g.maxScore = g.top[0].score
	for {
		if !g.titleScreen() {
			return
		}
		g.game()
		// StopAllSound()
		if g.life <= 0 || g.score > g.maxScore {
			g.gameEnd()
		}
	}
}

// zeros returns n with at least digits digits ("%07d").
func zeros(n, digits int) string {
	var buf [12]byte
	i := len(buf)
	for n > 0 || digits > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
		digits--
	}
	return string(buf[i:])
}

// itoa returns n in decimal (DecNum).
func itoa(n int) string {
	if n < 0 {
		return "-" + zeros(-n, 1)
	}
	return zeros(n, 1)
}
