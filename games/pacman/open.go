package pacman

import "github.com/sat0ken/tinygo-cyd/picopad"

// The open screen (src/open.cpp).

var (
	openCharName = [4]string{"SHADOW", "SPEEDY", "BASHFUL", "POKEY"}
	openCharNick = [4]string{"\"BLINKY\"", "\"PINKY\"", "\"INKY\"", "\"CLYDE\""}
)

// dispWait updates the display and waits ms from the last time. Y sets
// quit (the C source resets to the boot loader): the open screen then
// ends without waiting or drawing.
func (g *Game) dispWait(ms uint32) {
	if g.quit {
		return
	}
	ms *= 1000
	g.DispUpdate()
	var t uint32
	for {
		t = g.Time()
		if g.KeyGet() == picopad.KeyY {
			g.quit = true
			return
		}
		if t-g.lastTime >= ms {
			break
		}
	}
	g.lastTime = t
}

// animPellet blinks the pellet at y1.
func (g *Game) animPellet(y1 int) {
	g.blink++
	if g.blink <= 3 {
		g.DrawRect(120, y1, tileW, tileH, picopad.ColBlack)
	} else {
		if g.blink == 6 {
			g.blink = 0
		}
		g.DrawImg4Pal(tilesImg, tilesImgPal, tilePellet*tileW, 0, 120, y1, tileW, tileH, tilesImgW)
	}
}

// sprite draws a sprite.
func (g *Game) sprite(n, x, y int) {
	g.DrawImg4Pal(spritesImg, spritesImgPal, n*spriteW, 0, x, y, spriteW, spriteH, spriteImgW)
}

// openScreen shows the characters and the points.
func (g *Game) openScreen() {
	// StopSound()
	g.DrawClear()
	g.SelFont8x8()

	// the logo
	y := 3
	g.DrawImg4Pal(logoImg, logoImgPal, 0, 0, (width-logoW)/2, y, logoW, logoH, logoW)
	y += logoH
	g.dispWait(200)
	g.DrawText("for Raspberry Pico", (width-18*8)/2, y, picopad.ColGreen)
	y += 25
	g.dispWait(400)

	// the title
	g.DrawText("CHARACTER / NICKNAME", 90, y, picopad.ColWhite)
	y += 20
	g.dispWait(400)

	// the characters
	cols := [4]picopad.Color{picopad.ColRed, picopad.RGB(255, 166, 195), picopad.RGB(0, 230, 255), picopad.RGB(255, 166, 0)}
	for i := 0; i < 4; i++ {
		g.sprite(spriteBlinky+4+i*8, 70, y-6)
		g.dispWait(400)
		g.DrawText(openCharName[i], 110, y, cols[i])
		g.dispWait(200)
		g.DrawText(openCharNick[i], 180, y, cols[i])
		g.dispWait(200)
		y += 20
	}
	g.dispWait(400)
	y += 35

	// the points of a dot and a pellet
	g.DrawImg4Pal(tilesImg, tilesImgPal, tileDot*tileW, 0, 120, y, tileW, tileH, tilesImgW)
	g.DrawText("10 pts", 140, y+2, picopad.ColWhite)
	y += 20
	y1 := y
	g.DrawImg4Pal(tilesImg, tilesImgPal, tilePellet*tileW, 0, 120, y1, tileW, tileH, tilesImgW)
	g.DrawText("50 pts", 140, y+2, picopad.ColWhite)
	y -= 40
	g.dispWait(400)

	// the pellet the pacman goes to
	y2 := y
	g.DrawImg4Pal(tilesImg, tilesImgPal, tilePellet*tileW, 0, 73, y2, tileW, tileH, tilesImgW)
	g.dispWait(400)

	x := 320
	phase := 0
	g.blink = 0
	// PLAYSOUND(IntermissionSnd)

	// the pacman and the ghosts go left
	for {
		x -= charSpeed
		g.animPellet(y1)
		if g.blink <= 3 {
			g.DrawRect(73, y2, tileW, tileH, picopad.ColBlack)
		} else {
			g.DrawImg4Pal(tilesImg, tilesImgPal, tilePellet*tileW, 0, 73, y2, tileW, tileH, tilesImgW)
		}
		g.sprite(spritePacL+phase, x, y-6)
		g.sprite(spriteBlinky+phase, x+spriteW, y-6)
		g.sprite(spritePinky+phase, x+2*spriteW, y-6)
		g.sprite(spriteInky+phase, x+3*spriteW, y-6)
		g.sprite(spriteClyde+phase, x+4*spriteW, y-6)
		g.DrawRect(x+5*spriteW, y-6, spriteW, spriteH, picopad.ColBlack)
		g.dispWait(speed)
		phase = (phase + 1) & 1
		if x <= 70 {
			break
		}
	}

	// the pellets are eaten
	g.DrawRect(120, y1, tileW, tileH, picopad.ColBlack)
	g.DrawRect(73, y2, tileW, tileH, picopad.ColBlack)

	// the pacman eats the frightened ghosts going right
	x2 := x
	var eat [4]bool
	for {
		x += charSpeed
		x2 += frighSpeed
		g.animPellet(y1)

		// eat a ghost: its points instead
		wait := false
		g.DrawRect(x-spriteW, y-6, 2*spriteW, spriteH, picopad.ColBlack)
		for k := 0; k < 4; k++ {
			if x2 < x-spriteW/2-k*spriteW && !eat[k] {
				g.sprite(sprite200+k, x2+(k+1)*spriteW, y-6)
				wait = true
				eat[k] = true
				break
			}
		}

		// the ghosts and the pacman
		for k := 0; k < 4; k++ {
			if !eat[k] {
				g.sprite(spriteFrigh+phase, x2+(k+1)*spriteW, y-6)
			}
		}
		if !wait {
			g.sprite(spritePacR+phase, x, y-6)
		}

		g.dispWait(speed)
		if wait {
			for i := 1000 / speed; i > 0; i-- {
				g.animPellet(y1)
				g.dispWait(speed)
			}
		}
		phase = (phase + 1) & 1
		if x >= width {
			break
		}
	}
	// while (PlayingSound()) { AnimPellet(y1); DispWait(SPEED); }
}
