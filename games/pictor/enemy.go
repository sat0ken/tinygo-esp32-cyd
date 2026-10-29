package pictor

import (
	"math"

	"github.com/sat0ken/tinygo-cyd/picopad"
)

// The enemies and the explosions (enemy.cpp).

const (
	enemySpeed     = 3
	enemySlowSpeed = 1
	enemyBorderY   = 10
	enemyHitDist   = 20
	sugarHitRadius = 70

	explosionsMax   = 20
	explosionW      = 32
	explosionWall   = 256
	explosionH      = 32
	explosionPhases = 8
)

// moveSeg is a part of a way: steps frames (0: for ever) of dx, dy
// (sMoveSeg).
type moveSeg struct {
	steps  int
	dx, dy int
}

func moveL(n int) moveSeg   { return moveSeg{n, -enemySpeed, 0} }
func moveLSU(n int) moveSeg { return moveSeg{n, -enemySpeed, -enemySlowSpeed} }
func moveLSD(n int) moveSeg { return moveSeg{n, -enemySpeed, enemySlowSpeed} }
func moveLD(n int) moveSeg  { return moveSeg{n, -enemySpeed, enemySpeed} }
func moveD(n int) moveSeg   { return moveSeg{n, 0, enemySpeed} }
func moveU(n int) moveSeg   { return moveSeg{n, 0, -enemySpeed} }
func moveLU(n int) moveSeg  { return moveSeg{n, -enemySpeed, -enemySpeed} }

// The ways of the enemies; an enemy goes from segment to segment.
var (
	move1  = []moveSeg{moveL(0)}   // straight left
	move1U = []moveSeg{moveLSU(0)} // left, slowly up
	move1D = []moveSeg{moveLSD(0)} // left, slowly down

	moves = [...][]moveSeg{
		move1, move1U, move1D,
		{moveL(20), moveLD(30), moveL(5), moveLU(10), moveL(0)},                                             // left down
		{moveL(20), moveLU(30), moveL(5), moveLD(10), moveL(0)},                                             // left up
		{moveL(10), moveLU(10), moveL(5), moveLD(20), moveL(5), moveLU(20), moveL(5), moveLD(10), moveL(0)}, // saw
		{moveL(10), moveLD(10), moveL(5), moveLU(20), moveL(5), moveLD(20), moveL(5), moveLU(10), moveL(0)}, // saw 2
		{moveL(40), moveU(20), moveL(0)},                                                                    // up
		{moveL(40), moveD(20), moveL(0)},                                                                    // down
	}
)

// enemyScore are the points of the enemies 1, 2, 3 of a scene.
var enemyScore = [3]int{10, 20, 50}

// enemyTemp is an enemy (sEnemyTemp): the phase is (anim >> animShift) &
// animMask, anim going round to animMax.
type enemyTemp struct {
	name      string
	pal       picopad.Palette
	img       string
	w, wall   int
	h         int
	animMax   int
	animShift int
	animMask  int
}

// enemyTemps are the enemies of the scenes, three each (the transparent
// colour is magenta).
var enemyTemps = [enemyNum]enemyTemp{
	// 1
	{"Fly", flyImgPal, flyImg, 50, 100, 27, 1, 0, 1},
	{"Wasp", waspImgPal, waspImg, 50, 100, 33, 1, 0, 1},
	{"Hornet", hornetImgPal, hornetImg, 50, 100, 43, 1, 0, 1},
	// 2
	{"Dragonfly", dragonflyImgPal, dragonflyImg, 60, 120, 44, 1, 0, 1},
	{"Moth", mothImgPal, mothImg, 40, 80, 46, 1, 0, 1},
	{"Kestrel", kestrelImgPal, kestrelImg, 50, 100, 45, 3, 1, 1},
	// 3
	{"Maggot", maggotImgPal, maggotImg, 37, 37, 50, 0, 0, 0},
	{"Eagle", eagleImgPal, eagleImg, 50, 100, 53, 3, 1, 1},
	{"Fire Ant", fireantImgPal, fireantImg, 50, 100, 46, 3, 1, 1}, // C: Antman
	// 4
	{"Fish", fishImgPal, fishImg, 50, 100, 25, 3, 1, 1},
	{"Anglerfish", anglerfishImgPal, anglerfishImg, 60, 120, 37, 3, 1, 1},
	{"Shark", sharkImgPal, sharkImg, 50, 100, 25, 3, 1, 1},
	// 5
	{"Penguin", penguinImgPal, penguinImg, 30, 60, 35, 1, 0, 1},
	{"Polar Bear", polarbearImgPal, polarbearImg, 60, 120, 40, 3, 1, 1},
	{"Snowflake", snowflakeImgPal, snowflakeImg, 30, 30, 33, 0, 0, 0},
	// 6
	{"Phoenix", phoenixImgPal, phoenixImg, 50, 100, 35, 3, 1, 1},
	{"Fire Ball", fireballImgPal, fireballImg, 30, 60, 35, 3, 1, 1},
	{"Evil", evilImgPal, evilImg, 60, 120, 27, 3, 1, 1},
	// 7
	{"Pumpkin", pumpkinImgPal, pumpkinImg, 30, 60, 28, 3, 1, 1},
	{"Zombie", zombieImgPal, zombieImg, 30, 30, 35, 0, 0, 0},
	{"Death", deathImgPal, deathImg, 65, 130, 35, 3, 1, 1},
	// 8
	{"Camera", cameraImgPal, cameraImg, 35, 35, 16, 0, 0, 0},
	{"Explorer", explorerImgPal, explorerImg, 30, 60, 25, 3, 1, 1},
	{"War Dron", dronImgPal, dronImg, 50, 50, 36, 0, 0, 0},
	// 9 (C: the starship Enterprise, a Klingon warship, the Death Star)
	{"Star Cruiser", cruiserImgPal, cruiserImg, 60, 60, 17, 0, 0, 0},
	{"Warship", warshipImgPal, warshipImg, 50, 50, 26, 0, 0, 0},
	{"Space Mine", mineImgPal, mineImg, 40, 40, 40, 0, 0, 0},
	// 10
	{"Alien", alienImgPal, alienImg, 40, 80, 33, 1, 0, 1},
	{"Alien 2", xenomorphImgPal, xenomorphImg, 45, 90, 48, 1, 0, 1},
	{"Meteor", meteorImgPal, meteorImg, 50, 100, 27, 3, 1, 1},
	// 11
	{"Candy Cane", candycaneImgPal, candycaneImg, 50, 100, 26, 3, 1, 1},
	{"Lollipop", lollipopImgPal, lollipopImg, 50, 100, 32, 3, 1, 1},
	{"Fat Man", fatmanImgPal, fatmanImg, 50, 100, 27, 3, 1, 1},
	// 12
	{"Cloud", cloudImgPal, cloudImg, 35, 35, 29, 0, 0, 0},
	{"Star", starImgPal, starImg, 35, 35, 32, 0, 0, 0},
	{"Moon", moonImgPal, moonImg, 35, 35, 36, 0, 0, 0},
}

// enemy is an enemy on the screen (sEnemy); temp nil: not used.
type enemy struct {
	temp *enemyTemp
	x, y int // middle
	anim int
	move []moveSeg // the way from the current segment
	step int
	inx  int // which of the three
}

// explosion is an explosion (sExplosion); anim explosionPhases: not used.
type explosion struct {
	anim int
	x, y int
}

// initEnemy prepares the enemies of the scene.
func (g *Game) initEnemy() {
	for j := range g.enemyNow {
		g.enemyNow[j] = &enemyTemps[g.backInx*enemyLevNum+j]
	}
	for i := range g.enemy {
		g.enemy[i].temp = nil
	}
	for i := range g.explosion {
		g.explosion[i].anim = explosionPhases
	}
}

// addEnemy adds an enemy (in the first free place).
func (g *Game) addEnemy(temp *enemyTemp, x, y int, move []moveSeg, inx int) {
	for i := range g.enemy {
		if g.enemy[i].temp == nil {
			g.enemy[i] = enemy{temp: temp, x: x, y: y, move: move, inx: inx}
			return
		}
	}
}

// addExplosion adds an explosion.
func (g *Game) addExplosion(x, y int) {
	for i := range g.explosion {
		if e := &g.explosion[i]; e.anim >= explosionPhases {
			*e = explosion{0, x, y}
			return
		}
	}
}

// newEnemy makes a new enemy now and then: more of them later in the
// scene and in the later scenes, the second and third ones later.
func (g *Game) newEnemy() {
	gen := int(float32((enemyGenEnd-enemyGenBeg)*g.currentFrame/g.maxFrame+enemyGenBeg) *
		float32(math.Pow(float64(float32(enemyGenLevel)), float64(g.level)))) // f2i(... * powf(ENEMY_GEN_LEVEL, Level))
	if g.enemyRand() > uint16(gen) {
		return
	}

	// enemy 3 from the middle of the scene, enemy 2 more and more
	gen = 0
	if g.currentFrame*2 >= g.maxFrame {
		gen = (g.currentFrame*2 - g.maxFrame) * 65536 / g.maxFrame
	}
	inx := 0
	if int(g.enemyRand()) < gen {
		inx = 2
	} else if gen = g.currentFrame * 50000 / g.maxFrame; int(g.enemyRand()) < gen {
		inx = 1
	}

	e := g.enemyNow[inx]
	ymin := bgYMin + e.h/2 + enemyBorderY
	ymax := bgYMax - e.h/2 - enemyBorderY
	y := int(g.enemyRandMinMax(uint16(ymin), uint16(ymax)))

	// enemy 1 goes straight, enemy 2 also diagonally, enemy 3 any way
	n := len(moves) - 1
	switch inx {
	case 0:
		n = 2
	case 1:
		n = 4
	}
	g.addEnemy(e, width+e.w/2+2, y, moves[g.enemyRandMax(uint16(n))], inx)
}

// enemyShift moves the enemies, which shoot bullets.
func (g *Game) enemyShift() {
	for i := range g.enemy {
		e := &g.enemy[i]
		temp := e.temp
		if temp == nil {
			continue
		}
		e.anim++
		if e.anim > temp.animMax {
			e.anim = 0
		}

		move := &e.move[0]
		e.x += move.dx
		e.y += move.dy

		ymin := bgYMin + temp.h/2 + enemyBorderY
		ymax := bgYMax - temp.h/2 - enemyBorderY
		if e.y < ymin {
			e.y = ymin
			e.step = move.steps // the next segment
			if &e.move[0] == &move1U[0] {
				e.move = move1D // turn
			}
		}
		if e.y > ymax {
			e.y = ymax
			e.step = move.steps
			if &e.move[0] == &move1D[0] {
				e.move = move1U
			}
		}

		// the next step (the segments that turn have no steps)
		if move.steps != 0 {
			e.step++
			if e.step >= move.steps {
				e.step = 0
				e.move = e.move[1:]
			}
		}

		if e.x+temp.w/2 <= -2 {
			e.temp = nil
		} else if g.RandU16() < enemyGenBul {
			g.addBullet(&bulletTemps[e.inx], e.x-temp.w/2, e.y)
		}
	}
}

// explosionShift animates the explosions.
func (g *Game) explosionShift() {
	for i := range g.explosion {
		if g.explosion[i].anim < explosionPhases {
			g.explosion[i].anim++
		}
	}
}

// enemyDisp draws the enemies.
func (g *Game) enemyDisp() {
	for i := range g.enemy {
		e := &g.enemy[i]
		if t := e.temp; t != nil {
			g.DrawBlitPal(t.img, t.pal, ((e.anim>>t.animShift)&t.animMask)*t.w, 0, e.x-t.w/2, e.y-t.h/2,
				t.w, t.h, t.wall, picopad.ColMagenta)
		}
	}
}

// explosionDisp draws the explosions.
func (g *Game) explosionDisp() {
	for i := range g.explosion {
		if e := &g.explosion[i]; e.anim < explosionPhases {
			g.DrawBlit4Pal(explosionImg, explosionImgPal, e.anim*explosionW, 0, e.x-explosionW/2, e.y-explosionH/2,
				explosionW, explosionH, explosionWall, picopad.ColMagenta)
		}
	}
}

// destroy destroys enemy e: its points and maybe an explosion.
func (g *Game) destroy(e *enemy, explode bool) {
	e.temp = nil
	g.score += enemyScore[e.inx]
	g.addPoint(enemyScore[e.inx], e.x, e.y)
	if explode {
		g.addExplosion(e.x, e.y)
	}
}

// enemyHit: the missiles hit the enemies (the black hole also takes the
// bullets, the sugar bomb the enemies around).
func (g *Game) enemyHit() {
	for i := range g.missile {
		m := &g.missile[i]
		if m.temp == nil {
			continue
		}
		xm, ym := m.x, m.y
		for j := range g.enemy {
			e := &g.enemy[j]
			te := e.temp
			if te == nil || e.x-te.w/2 >= width { // not out of the screen
				continue
			}
			xe, ye := e.x, e.y
			if dx, dy := xe-xm, ye-ym; dx*dx+dy*dy > enemyHitDist*enemyHitDist {
				continue
			}
			if m.actInx == actDevilBird {
				g.destroy(e, false) // sucked in
			} else {
				g.destroy(e, true)
				if m.actInx == actGinger { // the sugar bomb
					for k := range g.enemy {
						e2 := &g.enemy[k]
						if e2.temp == nil {
							continue
						}
						if dx, dy := e2.x-xe, e2.y-ye; dx*dx+dy*dy <= sugarHitRadius*sugarHitRadius {
							g.destroy(e2, true)
						}
					}
				}
			}
			m.hits--
			if m.hits <= 0 {
				m.temp = nil
				break
			}
		}

		// the black hole takes the bullets
		if m.actInx == actDevilBird {
			for j := range g.bullet {
				b := &g.bullet[j]
				if b.temp == nil {
					continue
				}
				if dx, dy := b.x-xm, b.y-ym; dx*dx+dy*dy <= enemyHitDist*enemyHitDist {
					b.temp = nil
				}
			}
		}
	}
}
