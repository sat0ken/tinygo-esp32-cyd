package zoom

import "github.com/sat0ken/tinygo-cyd/picopad"

// The enemies (enemy.cpp): saucers, walkers, the mother ship, their bombs
// and the explosions, on tracks of steps in time.

// object types
const (
	enemyNo    = 0
	enemyDisk  = 1 // a saucer
	enemyBig   = 2 // a walker
	enemyBoss  = 3 // the mother ship
	enemyCrash = 4 // an explosion
	enemyBomb  = 5
)

const (
	speedShift        = 3  // the speeds are in Z << (frac+speedShift) a ms
	enemyDiskSpeed    = 15 // << speedShift
	enemyBigSpeed     = 7
	enemyBossSpeed    = 12
	enemyBombSpeed    = 15
	enemyBossLives    = 5
	enemyDiskDelay    = 500000  // between the saucers of a group [µs]
	enemyProtDelay    = 1000000 // an enemy cannot be hit at first [µs]
	enemyBigNextBomb  = 3000000
	enemyBossNextBomb = 1000000
	enemyCrashTime    = 200 // a step of an explosion [ms]
	enemyHitDist      = 20
	objMax            = 20
)

// the tracks (moves)
const (
	enemyDisk1Move = 0 // a saucer on the left
	enemyDisk2Move = 1 // on the right
	enemyBigMove   = 2
	enemyBossMove  = 3
	enemyCrashMove = 4
	enemyBombLMove = 5 // a bomb to the left
	enemyBombRMove = 6
	enemyBombDMove = 7 // down
)

// enemyPoint are the points of a saucer, a walker and the mother ship.
var enemyPoint = [4]int{0, 200 << frac, 300 << frac, 600 << frac}

// step is a step of a track: dx, dz a ms for dt ms (sStep).
type step struct {
	dx, dz int16
	dt     int16
}

// steps are the steps of the tracks.
var steps = [...]step{
	// a saucer on the left
	{0, enemyDiskSpeed, 200},
	{-4 * enemyDiskSpeed, enemyDiskSpeed, 500},
	{0, enemyDiskSpeed, 3500},
	{enemyDiskSpeed, 0, 2000},
	{0, -2 * enemyDiskSpeed, 800},
	{-2 * enemyDiskSpeed, -enemyDiskSpeed, 1200},
	{2 * enemyDiskSpeed, -enemyDiskSpeed, 1400},
	// a saucer on the right
	{0, enemyDiskSpeed, 200},
	{4 * enemyDiskSpeed, enemyDiskSpeed, 500},
	{0, enemyDiskSpeed, 3500},
	{-enemyDiskSpeed, 0, 2000},
	{0, -2 * enemyDiskSpeed, 800},
	{2 * enemyDiskSpeed, -enemyDiskSpeed, 1200},
	{-2 * enemyDiskSpeed, -enemyDiskSpeed, 1400},
	// a walker
	{0, -enemyBigSpeed, 3000},
	{enemyBigSpeed, -enemyBigSpeed, 800},
	{-2 * enemyBigSpeed, -enemyBigSpeed, 1000},
	{2 * enemyBigSpeed, -enemyBigSpeed, 1400},
	{0, -2 * enemyBigSpeed, 1000},
	{-enemyBigSpeed, -enemyBigSpeed, 400},
	{2 * enemyBigSpeed, -enemyBigSpeed, 400},
	{-enemyBigSpeed, -enemyBigSpeed, 800},
	{2 * enemyBigSpeed, -enemyBigSpeed, 1000},
	{-enemyBigSpeed, -enemyBigSpeed, 100},
	{0, -enemyBigSpeed, 500},
	// the mother ship
	{0, enemyBossSpeed, 1500},
	{-enemyBossSpeed, enemyBossSpeed, 800},
	// ... repeated
	{-enemyBossSpeed, -enemyBossSpeed, 800},
	{enemyBossSpeed, -enemyBossSpeed, 800},
	{enemyBossSpeed, enemyBossSpeed, 1600},
	{enemyBossSpeed, -enemyBossSpeed, 800},
	{-enemyBossSpeed, -enemyBossSpeed, 800},
	{-enemyBossSpeed, enemyBossSpeed, 1600},
	// an explosion
	{0, 0, enemyCrashTime},
	{0, 0, enemyCrashTime},
	{0, 0, enemyCrashTime},
	{0, 0, enemyCrashTime},
	{0, 0, enemyCrashTime},
	{0, 0, enemyCrashTime},
	{0, 0, enemyCrashTime},
	{0, 0, enemyCrashTime},
	// bombs: to the left, to the right, down
	{-3 * enemyBombSpeed, -enemyBombSpeed / 2, 2000},
	{3 * enemyBombSpeed, -enemyBombSpeed / 2, 2000},
	{0, -enemyBombSpeed, 5000},
}

// move is a track (sMove): numStep steps from startStep, then again from
// repStep (-1: the end).
type move struct {
	startStep, numStep, repStep int
}

var moves = [...]move{
	{0, 7, -1},   // a saucer on the left
	{7, 7, -1},   // on the right
	{14, 11, -1}, // a walker
	{25, 8, 27},  // the mother ship
	{33, 8, -1},  // an explosion
	{41, 1, -1},  // bombs
	{42, 1, -1},
	{43, 1, -1},
}

// obj is an object (sObj).
type obj struct {
	typ       int
	move      int
	istep     int    // the step of the track; of an explosion, its phase
	startTime uint32 // [µs]
	time      uint32 // the start of the step
	nextBomb  uint32
	sx, sz    int16 // the start of the step, with frac
	x, z      int16 // with frac
}

func (g *Game) initObj() {
	for i := range g.obj {
		g.obj[i].typ = enemyNo
	}
}

// size returns the picture (0 the nearest, 7) of an object at z.
func size(z int16) int {
	return min(max(int(z)>>(4+frac), 0), 7)
}

// dispObj draws the objects.
func (g *Game) dispObj() {
	for i := range g.obj {
		o := &g.obj[i]
		if o.typ == enemyNo {
			continue
		}
		y := g.getPerspTab(int(o.z))
		n := size(o.z)
		x := int(o.x) >> frac
		switch o.typ {
		case enemyDisk:
			g.DrawBlitPal(ufoImg, ufoImgPal, 0, n*ufoH, x-ufoW/2, y-ufoH, ufoW, ufoH, ufoW, picopad.ColMagenta)
		case enemyBig:
			g.DrawBlitPal(walkerImg, walkerImgPal, n*walkerW, 0, x-walkerW/2, y-walkerH, walkerW, walkerH, walkerW*8, picopad.ColMagenta)
		case enemyBoss:
			g.DrawBlitPal(bossImg, bossImgPal, 0, n*bossH, x-bossW/2, y-bossH, bossW, bossH, bossW, picopad.ColMagenta)
		case enemyCrash:
			g.DrawBlitPal(explosionImg, explosionImgPal, 0, o.istep*explosionH, x-explosionW/2, y-explosionH, explosionW, explosionH, explosionW, picopad.ColMagenta)
		case enemyBomb:
			g.DrawBlitPal(bombImg, bombImgPal, 0, n*bombH, x-bombW/2, y-bombH-15, bombW, bombH, bombW, picopad.ColMagenta)
		}
	}
}

// getFreeObj returns a free object (nil: none).
func (g *Game) getFreeObj() *obj {
	for i := range g.obj {
		if g.obj[i].typ == enemyNo {
			return &g.obj[i]
		}
	}
	return nil
}

// genBomb drops a bomb.
func (g *Game) genBomb(mv int, x, z int16) {
	o := g.getFreeObj()
	if o == nil {
		return
	}
	now := g.Time()
	o.typ = enemyBomb
	o.move = mv
	o.istep = 0
	o.startTime = now
	o.time = now
	o.sx, o.x = x, x
	o.sz, o.z = z, z
}

// doObj makes new enemies and moves the objects.
func (g *Game) doObj() {
	// the mother ship comes for the last goal of the part in space
	boss := g.subLevel == 3 && g.goals <= 1
	if boss && !g.bossIsGen {
		g.bossIsGen = true
		if o := g.getFreeObj(); o != nil { // (C: never full here)
			now := g.Time()
			x, z := int16((width/2)<<frac), int16(-150)
			*o = obj{typ: enemyBoss, move: enemyBossMove, startTime: now, time: now, nextBomb: now + enemyBossNextBomb,
				sx: x, x: x, sz: z, z: z}
		}
	}

	// a group of saucers every 8 s (not in part 0)
	if !boss && g.subLevel != 0 && g.Time()-g.timeLastDisk > 8000000 {
		n := int(g.RandU8MinMax(1, 4))
		x := int(g.RandS16MinMax(100, width-100)) << frac
		mv := enemyDisk1Move
		if x >= (width/2)<<frac {
			mv = enemyDisk2Move
		}
		for i := 0; i < n; i++ {
			o := g.getFreeObj()
			if o == nil {
				break
			}
			g.timeLastDisk = g.Time()
			t := g.Time() + uint32(enemyDiskDelay*i)
			*o = obj{typ: enemyDisk, move: mv, startTime: t, time: t, sx: int16(x), x: int16(x), sz: -150, z: -150}
		}
	}

	// a walker every 12 s (only in part 2)
	if g.subLevel == 2 && g.Time()-g.timeLastBig > 12000000 {
		if o := g.getFreeObj(); o != nil {
			g.timeLastBig = g.Time()
			x, z := int16(int(g.RandS16MinMax(80, width-80))<<frac), int16(150<<frac)
			now := g.Time()
			*o = obj{typ: enemyBig, move: enemyBigMove, startTime: now, time: now, nextBomb: now + enemyBigNextBomb,
				sx: x, x: x, sz: z, z: z}
		}
	}

	// move the objects
	for i := range g.obj {
		o := &g.obj[i]
		if o.typ == enemyNo {
			continue
		}
		m := &moves[o.move]
		n := m.startStep + o.istep
		s := &steps[n]
		dt := int(int32(g.Time()-o.time)) >> 10 // [ms]
		if dt < 0 {
			dt = 0
		}
		for dt >= int(s.dt) { // the next step (from where the whole dt leads, as in C)
			o.sx = int16(int(o.sx) + int(s.dx)*dt>>speedShift)
			o.x = o.sx
			o.sz = int16(int(o.sz) + int(s.dz)*dt>>speedShift)
			o.z = o.sz
			dt -= int(s.dt)
			o.time = g.Time()
			o.istep++
			n++
			if o.istep >= m.numStep {
				n = m.repStep
				o.istep = n - m.startStep
				if n < 0 { // the end of the track
					o.typ = enemyNo
					break
				}
			}
			s = &steps[n]
		}
		if o.typ == enemyNo {
			continue
		}
		o.x = int16(int(o.sx) + int(s.dx)*dt>>speedShift)
		o.z = int16(int(o.sz) + int(s.dz)*dt>>speedShift)

		// the bombs of the walkers (from sector 2) and of the mother ship
		if o.typ == enemyBig && g.level > 0 && int32(g.Time()-o.nextBomb) >= 0 {
			g.genBomb(enemyBombDMove, o.x, o.z)
			o.nextBomb = g.Time() + enemyBigNextBomb
		}
		if o.typ == enemyBoss && int32(g.Time()-o.nextBomb) >= 0 {
			g.genBomb(enemyBombDMove, o.x, o.z)
			o.nextBomb = g.Time() + enemyBossNextBomb
		}
	}
}

// enemyHit reports whether a shot at x, z (with frac) hits an enemy, which
// explodes (the mother ship after 5 hits).
func (g *Game) enemyHit(x, z int16) bool {
	for i := range g.obj {
		o := &g.obj[i]
		t := o.typ
		if t != enemyDisk && t != enemyBig && t != enemyBoss {
			continue
		}
		if int32(g.Time()-o.startTime) < enemyProtDelay {
			continue
		}
		x2, z2 := int(int16(o.x-x)), int(int16(o.z-z))
		if x2*x2+z2*z2 > (enemyHitDist<<frac)*(enemyHitDist<<frac) {
			continue
		}
		if t == enemyBoss {
			g.bossLives--
			if g.bossLives > 0 {
				return true
			}
		}

		// the explosion, from the size of the enemy
		now := g.Time()
		o.typ = enemyCrash
		o.move = enemyCrashMove
		o.istep = size(o.z)
		o.startTime = now
		o.time = now
		o.sx, o.sz = o.x, o.z
		// PLAYSOUND(ExplosionSnd)

		// in space only the mother ship takes the last goal
		if g.subLevel != 3 || g.goals > 1 || t == enemyBoss {
			g.decGoals()
			g.addScore(enemyPoint[t])
		}
		return true
	}
	return false
}
