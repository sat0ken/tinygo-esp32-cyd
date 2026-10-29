// Package ghostrac is the Ghost Racing of PicoLibSDK
// (PicoPad/GAME/GHOSTRAC, by Miroslav Nemecek, Unlicense), ported from C
// to package picopad. It follows the C source (src/main.cpp); the C
// globals are the fields of Game.
//
// A car race on a track seen in perspective (DrawTile4PalMap with a
// Mat2D): UP or X speeds up, DOWN or A brakes, LEFT/RIGHT turn, Y ends Run
// (on the PicoPad: back to the boot loader). A lap starts at the start
// line and passes the three check points; the ghost drives the best lap
// again. There is no engine sound yet.
package ghostrac

import (
	"math"

	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/picopad"
)

//go:generate sh -c "go run ../../tools/picoimg -pkg ghostrac -root $PICOLIBSDK -o img.go PicoPad/GAME/GHOSTRAC/img/tiles_pal.cpp PicoPad/GAME/GHOSTRAC/img/skyline.cpp $(for i in $(seq 0 16); do echo PicoPad/GAME/GHOSTRAC/img/car$i.cpp; done) $(for i in $(seq 0 7); do echo PicoPad/GAME/GHOSTRAC/img/ghost$i.cpp; done)"

const (
	width  = picopad.Width
	height = picopad.Height

	infoY    = 0  // Y coordinate of the info row
	infoH    = 16 // height of the info row
	skylineY = infoH
	skylineW = 320
	skylineH = 29
	horizonH = skylineH

	tileBits = 7             // bits of the tile size
	tileSize = 1 << tileBits // tile size (128)
	mapY     = skylineY + skylineH
	mapWBits = 5
	mapW     = 1 << mapWBits // 32
	mapHBits = 5
	mapH     = 1 << mapHBits // 32
	mapSize  = mapW * mapH
	horiz    = 30 // terrain horizon offset

	carImgW   = 160
	carImgH   = 96
	carImgX   = (width - carImgW) / 2
	carImgY   = 100 + skylineY + skylineH
	carImgNum = 17
	carTrans  = picopad.Color(0x630C) // transparent colour of the cars

	ghostW = 128
	ghostH = 64

	grass    = 0 // the tiles: grass, start, the check points
	start    = 1
	checkA   = 2
	checkB   = 3
	checkC   = 4
	checkNum = 3

	speedCoef = float32(tileSize) / 15
	accel     = speedCoef * 5
	accelSlow = accel * 0.2  // slowing down by itself
	grassSlow = accel * 0.01 // slowing down on the grass
	turnAccel = 0.02
	autoTurn  = 0.01
	turnMax   = 0.02
	speedMax  = 400
	recordMax = 1000

	pi  = math.Pi
	pi2 = 2 * math.Pi

	timeTxt = "TIME:"
	lastTxt = "LAST:"
	bestTxt = "BEST:"
)

// tileMap is the track:
//
//	0 grass, 1 start, 2 3 4 the check points (blue, yellow, red),
//	5 | straight, 6 -- straight, 7..22 the curves (doc/tiles_num.bmp)
const tileMap = "" +
	"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" + // 0
	"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" + // 1
	"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" + // 2
	"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" + // 3
	"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" + // 4
	"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" + // 5
	"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" + // 6
	"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" + // 7
	"\x00\x00\x00\x00\x00\x00\x00\x00\x07\x08\x06\x06\x06\x06\x06\x06\x06\x06\x06\x06\x06\x06\x0b\x0c\x00\x00\x00\x00\x00\x00\x00\x00" + // 8
	"\x00\x00\x00\x00\x00\x00\x00\x00\x09\x0a\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x0d\x0e\x00\x00\x00\x00\x00\x00\x00\x00" + // 9
	"\x00\x00\x00\x00\x00\x00\x00\x00\x05\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x05\x00\x00\x00\x00\x00\x00\x00\x00" + // 10
	"\x00\x00\x00\x00\x00\x00\x00\x00\x05\x00\x00\x07\x08\x06\x06\x06\x0b\x0c\x00\x00\x00\x00\x00\x02\x00\x00\x00\x00\x00\x00\x00\x00" + // 11
	"\x00\x00\x00\x00\x00\x00\x00\x00\x05\x00\x00\x09\x0a\x00\x00\x00\x0d\x0e\x00\x00\x00\x00\x00\x05\x00\x00\x00\x00\x00\x00\x00\x00" + // 12
	"\x00\x00\x00\x00\x00\x00\x00\x00\x05\x00\x00\x05\x00\x00\x00\x00\x00\x0f\x10\x00\x00\x00\x13\x14\x00\x00\x00\x00\x00\x00\x00\x00" + // 13
	"\x00\x00\x00\x00\x00\x00\x00\x00\x01\x00\x00\x03\x00\x00\x00\x00\x00\x11\x12\x06\x06\x06\x15\x16\x00\x00\x00\x00\x00\x00\x00\x00" + // 14
	"\x00\x00\x00\x00\x00\x00\x00\x00\x05\x00\x00\x05\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" + // 15
	"\x00\x00\x00\x00\x00\x00\x00\x00\x05\x00\x00\x0f\x10\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" + // 16
	"\x00\x00\x00\x00\x00\x00\x00\x00\x05\x00\x00\x11\x12\x06\x06\x06\x06\x06\x06\x06\x06\x0b\x0c\x00\x00\x00\x00\x00\x00\x00\x00\x00" + // 17
	"\x00\x00\x00\x00\x00\x00\x00\x00\x05\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x0d\x0e\x00\x00\x00\x00\x00\x00\x00\x00\x00" + // 18
	"\x00\x00\x00\x00\x00\x00\x00\x00\x0f\x10\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x05\x00\x00\x00\x00\x00\x00\x00\x00\x00" + // 19
	"\x00\x00\x00\x00\x00\x00\x00\x00\x11\x12\x06\x06\x06\x06\x06\x06\x06\x0b\x0c\x00\x00\x00\x04\x00\x00\x00\x00\x00\x00\x00\x00\x00" + // 20
	"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x0d\x0e\x00\x00\x00\x05\x00\x00\x00\x00\x00\x00\x00\x00\x00" + // 21
	"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x0f\x10\x00\x13\x14\x00\x00\x00\x00\x00\x00\x00\x00\x00" + // 22
	"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x11\x12\x06\x15\x16\x00\x00\x00\x00\x00\x00\x00\x00\x00" + // 23
	"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" + // 24
	"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" + // 25
	"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" + // 26
	"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" + // 27
	"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" + // 28
	"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" + // 29
	"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" + // 30
	"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" // 31

// the images of the car (turning from left to right) and the ghost
// (seen from 8 directions)
var (
	carImg      = [carImgNum]string{car0Img, car1Img, car2Img, car3Img, car4Img, car5Img, car6Img, car7Img, car8Img, car9Img, car10Img, car11Img, car12Img, car13Img, car14Img, car15Img, car16Img}
	carImgPal   = [carImgNum]picopad.Palette{car0ImgPal, car1ImgPal, car2ImgPal, car3ImgPal, car4ImgPal, car5ImgPal, car6ImgPal, car7ImgPal, car8ImgPal, car9ImgPal, car10ImgPal, car11ImgPal, car12ImgPal, car13ImgPal, car14ImgPal, car15ImgPal, car16ImgPal}
	ghostImg    = [8]string{ghost0Img, ghost1Img, ghost2Img, ghost3Img, ghost4Img, ghost5Img, ghost6Img, ghost7Img}
	ghostImgPal = [8]picopad.Palette{ghost0ImgPal, ghost1ImgPal, ghost2ImgPal, ghost3ImgPal, ghost4ImgPal, ghost5ImgPal, ghost6ImgPal, ghost7ImgPal}
	checkCol    = [checkNum]picopad.Color{picopad.ColCyan, picopad.ColYellow, picopad.ColRed}
)

// Game is Ghost Racing on a console.
type Game struct {
	*picopad.Console

	speed   float32 // current speed
	carX    float32
	carY    float32
	carDir  float32 // direction (radians)
	carTurn float32
	curTile int // the tile under the car

	startTime uint32 // start of the lap
	thisTime  int    // this lap [s]
	lastTime  int    // last lap
	bestTime  int    // best lap
	newBest   bool
	checkNext int // the next check point (grass: the race has not started)
	checkOK   [checkNum]bool

	// this lap: a record every second
	recordStep uint32
	recordInx  int
	recordX    [recordMax]float32
	recordY    [recordMax]float32
	recordDir  [recordMax]float32

	// the best lap: the ghost
	recordBestNum int
	recordBestX   [recordMax]float32
	recordBestY   [recordMax]float32
	recordBestDir [recordMax]float32

	mat  picopad.Mat2D // the terrain
	mat2 picopad.Mat2D // the ghost
}

// New returns the game on d (at least 320x240) with the touch panel t.
// Set Pad to use a game controller.
func New(d hal.Display, t hal.Touch) *Game {
	return &Game{Console: picopad.New(d, t, 0)}
}

// infoTime displays a time m:ss.
func (g *Game) infoTime(x, t int, col picopad.Color) {
	t = min(t, 99*60+59)
	var buf [5]byte
	n := 0
	if m := t / 60; m >= 10 {
		buf[n] = byte('0' + m/10)
		n++
		buf[n] = byte('0' + m%10)
		n++
	} else {
		buf[n] = byte('0' + m)
		n++
	}
	buf[n] = ':'
	buf[n+1] = byte('0' + t%60/10)
	buf[n+2] = byte('0' + t%10)
	for i := 0; i < n+3; i++ {
		g.DrawChar(buf[i], x+i*8, infoY, col)
	}
}

// infoRow displays the times and the check points.
func (g *Game) infoRow() {
	g.DrawRect(0, infoY, width, infoH, picopad.ColBlack)
	x := 0
	g.DrawText(timeTxt, x, infoY, picopad.ColYellow)
	x += 5 * 8
	g.infoTime(x, g.thisTime, picopad.ColWhite)
	x += 7 * 8
	col := picopad.ColWhite
	if g.newBest {
		col = picopad.ColGreen
	}
	g.DrawText(lastTxt, x, infoY, picopad.ColYellow)
	x += 5 * 8
	g.infoTime(x, g.lastTime, col)
	x += 7 * 8
	g.DrawText(bestTxt, x, infoY, picopad.ColYellow)
	x += 5 * 8
	g.infoTime(x, g.bestTime, col)
	x += 7 * 8
	for i := 0; i < checkNum; i++ {
		ch := byte(2) // the empty and the full face of the font
		if g.checkOK[i] {
			ch = 3
		}
		g.DrawChar(ch, x, infoY, checkCol[i])
		x += 8
	}
}

// reset is the state at the start of the C program.
func (g *Game) reset() {
	g.speed = 0
	g.carX = 8.5 * tileSize
	g.carY = 15.5 * tileSize
	g.carDir, g.carTurn = 0, 0
	g.curTile = 0
	g.thisTime, g.lastTime, g.bestTime = 0, 0, 0
	g.newBest = false
	g.checkNext = grass
	g.checkOK = [checkNum]bool{}
	g.recordInx = 0
	g.recordBestNum = 0
}

// Run plays until Y (main of the C source, which then resets to the boot
// loader).
func (g *Game) Run() {
	g.reset()

	// the skyline has 256 colours: the terrain, the cars and the ghosts
	// get their exact colours first (the frame buffer has 256)
	g.PrepPalette(tilesPalImgPal)
	for i := range carImgPal {
		g.PrepPalette(carImgPal[i])
	}
	for i := range ghostImgPal {
		g.PrepPalette(ghostImgPal[i])
	}

	// the ghost of the last frame (C: variables of main)
	var a float32
	var wd, hd, x0, y0 int
	var img string
	var pal picopad.Palette

	t0 := int32(g.Time())
	for {
		// the time since the last frame [s]
		t2 := int32(g.Time())
		dt := float32(t2-t0) / 1000000
		t0 = t2

		if g.KeyPressed(picopad.KeyX) || g.KeyPressed(picopad.KeyUp) {
			g.speed += accel * dt
		}
		if g.KeyPressed(picopad.KeyA) || g.KeyPressed(picopad.KeyDown) {
			g.speed -= accel * dt * 2
		}
		if g.KeyGet() == picopad.KeyY {
			return
		}

		g.speed -= accelSlow * dt
		if tileMap[g.curTile] == grass {
			g.speed -= grassSlow * dt * g.speed
		}
		g.speed = min(max(g.speed, 0), speedMax)

		// turning
		if g.speed > 0 {
			if g.KeyPressed(picopad.KeyLeft) {
				g.carTurn -= turnAccel * dt
			}
			if g.KeyPressed(picopad.KeyRight) {
				g.carTurn += turnAccel * dt
			}
			if g.carTurn > 0 {
				g.carTurn = max(g.carTurn-autoTurn*dt, 0)
			}
			if g.carTurn < 0 {
				g.carTurn = min(g.carTurn+autoTurn*dt, 0)
			}
		}
		g.carTurn = min(max(g.carTurn, -turnMax), turnMax)

		g.carDir += g.carTurn * dt * g.speed
		if g.carDir < 0 {
			g.carDir = float32(float64(g.carDir) + pi2)
		}
		if g.carDir >= pi2 {
			g.carDir = float32(float64(g.carDir) - pi2)
		}
		// if (Speed == 0) StopSound(); else PLAYSOUNDREP(EngineSnd), SpeedSound

		// move
		s := float32(math.Sin(float64(g.carDir)))
		c := float32(math.Cos(float64(g.carDir)))
		g.carY += -g.speed * dt * c
		g.carX += g.speed * dt * s

		xi := int((g.carX+1024*tileSize)/tileSize) & (mapW - 1)
		yi := int((g.carY+1024*tileSize)/tileSize) & (mapH - 1)
		g.curTile = xi + yi*mapW

		// the skyline and the terrain
		offx := int(float64(g.carDir*2*skylineW) / pi)
		for offx >= skylineW {
			offx -= skylineW
		}
		g.mat.PrepDrawImg(tileSize, tileSize, int(g.carX), int(g.carY), width, width, 0, 0, g.carDir, 0, tileSize)

		// the ghost: where the best lap was this time after its start
		dif := int32(g.Time() - g.recordStep)
		ghost := g.recordBestNum > 0 && g.recordInx+1 < g.recordBestNum
		if ghost {
			i := g.recordInx
			x := (g.recordBestX[i+1]-g.recordBestX[i])*float32(dif)/1000000 + g.recordBestX[i] - g.carX
			y := (g.recordBestY[i+1]-g.recordBestY[i])*float32(dif)/1000000 + g.recordBestY[i] - g.carY
			d := g.recordBestDir[i]

			a = float32(math.Atan2(float64(y), float64(x))) // the angle of the ghost
			a0 := a
			k := float32(math.Sqrt(float64(y*y + x*x))) // the distance

			// the view angle
			a = float32(float64(a) - float64(g.carDir) + pi/2)
			for a < -pi {
				a = float32(float64(a) + pi2)
			}
			for a > pi {
				a = float32(float64(a) - pi2)
			}
			if a >= -pi/3 && a <= pi/3 {
				// the size of the ghost
				s := 200 / (k + 50)
				wd = int(ghostW * s)
				hd = int(ghostH * s)
				if wd < 30 || wd > 160 {
					wd = 0
				}

				// the direction of the ghost
				d = float32(float64(d) - (float64(a0) - pi/2))
				d = float32(float64(d) + pi/8) // rounding
				for d < -pi {
					d = float32(float64(d) + pi2)
				}
				for d > pi {
					d = float32(float64(d) - pi2)
				}
				i := min(max(int((float64(d)+pi)*8/pi2), 0), 7)
				img, pal = ghostImg[i], ghostImgPal[i]

				// on the screen
				x0 = int((math.Tan(float64(a))+1)*width/2) - wd/2
				y0 = infoH + skylineH - 20 + hd/2
			}
		}

		// the car turning
		turn := int(float64((g.carTurn+turnMax)/(2*turnMax)*carImgNum) - 0.5)
		turn = min(max(turn, 0), carImgNum-1)

		// display
		g.infoRow()
		g.DrawImgPal(skylineImg, skylineImgPal, 0, 0, -offx, skylineY, skylineW, skylineH, skylineW)
		g.DrawImgPal(skylineImg, skylineImgPal, 0, 0, width-offx, skylineY, skylineW, skylineH, skylineW)
		g.DrawTile4PalMap(tilesPalImg, tilesPalImgPal, tileMap, mapWBits, mapHBits, tileBits,
			0, mapY, width, height-horizonH, &g.mat, horiz)
		if ghost && a >= -pi/3 && a <= pi/3 && wd > 0 && hd > 0 &&
			x0 >= -wd && x0 < width && y0 >= -hd && y0 < height {
			g.mat2.PrepDrawImg(ghostW, ghostH, ghostW/2, ghostH/2, wd, hd, 0, 0, 0, 0, 0)
			g.DrawImg4PalMat(img, pal, ghostW, ghostH, x0, y0, wd, hd, &g.mat2, picopad.DrawImgTransp, carTrans)
		}
		g.DrawBlit4Pal(carImg[turn], carImgPal[turn], 0, 0, carImgX, carImgY, carImgW, carImgH, carImgW, carTrans)
		g.DispUpdate()

		// the time of the lap
		if g.checkNext != grass {
			g.thisTime = int((g.Time() - g.startTime) / 1000000)
		}

		// the start and the check points
		tile := int(tileMap[g.curTile])
		if tile == g.checkNext || (tile == start && g.checkNext == grass) {
			switch tile {
			case start:
				if g.checkNext != grass { // a lap
					g.lastTime = g.thisTime
					g.newBest = false
					if g.lastTime < g.bestTime || g.bestTime == 0 {
						g.bestTime = g.lastTime
						g.newBest = true
						g.recordBestNum = g.recordInx
						copy(g.recordBestX[:], g.recordX[:g.recordInx])
						copy(g.recordBestY[:], g.recordY[:g.recordInx])
						copy(g.recordBestDir[:], g.recordDir[:g.recordInx])
					}
				}
				g.startTime = g.Time()
				g.checkNext = checkA
				g.checkOK = [checkNum]bool{}
				g.recordInx = 0
				g.recordStep = g.Time()
			case checkA:
				g.checkOK[0] = true
				g.checkNext = checkB
			case checkB:
				g.checkOK[1] = true
				g.checkNext = checkC
			case checkC:
				g.checkOK[2] = true
				g.checkNext = start
			}
		}

		// record the lap every second
		if g.checkNext != grass && dif >= 1000000 {
			g.recordStep += 1000000
			if i := g.recordInx; i < recordMax {
				g.recordX[i] = g.carX
				g.recordY[i] = g.carY
				g.recordDir[i] = g.carDir
				g.recordInx = i + 1
			}
		}
		g.DispUpdate()
	}
}
