// Package raptor is the Raptor of PicoLibSDK (PicoPad/GAME/RAPTOR, by
// Miroslav Nemecek, Unlicense), a shooting game over a scrolling
// landscape, ported from C to package picopad. It follows the C source
// function by function; the C globals are the fields of Game.
//
// The arrows move the ship, A shoots (the machine gun and the selected
// weapon), B selects the weapon, Y ends Run (on the PicoPad: back to the
// boot loader). A new level comes with a new weapon (green and silver
// rockets, laser, artillery, sonar bomb, black hole); a shield is lost
// when the energy runs out. Without shields the ship burns, and the game
// goes on until Y as in the C source. There is no sound yet: the
// PlaySoundChan calls of the C source are comments.
package raptor

import (
	"math"

	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/picopad"
)

//go:generate go run ../../tools/picoimg -pkg raptor -root $PICOLIBSDK -o img.go PicoPad/GAME/RAPTOR/img/landscape.cpp PicoPad/GAME/RAPTOR/img/shadows.cpp PicoPad/GAME/RAPTOR/img/sprites.cpp PicoPad/GAME/RAPTOR/img/tiles.cpp

const (
	width  = picopad.Width
	height = picopad.Height

	shieldMax = 7
	energyMax = 30
	shipSpeed = 10
	weaponNum = 7 // number of weapons
	enemyNum  = 5 // number of enemies

	weaponArtillery = 4  // the artillery
	artilleryNum    = 15 // missiles of the artillery
	weaponSonarBomb = 5  // the sonar bomb
	weaponBlackHole = 6  // the black hole

	landscapeW   = 320
	landscapeH   = 320
	shadowW      = 32
	shadowH      = 32
	shadowShiftX = 30
	shadowShiftY = 30
	shadowShip   = 0 // the shadows: the spaceship, then the enemies
	shadowEnemy1 = 1
	spriteW      = 32
	spriteH      = 32
	tileW        = 16
	tileH        = 16

	// sprites
	spriteShip      = 0 // standing (2 phases)
	spriteShipD     = 0 // moving down
	spriteShipU     = 1 // moving up
	spriteShipR     = 2 // moving right
	spriteShipL     = 3 // moving left
	spriteWeapon0   = 4 // the tile of weapon 0, its missile after
	spriteWeapon1   = 6
	spriteWeapon2   = 8
	spriteWeapon3   = 10
	spriteWeapon4   = 12
	spriteWeapon5   = 14
	spriteWeapon5N  = 8 // animation phases of weapon 5
	spriteWeapon6   = 23
	spriteWeapon6N  = 6 // animation phases of weapon 6
	spriteExplose1  = 30
	spriteExplose5  = 34
	spriteEnemy1    = 38 // the enemies, each followed by its missile
	spriteEnemy1Mis = 39
	spriteHit0      = 50 // the hit missile (4 phases)

	// tiles
	tileEnergy2 = 0 // energy, 2 points
	tileEnergy1 = 1 // energy, 1 point
	tileShield  = 2
	tileDigit0  = 3

	// moving
	moveNo = 0
	moveL  = 1
	moveR  = 2
	moveU  = 3
	moveD  = 4

	myMissileMax = 50
	enMissileMax = 50
	enemyMax     = 50
)

var (
	reloadList         = [weaponNum]int{2, 6, 5, 3, 12, 12, 50}             // weapon reload time
	speedList          = [weaponNum]float32{30, 15, 15, 30, 15, 5, 1.0}     // weapon speed
	weaponEnergy       = [weaponNum]int{26, 60, 120, 330, 80, 10000, 10000} // weapon energy
	enemySpeed         = [enemyNum]float32{2, 3, 4, 3, 2}                   // enemy speed
	enemyEnergyList    = [enemyNum]int{125, 250, 500, 1000, 2000}           // enemy energy
	enemyPoints        = [enemyNum]int{20, 50, 100, 200, 500}               // enemy points
	enemyMissileSpeed  = [enemyNum]float32{4, 6, 8, 6, 5}                   // enemy missile speed
	enemyMissileEnergy = [enemyNum]int{3, 5, 10, 20, 10}                    // enemy missile energy
	weaponList         = [weaponNum + 1]uint8{spriteWeapon0, spriteWeapon1, spriteWeapon2, spriteWeapon3, spriteWeapon4, spriteWeapon5, spriteWeapon6, spriteHit0 - 1}
)

// missile is a flying missile (sMissile).
type missile struct {
	typ    int // -1: not used, weaponNum: hit
	x, y   float32
	dx, dy float32
	anim   int
}

// enemy is an enemy (sEnemy).
type enemy struct {
	typ    int // -1: not used, enemyNum: dead
	x, y   float32
	dx, dy float32
	anim   int
	energy int
}

// Game is Raptor on a console.
type Game struct {
	*picopad.Console

	landscapeY  int
	lastTime    uint32
	level       int // current level (1..)
	weapon      int // current weapon (0: machine gun)
	weaponMax   int // max. weapon
	shields     int // -1: the spaceship is dead
	energy      int
	score       int
	shipX       int
	shipY       int
	move        uint8 // moving state
	gameFrame   int   // frame counter (animations)
	nextShield  int   // score to the next shield
	reloadNext0 int   // reload time of the machine gun
	reloadNext  int   // reload time of the current weapon

	myMissile [myMissileMax]missile
	enMissile [enMissileMax]missile
	enemy     [enemyMax]enemy

	keys func(key uint8) bool // the held buttons for the tests (nil: KeyPressed)
}

// pressed returns whether key is held.
func (g *Game) pressed(key uint8) bool {
	if g.keys != nil {
		return g.keys(key)
	}
	return g.KeyPressed(key)
}

// New returns the game on d (at least 320x240) with the touch panel t;
// seed starts the random numbers. Set Pad to use a game controller.
func New(d hal.Display, t hal.Touch, seed uint32) *Game {
	return &Game{Console: picopad.New(d, t, seed)}
}

// newGame starts a new game.
func (g *Game) newGame() {
	g.lastTime = g.Time()
	g.level = 1
	g.weapon = 0
	g.weaponMax = 0
	g.shields = 5
	g.energy = energyMax
	g.score = 0
	g.shipX = (width - spriteW) / 2
	g.shipY = height - spriteH
	g.move = moveNo
	g.gameFrame = 0
	g.nextShield = 10000
	g.reloadNext0 = 0
	g.reloadNext = 0
	for i := range g.myMissile {
		g.myMissile[i].typ = -1
	}
	for i := range g.enMissile {
		g.enMissile[i].typ = -1
	}
	for i := range g.enemy {
		g.enemy[i].typ = -1
	}
}

// doLandscape scrolls the landscape.
func (g *Game) doLandscape() {
	g.landscapeY++
	if g.landscapeY >= landscapeH {
		g.landscapeY -= landscapeH
	}
}

// doKeys selects the weapon; it returns true on Y.
func (g *Game) doKeys() bool {
	for {
		ch := g.KeyGet()
		if ch == picopad.NoKey {
			return false
		}
		if ch == picopad.KeyY {
			return true
		}
		if ch == picopad.KeyB && g.weaponMax > 0 && g.shields >= 0 {
			g.weapon++
			if g.weapon > g.weaponMax {
				g.weapon = 1
			}
			g.reloadNext = 0
		}
	}
}

// doShip moves the spaceship.
func (g *Game) doShip() {
	if g.shields < 0 {
		return // dead
	}
	g.move = moveNo
	if g.pressed(picopad.KeyUp) {
		g.shipY -= shipSpeed
		if g.shipY < tileH {
			g.shipY = tileH
		} else {
			g.move = moveU
		}
	}
	if g.pressed(picopad.KeyDown) {
		g.shipY += shipSpeed
		if g.shipY > height-spriteH {
			g.shipY = height - spriteH
		} else {
			g.move = moveD
		}
	}
	if g.pressed(picopad.KeyLeft) {
		g.shipX -= shipSpeed
		if g.shipX < tileW {
			g.shipX = tileW
		} else {
			g.move = moveL
		}
	}
	if g.pressed(picopad.KeyRight) {
		g.shipX += shipSpeed
		if g.shipX > width-tileW-spriteW {
			g.shipX = width - tileW - spriteW
		} else {
			g.move = moveR
		}
	}
}

// doShooting shoots the machine gun and the current weapon.
func (g *Game) doShooting() {
	if g.shields < 0 {
		return // dead
	}
	if g.reloadNext0 > 0 {
		g.reloadNext0--
	}
	if g.reloadNext > 0 {
		g.reloadNext--
	}
	if !g.pressed(picopad.KeyA) {
		return // StopSoundChan(SNDCHAN_GUN)
	}
	// PlaySoundChan(SNDCHAN_GUN, MachinegunSnd ...)

	// a missile of the machine gun
	if g.reloadNext0 <= 0 {
		for i := range g.myMissile {
			if m := &g.myMissile[i]; m.typ < 0 {
				*m = missile{typ: 0, x: float32(g.shipX), y: float32(g.shipY), dy: speedList[0]}
				g.reloadNext0 = reloadList[0]
				break
			}
		}
	}

	// a missile of the current weapon (the artillery: more)
	if g.weapon != 0 && g.reloadNext <= 0 {
		n := 1
		if g.weapon == weaponArtillery {
			n = artilleryNum
		}
		for i := 0; i < myMissileMax && n > 0; i++ {
			m := &g.myMissile[i]
			if m.typ >= 0 {
				continue
			}
			*m = missile{typ: g.weapon, x: float32(g.shipX), y: float32(g.shipY), dy: speedList[g.weapon]}
			if g.weapon == weaponArtillery {
				s := speedList[g.weapon] * g.RandFloatMinMax(0.9, 1.1) // random speed
				m.dx = 0.7 * g.RandFloatMinMax(-1.0, +1.0)             // random direction
				m.dy = float32(math.Sqrt(float64(1 - m.dx*m.dx)))
				m.dx *= s
				m.dy *= s
			}
			n--
		}
		g.reloadNext = reloadList[g.weapon]
		// PlaySoundChan(SNDCHAN_WEAPON, SoundList[Weapon] ...)
	}
}

// hitEnemy: the missile m of weapon j hit e.
func (g *Game) hitEnemy(e *enemy, j int) {
	e.energy -= weaponEnergy[j]
	if e.energy > 0 {
		return
	}
	g.score += enemyPoints[e.typ]
	if j == weaponBlackHole {
		e.typ = -1 // sucked through the black hole
	} else {
		e.typ = enemyNum // explosion
		e.anim = 0
	}

	// a shield every 10000
	if g.score >= g.nextShield && g.shields >= 0 {
		g.nextShield += 10000
		g.shields++
	}

	// the next level at 1000, 2000, 5000, 10000, 17000...: a new weapon
	if g.score >= 1000*(1+(g.level-1)*(g.level-1)) {
		g.level++
		if g.level <= weaponNum {
			g.weaponMax = g.level - 1
			g.weapon = g.weaponMax
		}
	}
}

// moveMissile moves the missiles of the spaceship.
func (g *Game) moveMissile() {
	for i := range g.myMissile {
		m := &g.myMissile[i]
		j := m.typ
		if j < 0 || j >= weaponNum {
			continue
		}

		// the sonar bomb goes to the nearest enemy
		if j == weaponSonarBomb {
			m.dx, m.dy = 0, 0
			kbest := -1
			var distbest float32
			for k := range g.enemy {
				if e := &g.enemy[k]; e.typ >= 0 && e.typ < enemyNum {
					dx, dy := e.x-m.x, e.y-m.y
					if kbest < 0 || dx*dx+dy*dy < distbest {
						kbest, distbest = k, dx*dx+dy*dy
					}
				}
			}
			if kbest >= 0 {
				e := &g.enemy[kbest]
				// the float functions of C: in float64, rounded to float32
				a := float32(math.Atan2(float64(e.y-m.y), float64(e.x-m.x)))
				m.dx = float32(math.Cos(float64(a))) * speedList[weaponSonarBomb]
				m.dy = -float32(math.Sin(float64(a))) * speedList[weaponSonarBomb]
			}
		}

		m.x += m.dx
		m.y -= m.dy
		if m.y < -spriteH || m.y > height || m.x < -spriteW || m.x > width {
			m.typ = -1 // out of the screen
			continue
		}

		// hit an enemy
		for k := range g.enemy {
			e := &g.enemy[k]
			if e.typ < 0 || e.typ >= enemyNum {
				continue
			}
			if dx, dy := e.x-m.x, e.y-m.y; dx*dx+dy*dy < spriteW*spriteH {
				g.hitEnemy(e, j)
				if j != weaponBlackHole { // the missile hits
					m.typ = weaponNum
					m.anim = 0
					break
				}
			}
		}

		// the black hole takes the enemy missiles
		if j == weaponBlackHole {
			for k := range g.enMissile {
				if m2 := &g.enMissile[k]; m2.typ >= 0 {
					if dx, dy := m2.x-m.x, m2.y-m.y; dx*dx+dy*dy < spriteW*spriteH {
						m2.typ = -1
					}
				}
			}
		}
	}
}

// doEnemy makes and moves the enemies.
func (g *Game) doEnemy() {
	// a new enemy
	if int(g.RandU16()) < 4000+400*g.level {
		for i := range g.enemy {
			e := &g.enemy[i]
			if e.typ >= 0 {
				continue
			}
			j := min(g.level-1, enemyNum-1)
			e.typ = int(g.RandU8Max(uint8(j)))
			e.x = float32(g.RandU16MinMax(spriteW, width-tileW-spriteW))
			e.y = -2 * spriteH
			e.dx = 0
			e.dy = enemySpeed[e.typ]
			e.anim = 0
			e.energy = enemyEnergyList[e.typ]
			if e.typ > 1 { // towards the spaceship
				e.dx = (float32(g.shipX) - e.x) * 0.01
			}
			break
		}
	}

	// move the enemies (and the dead ones)
	for i := range g.enemy {
		if e := &g.enemy[i]; e.typ >= 0 {
			e.x += e.dx
			e.y += e.dy
			if e.y > height {
				e.typ = -1
			}
		}
	}
}

// doEnemyShooting: the enemies shoot.
func (g *Game) doEnemyShooting() {
	for i := range g.enemy {
		e := &g.enemy[i]
		j := e.typ
		if j < 0 || j >= enemyNum || g.RandU16() >= 1000 {
			continue
		}
		for k := range g.enMissile {
			if m := &g.enMissile[k]; m.typ < 0 {
				*m = missile{typ: j, x: e.x, y: e.y, dy: enemyMissileSpeed[j]}
				if j > 1 { // towards the spaceship
					m.dx = (float32(g.shipX) - m.x) * 0.02
				}
				break
			}
		}
	}
}

// moveEnemyMissile moves the enemy missiles; one hitting the spaceship
// takes energy, and a shield when the energy runs out.
func (g *Game) moveEnemyMissile() {
	for i := range g.enMissile {
		m := &g.enMissile[i]
		j := m.typ
		if j < 0 {
			continue
		}
		m.x += m.dx
		m.y += m.dy
		if m.y < -spriteH || m.y > height || m.x < -spriteW || m.x > width {
			m.typ = -1
			continue
		}
		dx, dy := float32(g.shipX)-m.x, float32(g.shipY)-m.y
		if float64(dx*dx+dy*dy) < spriteW*spriteH*0.4 && g.shields >= 0 {
			g.energy -= enemyMissileEnergy[j]
			m.typ = -1
			if g.energy < 0 {
				g.shields--
				if g.shields >= 0 {
					g.energy = energyMax
				} else {
					g.energy = 0
				}
			}
		}
	}
}

// dispSprite draws a sprite.
func (g *Game) dispSprite(sprite, x, y int) {
	g.DrawBlitPal(spritesImg, spritesImgPal, 0, sprite*spriteH, x, y, spriteW, spriteH, spriteW, picopad.RGB(149, 34, 140))
}

// dispTile draws a tile.
func (g *Game) dispTile(tile, x, y int) {
	g.DrawBlit4Pal(tilesImg, tilesImgPal, 0, tile*tileH, x, y, tileW, tileH, tileW, picopad.RGB(149, 34, 140))
}

// dispShadow draws a shadow down right of x, y.
func (g *Game) dispShadow(shadow, x, y int) {
	g.DrawBlit1Shadow(shadowsImg, 0, shadow*shadowH, x+shadowShiftX, y+shadowShiftY, shadowW, shadowH, shadowW, 13)
}

// dispLandscape draws the scrolled landscape.
func (g *Game) dispLandscape() {
	g.DrawImg4Pal(landscapeImg, landscapeImgPal, 0, 0, 0, g.landscapeY, landscapeW, landscapeH, landscapeW)
	g.DrawImg4Pal(landscapeImg, landscapeImgPal, 0, 0, 0, g.landscapeY-landscapeH, landscapeW, landscapeH, landscapeW)
}

// dispShadows draws the shadows of the spaceship and the enemies.
func (g *Game) dispShadows() {
	if g.shields >= 0 {
		g.dispShadow(shadowShip, g.shipX, g.shipY)
	}
	for i := range g.enemy {
		if e := &g.enemy[i]; e.typ >= 0 && e.typ < enemyNum {
			g.dispShadow(shadowEnemy1+e.typ, int(e.x), int(e.y))
		}
	}
}

// dispShooting draws the missiles of the spaceship.
func (g *Game) dispShooting() {
	for i := range g.myMissile {
		m := &g.myMissile[i]
		j := m.typ
		if j < 0 {
			continue
		}
		n := int(weaponList[j]) + 1
		switch j {
		case weaponSonarBomb:
			m.anim++
			if m.anim >= spriteWeapon5N {
				m.anim = 0
			}
			n += m.anim
		case weaponBlackHole:
			m.anim++
			if m.anim >= spriteWeapon6N {
				m.anim = 0
			}
			n += spriteWeapon6N - 1 - m.anim
		case weaponNum: // hit
			n += m.anim
			m.anim++
			if m.anim >= 4 {
				m.typ = -1
			}
		}
		g.dispSprite(n, int(m.x), int(m.y))
	}
}

// dispEnemy draws the enemies and the explosions.
func (g *Game) dispEnemy() {
	for i := range g.enemy {
		e := &g.enemy[i]
		j := e.typ
		if j < 0 {
			continue
		}
		if j >= enemyNum { // explosion
			g.dispSprite(spriteExplose1+e.anim, int(e.x), int(e.y))
			e.anim++
			if e.anim >= 8 {
				e.typ = -1
			}
			continue
		}
		j *= 2
		if j == 4*2 { // enemy 5 is animated
			e.anim++
			if e.anim >= 3 {
				e.anim = 0
			}
			if e.anim != 0 {
				j += e.anim + 1
			}
		}
		g.dispSprite(spriteEnemy1+j, int(e.x), int(e.y))
	}
}

// dispShip draws the spaceship (burning when dead).
func (g *Game) dispShip() {
	if g.shields < 0 {
		g.dispSprite(spriteExplose5+(g.gameFrame&3), g.shipX, g.shipY)
		return
	}
	var inx int
	switch g.move {
	case moveL:
		inx = spriteShipL
	case moveR:
		inx = spriteShipR
	case moveD:
		inx = spriteShipD
	case moveU:
		inx = spriteShipU
	default:
		inx = spriteShip + (g.gameFrame & 1)
	}
	g.dispSprite(inx, g.shipX, g.shipY)
}

// dispEnemyShooting draws the enemy missiles.
func (g *Game) dispEnemyShooting() {
	for i := range g.enMissile {
		if m := &g.enMissile[i]; m.typ >= 0 {
			g.dispSprite(m.typ*2+spriteEnemy1Mis, int(m.x), int(m.y))
		}
	}
}

// dispStatus draws the shields, the score, the weapon and the energy.
func (g *Game) dispStatus() {
	for i := 0; i < g.shields; i++ {
		g.dispTile(tileShield, 0, i*tileH)
	}

	s := g.score
	for i := 5; i >= 0 || s > 0; i-- {
		s2 := s / 10
		g.dispTile(tileDigit0+s-s2*10, (width-6*tileW-spriteW/2)/2+i*tileW, 0)
		s = s2
	}

	g.dispSprite(int(weaponList[g.weapon]), width-tileW-spriteW, 2)

	for i := 0; i < g.energy; i += 2 {
		if i <= g.energy-2 {
			g.dispTile(tileEnergy2, width-tileW, i*tileH/2)
		} else {
			g.dispTile(tileEnergy1, width-tileW, i*tileH/2)
		}
	}
}

// waitTime updates the display and waits ms from the last time.
func (g *Game) waitTime(ms uint32) {
	ms *= 1000
	g.DispUpdate()
	var t uint32
	for {
		t = g.Time()
		if t-g.lastTime >= ms {
			break
		}
	}
	g.lastTime = t
}

// step is a frame of the game; it returns false on Y.
func (g *Game) step() bool {
	g.doLandscape()
	if g.doKeys() {
		return false
	}
	g.doShip()
	g.doShooting()
	g.moveMissile()
	g.doEnemy()
	g.doEnemyShooting()
	g.moveEnemyMissile()
	g.display()
	g.waitTime(100)
	g.gameFrame++
	return true
}

// display draws a frame (and ends the hits and the explosions).
func (g *Game) display() {
	g.dispLandscape()
	g.dispShadows()
	g.dispShooting()
	g.dispEnemy()
	g.dispShip()
	g.dispEnemyShooting()
	g.dispStatus()
	g.DispUpdate()
}

// Run plays until Y (main of the C source, which then resets to the boot
// loader).
func (g *Game) Run() {
	g.newGame()
	for g.step() {
	}
}
