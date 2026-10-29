package pictor

import "github.com/sat0ken/tinygo-cyd/picopad"

// The start of a scene (gamestart.cpp), the game menu (gamemenu.cpp) and
// the end of a game (gameend.cpp).

const (
	soundModeOff   = 0
	soundModeSound = 1
	soundModeMusic = 2

	openX      = 190
	topNameLen = 8
	topNum     = 8
)

// topEntry is an entry of the top list (sTop).
type topEntry struct {
	score int
	name  [topNameLen]byte
}

// dispGameStart displays the start of a scene: the dark scene, the actor,
// the sector, its name and its new helper.
func (g *Game) dispGameStart(prompt bool) {
	// the actor at the start
	g.actorX = g.actorMinX
	switch {
	case g.actor.walk && g.actInx == actJill:
		g.actorY = g.actorMaxY + 6
		g.actorPhase = jillStand
	case g.actor.walk:
		g.actorY = g.actorMaxY
		g.actorPhase = 0
	default:
		g.actorY = (g.actorMinY + g.actorMaxY) / 2
		g.shiftActor()
	}

	b := &g.back
	g.DrawImgPalDbl(b.img1, b.pal1, 0, 0, 0, bgY, width, bgHeight, bgWidth)
	g.DrawBlitPal(b.img2, b.pal2, 0, 0, 0, bg2Y+b.dy, width, bg2Height-b.dy, bg2Width, picopad.ColWhite)
	g.DrawBlitPal(b.img3, b.pal3, 0, 0, 0, bg3Y, width, bg3Height, bg3Width, picopad.ColWhite)
	g.dispHeader()
	g.dispFoot()
	g.dispShadow()
	g.dispActor()

	y := 60
	if g.level < bgNum {
		y = 20
	}
	n := itoa(g.level + 1)
	x := openX - (len(n)+16)*8/2
	g.DrawText("Entering Sector", x, y, picopad.ColWhite)
	g.DrawText(n, x+16*8, y, picopad.ColWhite)
	y += 16
	g.DrawText2(b.name, openX-len(b.name)*16/2, y, colAzure)

	// the new helper
	if g.level < bgNum {
		y += 32
		a := &actorTemps[g.backInx+1]
		y += (122 - (16 + 16 + a.h + 16)) / 2
		g.DrawText("New Helper", openX-10*8/2, y, picopad.ColWhite)
		y += 16
		g.DrawTextW(a.name, openX-len(a.name)*16/2, y, picopad.ColGreen)
		y += 16
		g.DrawBlitPal(a.img, a.pal, 0, 0, openX-a.w/2, y, a.w, a.h, a.wall, a.trans)
		y += a.h
		x = openX - (len(a.missile)+7)*8/2
		g.DrawText("Attack:", x, y, picopad.ColRed)
		g.DrawText(a.missile, x+7*8, y, picopad.ColRed)
	}

	if prompt {
		g.DrawText("Press B/X to transform", openX-22*8/2, 190, picopad.ColYellow)
		g.DrawText("Press A to start game", openX-21*8/2, 205, picopad.ColYellow)
	}
	g.DispUpdate()
}

// gameStart shows the start of a scene until A; it returns false to end
// the game.
func (g *Game) gameStart() bool {
	g.dispGameStart(false)
	g.WaitMs(1000)
	g.KeyFlush()
	for {
		g.dispGameStart(true)
		switch g.KeyGet() {
		case picopad.KeyA:
			for g.KeyPressed(picopad.KeyA) {
			}
			return true
		case picopad.KeyB:
			g.setActor(g.actInx + 1)
			g.actorReload = 0
		case picopad.KeyX:
			g.setActor(g.actInx - 1)
			g.actorReload = 0
		case picopad.KeyY:
			if !g.gameMenu() {
				return false
			}
			g.scenePalette()
		}
		g.WaitMs(45) // as on the VGA PicoPad (the actor's animation)
	}
}

// loadTop clears the top list (C: loads PICTOR.TOP from the SD card).
func (g *Game) loadTop() {
	for i := range g.top {
		g.top[i].score = 0
		for k := range g.top[i].name {
			g.top[i].name[k] = ' '
		}
	}
}

// dispTop displays the top list.
func (g *Game) dispTop() {
	y := 105
	g.DrawTextW("Top List", (width-8*16)/2, y, picopad.ColGreen)
	y += 15
	for i := range g.top {
		t := &g.top[i]
		g.DrawTextW(itoa(i+1)+": "+zeros(t.score, 7)+" "+string(t.name[:]), 0, y, picopad.ColWhite)
		y += 15
	}
}

// gameMenu is the game menu: Y continues (true), X ends (false), B
// changes the sound mode, A (C: a screenshot) continues.
func (g *Game) gameMenu() bool {
	// StopAllSound()
	g.KeyFlush()
	for {
		g.DrawClear()
		y := 0
		g.DrawText2("Game Menu", (width-9*16)/2, y, picopad.ColGreen)
		y += 32
		item := func(key, label string) {
			g.DrawTextW(key, 0, y, picopad.ColYellow)
			g.DrawTextW(label, 32, y, picopad.ColWhite)
			y += 15
		}
		item("Y", "... Continue")
		item("X", "... Exit")
		mode := [...]string{"off", "ON", "ALL"}[g.soundMode]
		g.DrawTextW(mode, 15*16, y, picopad.ColWhite)
		item("B", "... Sound is")
		item("A", "... Screenshot")
		g.dispTop()
		g.DispUpdate()

		key := g.KeyGet()
		for key == picopad.NoKey {
			key = g.KeyGet()
		}
		switch key {
		case picopad.KeyX:
			return false
		case picopad.KeyB:
			g.soundMode = (g.soundMode + 1) % 3
		case picopad.KeyA:
			for g.KeyPressed(picopad.KeyA) {
			}
			return true
		case picopad.KeyY:
			return true
		}
	}
}

// dispGameEnd displays the end of the game (with the name of a new max.
// score).
func (g *Game) dispGameEnd(prompt bool) {
	g.DrawClear()
	g.DrawText2("Game Over", (width-9*16)/2, 0, picopad.ColRed)
	g.dispTop()
	if g.score > g.maxScore {
		y := 35
		g.DrawTextW("New MaxScore "+zeros(g.score, 7), (width-20*16)/2, y, picopad.ColMagenta)
		if prompt {
			y += 15
			g.DrawText("Enter your name and press A to continue", (width-39*8)/2, y, picopad.ColYellow)
			y += 15
			g.DrawText2(string(g.playerName[:]), (width-8*16)/2, y, colAzure)
			if (g.Time()>>18)&1 == 0 {
				g.DrawText2("_", (width-8*16)/2+g.playerNameInx*16, y+4, colAzure)
			}
		}
	} else {
		g.DrawTextW("Your Score "+zeros(g.score, 7), (width-18*16)/2, 35, picopad.ColMagenta)
		if prompt {
			g.DrawText("Press any key...", (width-16*8)/2, 70, picopad.ColYellow)
		}
	}
	g.DispUpdate()
}

// gameEnd shows the end of the game; a new max. score enters the top list
// with a name (UP/DOWN the letter, LEFT/RIGHT the place, A).
func (g *Game) gameEnd() {
	g.playerName = g.top[0].name
	g.dispGameEnd(false)
	// StopAllSound()
	g.WaitMs(1000)
	g.KeyFlush()
	for {
		g.dispGameEnd(true)
		g.WaitMs(30)
		key := g.KeyGet()
		if g.score <= g.maxScore {
			if key != picopad.NoKey {
				return
			}
			continue
		}
		ch := &g.playerName[g.playerNameInx]
		switch key {
		case picopad.KeyLeft:
			if g.playerNameInx > 0 {
				g.playerNameInx--
			}
		case picopad.KeyRight:
			if g.playerNameInx < topNameLen-1 {
				g.playerNameInx++
			}
		case picopad.KeyUp: // space, 0..9, A..Z
			switch *ch++; *ch {
			case ' ' + 1:
				*ch = '0'
			case '9' + 1:
				*ch = 'A'
			case 'Z' + 1:
				*ch = ' '
			}
		case picopad.KeyDown:
			switch *ch--; *ch {
			case ' ' - 1:
				*ch = 'Z'
			case '0' - 1:
				*ch = ' '
			case 'A' - 1:
				*ch = '9'
			}
		case picopad.KeyA:
			copy(g.top[1:], g.top[:topNum-1])
			g.top[0] = topEntry{g.score, g.playerName}
			g.maxScore = g.score
			// SaveTop(): the SD card
			return
		case picopad.KeyY:
			return
		}
	}
}
