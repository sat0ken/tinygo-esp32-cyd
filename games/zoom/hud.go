package zoom

import (
	"math"

	"github.com/sat0ken/tinygo-cyd/picopad"
)

// The ground (ground.cpp), the stars (stars.cpp), the score (score.cpp),
// the fuel (fuel.cpp), the goals (goals.cpp) and the lives (lives.cpp).

const (
	groundDY   = 15 // between the lines of the ground
	grassYMin  = horizon
	grassH     = 184
	grassYMax  = grassYMin + grassH - 1
	horizLow   = 50 // the horizon lowered
	distMin    = 0  // the distances of the objects from the projection plane
	distMax    = 255
	distViewer = 40 // the viewer before the projection plane

	starsMax  = 100
	starFree  = -32768
	starTime  = 50000 // a new star [µs]
	starMin   = 500   // speeds
	starMax   = 800
	scoreDigs = 6
	scoreLive = 20000 << frac // a new life
	fuelMax   = 256
)

// goalsInit are the goals of the parts of a sector.
var goalsInit = [4]int{10, 15, 25, 10}

// The grass of the sectors (gradients, from the C source) and the colour of
// its lines (COL_LTGREEN, COL_LTYELLOW, COL_LTRED, COL_WHITE).
var (
	grassImgs = [4]string{grass1Img, grass2Img, grass3Img, grass4Img}
	grassPals = [4]picopad.Palette{grass1ImgPal, grass2ImgPal, grass3ImgPal, grass4ImgPal}
	grassCols = [4]picopad.Color{picopad.RGB(127, 255, 127), picopad.RGB(255, 255, 127), picopad.RGB(255, 127, 127), picopad.ColWhite}
)

// calcPerspTab computes the Y of the distances on the screen:
// y = grassYMax - (grassH+horizLow)/(distViewer+z)*z.
func (g *Game) calcPerspTab() {
	for i := range g.perspTab {
		z := i + distMin
		y := grassYMax - ((grassH+horizLow)*z+(distViewer+z)/2)/(distViewer+z)
		g.perspTab[i] = int16(min(max(y, grassYMin), grassYMax))
	}
}

// getPerspTab returns the Y of distance z (with frac), between the
// entries.
func (g *Game) getPerspTab(z int) int {
	z -= distMin << frac
	if z < 0 {
		return height - z>>3
	}
	last := len(g.perspTab) - 1
	z = min(z, last<<frac)
	z1 := z >> frac
	z2 := min(z1+1, last)
	y1, y2 := int(g.perspTab[z1]), int(g.perspTab[z2])
	d := z & (1<<frac - 1)
	return y1 + (y2-y1)*d/(1<<frac)
}

// dispGround draws the grass and its lines.
func (g *Game) dispGround() {
	n := g.level & 3
	g.DrawImgRle(grassImgs[n], grassPals[n], 0, horizon, width, grassH)
	c := grassCols[n]
	for d := distMin<<frac + int(g.grassPhase); d <= distMax<<frac; d += groundDY << frac {
		g.DrawRect(0, g.getPerspTab(d), width, 1, c)
	}
}

// star is a star flying from the middle (sStar): x starFree is unused.
type star struct {
	x, y   int16 // with frac
	dx, dy int16
}

// newStar1 makes a star.
func (g *Game) newStar1() {
	g.lastStarTime += starTime
	for i := range g.stars {
		s := &g.stars[i]
		if s.x != starFree {
			continue
		}
		s.x = (width / 2) << frac
		s.y = (height / 2) << frac
		d := float32(g.RandU16MinMax(starMin, starMax))
		a := g.RandFloatMax(2 * math.Pi)
		s.dx = int16(d * float32(math.Sin(float64(a))))
		s.dy = int16(d * float32(math.Cos(float64(a))))
		return
	}
}

// newStars makes the stars of the time gone.
func (g *Game) newStars() {
	for int32(g.Time()-g.lastStarTime) > 0 {
		g.newStar1()
	}
}

func (g *Game) initStars() {
	g.lastStarTime = g.Time()
	for i := range g.stars {
		g.stars[i].x = starFree
	}
}

// shiftStars moves the stars a step.
func (g *Game) shiftStars() {
	for i := range g.stars {
		s := &g.stars[i]
		if s.x == starFree {
			continue
		}
		s.x += s.dx
		s.y += s.dy
		if s.x < -4<<frac || s.y < -4<<frac || s.x > (width+4)<<frac || s.y > (height+4)<<frac {
			s.x = starFree
		}
	}
}

// dispStars draws the stars, brighter away from the middle.
func (g *Game) dispStars() {
	for i := range g.stars {
		s := &g.stars[i]
		if s.x == starFree {
			continue
		}
		x, y := int(s.x)>>frac, int(s.y)>>frac
		k1, k2 := x-width/2, y-height/2
		k := min((k1*k1+k2*k2)>>5, 255)
		g.DrawPoint(x, y, picopad.RGB(uint8(k), uint8(k), uint8(k)))
	}
}

// dispScore draws the score with the big digits.
func (g *Game) dispScore() {
	n := g.score >> frac
	x := digitW*(scoreDigs-1) + 4
	for i := 0; i < scoreDigs; i++ {
		d := n % 10
		n /= 10
		g.DrawBlitPal(digitImg, digitImgPal, d*digitW, 0, x, 0, digitW, digitH, digitW*10, picopad.ColMagenta)
		x -= digitW
	}
}

// addScore adds points (with frac); every scoreLive gives a life.
func (g *Game) addScore(add int) {
	g.score += add
	if g.score-g.lastScore >= scoreLive {
		g.lastScore += scoreLive
		g.lives++
		g.dispAll()
		// PLAYSOUND(LivesSnd)
		g.delay(2500)
	}
}

// dispFuel draws the fuel (and LOW FUEL).
func (g *Game) dispFuel() {
	x := (width - fuelMax - 2) / 2
	j := g.fuel >> frac
	bg, fg := picopad.ColBlack, picopad.ColGreen
	if g.subLevel == 3 {
		bg, fg = picopad.ColBlue, picopad.ColYellow
	}
	g.DrawRect(x, 34, fuelMax+2, 4, bg)
	g.DrawRect(x+1, 35, j, 2, fg)
	switch {
	case j >= 32:
	case j >= 12:
		g.DrawTextBg("LOW FUEL", (width-8*8)/2, 32, picopad.ColYellow, picopad.ColBlack)
	case (g.Time()>>18)&1 == 0: // blinking
		g.DrawTextBg("        ", (width-8*8)/2, 32, picopad.ColRed, picopad.ColBlack)
	default:
		g.DrawTextBg("LOW FUEL", (width-8*8)/2, 32, picopad.ColRed, picopad.ColBlack)
	}
}

// subFuel uses fuel; without it the ship explodes.
func (g *Game) subFuel(sub int) {
	g.fuel -= sub
	if g.fuel <= 0 {
		g.fuel = 0
		g.shipCrash()
	}
	// the sound of low fuel
}

func (g *Game) dispGoals() {
	for i := 0; i < g.goals; i++ {
		g.DrawBlitPal(goalImg, goalImgPal, 0, 0, i*goalW, 23, goalW, goalH, goalW, picopad.ColMagenta)
	}
}

func (g *Game) decGoals() {
	if g.goals > 0 {
		g.goals--
		g.goalTime = g.Time()
	}
}

func (g *Game) dispLives() {
	for i := 0; i < g.lives && i < 5; i++ {
		g.DrawBlitPal(liveImg, liveImgPal, 0, 0, 200+i*24, 10, liveW, liveH, liveW, picopad.ColMagenta)
	}
}
