// Package picoinvaders is the Space Invaders of PicoLibSDK
// (PicoPad/GAME/INVADERS, by Miroslav Nemecek, Unlicense), ported from C
// to package picopad (package invaders is another game, example
// 08_invaders). It follows the C source function by function; the C
// globals are the fields of Game. All the graphics and movements are
// multiples of 4 pixels.
//
// LEFT and RIGHT move the ship, A shoots, Y ends Run (on the PicoPad:
// back to the boot loader). After GAME OVER a key starts a new game. There
// is no sound yet: the PLAYSOUND and PlaySoundChan calls of the C source
// are comments.
package picoinvaders

import (
	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/picopad"
)

//go:generate go run ../../tools/picoimg -pkg picoinvaders -root $PICOLIBSDK -o sprites.go PicoPad/GAME/INVADERS/img/sprites.cpp

const (
	width       = picopad.Width
	height      = picopad.Height
	spritesImgW = 36 // width of the image with the sprites

	// aliens
	alienW      = 12                              // alien width
	alienH      = 8                               // alien height
	alienNumX   = 11                              // number of aliens in X direction
	alienNumY   = 5                               // number of aliens in Y direction
	alienNum    = alienNumX * alienNumY           // number of aliens (55)
	alienSpaceX = 4                               // space between aliens in X direction
	alienDX     = alienW + alienSpaceX            // alien delta X (16)
	alienTotW   = alienNumX*alienDX - alienSpaceX // alien block width (172)
	alienTotX   = ((width - alienTotW) / 2) &^ 3  // alien block initial X (72)
	alienSpaceY = 8                               // space between aliens in Y direction
	alienDY     = alienH + alienSpaceY            // alien delta Y (16)
	alienTotH   = alienNumY*alienDY - alienSpaceY // alien block height (72)
	alienTotY   = 40                              // alien block initial Y
	alienMaxY   = 112                             // alien block max. initial Y
	alienMinX   = 0                               // alien min. X coordinate
	alienMaxX   = width                           // alien max. X coordinate

	// enemy missiles
	missileNum     = 10 // max. number of missiles
	missileW       = 4
	missileH       = 8
	missile1ImgX   = 24
	missile1ImgY   = 32
	missile2ImgX   = 24
	missile2ImgY   = 40
	missileExpW    = 8
	missileExpH    = 8
	missileInvImgX = 28 // inverted explosion
	missileInvImgY = 40

	// spaceship
	shipW       = 12
	shipH       = 8
	shipImgX    = 0
	shipImgY    = 24
	shipExpImgX = 24
	shipExpImgY = 24
	shipY       = height - 16 - 8 - shipH    // ship Y coordinate (208)
	shipX       = ((width - shipW) / 2) &^ 3 // ship initial X coordinate (152)
	shipDX      = 16                         // delta X of the ship lives

	// spaceship shoots
	shootW     = 4
	shootH     = 8
	shootImgX  = 12
	shootImgY  = 24
	shootExpW  = 4
	shootExpH  = 8
	shootExpX  = 16
	shootExpY  = 24
	shootExp2X = 20 // inverted explosion
	shootExp2Y = 24

	// score
	lineY1    = 20
	lineY2    = height - 20
	scoreY    = 0
	scoreX    = 0
	maxScoreX = width - 15*8
	levelY    = height - 16
	levelX    = width - 9*8

	// houses
	houseW      = 24
	houseH      = 16
	houseImgX   = 0
	houseImgY   = 32
	houseY      = shipY - 8 - houseH // (184)
	houseNum    = 4
	houseDX     = 3 * alienDX                    // (48)
	houseSpaceX = houseDX - houseW               // (24)
	houseTotW   = houseNum*houseDX - houseSpaceX // (168)
	houseX      = ((width - houseTotW) / 2) &^ 3 // (76)

	// mothership
	motherY       = 24
	motherW       = 16
	motherH       = 8
	motherImgX    = 0
	motherImgY    = 48
	motherExpImgX = 16
	motherExpImgY = 48
	nextMotherN   = 500 // steps to a new mothership
)

var (
	alienType  = [alienNumY]uint8{0, 1, 1, 2, 2}    // types of aliens
	alienScore = [alienNumY]int{30, 20, 20, 10, 10} // score of aliens
)

// Game is Space Invaders on a console.
type Game struct {
	*picopad.Console

	oldTime uint32

	// spaceship
	lives    int // number of remaining lives
	shipX    int // spaceship X coordinate
	score    int
	maxScore int
	level    int
	colHouse picopad.Color // house color
	nextLive int           // score to the next live

	// aliens
	alienLive  [alienNum]bool
	alienNum   int // number of aliens
	alienX     int // X coordinate of the alien block
	alienY     int // Y coordinate of the alien block
	alienPhase int // animation phase (0 or 1)
	alienMinX  int
	alienMaxX  int
	alienLeft  bool // the aliens are moving left
	alienTimer int  // timer to the next step
	alienTone  int  // current alien tone

	// alien missiles
	missileOn    [missileNum]bool
	missileX     [missileNum]int
	missileY     [missileNum]int
	enemyLast    [alienNumX]bool // last enemy
	missilePhase int

	// spaceship shoot
	shootX, shootY int
	shootOn        bool

	// mothership
	motherOn   bool
	motherLeft bool
	motherX    int
	nextMother int // counter to the next mothership
}

// New returns the game on d (at least 320x240) with the touch panel t;
// seed starts the random numbers. Set Pad to use a game controller.
func New(d hal.Display, t hal.Touch, seed uint32) *Game {
	return &Game{Console: picopad.New(d, t, seed)}
}

// dispSprite displays a sprite.
func (g *Game) dispSprite(xs, ys, x, y, w, h int) {
	g.DrawImg4Pal(spritesImg, spritesImgPal, xs, ys, x, y, w, h, spritesImgW)
}

func (g *Game) dispShip() { g.dispSprite(shipImgX, shipImgY, g.shipX, shipY, shipW, shipH) }

func (g *Game) clearShip() { g.DrawRect(g.shipX, shipY, shipW, shipH, picopad.ColBlack) }

// dispLives displays the remaining lives.
func (g *Game) dispLives() {
	g.DrawRect(0, height-shipH-4, 10*shipDX, shipH, picopad.ColBlack)
	i := g.lives
	if i > 10 {
		i = 10
	}
	for i--; i >= 0; i-- {
		g.dispSprite(shipImgX, shipImgY, i*shipDX, height-shipH-4, shipW, shipH)
	}
}

// lost loses a live.
func (g *Game) lost() {
	g.dispSprite(shipExpImgX, shipExpImgY, g.shipX, shipY, shipW, shipH)
	// PLAYSOUND(CrashSnd)
	g.DispUpdate()
	g.WaitMs(1000)
	g.DrawRect(g.shipX, shipY, shipW, shipH, picopad.ColBlack)
	g.lives--
	g.shipX = shipX
	g.dispShip()
	g.dispLives()
	g.DispUpdate()
}

// moveMother moves the mothership.
func (g *Game) moveMother() {
	// a new mothership
	g.nextMother--
	if g.nextMother <= 0 && !g.motherOn {
		g.nextMother = nextMotherN
		g.motherOn = true
		g.motherLeft = !g.motherLeft
		if g.motherLeft {
			g.motherX = width
		} else {
			g.motherX = -motherW
		}
		// PlaySoundChan(MOTHERCHAN, MotherSnd, ...)
	}

	if g.motherOn {
		g.DrawRect(g.motherX, motherY, motherW, motherH, picopad.ColBlack)
		if g.motherLeft {
			g.motherX -= 2
			if g.motherX < -motherW {
				g.motherOn = false // StopSoundChan(MOTHERCHAN)
				return
			}
		} else {
			g.motherX += 2
			if g.motherX >= width {
				g.motherOn = false // StopSoundChan(MOTHERCHAN)
				return
			}
		}
		g.dispSprite(motherImgX, motherImgY, g.motherX, motherY, motherW, motherH)
		g.DispUpdate()
	}
}

// dispLine displays the lines.
func (g *Game) dispLine() {
	g.DrawRect(0, lineY1, width, 1, picopad.ColWhite)
	g.DrawRect(0, lineY2, width, 1, picopad.ColWhite)
}

// limitAlien prepares the limits of the alien block.
func (g *Game) limitAlien() {
	xmin, xmax := width, 0
	for i := 0; i < alienNum; i++ {
		if g.alienLive[i] {
			x := (i % alienNumX) * alienDX
			if x < xmin {
				xmin = x
			}
			if x > xmax {
				xmax = x
			}
		}
	}
	g.alienMinX = alienMinX - xmin
	g.alienMaxX = alienMaxX - alienDX - xmax + alienSpaceX
}

// clearAlien clears one alien.
func (g *Game) clearAlien(inx int) {
	y := inx / alienNumX
	x := inx - y*alienNumX
	g.DrawRect(g.alienX+x*alienDX, g.alienY+y*alienDY, alienW, alienH, picopad.ColBlack)
}

// clearAliens clears the aliens.
func (g *Game) clearAliens() {
	for i := 0; i < alienNum; i++ {
		if g.alienLive[i] {
			g.clearAlien(i)
		}
	}
}

// dispAlien displays one alien (its explosion if it is dead).
func (g *Game) dispAlien(inx int) {
	y := inx / alienNumX
	x := inx - y*alienNumX
	ys := int(alienType[y]) * alienH
	y = g.alienY + y*alienDY
	x = g.alienX + x*alienDX

	if g.alienLive[inx] {
		g.dispSprite(g.alienPhase*alienW, ys, x, y, alienW, alienH)
	} else {
		g.dispSprite(2*alienW, ys, x, y, alienW, alienH)
	}

	// the aliens landed: a live lost, they go back up
	if y >= shipY {
		g.lost()
		g.clearAliens()
		g.alienY -= 4 * 10
	}
}

// dispAliens displays the aliens.
func (g *Game) dispAliens() {
	for i := 0; i < alienNum; i++ {
		if g.alienLive[i] {
			g.dispAlien(i)
		}
	}
}

// checkHouse returns the number of house pixels in the area 4x8 at x, y.
func (g *Game) checkHouse(x, y int) int {
	if y+8 <= houseY || y >= houseY+houseH {
		return 0
	}
	if x+4 <= houseX || x >= houseX+houseTotW {
		return 0
	}
	if (x-houseX)%houseDX >= houseW {
		return 0
	}

	// count the pixels of the house (C: FrameBuf)
	num := 0
	for i := 0; i < 8; i++ {
		for j := 0; j < 4; j++ {
			if g.DrawGetPoint(x+j, y+i) == g.colHouse {
				num++
			}
		}
	}
	return num
}

// shootHitHouse: the shoot hits a house.
func (g *Game) shootHitHouse() {
	if g.shootOn && g.checkHouse(g.shootX, g.shootY) > 15 {
		g.shootOn = false
		g.DrawBlit4Pal(spritesImg, spritesImgPal, shootExp2X, shootExp2Y,
			g.shootX, g.shootY, shootW, shootH, spritesImgW, picopad.ColWhite)
	}
}

// missileHitHouse: the missiles hit a house.
func (g *Game) missileHitHouse() {
	for i := 0; i < missileNum; i++ {
		if g.missileOn[i] && g.checkHouse(g.missileX[i], g.missileY[i]) > 10 {
			g.missileOn[i] = false
			g.DrawBlit4Pal(spritesImg, spritesImgPal, missileInvImgX, missileInvImgY,
				g.missileX[i]-2, g.missileY[i], missileExpW, missileExpH, spritesImgW, picopad.ColYellow)
		}
	}
}

// drawNum draws n with digits digits, leading zeros (by characters: a
// string would allocate).
func (g *Game) drawNum(n, digits, x, y int, col picopad.Color) {
	for i := digits - 1; i >= 0; i-- {
		g.DrawCharBg(byte('0'+n%10), x+i*8, y, col, picopad.ColBlack)
		n /= 10
	}
}

// dispScore displays the score; a new live at each next live score.
func (g *Game) dispScore() {
	g.drawNum(g.score, 5, scoreX+7*8, scoreY, picopad.ColWhite)
	if g.score >= g.nextLive {
		g.lives++
		g.dispLives()
		g.nextLive *= 2
	}
}

// dispMaxScore displays the max. score (green when it is this game's).
func (g *Game) dispMaxScore() {
	col := picopad.ColWhite
	if g.maxScore > 0 && g.maxScore == g.score {
		col = picopad.ColGreen
	}
	g.drawNum(g.maxScore, 5, maxScoreX+10*8, scoreY, col)
}

// dispLevel displays the level.
func (g *Game) dispLevel() {
	n := g.level + 1
	if n > 99 {
		n = 99
	}
	g.drawNum(n, 2, levelX+7*8, levelY, picopad.ColWhite)
}

// genMissile generates missiles: the lowest alien of a column shoots.
func (g *Game) genMissile() {
	for x := range g.enemyLast {
		g.enemyLast[x] = true
	}
	i := alienNum - 1
	for y := alienNumY - 1; y >= 0; y-- {
		for x := alienNumX - 1; x >= 0; x-- {
			if g.alienLive[i] && g.enemyLast[x] {
				g.enemyLast[x] = false
				if g.RandU16() < 300 && g.alienY+y*alienDY+8 < shipY {
					for j := 0; j < missileNum; j++ {
						if !g.missileOn[j] {
							g.missileOn[j] = true
							g.missileX[j] = g.alienX + x*alienDX + 4
							g.missileY[j] = g.alienY + y*alienDY + 8
							break
						}
					}
				}
			}
			i--
		}
	}
}

// moveMissile moves the missiles.
func (g *Game) moveMissile() {
	for i := 0; i < missileNum; i++ {
		if g.missileOn[i] {
			g.missileY[i] += 4
			if g.missileY[i] > shipY {
				g.missileOn[i] = false
			}
		}
	}
}

// clearMissile clears the missiles.
func (g *Game) clearMissile() {
	for i := 0; i < missileNum; i++ {
		if g.missileOn[i] {
			g.DrawRect(g.missileX[i], g.missileY[i], missileW, missileH, picopad.ColBlack)
		}
	}
}

// dispMissile displays the missiles.
func (g *Game) dispMissile() {
	for i := 0; i < missileNum; i++ {
		if g.missileOn[i] {
			if g.missilePhase != 0 {
				g.dispSprite(missile2ImgX, missile2ImgY, g.missileX[i], g.missileY[i], missileW, missileH)
			} else {
				g.dispSprite(missile1ImgX, missile1ImgY, g.missileX[i], g.missileY[i], missileW, missileH)
			}
		}
	}
	g.missilePhase = (g.missilePhase + 1) & 1
}

// newGame starts a level (0: a new game).
func (g *Game) newGame(level int) {
	g.DrawClear()

	// houses
	for i := 0; i < houseNum; i++ {
		g.dispSprite(houseImgX, houseImgY, houseX+i*houseDX, houseY, houseW, houseH)
	}
	g.colHouse = g.DrawGetPoint(houseX+houseW/2, houseY)

	for i := range g.missileOn {
		g.missileOn[i] = false
	}

	g.shipX = shipX
	g.shootOn = false
	g.dispShip()
	g.dispLine()

	if level == 0 {
		g.lives = 3
		g.nextLive = 1500
		g.score = 0
	}
	g.dispLives()

	// aliens
	g.alienX = alienTotX
	g.alienY = alienTotY + level*8
	if g.alienY > alienMaxY {
		g.alienY = alienMaxY
	}
	for i := range g.alienLive {
		g.alienLive[i] = true
	}
	g.alienNum = alienNum
	g.alienPhase = 0
	g.alienLeft = true
	g.limitAlien()
	g.dispAliens()
	g.alienTimer = 1
	g.alienTone = 0
	g.missilePhase = 0

	g.nextMother = nextMotherN
	g.motherOn = false
	g.motherLeft = true
	g.level = level
	g.SelFont8x16()
	g.DrawText("SCORE:", scoreX, scoreY, picopad.ColWhite)
	g.dispScore()
	g.DrawText("HI-SCORE:", maxScoreX, scoreY, picopad.ColWhite)
	g.dispMaxScore()
	g.DrawText("LEVEL:", levelX, levelY, picopad.ColWhite)
	g.dispLevel()
	g.DispUpdate()

	g.oldTime = g.Time()
}

func (g *Game) dispShoot() { g.dispSprite(shootImgX, shootImgY, g.shootX, g.shootY, shootW, shootH) }

func (g *Game) clearShoot() { g.DrawRect(g.shootX, g.shootY, shootW, shootH, picopad.ColBlack) }

// newMax displays the score, and the max. score if it is beaten.
func (g *Game) newMax() {
	g.dispScore()
	if g.score > g.maxScore {
		g.maxScore = g.score
		g.dispMaxScore()
	}
}

// shootCollision: the shoot hits an alien, a missile or the mothership.
func (g *Game) shootCollision() {
	if !g.shootOn {
		return
	}

	// an alien
	if g.shootX >= g.alienX && g.shootY >= g.alienY &&
		g.shootX < g.alienX+alienTotW && g.shootY < g.alienY+alienTotH {
		x := (g.shootX - g.alienX) / alienDX
		y := (g.shootY - g.alienY) / alienDY
		k := y
		i := x + y*alienNumX
		x = g.alienX + x*alienDX - g.shootX
		y = g.alienY + y*alienDY - g.shootY
		if g.alienLive[i] && x < alienW && y < alienH {
			g.alienLive[i] = false
			g.alienNum--
			g.shootOn = false
			g.clearShoot()
			// PLAYSOUND(HitSnd)
			g.dispAlien(i)
			g.score += alienScore[k]
			g.newMax()
			g.DispUpdate()
			g.WaitMs(100)
			g.clearAlien(i)
			g.DispUpdate()
			g.limitAlien()
			return
		}
	}

	// a missile
	for i := 0; i < missileNum; i++ {
		if g.missileOn[i] && g.missileX[i] < g.shootX+shootW && g.missileX[i]+missileW >= g.shootX &&
			g.missileY[i] < g.shootY+2*shootH && g.missileY[i]+missileH >= g.shootY {
			g.missileOn[i] = false
			g.clearShoot()
			g.DrawRect(g.missileX[i], g.missileY[i], missileW, missileH, picopad.ColBlack)
			g.shootOn = false
			g.dispSprite(shootExpX, shootExpY, g.shootX, g.shootY, shootExpW, shootExpH)
			g.DispUpdate()
			g.WaitMs(150)
			g.clearShoot()
			g.DispUpdate()
		}
	}
	if !g.shootOn {
		return
	}

	// the mothership
	if g.motherOn && g.shootY < motherY+motherH && g.shootX+4 >= g.motherX && g.shootX < g.motherX+motherW {
		g.motherOn = false
		g.shootOn = false
		g.clearShoot()
		g.dispSprite(motherExpImgX, motherExpImgY, g.motherX, motherY, motherW, motherH)
		// StopSoundChan(MOTHERCHAN); PLAYSOUND(HitSnd)
		g.score += 1000
		g.newMax()
		g.DispUpdate()
		g.WaitMs(100)
		g.DrawRect(g.motherX, motherY, motherW, motherH, picopad.ColBlack)
		g.DispUpdate()
	}
}

// moveShoot moves the shoot.
func (g *Game) moveShoot() {
	if g.shootOn {
		g.clearShoot()
		g.shootY -= 8
		if g.shootY < motherY {
			g.shootOn = false
		} else {
			g.shootHitHouse()
			if g.shootOn {
				g.dispShoot()
				g.shootCollision()
			}
		}
		g.DispUpdate()
	}
}

// spaceCtrl moves the ship and shoots.
func (g *Game) spaceCtrl() {
	if g.KeyPressed(picopad.KeyLeft) && g.shipX >= 4 {
		g.clearShip()
		g.shipX -= 4
		g.dispShip()
		g.DispUpdate()
	}
	if g.KeyPressed(picopad.KeyRight) && g.shipX < width-alienW {
		g.clearShip()
		g.shipX += 4
		g.dispShip()
		g.DispUpdate()
	}
	if g.KeyPressed(picopad.KeyA) && !g.shootOn {
		g.shootOn = true
		g.shootX = g.shipX + 4
		g.shootY = shipY - 8
		// PLAYSOUND(ShootSnd)
		g.dispShoot()
		g.DispUpdate()
	}
}

// alienMove moves the aliens.
func (g *Game) alienMove() {
	g.alienTimer--
	if g.alienTimer <= 0 {
		g.clearAliens()
		if g.alienLeft {
			g.alienX -= 8
			if g.alienX < g.alienMinX {
				g.alienX += 16
				g.alienLeft = false
				g.alienY += 8
			}
		} else {
			g.alienX += 8
			if g.alienX > g.alienMaxX {
				g.alienX -= 16
				g.alienLeft = true
				g.alienY += 8
			}
		}
		g.dispAliens()
		g.DispUpdate()
		g.alienTimer = g.alienNum/4 + 2
		g.alienPhase = (g.alienPhase + 1) & 1
		g.alienTone = (g.alienTone + 1) & 3 // PlaySoundChan(1, Tones[AlienTone], ...)
	}
}

// crash: a missile hits the ship.
func (g *Game) crash() {
	for i := 0; i < missileNum; i++ {
		if g.missileOn[i] && g.missileX[i] < g.shipX+shipW && g.missileX[i]+missileW >= g.shipX &&
			g.missileY[i] < shipY+shipH && g.missileY[i]+missileH >= shipY {
			g.missileOn[i] = false
			g.DrawRect(g.missileX[i], g.missileY[i], missileW, missileH, picopad.ColBlack)
			g.lost()
		}
	}
}

// game is the game loop; it returns false on Y, true after GAME OVER and
// a key.
func (g *Game) game() bool {
	for {
		g.moveShoot()
		g.spaceCtrl()
		g.alienMove()

		// missiles
		g.clearMissile()
		g.moveMissile()
		g.genMissile()
		g.missileHitHouse()
		g.dispMissile()
		g.DispUpdate()

		g.moveMother()
		g.shootCollision()
		g.crash()

		// time step
		var t uint32
		for {
			t = g.Time()
			if t-g.oldTime >= 50000 {
				break
			}
		}
		g.oldTime = t

		if g.KeyGet() == picopad.KeyY {
			return false
		}

		// new level
		if g.alienNum == 0 {
			g.DrawTextBg(" LEVEL UP ", (width-10*8)/2, (height-16)/2, picopad.ColGreen, picopad.ColBlack)
			g.DispUpdate()
			g.WaitMs(1000)
			g.KeyFlush()
			g.newGame(g.level + 1)
		}

		// end of the game
		if g.lives < 0 {
			g.DrawTextBg(" GAME OVER ", (width-11*8)/2, (height-16)/2, picopad.ColRed, picopad.ColWhite)
			g.DispUpdate()
			g.WaitMs(500)
			// StopSound()
			g.KeyFlush()
			for g.KeyGet() == picopad.NoKey {
			}
			return true
		}
	}
}

// Run plays until Y (main of the C source, which then resets to the boot
// loader). With OneGame it returns after GAME OVER (and a key) too.
func (g *Game) Run() {
	g.maxScore = 0
	for {
		g.newGame(0)
		g.WaitMs(2000)
		if !g.game() || g.OneGame {
			return
		}
	}
}
