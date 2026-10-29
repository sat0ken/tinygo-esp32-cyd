package pictor

import "github.com/sat0ken/tinygo-cyd/picopad"

// The missiles of the actors (missile.cpp), the bullets of the enemies
// (bullet.cpp), the bonus (bonus.cpp) and the points (point.cpp).

const (
	missileSpeedXMin  = 5
	missileSpeedXMid  = 10
	missileSpeedXMax  = 20
	missileSpeedYMin  = 2
	missileSpeedYMid  = 4
	missileSpeedYMax  = 10
	speedFlash0       = 10 // the flash of the stingray
	speedFlash1       = 9  // *cos(22.5)
	speedFlash2       = 7  // *cos(45)
	speedFlash3       = 4  // *cos(67.5)
	speedBlackHole    = 5
	lifetimeFlash     = 6
	lifetimeScream    = 12
	missileMax        = 30
	bulletSpeed       = 5
	bulletMax         = 30
	bulletW           = 10
	bonusScore        = 200
	bonusBigScore     = 10000
	bonusBigScoreText = "10000"
	bonusBigScoreLen  = 5
	bonusHeart        = 0
	bonusColFirst     = 1 // the parts of the PicoPad: top, CPU, display, battery
	bonusColLast      = 4
	bonusMaskAll      = 1<<1 | 1<<2 | 1<<3 | 1<<4
	picopadW          = 40 // the whole PicoPad
	picopadH          = 16
	bonusW            = 32 // the cloud of a bonus
	bonusWall         = 192
	bonusH            = 32
	bonusPhases       = 6
	pointNum          = 20
	pointSpeedY       = 2
	pointFrames       = 6
)

// missileTemp is the missile of an actor (sMissileTemp, 4-bit images).
type missileTemp struct {
	pal      picopad.Palette
	img      string
	w, wall  int
	h        int
	phaseNum int
	reload   int // frames to reload
	hits     int // enemies it can hit
	bounce   bool
}

// missileTemps are the missiles of the actors (transparent: magenta).
var missileTemps = [actNum]missileTemp{
	{shurikenImgPal, shurikenImg, 32, 128, 32, 4, 30, 10, true},
	{seedImgPal, seedImg, 8, 8, 8, 1, 6, 10, true},
	{berryImgPal, berryImg, 8, 8, 8, 1, 13, 10, true},
	{sandballImgPal, sandballImg, 16, 32, 16, 2, 8, 10, false},
	{flashImgPal, flashImg, 16, 32, 16, 2, 16, 1, false},
	{crystalImgPal, crystalImg, 16, 16, 16, 1, 20, 2, true},
	{fireImgPal, fireImg, 16, 64, 16, 4, 22, 2, false},
	{screamImgPal, screamImg, 8, 8, 16, 1, 25, 3, true},
	{laserImgPal, laserImg, 8, 8, 8, 1, 4, 1, false},
	{phaserImgPal, phaserImg, 8, 8, 8, 1, 4, 1, false},
	{rocketImgPal, rocketImg, 32, 64, 16, 2, 4, 1, true},
	{sugarbombImgPal, sugarbombImg, 16, 64, 16, 4, 22, 1, true},
	{spiralImgPal, spiralImg, 32, 192, 32, 6, 20, 10, true},
}

// missile is a missile in the air (sMissile); temp nil: not used.
type missile struct {
	temp     *missileTemp
	x, y     int
	dx, dy   int
	actInx   int
	phase    int
	lifetime int
	hits     int
	target   *enemy // the enemy it goes for (nil: none)
}

func (g *Game) initMissile() {
	for i := range g.missile {
		g.missile[i].temp = nil
	}
	g.actorReload = 0
}

// addMissile adds a missile.
func (g *Game) addMissile(temp *missileTemp, actinx, x, y, dx, dy int, target *enemy) {
	for i := range g.missile {
		m := &g.missile[i]
		if m.temp != nil {
			continue
		}
		lifetime := lifetimeScream
		if actinx == actStingray {
			lifetime = lifetimeFlash
		}
		*m = missile{temp: temp, x: x, y: y, dx: dx, dy: dy, actInx: actinx, lifetime: lifetime, hits: temp.hits, target: target}
		return
	}
}

// missileShift moves the missiles: some go for an enemy, the sand balls
// fall, some bounce from the sky and the ground.
func (g *Game) missileShift() {
	for i := range g.missile {
		m := &g.missile[i]
		temp := m.temp
		if temp == nil {
			continue
		}
		precise := m.actInx == actSquirrel || m.actInx == actGinger // the better targeting
		if e := m.target; e != nil {
			if e.temp == nil || (g.RandU16() < 20000 && !precise) {
				m.target = nil // destroyed, or lost
			} else if dx := e.x - m.x; dx > 0 {
				dy := e.y - m.y
				speed := missileSpeedXMax
				if precise {
					speed = missileSpeedXMid
				}
				switch {
				case -dy >= dx: // up
					m.dy = -speed
					m.dx = dx * speed / -dy
				case dy >= dx: // down
					m.dy = speed
					m.dx = dx * speed / dy
				default:
					m.dx = speed
					m.dy = dy * speed / dx
				}
			}
		}

		m.x += m.dx
		m.y += m.dy
		if m.actInx == actScarabeus {
			m.dy++ // falling
		}

		// the sky and the ground
		if k := bgYMin + temp.h/2 + 1; m.y < k {
			if !temp.bounce || m.dx == 0 {
				m.temp = nil
			}
			m.dy = -m.dy
			m.y = k
		}
		if k := bgYMax - temp.h/2 - 1; m.y > k {
			if !temp.bounce || m.dx == 0 {
				m.temp = nil
			}
			m.dy = -m.dy
			m.y = k
		}

		if m.actInx == actStingray || m.actInx == actGhost {
			m.lifetime--
			if m.lifetime < 0 {
				m.temp = nil
			}
		}
		m.phase++
		if m.phase >= temp.phaseNum {
			m.phase = 0
		}
		if m.x-temp.w/2 >= width || m.x < 0 {
			m.temp = nil
		}
	}
	if g.actorReload > 0 {
		g.actorReload--
	}
}

// missileDisp draws the missiles.
func (g *Game) missileDisp() {
	for i := range g.missile {
		m := &g.missile[i]
		if t := m.temp; t != nil {
			g.DrawBlit4Pal(t.img, t.pal, m.phase*t.w, 0, m.x-t.w/2, m.y-t.h/2, t.w, t.h, t.wall, picopad.ColMagenta)
		}
	}
}

// target returns an enemy ahead of x on the screen: tries random ones,
// then (inOrder) the first one.
func (g *Game) target(x, tries int, inOrder bool) *enemy {
	ok := func(e *enemy) bool { return e.temp != nil && e.x > x && e.x-e.temp.w/2 < width }
	for j := tries; j > 0; j-- {
		if e := &g.enemy[g.RandU16Max(enemyMax-1)]; ok(e) {
			return e
		}
	}
	if inOrder {
		for j := range g.enemy {
			if ok(&g.enemy[j]) {
				return &g.enemy[j]
			}
		}
	}
	return nil
}

// genMissile shoots the missile(s) of the actor when it is loaded.
func (g *Game) genMissile() {
	if g.actorReload > 0 {
		return
	}
	a := g.actInx
	g.actorReload = missileTemps[a].reload
	x := g.actorX + g.actor.w/2
	y := g.actorY
	t := &missileTemps[a]
	add := func(dx, dy int) { g.addMissile(t, a, x, y, dx, dy, nil) }

	switch a {
	case 0: // Bara: three shurikens
		add(missileSpeedXMid, -missileSpeedYMid)
		add(missileSpeedXMid, 0)
		add(missileSpeedXMid, missileSpeedYMid)
	case 1: // bird
		add(missileSpeedXMid, 0)
	case 2: // parrot
		add(missileSpeedXMid, -missileSpeedYMin)
		add(missileSpeedXMid, missileSpeedYMin)
	case 3: // scarabeus: a sand ball thrown up
		// dx first: GCC for ARM evaluates the arguments of AddMissile from
		// left to right
		dx := int(g.RandS16MinMax(missileSpeedXMin, missileSpeedXMid))
		dy := int(g.RandS16MinMax(-missileSpeedYMax, -missileSpeedYMin))
		g.addMissile(t, a, x-g.actor.w/4, y-g.actor.h/2, dx, dy, nil)
	case 4: // stingray: 16 flashes around
		add(speedFlash0, 0)
		add(speedFlash1, -speedFlash3)
		add(speedFlash2, -speedFlash2)
		add(speedFlash3, -speedFlash1)
		add(0, -speedFlash0)
		add(-speedFlash3, -speedFlash1)
		add(-speedFlash2, -speedFlash2)
		add(-speedFlash1, -speedFlash3)
		add(-speedFlash0, 0)
		add(-speedFlash1, speedFlash3)
		add(-speedFlash2, speedFlash2)
		add(-speedFlash3, speedFlash1)
		add(0, speedFlash0)
		add(speedFlash3, speedFlash1)
		add(speedFlash2, speedFlash2)
		add(speedFlash1, speedFlash3)
	case 5: // ice bird
		add(missileSpeedXMid, -missileSpeedYMin)
		add(missileSpeedXMid, 0)
		add(missileSpeedXMid, missileSpeedYMin)
	case 6, 7: // dragon, ghost: five ahead
		add(speedFlash2, -speedFlash2)
		add(speedFlash1, -speedFlash3)
		add(speedFlash0, 0)
		add(speedFlash1, speedFlash3)
		add(speedFlash2, speedFlash2)
	case 8, 9: // robot (from the head), UFO: going for an enemy
		if a == 8 {
			y -= g.actor.h / 2
		}
		g.addMissile(t, a, x, y, missileSpeedXMid, 0, g.target(x, enemyMax, false))
	case 10, 11: // squirrel, gingerbread man: surely for an enemy
		g.addMissile(t, a, x, y, missileSpeedXMid, 0, g.target(x, enemyMax*3, true))
	case 12: // devil bird: the black hole
		add(speedBlackHole, 0)
	}
	// PlaySoundChan(SOUNDCHAN_MISSILE, temp->sound ...)
}

// bulletTemp is a bullet of the enemies (sBulletTemp, 4-bit, 10 x 10).
type bulletTemp struct {
	pal picopad.Palette
	img string
}

var bulletTemps = [3]bulletTemp{
	{bullet1ImgPal, bullet1Img},
	{bullet2ImgPal, bullet2Img},
	{bullet3ImgPal, bullet3Img},
}

// bullet is a bullet in the air (sBullet); temp nil: not used.
type bullet struct {
	temp *bulletTemp
	x, y int
}

func (g *Game) initBullet() {
	for i := range g.bullet {
		g.bullet[i].temp = nil
	}
}

func (g *Game) addBullet(temp *bulletTemp, x, y int) {
	for i := range g.bullet {
		if g.bullet[i].temp == nil {
			g.bullet[i] = bullet{temp, x, y}
			return
		}
	}
}

func (g *Game) bulletShift() {
	for i := range g.bullet {
		b := &g.bullet[i]
		if b.temp != nil {
			b.x -= bulletSpeed
			if b.x+bulletW/2 <= 0 {
				b.temp = nil
			}
		}
	}
}

func (g *Game) bulletDisp() {
	for i := range g.bullet {
		b := &g.bullet[i]
		if b.temp != nil {
			g.DrawBlit4Pal(b.temp.img, b.temp.pal, 0, 0, b.x-bulletW/2, b.y-bulletW/2, bulletW, bulletW, bulletW, picopad.ColMagenta)
		}
	}
}

// bonusTemp is a bonus (sBonusTemp, 4-bit): a heart or a part of the
// PicoPad, and the whole PicoPad.
type bonusTemp struct {
	pal     picopad.Palette
	img     string
	w, wall int
	h       int
	trans   picopad.Color
}

var bonusTemps = [6]bonusTemp{
	{heartImgPal, heartImg, 16, 16, 16, picopad.ColMagenta},
	{picopad1ImgPal, picopad1Img, 39, 39, 16, picopad.ColMagenta}, // top cover
	{picopad2ImgPal, picopad2Img, 26, 26, 10, picopad.ColMagenta}, // CPU
	{picopad3ImgPal, picopad3Img, 22, 22, 14, picopad.ColMagenta}, // display
	{picopad4ImgPal, picopad4Img, 9, 9, 14, picopad.ColMagenta},   // battery
	{picopadImgPal, picopadImg, picopadW, picopadW, picopadH, picopad.ColMagenta},
}

// bonusLevel are the bonuses of the scenes.
var bonusLevel = [bgNum]int{1, 2, bonusHeart, 3, 4, bonusHeart, 1, 2, bonusHeart, 3, 4, bonusHeart}

// initBonus puts the bonus of the scene in the middle of it.
func (g *Game) initBonus() {
	g.bonusInx = bonusLevel[g.backInx]
	g.bonus = &bonusTemps[g.bonusInx]
	g.bonusX = g.maxFrame/2*bgSpeed3 + width
	g.bonusY = int(g.enemyRandMinMax(bgYMin+100, bgYMax-40))
	g.bonusPhase = 0
}

func (g *Game) bonusShift() {
	g.bonusPhase++
	if g.bonusPhase >= bonusPhases {
		g.bonusPhase = 0
	}
	g.bonusX -= bgSpeed3
}

// bonusVisible reports whether the bonus is there and near the screen.
func (g *Game) bonusVisible() bool {
	return g.bonusX >= -30 && g.bonusInx >= 0 && g.bonusX < width+30
}

func (g *Game) bonusDisp() {
	if !g.bonusVisible() {
		return
	}
	x, y, b := g.bonusX, g.bonusY, g.bonus
	g.DrawBlit4Pal(bonuscloudImg, bonuscloudImgPal, g.bonusPhase*bonusW, 0, x-bonusW/2, y-bonusH/2, bonusW, bonusH, bonusWall, picopad.ColMagenta)
	g.DrawBlit4Pal(b.img, b.pal, 0, 0, x-b.w/2, y-b.h/2, b.w, b.h, b.wall, b.trans)
}

// bonusCollect: the actor takes the bonus (a heart: life; a part: points,
// and the whole PicoPad 10000).
func (g *Game) bonusCollect() {
	if !g.bonusVisible() {
		return
	}
	x := g.bonusX
	if !(g.actorX+g.actor.w/2 >= x && g.actorX-g.actor.w/2 < x &&
		g.actorY+g.actor.h/2 >= g.bonusY && g.actorY-g.actor.h/2 < g.bonusY) {
		return
	}
	if g.bonusInx == bonusHeart {
		g.life += 100
	} else {
		g.score += bonusScore
		g.addPoint(bonusScore, g.bonusX, g.bonusY)
	}
	// PlaySoundChan(SOUNDCHAN_ACTORHIT, ZingSnd ...)
	if g.bonusInx >= bonusColFirst && g.bonusInx <= bonusColLast {
		g.bonusMask |= 1 << g.bonusInx
		if g.bonusMask == bonusMaskAll { // the PicoPad is complete
			g.score += bonusBigScore
			g.dispHeader()
			g.DispUpdate()
			g.WaitMs(1000)
			g.bonusMask = 0
			g.dispHeader()
			g.DispUpdate()
		}
	}
	g.bonusInx = -1
}

// point is points flying up (sPoint); text "": not used.
type point struct {
	text   string
	x, y   int
	frames int
}

func (g *Game) initPoint() {
	for i := range g.point {
		g.point[i].text = ""
	}
}

// pointText returns the text of points (the ones there are without an
// allocation).
func pointText(val int) string {
	switch val {
	case 10:
		return "10"
	case 20:
		return "20"
	case 50:
		return "50"
	case bonusScore:
		return "200"
	}
	return itoa(val)
}

func (g *Game) addPoint(val, x, y int) {
	for i := range g.point {
		if p := &g.point[i]; p.text == "" {
			*p = point{pointText(val), x, y, pointFrames}
			return
		}
	}
}

func (g *Game) pointShift() {
	for i := range g.point {
		if p := &g.point[i]; p.text != "" {
			p.y -= pointSpeedY
			p.frames--
			if p.frames <= 0 {
				p.text = ""
			}
		}
	}
}

func (g *Game) pointDisp() {
	for i := range g.point {
		if p := &g.point[i]; p.text != "" {
			g.DrawText(p.text, p.x-len(p.text)*4, p.y-8, colAzure)
		}
	}
}
