package ants

import (
	"math"
	"strconv"

	"github.com/sat0ken/tinygo-cyd/picopad"
)

// The game engine (src/game.cpp) without the USB multiplayer.

const (
	transCol  = picopad.Color(0xA158) // transparent color
	castleMax = 100                   // castle and fence max. height
	cardNum   = 8                     // cards of one player
	cardTypes = 30                    // card types
	cardBack  = cardTypes             // card background
	cardDisc  = cardTypes + 2         // card discard
	midX1     = width/2 - cardW       // middle card 1
	midX2     = width / 2             // middle card 2
	animSteps = 16                    // steps of the card animation

	soundOff  = 0
	soundSome = 1
	soundAll  = 2

	// the parameters of a player (the order of the status)
	parBuilders = 0
	parBricks   = 1
	parSoldiers = 2
	parWeapons  = 3
	parWizards  = 4
	parCrystals = 5
	parCastle   = 6
	parFence    = 7
	parNum      = 8

	playerHuman = 0
	playerComp  = 1
)

var (
	castleX = [2]int{width/2 - 70, width/2 + 70} // castle middle X
	fenceX  = [2]int{width/2 - 20, width/2 + 20} // fence middle X
	stateX  = [2]int{0, width - stateW}          // state X

	// the colors of the parameter values
	parFgCol = [parNum]picopad.Color{picopad.ColYellow, picopad.ColWhite, picopad.ColYellow, picopad.ColWhite,
		picopad.ColYellow, picopad.ColWhite, picopad.ColWhite, picopad.ColWhite}

	// the cards: type (what it costs), costs, the parameter it changes
	// (parNum: extra), the change (+ own, - enemy)
	cardTab = [cardTypes]card{
		// bricks
		{parBricks, 1, parFence, +3},    // 0: wall
		{parBricks, 1, parCastle, +2},   // 1: base
		{parBricks, 3, parFence, +6},    // 2: defense
		{parBricks, 3, parNum, 0},       // 3: reserve (castle +8, fence -4)
		{parBricks, 5, parCastle, +5},   // 4: tower
		{parBricks, 8, parBuilders, +1}, // 5: school
		{parBricks, 10, parNum, 0},      // 6: wain (castle +8, enemy castle -4)
		{parBricks, 12, parFence, +22},  // 7: fence
		{parBricks, 18, parCastle, +20}, // 8: fort
		{parBricks, 39, parCastle, +32}, // 9: Babylon

		// weapons
		{parWeapons, 1, parFence, -2},    // 10: archer
		{parWeapons, 2, parFence, -3},    // 11: knight
		{parWeapons, 2, parFence, -4},    // 12: rider
		{parWeapons, 4, parFence, -6},    // 13: platoon
		{parWeapons, 8, parSoldiers, +1}, // 14: recruit
		{parWeapons, 10, parFence, -12},  // 15: attack
		{parWeapons, 12, parNum, 0},      // 16: saboteur (enemy stocks -4)
		{parWeapons, 15, parNum, 0},      // 17: thief (transfer enemy stocks 5)
		{parWeapons, 18, parCastle, -10}, // 18: swat
		{parWeapons, 28, parFence, -32},  // 19: banshee

		// crystals
		{parCrystals, 4, parBricks, +8},   // 20: conjure bricks
		{parCrystals, 4, parBricks, -8},   // 21: crush bricks
		{parCrystals, 4, parWeapons, +8},  // 22: conjure weapons
		{parCrystals, 4, parWeapons, -8},  // 23: crush weapons
		{parCrystals, 4, parCrystals, +8}, // 24: conjure crystals
		{parCrystals, 4, parCrystals, -8}, // 25: crush crystals
		{parCrystals, 8, parWizards, +1},  // 26: sorcerer
		{parCrystals, 21, parFence, -25},  // 27: dragon
		{parCrystals, 22, parCastle, +22}, // 28: pixies
		{parCrystals, 25, parNum, 0},      // 29: curse (all +1, enemy's all -1)
	}
)

// card is a card type (sCard).
type card struct {
	typ   int // what it costs (par*)
	costs int
	param int // the parameter it changes (parNum: extra)
	value int // the change (+ own, - enemy)
}

// player is a player (sPlayer).
type player struct {
	player  int          // player*
	par     [parNum]int  // parameters
	add     [parNum]int  // parameter additions
	cards   [cardNum]int // cards (-1: none)
	disable [cardNum]bool
}

// newCard returns a random card: the cheap ones more often.
func (g *Game) newCard() int {
	c := int(float32(math.Pow(float64(g.RandFloat()*0.99), float64(float32(1.6)))) * cardTypes) // powf
	// three groups from the weakest to the strongest cards
	return c/3 + (c%3)*10
}

// permission disables the cards that cost too much.
func (g *Game) permission() {
	for i := range g.players {
		p := &g.players[i]
		for j := range p.cards {
			if c := p.cards[j]; c < 0 {
				p.disable[j] = true
			} else {
				p.disable[j] = cardTab[c].costs > p.par[cardTab[c].typ]
			}
		}
	}
}

// dispState1 displays the state of a player.
func (g *Game) dispState1(pl int) {
	p := &g.players[pl]
	g.DrawImgPal(stateImg, stateImgPal, 0, 0, stateX[pl], stateY, stateW, stateH, stateW)
	g.SelFont6x8()
	for i := 0; i < parNum; i++ {
		s := strconv.Itoa(min(p.par[i], 999))
		g.DrawText(s, stateX[pl]+30-len(s)*3, stateY+i*stateDY+2, parFgCol[i])
	}
}

func (g *Game) dispState() {
	g.dispState1(0)
	g.dispState1(1)
}

func (g *Game) dispGrass(pl int) {
	g.DrawBlit4Pal(grassImg, grassImgPal, 0, 0, castleX[pl]-grassW/2, grassY, grassW, grassH, grassW, transCol)
}

func (g *Game) dispCastle(pl int) {
	h := min(g.players[pl].par[parCastle], castleMax)
	pixh := h + castleH - castleMax
	g.DrawBlitPal(castleImg, castleImgPal, pl*castleW, 0, castleX[pl]-castleW/2,
		grassY+8-pixh, castleW, pixh, castleAllW, transCol)
}

func (g *Game) dispFence(pl int) {
	h := min(g.players[pl].par[parFence], castleMax)
	pixh := h + fenceH - castleMax
	g.DrawImg4Pal(fenceImg, fenceImgPal, 0, 0, fenceX[pl]-fenceW/2, grassY+8-pixh, fenceW, pixh, fenceW)
}

// dispTitle shows whose turn it is.
func (g *Game) dispTitle() {
	ys := 0
	if g.player == 0 {
		ys = blacksRedsH
	}
	g.DrawImg4Pal(blacksRedsImg, blacksRedsImgPal, 0, ys, 0, 0, blacksRedsW, blacksRedsH, blacksRedsAllW)
	ys = 0
	if g.player == 1 {
		ys = blacksRedsH
	}
	g.DrawImg4Pal(blacksRedsImg, blacksRedsImgPal, blacksRedsW, ys, width-blacksRedsW, 0, blacksRedsW, blacksRedsH, blacksRedsAllW)
}

// dispFlag animates the flags on the castles.
func (g *Game) dispFlag() {
	t := int(g.Time()>>18) & 3
	for pl := 0; pl < 2; pl++ {
		h := min(g.players[pl].par[parCastle], castleMax)
		pixh := h + 20
		x := castleX[pl] - 10
		y := grassY + 8 - pixh
		g.DrawImg4Pal(backgroundImg, backgroundImgPal, x, y, x, y, flagsW, flagsH-5, width)
		g.DrawBlit4Pal(flagsImg, flagsImgPal, (t+pl*4)*flagsW, 0, x, y, flagsW, flagsH, flagsAllW, transCol)
	}
}

// waitFlag waits del times 50ms animating the flags.
func (g *Game) waitFlag(del int) {
	for ; del > 0; del-- {
		g.dispFlag()
		g.DispUpdate()
		g.WaitMs(50)
	}
}

// dispCard displays a card (type < 0: none).
func (g *Game) dispCard(typ, x, y, shadow int, disable, back, discard bool) {
	if typ < 0 {
		return
	}
	if shadow != 0 {
		g.DrawBlit1Shadow(shadowImg, 0, 0, x+shadow, y+shadow, cardW, cardH, cardW, 10)
	}
	if back {
		g.DrawBlitPal(cardsImg, cardsImgPal, cardBack*cardW, 0, x, y, cardW, cardH, cardsAllW, transCol)
		return
	}
	g.DrawBlitPal(cardsImg, cardsImgPal, typ*cardW, 0, x, y, cardW, cardH, cardsAllW, transCol)
	if disable || discard {
		g.DrawBlit1Shadow(shadowImg, 0, 0, x, y, cardW, cardH, cardW, 10)
	}
	if discard {
		g.DrawBlitPal(cardsImg, cardsImgPal, cardDisc*cardW, 0, x, y, cardW, cardH, cardsAllW, transCol)
	}
}

// animCard moves a card with its shadow from x1, y1 to x2, y2.
func (g *Game) animCard(typ, x1, y1, x2, y2 int, back, discard bool) {
	// if (SoundMode > 0) PLAYSOUND(CardSnd)
	for i := 0; i < animSteps; i++ {
		x := (x2-x1)/animSteps*i + x1
		y := (y2-y1)/animSteps*i + y1
		s := int(4 + 60*float32(math.Sin(float64(float32(math.Pi*float64(i)/animSteps)))))
		xs, ys := x+s, y+s

		g.dispFlag()
		g.SaveRect(g.saveCardBuf[:], xs, ys, cardW, cardH)
		g.SaveRect(g.saveCardBuf2[:], x, y, cardW, cardH)
		g.dispCard(typ, x, y, s, false, back, discard)
		g.DispUpdate()
		g.RestoreRect(g.saveCardBuf2[:], x, y, cardW, cardH)
		g.RestoreRect(g.saveCardBuf[:], xs, ys, cardW, cardH)
	}
}

// dispCards displays the cards of the active player.
func (g *Game) dispCards() {
	p := &g.players[g.player]
	x := 0
	for i := 0; i < cardNum; i++ {
		if p.cards[i] < 0 {
			g.DrawImg4Pal(backgroundImg, backgroundImgPal, x, height-cardH, x, height-cardH, cardW, cardH, width)
		} else {
			g.dispCard(p.cards[i], x, height-cardH, 0, p.disable[i], p.player != playerHuman && !g.demo, false)
		}
		x += cardW
	}
}

func (g *Game) cursorOn() {
	if g.players[g.player].player == playerHuman {
		g.DrawBlit4Pal(cursorImg, cursorImgPal, 0, 0, g.selCard*cardW+(cardW-cursorW)/2,
			height-cardH+(cardH-cursorH)/2+6, cursorW, cursorH, cursorW, transCol)
	}
}

func (g *Game) cursorOff() {
	p := &g.players[g.player]
	g.dispCard(p.cards[g.selCard], g.selCard*cardW, height-cardH, 0, p.disable[g.selCard],
		p.player != playerHuman && !g.demo, false)
}

// takeCard gives the active player a new card.
func (g *Game) takeCard() {
	c := g.newCard()
	p := &g.players[g.player]
	g.animCard(c, midX1, 0, g.freePos*cardW, height-cardH, p.player != playerHuman && !g.demo, false)
	p.cards[g.freePos] = c
	p.disable[g.freePos] = false
	g.permission()
	g.dispCards()
	g.waitFlag(5)
}

// changeAnim shows the changes of the parameters in clouds.
func (g *Game) changeAnim() {
	g.SelFont8x8()
	for i := 3; i > 0; i-- {
		save := 0
		for pl := 0; pl < 2; pl++ {
			x := stateW + 2
			if pl == 1 {
				x = width - stateW - cloudW - 2
			}
			y := stateY - 1
			p := &g.players[pl]
			for par := 0; par < parNum; par++ {
				if a := p.add[par]; a != 0 {
					g.SaveRect(g.saveCloudBuf[save:save+cloudW*cloudH], x, y, cloudW, cloudH)
					g.DrawBlit4Pal(cloudImg, cloudImgPal, 0, 0, x, y, cloudW, cloudH, cloudW, transCol)
					if a >= 0 {
						s := "+" + strconv.Itoa(a)
						g.DrawText(s, x+15-len(s)*4, y+6, picopad.RGB(0, 0, 220))
					} else {
						s := strconv.Itoa(a)
						g.DrawText(s, x+15-len(s)*4, y+6, picopad.RGB(255, 0, 0))
					}
					save += cloudW * cloudH
				}
				y += stateDY
			}
		}
		g.DispUpdate()
		g.waitFlag(5)

		save = 0
		for pl := 0; pl < 2; pl++ {
			x := stateW + 2
			if pl == 1 {
				x = width - stateW - cloudW - 2
			}
			y := stateY - 1
			p := &g.players[pl]
			for par := 0; par < parNum; par++ {
				if p.add[par] != 0 {
					g.RestoreRect(g.saveCloudBuf[save:save+cloudW*cloudH], x, y, cloudW, cloudH)
					save += cloudW * cloudH
				}
				y += stateDY
			}
		}
		g.DispUpdate()
		g.waitFlag(1)
	}
}

// dispAll displays the whole game area.
func (g *Game) dispAll() {
	g.DrawImg4Pal(backgroundImg, backgroundImgPal, 0, 0, 0, 0, width, height, width)
	g.dispTitle()
	for pl := 0; pl < 2; pl++ {
		g.dispGrass(pl)
		g.dispCastle(pl)
		g.dispFence(pl)
	}
	g.dispFlag()
	g.dispCard(0, midX1, 0, 0, false, true, false)
	g.dispCard(g.last, midX2, 0, 0, false, false, g.lastDiscard)
	g.dispState()
	g.dispCards()
	g.DispUpdate()
}

// newGame starts a game.
func (g *Game) newGame(player1, player2 int) {
	g.players[0].player = player1
	g.players[1].player = player2
	g.demo = player1 == playerComp && player2 == playerComp
	g.player = 0 // the black ants start
	g.last = -1
	g.lastDiscard = false

	for i := range g.players {
		p := &g.players[i]
		p.par = [parNum]int{2, 5, 2, 5, 2, 5, 30, 10} // builders, bricks ... castle, fence
		for j := range p.cards {
			p.cards[j] = g.newCard()
		}
	}
	g.selCard = 3
	g.permission()
	g.dispAll()
	g.cursorOn()
}

// winGame shows the winner; it returns after a key (the demo: after a
// while) and starts a new game (not with OneGame).
func (g *Game) winGame(pl int) {
	g.endGame = pl
	h := min(g.players[pl].par[parCastle], castleMax)
	pixh := h + castleH - castleMax
	x := castleX[pl] - trumpetW/2 + 23
	y := grassY + 8 - pixh - 17

	// the trumpet while the fanfare plays (FanfaresSnd: 59,678 samples at
	// 22,050 Hz, 2.7 seconds; none when the sound is off)
	if g.soundMode > soundOff {
		for t0 := g.Time(); g.Time()-t0 < 2700000; {
			g.dispFlag()
			g.DrawBlit4Pal(trumpetImg, trumpetImgPal, pl*trumpetW, 0, x, y, trumpetW, trumpetH, trumpetAllW, transCol)
			g.DispUpdate()
			g.WaitMs(30)
		}
	}
	g.dispAll()

	// the winners dance (PLAYSOUNDREP(ApplauseSnd))
	y = grassY + 8 - 16
	x = castleX[pl] - winW/2
	anim := 0
	g.KeyFlush()
	n := 1000000000
	if g.demo {
		n = 200
	}
	for ; n > 0; n-- {
		if g.KeyGet() != picopad.NoKey {
			break
		}
		g.SaveRect(g.saveWinBuf[:], x, y, winW, winH)
		g.DrawBlit4Pal(winImg, winImgPal, pl*4*winW+(anim>>2)*128, 0, x, y, winW, winH, winAllW, transCol)
		anim++
		if anim >= 16 {
			anim = 0
		}
		g.dispFlag()
		g.DispUpdate()
		g.WaitMs(30)
		g.RestoreRect(g.saveWinBuf[:], x, y, winW, winH)
	}
	if g.OneGame { // game returns
		return
	}
	g.newGame(g.players[0].player, g.players[1].player)
}

// putCard plays (or discards) the card at pos.
func (g *Game) putCard(pos int, discard bool) {
	p := &g.players[g.player]
	if p.disable[pos] {
		discard = true
	}
	g.freePos = pos
	cd := p.cards[pos]
	c := &cardTab[cd]

	// pay the card
	if !discard {
		p.par[c.typ] -= c.costs
		g.dispState1(g.player)
		g.permission()
	}

	p.cards[pos] = -1
	g.dispCards()
	g.animCard(cd, pos*cardW, height-cardH, midX2, 0, false, discard)
	g.last = cd
	g.lastDiscard = discard
	g.dispCard(g.last, midX2, 0, 0, false, false, g.lastDiscard)

	if !discard {
		g.playCard(cd, c)
	}

	// a victory
	if g.players[0].par[parCastle] >= 100 || g.players[1].par[parCastle] <= 0 {
		g.winGame(0)
		return
	}
	if g.players[1].par[parCastle] >= 100 || g.players[0].par[parCastle] <= 0 {
		g.winGame(1)
		return
	}

	g.takeCard()

	// the other player, with the new stocks
	g.player ^= 1
	p = &g.players[g.player]
	p.par[parBricks] += p.par[parBuilders]
	p.par[parWeapons] += p.par[parSoldiers]
	p.par[parCrystals] += p.par[parWizards]
	g.dispState1(g.player)
	g.permission()
	g.dispCards()
	g.dispTitle()
}

// playCard does what the card does.
func (g *Game) playCard(cd int, c *card) {
	castle1 := g.players[0].par[parCastle]
	castle2 := g.players[1].par[parCastle]
	fence1 := g.players[0].par[parFence]
	fence2 := g.players[1].par[parFence]
	for i := range g.players {
		g.players[i].add = [parNum]int{}
	}
	p := &g.players[g.player]
	enemy := &g.players[g.player^1]

	if c.param != parNum {
		if c.value < 0 {
			p = enemy
		}
		// the sounds of the C source: power, castle, fence, stocks
		p.add[c.param] = c.value
	} else {
		switch cd {
		case 3: // reserve (castle +8, fence -4)
			p.add[parCastle] = 8
			p.add[parFence] = -min(p.par[parFence], 4)
		case 6: // wain (castle +8, enemy castle -4)
			p.add[parCastle] = 8
			enemy.add[parCastle] = -4
		case 16: // saboteur (enemy stocks -4)
			enemy.add[parBricks] = -4
			enemy.add[parWeapons] = -4
			enemy.add[parCrystals] = -4
		case 17: // thief (transfer enemy stocks 5)
			bricks := min(enemy.par[parBricks], 5)
			weapons := min(enemy.par[parWeapons], 5)
			crystals := min(enemy.par[parCrystals], 5)
			enemy.add[parBricks] = -bricks
			enemy.add[parWeapons] = -weapons
			enemy.add[parCrystals] = -crystals
			p.add[parBricks] = bricks
			p.add[parWeapons] = weapons
			p.add[parCrystals] = crystals
		case 29: // curse (all +1, enemy's all -1)
			for i := range p.add {
				p.add[i] = 1
			}
			for i := range enemy.add {
				if enemy.par[i] > 0 {
					enemy.add[i] = -1
				}
			}
		}
	}

	g.dispState()
	g.changeAnim()

	// update the parameters
	for i := range g.players {
		p := &g.players[i]
		for j := 0; j < parNum; j++ {
			p.par[j] += p.add[j]
			switch j {
			case parFence: // the rest goes to the castle
				if p.par[j] < 0 {
					p.par[parCastle] = max(p.par[parCastle]+p.par[j], 0)
					p.par[j] = 0
				}
			case parBuilders, parSoldiers, parWizards:
				p.par[j] = max(p.par[j], 1)
			default:
				p.par[j] = max(p.par[j], 0)
			}
		}
	}
	g.permission()

	// redraw all only when a castle or a fence changes
	if castle1 != g.players[0].par[parCastle] || castle2 != g.players[1].par[parCastle] ||
		fence1 != g.players[0].par[parFence] || fence2 != g.players[1].par[parFence] {
		g.dispAll()
	} else {
		g.dispState()
		g.dispCards()
	}
	g.waitFlag(4)
}

// findCard returns the best card of the active player: an expensive one,
// workers first; else the most expensive one to discard.
func (g *Game) findCard() int {
	p := &g.players[g.player]
	foundinx, foundval := -1, 0
	for i := 0; i < cardNum; i++ {
		if p.disable[i] {
			continue
		}
		c := &cardTab[p.cards[i]]
		testval := c.costs
		if c.param == parBuilders || c.param == parSoldiers || c.param == parWizards {
			testval = 100
		}
		if testval > foundval || (testval == foundval && g.RandU8() < 0x60) {
			foundinx, foundval = i, testval
		}
	}
	if foundinx < 0 {
		foundinx = 0
		for i := 0; i < cardNum; i++ {
			if p.disable[i] {
				if testval := cardTab[p.cards[i]].costs; testval > foundval {
					foundinx, foundval = i, testval
				}
			}
		}
	}
	return foundinx
}

// autoPlay is the turn of the computer.
func (g *Game) autoPlay() {
	g.waitFlag(8)
	i := g.findCard()
	g.putCard(i, g.players[g.player].disable[i])
}

// game plays until Y (a key in the demo); with OneGame until the winner
// has danced.
func (g *Game) game(player1, player2 int) {
	g.endGame = -1
	g.newGame(player1, player2)
	for {
		if g.players[g.player].player == playerComp {
			g.autoPlay()
			g.cursorOn()
			if !g.demo {
				g.KeyFlush()
			} else if g.KeyGet() != picopad.NoKey {
				return
			}
		} else {
			switch g.KeyGet() {
			case picopad.KeyY:
				return
			case picopad.KeyLeft:
				g.cursorOff()
				g.selCard--
				if g.selCard < 0 {
					g.selCard = cardNum - 1
				}
				g.cursorOn()
				g.KeyFlush()
			case picopad.KeyRight:
				g.cursorOff()
				g.selCard++
				if g.selCard >= cardNum {
					g.selCard = 0
				}
				g.cursorOn()
				g.KeyFlush()
			case picopad.KeyUp, picopad.KeyB: // discard
				g.putCard(g.selCard, true)
				g.cursorOn()
				g.KeyFlush()
			case picopad.KeyDown, picopad.KeyA: // play
				g.putCard(g.selCard, g.players[g.player].disable[g.selCard])
				g.cursorOn()
				g.KeyFlush()
			case picopad.KeyX: // help
				g.cursorOff()
				g.selCard = g.findCard()
				g.cursorOn()
				g.KeyFlush()
			case picopad.NoKey:
			default:
				g.KeyFlush()
			}
		}
		if g.OneGame && g.endGame >= 0 {
			return
		}
		g.dispFlag()
		g.DispUpdate()
	}
}
