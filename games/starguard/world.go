package starguard

// The world: the camera (the ship) is at the origin looking along +z; x
// grows to the right, y downwards. A point at (x, y, z) is drawn at
// (cx + x*focal/z, cy + y*focal/z).
const (
	numStars  = 90
	starFar   = 1500
	starSpeed = 320 // world units per second: the ship's speed

	maxEnemies    = 8
	spawnFar      = 1200
	nearZ         = 40 // an enemy this close hits the ship
	rockRadius    = 30
	robotRadius   = 26
	capsuleRadius = 20

	shield0      = 100
	capsuleBoost = 25 // shield back for shooting a capsule
	clearBoost   = 20 // shield back after a stage
	boomTime     = 0.45
	flashTime    = 0.2
	laserTime    = 0.08
	fireEvery    = 0.18
)

// Level is how hard the game is.
type Level struct {
	Name   string
	Speed  float32 // times the enemies' speed
	Spawn  float32 // times the time between enemies
	Damage float32 // times the shield lost when one gets through
	BossHP int
}

var levels = [...]Level{
	{"EASY", 0.8, 1.25, 0.7, 16},
	{"NORMAL", 1, 1, 1, 24},
	{"HARD", 1.2, 0.8, 1.3, 32},
}

// stage is one of the three stages before the mothership: destroy count
// enemies; they come every every seconds, robots percent of them.
type stage struct {
	count  int
	every  float32
	robots uint32
	speed  float32
}

var stages = [...]stage{
	{10, 1.4, 20, 1},
	{14, 1.1, 35, 1.15},
	{18, 0.9, 50, 1.3},
}

// bossStage is the stage of the mothership.
const bossStage = len(stages)

type star struct{ x, y, z float32 }

type kind uint8

const (
	rock kind = iota
	robot
	drone   // a robot launched by the mothership
	capsule // shooting it gives shield back
)

// enemy is a rock, a robot ship, a drone or a capsule coming at the ship,
// or the explosion of one (boom > 0). It flies from (x0, y0) far away to
// (tx, ty) next to the ship, in a straight line in depth: on the screen it
// stays about where it appeared and comes to the middle only at the end.
// (Flying straight ahead, it would leave the view as it comes closer and
// hit the ship unseen.)
type enemy struct {
	alive   bool
	kind    kind
	x, y, z float32
	x0, y0  float32
	z0      float32
	tx, ty  float32
	vz      float32 // speed towards the ship
	phase   float32 // robots weave from side to side
	boom    float32 // seconds left of the explosion, 0: none
	big     float32 // the explosion's size, times the radius
}

func (e *enemy) radius() float32 {
	switch e.kind {
	case robot, drone:
		return robotRadius
	case capsule:
		return capsuleRadius
	}
	return rockRadius
}

// mothership is the boss of the last stage.
type mothership struct {
	active  bool
	x, y, z float32
	phase   float32
	hp, max int
	launchT float32
	dying   float32 // seconds left of its explosions, 0: alive
}

const (
	bossRadius = 80 // for hitting it: half its width is about 1.4 times this
	bossStop   = 380
	bossSpeed  = 160
	bossLaunch = 1.6 // seconds between drones
	bossDying  = 1.6
)

// rand is a xorshift32 generator, so a game can be repeated from a seed.
func (g *Game) rand() uint32 {
	g.rng ^= g.rng << 13
	g.rng ^= g.rng >> 17
	g.rng ^= g.rng << 5
	return g.rng
}

// randf returns a number in [lo, hi).
func (g *Game) randf(lo, hi float32) float32 {
	return lo + (hi-lo)*float32(g.rand()%10000)/10000
}

// newStar puts star s somewhere ahead, at depth z.
func (g *Game) newStar(s *star, z float32) {
	s.x = g.randf(-900, 900)
	s.y = g.randf(-700, 700)
	s.z = z
}

// freeEnemy returns an unused enemy, or nil.
func (g *Game) freeEnemy() *enemy {
	for i := range g.enemies {
		if e := &g.enemies[i]; !e.alive && e.boom <= 0 {
			return e
		}
	}
	return nil
}

// launch sends e from (x, y, z) to a point next to the ship.
func (g *Game) launch(e *enemy, k kind, x, y, z, vz float32) {
	*e = enemy{alive: true, kind: k, x: x, y: y, x0: x, y0: y, z: z, z0: z, vz: vz,
		tx: g.randf(-12, 12), ty: g.randf(-8, 8), phase: g.randf(0, 6.28)}
}

// spawn sends a new enemy (or a capsule) of the stage from far away,
// inside the view.
func (g *Game) spawn() {
	e := g.freeEnemy()
	if e == nil {
		return
	}
	st, lv := &stages[g.stage], &levels[g.level]
	halfW := float32(g.w) / 2 * spawnFar / g.l.focal * 0.7
	halfH := float32(g.l.viewH) / 2 * spawnFar / g.l.focal * 0.6
	x, y := g.randf(-halfW, halfW), g.randf(-halfH, halfH)
	switch {
	case g.shield < 70 && g.rand()%8 == 0:
		g.launch(e, capsule, x, y, spawnFar, 170)
	case g.rand()%100 < st.robots:
		g.launch(e, robot, x, y, spawnFar, g.randf(200, 260)*st.speed*lv.Speed)
	default:
		g.launch(e, rock, x, y, spawnFar, g.randf(150, 220)*st.speed*lv.Speed)
	}
}

// moveWorld moves the stars, the enemies and the mothership by dt
// seconds; enemies that reach the ship hit its shield.
func (g *Game) moveWorld(dt float32) {
	for i := range g.stars {
		s := &g.stars[i]
		s.z -= starSpeed * dt
		if s.z < 10 {
			g.newStar(s, starFar)
		}
	}
	lv := &levels[g.level]
	for i := range g.enemies {
		e := &g.enemies[i]
		if e.boom > 0 {
			e.boom -= dt
			e.z -= e.vz * dt * 0.3 // the pieces drift on
			continue
		}
		if !e.alive {
			continue
		}
		e.z -= e.vz * dt
		f := (e.z - nearZ) / (e.z0 - nearZ) // 1 where it started, 0 at the ship
		e.x = e.tx + (e.x0-e.tx)*f
		e.y = e.ty + (e.y0-e.ty)*f
		if e.kind == robot || e.kind == drone {
			// The weave grows with the depth, so it looks the same size.
			e.phase += 2.2 * dt
			e.x += sinf(e.phase) * 0.25 * e.z
		}
		if e.z < nearZ {
			e.alive = false
			switch e.kind {
			case robot, drone:
				g.shield -= int(15 * lv.Damage)
				g.flash = flashTime
			case rock:
				g.shield -= int(10 * lv.Damage)
				g.flash = flashTime
			}
		}
	}
	if g.stage == bossStage {
		g.moveBoss(dt)
		return
	}
	g.spawnT += dt
	if every := stages[g.stage].every * lv.Spawn; g.spawnT >= every {
		g.spawnT -= every
		g.spawn()
	}
}

// moveBoss brings the mothership in, sways it and launches drones; when
// it is destroyed it blows up for a while.
func (g *Game) moveBoss(dt float32) {
	b := &g.boss
	if !b.active {
		return
	}
	if b.dying > 0 {
		b.dying -= dt
		if g.rand()%6 == 0 {
			if e := g.freeEnemy(); e != nil {
				*e = enemy{kind: rock, x: b.x + g.randf(-90, 90), y: b.y + g.randf(-30, 30), z: b.z - 10,
					boom: boomTime, big: g.randf(1, 2)}
			}
		}
		if b.dying <= 0 {
			b.active = false
		}
		return
	}
	if b.z > bossStop {
		b.z -= bossSpeed * dt
	}
	b.phase += 0.9 * dt
	b.x = sinf(b.phase) * 170
	b.y = -50 + sinf(b.phase*1.7)*25
	b.launchT += dt
	if every := bossLaunch * levels[g.level].Spawn; b.launchT >= every && b.z <= bossStop+100 {
		b.launchT -= every
		if e := g.freeEnemy(); e != nil {
			g.launch(e, drone, b.x, b.y+30, b.z-20, 240*levels[g.level].Speed)
		}
	}
}

// project returns where world point (x, y, z) is drawn and how big a
// radius r looks.
func (g *Game) project(x, y, z, r float32) (sx, sy, sr float32) {
	f := g.l.focal / z
	return float32(g.w)/2 + x*f, float32(g.l.viewH)/2 + y*f, r * f
}

// target returns the nearest live enemy under the crosshair, -2 for the
// mothership, or -1.
func (g *Game) target() int {
	best, bestZ := -1, float32(0)
	for i := range g.enemies {
		e := &g.enemies[i]
		if !e.alive {
			continue
		}
		sx, sy, sr := g.project(e.x, e.y, e.z, e.radius())
		dx, dy := g.aimX-sx, g.aimY-sy
		if sr < 4 {
			sr = 4 // far ones are hard enough
		}
		if dx*dx+dy*dy <= sr*sr && (best < 0 || e.z < bestZ) {
			best, bestZ = i, e.z
		}
	}
	if b := &g.boss; b.active && b.dying <= 0 && (best < 0 || b.z < bestZ) {
		sx, sy, sr := g.project(b.x, b.y, b.z, bossRadius)
		// The ship is wide and flat: an ellipse 1.4 times wider than high.
		dx, dy := (g.aimX-sx)/1.4, g.aimY-sy
		if dx*dx+dy*dy <= sr*sr/2 {
			best = -2
		}
	}
	return best
}

// fire shoots at the crosshair: the nearest enemy under it explodes.
func (g *Game) fire() {
	g.laser = laserTime
	i := g.target()
	switch {
	case i == -2:
		b := &g.boss
		b.hp--
		g.score += 5
		if b.hp <= 0 {
			b.dying = bossDying
			g.score += 500
		}
	case i >= 0:
		e := &g.enemies[i]
		e.alive, e.boom, e.big = false, boomTime, 1.4
		switch e.kind {
		case rock:
			g.score += 10
			g.killed++
		case robot:
			g.score += 30
			g.killed++
		case drone:
			g.score += 20
		case capsule:
			g.shield += capsuleBoost
			if g.shield > shield0 {
				g.shield = shield0
			}
		}
	}
}
