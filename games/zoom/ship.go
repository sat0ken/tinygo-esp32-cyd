package zoom

import "github.com/sat0ken/tinygo-cyd/picopad"

// The ship (ship.cpp), its shots (shot.cpp) and the gates (gates.cpp).

const (
	shipMinX      = 30 << frac
	shipMaxX      = width<<frac - shipMinX
	shipDefX      = (width / 2) << frac
	shipSpeedMin  = 30 << frac
	shipSpeedMax  = 120 << frac
	shipSpeedDef  = (shipSpeedMin + shipSpeedMax) / 2
	shipCrashNum  = 8  // pictures of the explosion
	shipCrashWait = 10 // steps of the explosion, with the wait at the end
	shipAnimTime  = 300000
	shipNewTime   = 2000000 // a new ship is protected [µs]
	shipHitDist   = 8

	// the pictures of the ship
	shipUp     = 0
	shipLeft   = 1
	shipRight  = 2
	shipShadow = 3

	shotsMax = 4
	shotFree = -32768 // x of an unused shot
	shotTime = 500000 // between the shots [µs]
	shotDist = 50 << frac

	gatesMax    = 3
	gateDistX   = 100 // between the pylons
	gateFree    = -32768
	gateDistZ   = 80
	gateInitZ   = 110
	gatePassDX  = 40  // through the gate
	gateCrashDX = 100 // into a pylon
)

// dispShip draws the ship and its shadow, or its explosion.
func (g *Game) dispShip() {
	y := height - 30 - int(g.speed)>>(frac+3)
	if g.subLevel == 3 {
		y = height - 40
	}
	x := int(g.shipX) >> frac
	if g.shipInCrash > 0 {
		if g.shipInCrash < shipCrashNum {
			g.DrawBlitPal(explosionImg, explosionImgPal, 0, int(g.shipInCrash)*explosionH, x-explosionW/2, y-15,
				explosionW, explosionH, explosionW, picopad.ColMagenta)
		}
		return
	}
	if g.subLevel != 3 {
		g.DrawBlitPal(shipImg, shipImgPal, 0, shipShadow*shipH, x-shipW/2, height-20, shipW, shipH, shipW, picopad.ColMagenta)
	}
	phase := shipUp
	if g.keyPressed(picopad.KeyLeft) {
		phase = shipLeft
	}
	if g.keyPressed(picopad.KeyRight) {
		phase = shipRight
	}
	g.DrawBlitPal(shipImg, shipImgPal, 0, phase*shipH, x-shipW/2, y, shipW, shipH, shipW, picopad.ColMagenta)
}

// shipCrash makes the ship explode (not a new one).
func (g *Game) shipCrash() {
	if g.shipInCrash > 0 || g.Time()-g.shipNewTime < shipNewTime {
		return
	}
	g.shipInCrash = 1
	g.shipCrashTime = g.Time()
	// PLAYSOUND(ExplosionSnd)
}

// shipEnemyCrash: an enemy or a bomb at the ship.
func (g *Game) shipEnemyCrash() {
	x, z := g.shipX, int16(5<<frac)
	for i := range g.obj {
		o := &g.obj[i]
		if t := o.typ; t != enemyDisk && t != enemyBig && t != enemyBoss && t != enemyBomb {
			continue
		}
		if int32(g.Time()-o.startTime) < enemyProtDelay {
			continue
		}
		x2, z2 := int(int16(o.x-x)), int(int16(o.z-z))
		if x2*x2+z2*z2 <= (shipHitDist<<frac)*(shipHitDist<<frac) {
			g.shipCrash()
		}
	}
}

// doShip moves the ship with the buttons, or its explosion and then a new
// ship.
func (g *Game) doShip(dt, dz uint32) {
	if g.shipInCrash > 0 {
		if g.Time()-g.shipCrashTime >= shipAnimTime {
			g.shipCrashTime = g.Time()
			g.shipInCrash++
			if g.shipInCrash == shipCrashWait && g.lives > 0 {
				g.lives--
				if g.fuel <= 0 {
					g.fuel = (fuelMax * 3 / 4) << frac
				}
				g.shipInCrash = 0
				g.shipNewTime = g.Time()
			}
		}
		return
	}
	d := int(dt >> 7)
	if g.keyPressed(picopad.KeyLeft) {
		g.shipX = int16(max(int(g.shipX)-d, shipMinX))
	}
	if g.keyPressed(picopad.KeyRight) {
		g.shipX = int16(min(int(g.shipX)+d, shipMaxX))
	}
	if g.keyPressed(picopad.KeyUp) && g.subLevel != 3 {
		g.speed = int16(min(int(g.speed)+d, shipSpeedMax))
	}
	if g.keyPressed(picopad.KeyDown) && g.subLevel != 3 {
		g.speed = int16(max(int(g.speed)-d, shipSpeedMin))
	}
}

// shot is a shot (sShot): x shotFree is unused.
type shot struct {
	x, z int16 // with frac
}

func (g *Game) resetShots() {
	for i := range g.shots {
		g.shots[i].x = shotFree
	}
}

func (g *Game) dispShots() {
	for i := range g.shots {
		s := &g.shots[i]
		if s.x == shotFree {
			continue
		}
		y := g.getPerspTab(int(s.z))
		n := min(int(s.z)>>(4+frac), 3)
		g.DrawBlitPal(shotImg, shotImgPal, 0, n*shotH, int(s.x)>>frac-shotW/2, y-shotH, shotW, shotH, shotW, picopad.ColMagenta)
	}
}

// shooting shoots with A (every 0.5 s, 4 shots at most) and moves the
// shots by dt µs.
func (g *Game) shooting(dt int) {
	if g.keyPressed(picopad.KeyA) && g.shipInCrash == 0 {
		if i := int32(g.Time() - g.lastShotTime); i < -1000000 || i >= shotTime {
			for k := range g.shots {
				if s := &g.shots[k]; s.x == shotFree {
					s.x, s.z = g.shipX, 12<<frac
					g.lastShotTime = g.Time()
					// PlaySoundChan(SNDCHAN_SHOT, LaserSnd ...)
					return // (the shots move from the next frame)
				}
			}
		}
	}
	for i := range g.shots {
		s := &g.shots[i]
		if s.x == shotFree {
			continue
		}
		s.z = int16(int(s.z) + dt>>8)
		if s.z >= shotDist {
			s.x = shotFree
		}
		if g.enemyHit(s.x, s.z) { // (also a shot just ended, as in C)
			s.x = shotFree
		}
	}
}

// gate is a gate of two pylons (sGate): x gateFree is unused.
type gate struct {
	x    int16 // the middle (pixels)
	z    int16 // with frac
	def  bool  // flown through
	bomb bool  // its bombs dropped
}

func (g *Game) resetGates() {
	for i := range g.gates {
		g.gates[i].x = gateFree
	}
	g.gateNext = 0
}

func (g *Game) dispGate() {
	if g.subLevel == 3 { // no gates in space
		return
	}
	for i := range g.gates {
		gt := &g.gates[i]
		if gt.x == gateFree {
			continue
		}
		n := min(max(int(gt.z)/840, 0), 7)
		y := g.getPerspTab(int(gt.z)) - gateH - n // the distant gates do not jump when zoomed in
		x := int(gt.x)
		g.DrawBlitPal(gateImg, gateImgPal, n*gateW, 0, x-gateW/2-gateDistX/2+n*4, y, gateW, gateH, gateW*8, picopad.ColMagenta)
		g.DrawBlitPal(gateImg, gateImgPal, n*gateW, 0, x-gateW/2+gateDistX/2-n*4, y, gateW, gateH, gateW*8, picopad.ColMagenta)
	}
}

// gating moves the gates by dz: the bombs (from sector 2), flying through
// (a goal) or into a pylon, a new gate every gateDistZ.
func (g *Game) gating(dz int) {
	if g.subLevel == 3 {
		return
	}
	for i := range g.gates {
		gt := &g.gates[i]
		if gt.x == gateFree {
			continue
		}
		gt.z = int16(int(gt.z) - dz)
		if gt.z < -4<<frac {
			gt.x = gateFree
			continue
		}
		x := int(gt.x)

		if !gt.bomb && gt.z < 15<<frac {
			gt.bomb = true
			switch {
			case g.level >= 2: // both ways
				g.genBomb(enemyBombLMove, int16((x-gateDistX/2)<<frac), gt.z)
				g.genBomb(enemyBombRMove, int16((x+gateDistX/2)<<frac), gt.z)
			case g.level == 1 && x >= width/2:
				g.genBomb(enemyBombLMove, int16((x-gateDistX/2)<<frac), gt.z)
			case g.level == 1:
				g.genBomb(enemyBombRMove, int16((x+gateDistX/2)<<frac), gt.z)
			}
		}

		if !gt.def && g.shipInCrash == 0 {
			if dx := x - int(g.shipX)>>frac; dx >= -gatePassDX && dx <= gatePassDX && gt.z < 5<<frac {
				gt.def = true
				g.decGoals()
				g.addScore(500 << frac)
				// PlaySoundChan(0, GateSnd ...)
			}
		}

		dx := x - int(g.shipX)>>frac
		if (dx < -gatePassDX && dx >= -gateCrashDX || dx > gatePassDX && dx <= gateCrashDX) && gt.z < 7<<frac && gt.z > 2<<frac {
			g.shipCrash()
		}
	}

	g.gateNext -= dz
	if g.gateNext > 0 {
		return
	}
	for i := range g.gates {
		if gt := &g.gates[i]; gt.x == gateFree {
			gt.z = gateInitZ << frac
			gt.x = g.RandS16MinMax((gateW+gateDistX)/2, width-(gateW+gateDistX)/2)
			gt.def = false
			gt.bomb = false
			g.gateNext = gateDistZ << frac
			return
		}
	}
}
