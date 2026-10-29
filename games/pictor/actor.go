package pictor

import "github.com/sat0ken/tinygo-cyd/picopad"

// The actors (actor.cpp): Bara and her helpers.

const (
	actJill      = 0 // Bara
	actScarabeus = 3
	actStingray  = 4
	actGhost     = 7
	actSquirrel  = 10
	actGinger    = 11
	actDevilBird = 12

	actMinX      = 3
	actMaxX      = width - 20
	actMinY      = titHeight + 3
	actMaxY      = height - footHeight - 3
	actSpeedX    = 5
	actSpeedY    = 5
	actShadowY   = 210
	actJumpSpeed = 10 // jump height 10+9+8... = 55
	actHitDist   = 20
	bloodTimeMax = 5
	actHitFly    = 6 // hit points of a flying actor
	actHitWalk   = 3 // of a walking one
	shadowW      = 32
	shadowH      = 7
	bloodW       = 16
	bloodH       = 16
	jillW        = 64
	jillRunNum   = 8  // run phases
	jillJumpUp   = 8  // jump up phase
	jillJumpDn   = 9  // jump down phase
	jillStand    = 10 // stand phase
)

// actorTemp is an actor (sActorTemp).
type actorTemp struct {
	name    string
	missile string // name of the missile
	pal     picopad.Palette
	img     string
	w, wall int // width of a phase, of the image
	h       int
	trans   picopad.Color
	walk    bool
}

// actorTemps are the actors: Bara and the helper of each scene.
var actorTemps = [actNum]actorTemp{
	{"Bara", "Shuriken", jillImgPal, jillImg, jillW, 704, 100, picopad.ColBlack, true},
	{"Bird", "Seed", birdImgPal, birdImg, 100, 400, 56, picopad.ColGreen, false},
	{"Parrot", "Berry", parrotImgPal, parrotImg, 80, 320, 59, picopad.ColMagenta, false},
	{"Scarabeus", "Sand Ball", scarabeusImgPal, scarabeusImg, 80, 320, 59, picopad.ColMagenta, true},
	{"Stingray", "Lightning", stingrayImgPal, stingrayImg, 100, 400, 48, picopad.ColMagenta, false},
	{"Ice Bird", "Ice Crystal", icebirdImgPal, icebirdImg, 80, 320, 44, picopad.ColMagenta, false},
	{"Dragon", "Fire Ball", dragonImgPal, dragonImg, 100, 400, 58, picopad.ColMagenta, false},
	{"Ghost", "Screaming", ghostImgPal, ghostImg, 50, 200, 55, picopad.ColMagenta, false},
	{"Robot", "Laser", robotImgPal, robotImg, 50, 200, 61, picopad.ColMagenta, true}, // C: R2-D2
	{"UFO", "Plasma", ufoImgPal, ufoImg, 80, 320, 50, picopad.ColMagenta, false},     // C: Phaser
	{"Rocket Squirrel", "Rocket", squirrelImgPal, squirrelImg, 60, 240, 60, picopad.ColMagenta, false},
	{"Gingerbread Man", "Sugar Bomb", gingerbreadImgPal, gingerbreadImg, 55, 220, 68, picopad.ColMagenta, true},
	{"Devil Bird", "Hole of Nothingness", devilImgPal, devilImg, 100, 400, 72, picopad.ColMagenta, false},
}

// setBlood shows blood at x, y.
func (g *Game) setBlood(x, y int) {
	g.bloodX = min(max(x, bloodW/2+1), width-bloodW/2-1)
	g.bloodY = min(max(y, bgYMin+bloodH/2+1), bgYMax-bloodH/2-1)
	g.bloodTime = bloodTimeMax
	// PlaySoundChan(SOUNDCHAN_ACTORHIT, FailSnd ...)
}

func (g *Game) dispBlood() {
	if g.bloodTime > 0 {
		g.DrawBlit4Pal(bloodImg, bloodImgPal, 0, 0, g.bloodX-bloodW/2, g.bloodY-bloodH/2, bloodW, bloodH, bloodW, picopad.ColMagenta)
	}
}

func (g *Game) shiftBlood() {
	if g.bloodTime > 0 {
		g.bloodTime--
	}
}

// setActor switches to actor actinx (of the ones the scene allows).
func (g *Game) setActor(actinx int) {
	actnum := min(g.level+2, actNum)
	if actinx < 0 {
		actinx += actnum
	}
	if actinx >= actnum {
		actinx -= actnum
	}
	actinx = min(max(actinx, 0), actnum-1)

	g.actorPhase = 0
	g.actorReload = missileTemps[actinx].reload // no cheating by switching
	g.actInx = actinx
	g.actor = actorTemps[actinx]

	g.actorMinX = actMinX + g.actor.w/2
	g.actorMaxX = actMaxX - g.actor.w/2
	g.actorX = min(max(g.actorX, g.actorMinX), g.actorMaxX)
	g.actorMinY = actMinY + g.actor.h/2
	g.actorMaxY = actMaxY - g.actor.h/2
	g.actorY = min(max(g.actorY, g.actorMinY), g.actorMaxY)
	g.actorJumpSpeed = 0
}

// initActor puts the actor at the start of a scene.
func (g *Game) initActor() {
	g.setActor(g.actInx)
	g.actorX = g.actorMinX
	if g.actor.walk {
		g.actorY = g.actorMaxY
	} else {
		g.actorY = (g.actorMinY + g.actorMaxY) / 2
	}
}

// dispActor draws the actor and its shadow.
func (g *Game) dispActor() {
	x := g.actorX - shadowW/2
	if g.actInx == actStingray {
		x -= 10
	}
	g.DrawBlit1Shadow(shadowImg, 0, 0, x, actShadowY, shadowW, shadowH, shadowW, 11)
	a := &g.actor
	g.DrawBlitPal(a.img, a.pal, g.actorPhase*a.w, 0, g.actorX-a.w/2, g.actorY-a.h/2, a.w, a.h, a.wall, a.trans)
}

// shiftActor animates the actor (not jumping).
func (g *Game) shiftActor() {
	if !g.actor.walk || g.actorY >= g.actorMaxY {
		g.actorPhase++
		n := 4
		if g.actInx == actJill {
			n = jillRunNum
		}
		if g.actorPhase >= n {
			g.actorPhase = 0
		}
	}
}

// fallActor moves a jumping actor.
func (g *Game) fallActor() {
	if !g.actor.walk {
		return
	}
	g.actorY += g.actorJumpSpeed
	if g.actorY >= g.actorMaxY {
		g.actorY = g.actorMaxY
		g.actorJumpSpeed = 0
		return
	}
	g.actorJumpSpeed++
	switch {
	case g.actInx != actJill:
		g.actorPhase = 0
	case g.actorJumpSpeed < 0:
		g.actorPhase = jillJumpUp
	default:
		g.actorPhase = jillJumpDn
	}
}

// ctrlActor moves the actor with the buttons; it returns false to end the
// game.
func (g *Game) ctrlActor() bool {
	switch g.keyGet() {
	case picopad.KeyB:
		g.setActor(g.actInx + 1)
	case picopad.KeyX:
		g.setActor(g.actInx - 1)
	case picopad.KeyY: // the game menu
		if !g.gameMenu() {
			return false
		}
		g.scenePalette()
		g.dispHeader()
		g.dispFoot()
		// MusicStart()
	}

	if g.keyPressed(picopad.KeyA) {
		g.genMissile()
	}
	if g.keyPressed(picopad.KeyLeft) {
		g.actorX = max(g.actorX-actSpeedX, g.actorMinX)
	}
	if g.keyPressed(picopad.KeyRight) {
		g.actorX = min(g.actorX+actSpeedX, g.actorMaxX)
	}
	if g.keyPressed(picopad.KeyUp) {
		if g.actor.walk {
			if g.actorY == g.actorMaxY {
				g.actorJumpSpeed = -actJumpSpeed
			}
		} else {
			g.actorY = max(g.actorY-actSpeedY, g.actorMinY)
		}
	}
	if !g.actor.walk && g.keyPressed(picopad.KeyDown) {
		g.actorY = min(g.actorY+actSpeedY, g.actorMaxY)
	}
	g.fallActor()
	return true
}

// hitActor: the bullets and the enemies that hit the actor take its life.
func (g *Game) hitActor() {
	hit := actHitFly
	if g.actor.walk {
		hit = actHitWalk
	}
	for i := range g.bullet {
		b := &g.bullet[i]
		if b.temp == nil {
			continue
		}
		if dx, dy := b.x-g.actorX, b.y-g.actorY; dx*dx+dy*dy <= actHitDist*actHitDist {
			b.temp = nil
			g.life = max(g.life-hit, 0)
			g.setBlood(b.x, b.y)
		}
	}
	for i := range g.enemy {
		e := &g.enemy[i]
		if e.temp == nil {
			continue
		}
		if dx, dy := e.x-g.actorX, e.y-g.actorY; dx*dx+dy*dy <= actHitDist*actHitDist {
			e.temp = nil
			g.score += enemyScore[e.inx]
			g.addPoint(enemyScore[e.inx], e.x, e.y)
			g.life = max(g.life-hit, 0)
			g.setBlood(e.x, e.y)
		}
	}
}
