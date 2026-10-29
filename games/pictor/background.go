package pictor

import "github.com/sat0ken/tinygo-cyd/picopad"

// The backgrounds (background.cpp), the header (header.cpp), the foot
// (foot.cpp) and the random numbers of the enemies (random.cpp).

// backTemp is a background: three layers scrolling at their speeds
// (sBackTemp). Layer 1 is drawn twice as large, layers 2 and 3 have the
// transparent colour white.
type backTemp struct {
	name string
	pal1 picopad.Palette
	img1 string
	pal2 picopad.Palette
	img2 string
	dy   int // Y correction of layer 2
	pal3 picopad.Palette
	img3 string
}

// backTemps are the backgrounds of the scenes (the heights of layer 2
// give dy).
var backTemps = [bgNum]backTemp{
	{"Meadow", meadowImgPal, meadowImg, meadow2ImgPal, meadow2Img, bg2Height - 64, meadow3ImgPal, meadow3Img},
	{"Jungle", jungleImgPal, jungleImg, jungle2ImgPal, jungle2Img, bg2Height - 38, jungle3ImgPal, meadow3Img},
	{"Sandy Beach", sandbeachImgPal, sandbeachImg, sandbeach2ImgPal, sandbeach2Img, bg2Height - 40, sandbeach3ImgPal, meadow3Img},
	{"Underwater", underwaterImgPal, underwaterImg, underwater2ImgPal, underwater2Img, bg2Height - 80, underwater3ImgPal, meadow3Img},
	{"Ice Land", icelandImgPal, icelandImg, iceland2ImgPal, iceland2Img, bg2Height - 38, iceland3ImgPal, meadow3Img},
	{"Fiery Land", firelandImgPal, firelandImg, fireland2ImgPal, fireland2Img, bg2Height - 48, fireland3ImgPal, meadow3Img},
	{"Haunted Hill", hauntedImgPal, hauntedImg, haunted2ImgPal, haunted2Img, bg2Height - 64, haunted3ImgPal, meadow3Img},
	{"Spacecraft", spacecraftImgPal, spacecraftImg, spacecraft2ImgPal, spacecraft2Img, bg2Height - 56, spacecraft3ImgPal, spacecraft3Img},
	{"Galaxy", galaxyImgPal, galaxyImg, galaxy2ImgPal, galaxy2Img, bg2Height - 66, galaxy3ImgPal, meadow3Img},
	{"Alien Planet", planetImgPal, planetImg, planet2ImgPal, planet2Img, bg2Height - 47, planet3ImgPal, meadow3Img},
	{"Candy Land", candylandImgPal, candylandImg, candyland2ImgPal, candyland2Img, bg2Height - 35, candyland3ImgPal, meadow3Img},
	{"Surreal Land", surrealImgPal, surrealImg, surreal2ImgPal, surreal2Img, bg2Height - 58, surreal3ImgPal, meadow3Img},
}

// initBack prepares the background of the scene and its palette.
func (g *Game) initBack() {
	g.backInx = g.level % bgNum
	g.back = backTemps[g.backInx]
	g.backPhase1, g.backPhase2, g.backPhase3 = 0, 0, 0
	g.scenePalette()
}

// dispLayer draws a scrolling layer: from phase, and from 0 again after
// the end of the image.
func (g *Game) dispLayer(img string, pal picopad.Palette, phase, y, h, ws int) {
	if phase < ws-width {
		g.DrawBlitPal(img, pal, phase, 0, 0, y, width, h, ws, picopad.ColWhite)
		return
	}
	g.DrawBlitPal(img, pal, phase, 0, 0, y, ws-phase, h, ws, picopad.ColWhite)
	g.DrawBlitPal(img, pal, 0, 0, ws-phase, y, width-(ws-phase), h, ws, picopad.ColWhite)
}

// dispBack draws the backgrounds.
func (g *Game) dispBack() {
	b := &g.back
	if p := g.backPhase1; p < bgWidth-width {
		g.DrawImgPalDbl(b.img1, b.pal1, p, 0, 0, bgY, width, bgHeight, bgWidth)
	} else {
		g.DrawImgPalDbl(b.img1, b.pal1, p, 0, 0, bgY, bgWidth-p, bgHeight, bgWidth)
		g.DrawImgPalDbl(b.img1, b.pal1, 0, 0, bgWidth-p, bgY, width-(bgWidth-p), bgHeight, bgWidth)
	}
	g.dispLayer(b.img2, b.pal2, g.backPhase2, bg2Y+b.dy, bg2Height-b.dy, bg2Width)
	g.dispLayer(b.img3, b.pal3, g.backPhase3, bg3Y, bg3Height, bg3Width)
}

// shiftBack scrolls the backgrounds.
func (g *Game) shiftBack() {
	g.backPhase1 = (g.backPhase1 + bgSpeed1) % bgWidth
	g.backPhase2 = (g.backPhase2 + bgSpeed2) % bg2Width
	g.backPhase3 = (g.backPhase3 + bgSpeed3) % bg3Width
}

// dispShadow makes the screen dark grey (the start of a scene).
func (g *Game) dispShadow() {
	g.MapRect(0, 0, width, height, func(c picopad.Color) picopad.Color {
		k := (c&0x1f + c>>5&0x3f + c>>11) >> 3
		return k | k<<6 | k<<11
	})
}

// drawNum draws n with digits digits (leading zeros) by characters (a
// string would allocate in every frame).
func (g *Game) drawNum(n, digits, x, y int, col, bg picopad.Color) {
	for i := digits - 1; i >= 0; i-- {
		g.DrawCharBg(byte('0'+n%10), x+i*8, y, col, bg)
		n /= 10
	}
}

// dispScore displays the score (green above the max. score).
func (g *Game) dispScore() {
	col := picopad.ColWhite
	if g.score > g.maxScore {
		col = picopad.ColGreen
	}
	g.drawNum(g.score, 7, 6*8+2, titY+2, col, headerCol)
}

// dispMaxScore displays the max. score.
func (g *Game) dispMaxScore() {
	col, n := picopad.ColWhite, g.maxScore
	if g.score > g.maxScore {
		col, n = picopad.ColGreen, g.score
	}
	g.drawNum(n, 7, width-7*8-2, titY+2, col, headerCol)
}

// dispPico displays the parts of the PicoPad collected, or the whole one.
func (g *Game) dispPico() {
	if g.bonusMask == bonusMaskAll {
		x := (width - picopadW - (bonusBigScoreLen+1)*8) / 2
		g.DrawBlit4Pal(picopadImg, picopadImgPal, 0, 0, x, titY+(titHeight-picopadH)/2,
			picopadW, picopadH, picopadW, picopad.ColMagenta)
		g.DrawText(bonusBigScoreText, x+picopadW+8, titY+2, picopad.ColYellow)
		return
	}
	x := 6*8 + 2 + 7*8 + 2 + 15
	for i := bonusColFirst; i <= bonusColLast; i++ {
		if g.bonusMask&(1<<i) != 0 {
			b := &bonusTemps[i]
			g.DrawBlit4Pal(b.img, b.pal, 0, 0, x, titY+(titHeight-b.h)/2, b.w, b.h, b.wall, b.trans)
			x += b.w + 2
		}
	}
}

// dispHeader displays the header.
func (g *Game) dispHeader() {
	g.DrawRect(1, titY+1, width-2, titHeight-2, headerCol)
	g.DrawFrame(0, titY, width, titHeight, headerFrame)
	g.DrawText("SCORE", 2, titY+2, picopad.ColYellow)
	g.DrawText("MAX", width-11*8-2, titY+2, picopad.ColYellow)
	g.dispScore()
	g.dispMaxScore()
	g.dispPico()
}

const (
	lifeX = 5*8 + 2 + 1
	lifeY = footY + 2 + 1
	lifeW = 70
	lifeH = footHeight - 4 - 2
	loadX = 164
	loadY = lifeY
	loadW = 30
	loadH = lifeH
	timeY = lifeY
	timeW = 70
	timeH = lifeH
	timeX = width - 2 - timeW - 2
)

// lifeCol are the colours of the hundreds of life.
var lifeCol = [...]picopad.Color{picopad.ColBlack, picopad.ColGreen, picopad.ColCyan, picopad.ColYellow,
	picopad.ColWhite, picopad.ColMagenta, picopad.ColMagenta}

// dispLife displays the life: a bar a hundred, in colours.
func (g *Game) dispLife() {
	life := min(g.life, lifeMax)
	segm := life / 100
	life -= segm * 100
	life = min((life*lifeW+50)/100, lifeW)
	g.DrawRect(lifeX, lifeY, life, lifeH, lifeCol[segm+1])
	g.DrawRect(lifeX+life, lifeY, lifeW-life, lifeH, lifeCol[segm])
}

// dispReload displays the loading of the weapon.
func (g *Game) dispReload() {
	reload := missileTemps[g.actInx].reload
	load := reload - min(g.actorReload, reload)
	load = min((load*loadW+reload/2)/reload, loadW)
	col := picopad.ColRed
	if g.actorReload == 0 {
		col = picopad.ColGreen
	}
	g.DrawRect(loadX, loadY, load, loadH, col)
	g.DrawRect(loadX+load, loadY, loadW-load, loadH, picopad.ColBlack)
}

// dispTime displays the time of the scene.
func (g *Game) dispTime() {
	frame := min((g.currentFrame*timeW+g.maxFrame/2)/g.maxFrame, timeW)
	g.DrawRect(timeX, timeY, frame, timeH, picopad.ColGreen)
	g.DrawRect(timeX+frame, timeY, timeW-frame, timeH, picopad.ColBlack)
}

// dispFoot displays the foot.
func (g *Game) dispFoot() {
	g.DrawRect(1, footY+1, width-2, footHeight-2, headerCol)
	g.DrawFrame(0, footY, width, footHeight, headerFrame)
	g.DrawText("LIFE", 2, footY+2, picopad.ColYellow)
	g.DrawText("LOAD", loadX-5*8, footY+2, picopad.ColYellow)
	g.DrawText("TIME", timeX-5*8, footY+2, picopad.ColYellow)
	g.DrawFrame(lifeX-1, lifeY-1, lifeW+2, lifeH+2, picopad.ColWhite)
	g.DrawFrame(loadX-1, loadY-1, loadW+2, loadH+2, picopad.ColWhite)
	g.DrawFrame(timeX-1, timeY-1, timeW+2, timeH+2, picopad.ColWhite)
	g.dispLife()
	g.dispReload()
	g.dispTime()
}

// enemyRand is the random number generator of the enemies, the same in
// each play of a scene.
func (g *Game) enemyRand() uint16 {
	g.enemyRandSeed = g.enemyRandSeed*214013 + 2531011
	return uint16(g.enemyRandSeed >> 16)
}

// enemyRandMax returns a random number from 0 to max.
func (g *Game) enemyRandMax(max uint16) uint16 {
	if max == 0 {
		return 0
	}
	m := max
	m |= m >> 1
	m |= m >> 2
	m |= m >> 4
	m |= m >> 8
	for {
		if r := g.enemyRand() & m; r <= max {
			return r
		}
	}
}

// enemyRandMinMax returns a random number from min to max.
func (g *Game) enemyRandMinMax(min, max uint16) uint16 { return g.enemyRandMax(max-min) + min }
