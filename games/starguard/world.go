package starguard

// The world: the camera (the ship) is at the origin looking along +z; x
// grows to the right, y downwards. A point at (x, y, z) is drawn at
// (cx + x*focal/z, cy + y*focal/z).
const (
	numStars  = 90
	starFar   = 1500
	starSpeed = 320 // world units per second: the ship's speed

	maxEnemies  = 8
	spawnFar    = 1200
	nearZ       = 40 // an enemy this close hits the ship
	rockRadius  = 30
	robotRadius = 26

	shield0    = 100
	rockHit    = 10 // shield lost when a rock reaches the ship
	robotHit   = 15
	boomTime   = 0.45
	flashTime  = 0.2
	laserTime  = 0.08
	fireEvery  = 0.18
	spawnEvery = 1.3
)

type star struct{ x, y, z float32 }

type kind uint8

const (
	rock kind = iota
	robot
)

// enemy is a rock or a robot ship coming at the ship, or the explosion
// of one (boom > 0). It flies from (x0, y0) far away to (tx, ty) next to
// the ship, in a straight line in depth: on the screen it stays about
// where it appeared and comes to the middle only at the end. (Flying
// straight ahead, it would leave the view as it comes closer and hit the
// ship unseen.)
type enemy struct {
	alive   bool
	kind    kind
	x, y, z float32
	x0, y0  float32
	tx, ty  float32
	vz      float32 // speed towards the ship
	phase   float32 // robots weave from side to side
	boom    float32 // seconds left of the explosion, 0: none
}

func (e *enemy) radius() float32 {
	if e.kind == robot {
		return robotRadius
	}
	return rockRadius
}

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

// spawn sends a new enemy from far away, inside the view.
func (g *Game) spawn() {
	for i := range g.enemies {
		e := &g.enemies[i]
		if e.alive || e.boom > 0 {
			continue
		}
		halfW := float32(g.w) / 2 * spawnFar / g.l.focal * 0.7
		halfH := float32(g.l.viewH) / 2 * spawnFar / g.l.focal * 0.6
		*e = enemy{alive: true, x0: g.randf(-halfW, halfW), y0: g.randf(-halfH, halfH), z: spawnFar,
			tx: g.randf(-12, 12), ty: g.randf(-8, 8)}
		e.x, e.y = e.x0, e.y0
		if g.rand()%3 == 0 {
			e.kind = robot
			e.vz = g.randf(200, 260)
			e.phase = g.randf(0, 6.28)
		} else {
			e.vz = g.randf(150, 220)
		}
		return
	}
}

// moveWorld moves the stars and the enemies by dt seconds; enemies that
// reach the ship hit its shield.
func (g *Game) moveWorld(dt float32) {
	for i := range g.stars {
		s := &g.stars[i]
		s.z -= starSpeed * dt
		if s.z < 10 {
			g.newStar(s, starFar)
		}
	}
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
		f := (e.z - nearZ) / (spawnFar - nearZ) // 1 far away, 0 at the ship
		e.x = e.tx + (e.x0-e.tx)*f
		e.y = e.ty + (e.y0-e.ty)*f
		if e.kind == robot {
			// The weave grows with the depth, so it looks the same size.
			e.phase += 2.2 * dt
			e.x += sinf(e.phase) * 0.25 * e.z
		}
		if e.z < nearZ {
			e.alive = false
			if e.kind == robot {
				g.shield -= robotHit
			} else {
				g.shield -= rockHit
			}
			g.flash = flashTime
		}
	}
	g.spawnT += dt
	if g.spawnT >= spawnEvery {
		g.spawnT -= spawnEvery
		g.spawn()
	}
}

// project returns where world point (x, y, z) is drawn and how big a
// radius r looks.
func (g *Game) project(x, y, z, r float32) (sx, sy, sr float32) {
	f := g.l.focal / z
	return float32(g.w)/2 + x*f, float32(g.l.viewH)/2 + y*f, r * f
}

// target returns the nearest live enemy under the crosshair, or -1.
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
	return best
}

// fire shoots at the crosshair: the nearest enemy under it explodes.
func (g *Game) fire() {
	g.laser = laserTime
	if i := g.target(); i >= 0 {
		e := &g.enemies[i]
		e.alive, e.boom = false, boomTime
		if e.kind == robot {
			g.score += 30
		} else {
			g.score += 10
		}
	}
}
