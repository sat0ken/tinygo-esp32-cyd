// Package vegaslot is the Vegaslot of PicoLibSDK (PicoPad/GAME/VEGASLOT,
// by Miroslav Nemecek, Unlicense), a slot machine, ported from C to
// package picopad. It follows the C source function by function; the C
// globals are the fields of Game.
//
// A starts a game, X autostarts, B changes the bet, LEFT plays the
// even/odd toss (and moves a win to the bank), RIGHT moves a win to the
// credit, UP fills 10 credits, DOWN moves the bank to the credit, Y ends
// Run (on the PicoPad: back to the boot loader). After a win, A risks it
// (double or half). Three same symbols on a win line win; the bet is the
// number of lines (6: the high game, 4 times the win). There is no sound
// yet: the PLAYSOUND calls of the C source are comments.
package vegaslot

import (
	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/picopad"
)

//go:generate go run ../../tools/picoimg -pkg vegaslot -root $PICOLIBSDK -o img.go PicoPad/GAME/VEGASLOT/img/back.cpp PicoPad/GAME/VEGASLOT/img/symbols.cpp PicoPad/GAME/VEGASLOT/img/blur.cpp PicoPad/GAME/VEGASLOT/img/digits.cpp

const (
	width  = picopad.Width
	height = picopad.Height

	symbolW   = 64 // symbol width
	symbolH   = 48 // symbol height
	symbolNum = 6  // number of symbol types

	reelNum = 3           // number of reels
	reelPos = 32          // number of reel positions
	reelW   = symbolW     // reel width
	reelH   = 3 * symbolH // reel height (144)
	reel1X  = 64          // X coordinate of the first reel
	reelY   = 48          // Y coordinate of the reels

	reelStateIn   = 1 // turning in
	reelStateOut  = 2 // turning out
	reelStateOver = 3 // overshoot

	reelTimeInc   = 80  // time of one animation step [ms]
	reelSlowSpeed = 100 // slow speed
	reelStartDel  = 2   // start delay in steps
	reelAccel     = 5   // acceleration/deceleration
	reelOverSteps = 150 // overshoot steps
	reelBlurSpeed = 200 // speed of the blurred image

	reelDirForw = 1 // reel direction forward (down), normal
	reelDirBack = 0 // reel direction back (up), bonus

	betHigh = 6   // bet max value: the high game
	maxWin  = 750 // max. win value

	digitW    = 16  // digit width
	digitH    = 24  // digit height
	digitY    = 16  // digit Y coordinate
	bankX     = 40  // BANK X coordinate
	bankNum   = 4   // BANK number of digits
	winX      = 120 // WIN X coordinate
	winNum    = 3   // WIN number of digits
	betX      = 184 // BET X coordinate
	betNum    = 1   // BET number of digits
	creditX   = 216 // CREDIT X coordinate
	creditNum = 4   // CREDIT number of digits
	digitImgW = 192 // width of the digit image

	// symbols
	che = 0 // cherry
	app = 1 // apple
	plu = 2 // plum
	pea = 3 // pear
	gra = 4 // grape
	bel = 5 // bell

	winLines = 5      // number of winning lines
	posMask  = 0x1f   // position mask (0..31)
	poshMask = 0x1fff // position mask * 256
)

// linesTab are the winning lines: the row of each reel.
var linesTab = [winLines * reelNum]int8{
	0, 0, 0, // 1: middle line
	1, 1, 1, // 2: top line
	-1, -1, -1, // 3: bottom line
	1, 0, -1, // 4: top diagonal
	-1, 0, 1, // 5: bottom diagonal
}

// winItem is a win of a winning table (sWinItem; u16 as in C: the tables
// are variables in RAM).
type winItem struct {
	value uint16 // value of the win (1 to 750)
	typ   uint16 // which symbols on how many lines (GetTypes)
	num   uint16 // number of wins
	rand  uint16 // randomness (0 to 0xffff)
}

// reel is a reel (sReel); the positions are "the symbol on line 0" * 256.
type reel struct {
	state int // reelState*
	dir   int // reelDir*
	speed int // current speed
	pos   int // current position * 256
	dest  int // destination position * 256
	rem   int // remaining steps * 256
	wait  int // wait to start (steps)
	over  int // overshoot steps * 256
	past  int // past steps * 256
}

// Game is Vegaslot on a console.
type Game struct {
	*picopad.Console

	bankVal    int  // current bank
	winVal     int  // current win
	userBetVal int  // user selected bet
	betVal     int  // current bet, betHigh: the high game, 0: no bet
	creditVal  int  // current credit
	bonus      int  // remaining bonus
	winLines   int  // current win lines
	betCredit  int  // bet from the credits (1..5)
	betBank    int  // bet from the bank (0: low, 15: high)
	lastHigh   bool // the last game was high
	autoStart  bool
	tossing    bool
	risking    bool
	riskEnable bool // risk start is enabled
	riskHalf   bool // the second half of the win remains in the risk
	gameIn     int
	gameOut    int

	reels    [reelNum]reel
	reelFast bool
}

// New returns the game on d (at least 320x240) with the touch panel t;
// seed starts the random numbers. Set Pad to use a game controller.
func New(d hal.Display, t hal.Touch, seed uint32) *Game {
	return &Game{Console: picopad.New(d, t, seed)}
}

// dispVal displays val with dig digits at x (-1: no digit).
func (g *Game) dispVal(val, x, dig int) {
	x += dig * digitW
	first := true
	if val < 0 {
		first = false
		val = 0
	}
	for ; dig > 0; dig-- {
		x -= digitW
		if val == 0 && !first {
			g.DrawImg4Pal(digitsImg, digitsImgPal, 0, 0, x, digitY, digitW, digitH, digitImgW)
		} else {
			k := val / 10
			val -= k * 10
			g.DrawImg4Pal(digitsImg, digitsImgPal, (val+1)*digitW, 0, x, digitY, digitW, digitH, digitImgW)
			val = k
			first = false
		}
	}
}

func (g *Game) dispBank() { g.dispVal(min(g.bankVal, 9999), bankX, bankNum) }

func (g *Game) dispWin() {
	k := min(g.winVal, 999)
	if k <= 0 {
		k = -1 // hide
	}
	g.dispVal(k, winX, winNum)
}

func (g *Game) dispBet() {
	k := g.betVal
	if k == 0 {
		k = -1 // no credit
	}
	if g.bonus != 0 {
		k = 0 // bonus game
	}
	if k == betHigh {
		g.DrawImg4Pal(digitsImg, digitsImgPal, 11*digitW, 0, betX, digitY, digitW, digitH, digitImgW)
	} else {
		g.dispVal(k, betX, betNum)
	}
}

func (g *Game) dispCredit() { g.dispVal(min(g.creditVal, 9999), creditX, creditNum) }

// dispReel displays a reel with the shadows.
func (g *Game) dispReel(inx int) {
	y := reelY
	x := reel1X + inx*reelW
	h := reelH
	r := &g.reels[inx]
	pos := r.pos/256 - 1
	ys := ((r.pos&0xff)*symbolH + 100) / 256
	tab := reelTab[inx*reelPos:]

	for h > 0 {
		pos &= posMask
		h2 := min(symbolH-ys, h)
		h -= h2
		if r.speed >= reelBlurSpeed {
			g.DrawImgPal(blurImg, blurImgPal, 0, int(tab[pos])*symbolH+ys, x, y, symbolW, h2, symbolW)
		} else {
			g.DrawImgPal(symbolsImg, symbolsImgPal, 0, int(tab[pos])*symbolH+ys, x, y, symbolW, h2, symbolW)
		}
		ys = 0
		y += h2
		pos++
	}

	// the darker top and bottom
	for i := 1; i <= 15; i++ {
		g.DrawRectShadow(x, reelY+i-1, symbolW, 1, uint8(i))
		g.DrawRectShadow(x, reelY+reelH-i, symbolW, 1, uint8(i))
	}

	// the lines without a bet
	if g.winLines < 2 {
		g.DrawRectShadow(x, reelY, symbolW, symbolH, 15)
	}
	if g.winLines < 1 {
		g.DrawRectShadow(x, reelY+symbolH, symbolW, symbolH, 15)
	}
	if g.winLines < 3 {
		g.DrawRectShadow(x, reelY+2*symbolH, symbolW, symbolH, 15)
	}
}

func (g *Game) dispReels() {
	for i := 0; i < reelNum; i++ {
		g.dispReel(i)
	}
}

func (g *Game) setWinLines(lines int) {
	g.winLines = lines
	g.dispReels()
}

// calcReel prepares turning the reels to their dest.
func (g *Game) calcReel(fast bool) {
	g.reelFast = fast
	for inx := range g.reels {
		r := &g.reels[inx]
		// whole turns: 1 of the 1st reel, 2 of the 2nd, 3 of the 3rd
		if fast {
			r.rem = (inx + 1) * reelPos * 256
		} else {
			r.rem = 0
		}
		r.wait = inx * reelStartDel
		i := r.dest - r.pos
		if r.dir == reelDirForw {
			i = -i
		}
		r.rem += i & poshMask
		r.over = 0
		r.speed = 0
		if fast {
			r.past = 0
			r.rem += reelOverSteps
			r.over = reelOverSteps
		}
		r.state = reelStateIn
	}
}

// turnReel turns the reels to their dest.
func (g *Game) turnReel(fast bool) {
	g.calcReel(fast)
	// if (fast) PlaySoundChan(1, ReelSnd ...); PLAYSOUND(StartSnd)

	t := g.Time()
	for {
		moving := false
		for inx := range g.reels {
			r := &g.reels[inx]
			if r.wait > 0 {
				r.wait--
			} else if r.rem > 0 || r.over > 0 {
				var shift int
				switch {
				case !fast:
					shift = min(reelSlowSpeed, r.rem)
					r.rem -= shift
				case r.state == reelStateIn: // accelerating
					shift = r.past/reelAccel + reelSlowSpeed
					if shift > r.rem {
						shift = r.rem / 2
					}
					r.past += shift
					r.rem -= shift
					if r.rem <= r.past {
						r.state = reelStateOut
					}
				case r.state == reelStateOut: // decelerating
					shift = r.rem/reelAccel + reelSlowSpeed
					if r.rem <= shift {
						shift = r.rem
						r.state = reelStateOver
					}
					r.rem -= shift
				default: // overshoot
					shift = reelSlowSpeed
					if shift >= r.over {
						shift = r.over
					}
					r.over -= shift
					shift = -shift
				}

				// move the reel (normally down: the position decreases)
				r.speed = shift
				if r.dir == reelDirForw {
					shift = -shift
				}
				r.pos = (r.pos + shift) & poshMask
				// if ((r->rem <= 0) && (r->over <= 0) && fast) PLAYSOUND(StopSnd)
			}

			if r.rem > 0 || r.over > 0 {
				moving = true
			} else {
				r.pos = r.dest
			}
		}

		g.dispReels()
		g.DispUpdate()

		var t2 uint32
		for {
			t2 = g.Time()
			if t2-t >= reelTimeInc*1000 {
				break
			}
		}
		t = t2
		if !moving {
			break
		}
	}
	// StopSoundChan(1)
}

func (g *Game) setWin(win int) {
	g.winVal = win
	g.dispWin()
}

// setBet sets the bet, lower if the credit or the bank is too low.
func (g *Game) setBet(bet int) {
	if bet < 1 {
		bet = 1
	}
	betcredit, betbank := bet, 0
	if bet == betHigh {
		betcredit, betbank = 5, 15
	}
	if g.creditVal < betcredit || g.bankVal < betbank {
		betbank = 0
		betcredit = min(g.creditVal, 5)
		bet = betcredit
	}
	g.betVal = bet
	g.setWinLines(betcredit)
	g.betCredit = betcredit
	g.betBank = betbank
	g.dispBet()
}

func (g *Game) addBank(add int) {
	if add != 0 {
		g.bankVal = max(g.bankVal+add, 0)
		g.dispBank()
		g.setBet(g.userBetVal)
	}
}

func (g *Game) addCredit(add int) {
	if add != 0 {
		g.creditVal = max(g.creditVal+add, 0)
		g.dispCredit()
		g.setBet(g.userBetVal)
	}
}

// subBet pays the bet (not in a bonus game).
func (g *Game) subBet() {
	if g.bonus == 0 {
		g.gameIn += g.betCredit
		g.addCredit(-g.betCredit)
		g.gameIn += g.betBank
		g.addBank(-g.betBank)
	}
}

// winToss chooses a win of the table of lines.
func (g *Game) winToss(lines int) *winItem {
	tab := winTabs[lines-1]
	rnd := int(g.RandU16())
	if rnd >= int(tab[0].rand) {
		return &tab[0] // not a win
	}
	k := 0
	for {
		k++
		rnd -= int(tab[k].rand)
		if rnd < 0 {
			return &tab[k]
		}
	}
}

// testSymbol returns the symbols on the win lines, 3 bits a line
// (symbolNum: no win).
func testSymbol(pos1, pos2, pos3 int) int {
	acc := 0
	for line := winLines - 1; line >= 0; line-- {
		acc <<= 3
		lin := linesTab[line*reelNum:]
		sym1 := reelTab[0*reelPos+((pos1+int(lin[0]))&posMask)]
		sym2 := reelTab[1*reelPos+((pos2+int(lin[1]))&posMask)]
		sym3 := reelTab[2*reelPos+((pos3+int(lin[2]))&posMask)]
		if sym1 == sym2 && sym1 == sym3 {
			acc |= int(sym1)
		} else {
			acc |= symbolNum
		}
	}
	return acc
}

// getPrize returns the win of the symbols on lines win lines.
func getPrize(lines, winsym int) int {
	acc := 0
	for ; lines > 0; lines-- {
		if sym := winsym & 7; sym != symbolNum {
			acc += winValTab[sym]
		}
		winsym >>= 3
	}
	return acc
}

// getTypes returns the winning types (winItem.typ) of the symbols on
// lines win lines: 3 bits the symbol and 3 bits its number of lines, for
// up to 3 symbols.
func getTypes(lines, winsym int) int {
	var num [symbolNum]int
	for ; lines > 0; lines-- {
		if sym := winsym & 7; sym != symbolNum {
			num[sym]++
		}
		winsym >>= 3
	}
	acc := 0
	for i := symbolNum - 1; i >= 0; i-- {
		if num[i] > 0 {
			acc = (acc<<3|num[i])<<3 | i
		}
	}
	return acc & 0xffff
}

// findReel finds a random position of the reels of the win item (sets
// the dest of the reels).
func (g *Game) findReel(lines int, item *winItem) {
	var i int
	if item.value == 0 {
		i = int(g.RandU16Max(uint16(reelPos*reelPos*reelPos - int(item.num) - 1)))
	} else {
		i = int(g.RandU16Max(item.num - 1))
	}
	var pos1, pos2, pos3 int
search:
	for pos3 = 0; pos3 < reelPos; pos3++ {
		for pos2 = 0; pos2 < reelPos; pos2++ {
			for pos1 = 0; pos1 < reelPos; pos1++ {
				winsym := testSymbol(pos1, pos2, pos3)
				prize := getPrize(lines, winsym)
				if (item.value == 0 && prize == 0) ||
					(item.value != 0 && prize == int(item.value) && getTypes(lines, winsym) == int(item.typ)) {
					i--
					if i < 0 {
						break search
					}
				}
			}
		}
	}
	g.reels[0].dest = pos1 * 256
	g.reels[1].dest = pos2 * 256
	g.reels[2].dest = pos3 * 256
}

// getReelPos chooses the next position of the reels.
func (g *Game) getReelPos(lines int) {
	var item *winItem
	if g.bonus > 0 {
		if g.RandU16()&3 == 0 {
			// lost in the bonus game
			for item = g.winToss(lines); item.value > 0; item = g.winToss(lines) {
			}
		} else {
			// win in the bonus game
			for item = g.winToss(lines); item.value == 0; item = g.winToss(lines) {
			}
		}
	} else {
		item = g.winToss(lines)
		// not a win: sometimes a bluff, a win on 5 lines but not on these
		if item.value == 0 && lines != 5 && g.RandU16()&3 == 0 {
			for {
				for item = g.winToss(5); item.value == 0; item = g.winToss(5) {
				}
				g.findReel(5, item)
				winsym := testSymbol(g.reels[0].dest/256, g.reels[1].dest/256, g.reels[2].dest/256)
				if getPrize(lines, winsym) == 0 {
					return
				}
			}
		}
	}
	g.findReel(lines, item)
}

// winInd blinks the win.
func (g *Game) winInd(prize int) {
	// PLAYSOUND(WinSnd)
	for i := 8; i > 0; i-- {
		g.setWin(prize)
		g.DispUpdate()
		g.WaitMs(100)
		g.setWin(0)
		g.DispUpdate()
		g.WaitMs(100)
	}
}

// doCash moves the bank to the credit (up to 100), or pays the credit out.
func (g *Game) doCash() {
	if i := g.bankVal; i > 0 {
		i = min(i, 100)
		g.addCredit(i)
		g.addBank(-i)
		// PLAYSOUND(CashSnd)
	} else if i := g.creditVal; i > 0 {
		g.addCredit(-min(i, 100))
		// PLAYSOUND(OutSnd)
	}
}

// winTrans moves a win to the bank or the credit in steps.
func (g *Game) winTrans(prize int, tobank bool, remain int) {
	// PLAYSOUND(TransferSnd)
	var trans int
	switch {
	case prize <= 20:
		trans = 2
	case prize <= 50:
		trans = 5
	case prize <= 100:
		trans = 10
	case prize <= 200:
		trans = 20
	default:
		trans = 50
	}
	for prize > 0 {
		trans = min(trans, prize)
		prize -= trans
		if tobank {
			g.addBank(trans)
		} else {
			g.addCredit(trans)
		}
		g.setWin(prize + remain)
		g.DispUpdate()
		g.WaitMs(100)
	}
}

// doRisk is the risk game after a win: LEFT moves it to the bank, RIGHT
// to the credit (the first half when it is more than 10), A risks it
// (double or half), X autostarts.
func (g *Game) doRisk(prize int) {
	if g.autoStart {
		g.winTrans(prize, true, 0)
		return
	}
	g.risking = true
	g.riskHalf = false
	for prize > 0 {
		g.riskEnable = prize <= maxWin/2 && prize >= 10

		// the win blinks
		if (g.Time()>>17)&1 == 0 {
			g.setWin(prize)
		} else {
			g.setWin(0)
		}

		switch c := g.KeyGet(); c {
		case picopad.KeyLeft, picopad.KeyRight: // to the bank, to the credit
			i := prize
			if !g.riskHalf && prize > 10 {
				i /= 2
				g.riskHalf = true
			}
			prize -= i
			g.winTrans(i, c == picopad.KeyLeft, prize)
			g.KeyFlush()
		case picopad.KeyA: // risk
			if g.riskEnable {
				if g.RandU16()&1 == 0 {
					prize /= 2 // PLAYSOUND(RiskLostSnd)
				} else {
					prize *= 2 // PLAYSOUND(RiskWinSnd)
				}
				g.riskHalf = false
			}
			g.KeyFlush()
		case picopad.KeyX: // autostart
			g.autoStart = true
			g.winTrans(prize, true, 0)
			g.KeyFlush()
			return
		}
		g.DispUpdate()
	}
	g.setWin(0)
	g.risking = false
}

// doToss is the even/odd game with the bet credits: A plays, an even
// number doubles the bet into the bank; LEFT ends it, X autostarts.
func (g *Game) doToss() {
	g.tossing = true
	g.autoStart = false
	// PlaySoundChan(1, TossSnd ...)
	for g.betCredit > 0 && g.betCredit <= g.creditVal {
		// one game
		t := g.Time()
		for {
			// a random number
			g.dispVal(int(g.RandU16()&7)+1, betX, betNum)
			g.DispUpdate()
			g.WaitMs(100)

			// the minimal game time is 1 second
			if g.Time()-t < 1000000 {
				continue
			}
			ch := g.KeyGet()
			if g.autoStart {
				if ch != picopad.NoKey {
					// PLAYSOUND(AutoOffSnd)
					g.autoStart = false
					ch = picopad.NoKey
					g.KeyFlush()
				} else {
					ch = picopad.KeyA
				}
			}

			if ch == picopad.KeyA { // roll
				n := g.betCredit
				g.addCredit(-n)
				k := int(g.RandU16()&7) + 1
				g.dispVal(k, betX, betNum)
				g.DispUpdate()
				if k&1 == 1 {
					n = 0 // odd: lost
				} else {
					n *= 2 // even: win
					g.addBank(n)
				}
				g.setWin(n)
				g.WaitMs(200)
				g.setWin(0)
				g.KeyFlush()
				break
			} else if ch == picopad.KeyX { // autostart
				// PLAYSOUND(AutoOnSnd)
				g.autoStart = true
				g.KeyFlush()
			} else if ch == picopad.KeyLeft { // exit
				g.KeyFlush()
				g.dispBet()
				return
			}
		}
	}
	g.dispBet()
}

// newGame draws the machine and starts with 100 credits.
func (g *Game) newGame() {
	g.DrawImgRle(backImg, backImgPal, 0, 0, width, height)

	g.bankVal = 0
	g.winVal = 0
	g.userBetVal = 3
	g.betVal = 3
	g.creditVal = 100
	g.bonus = 0
	g.winLines = 3
	g.betCredit = 3
	g.betBank = 0
	g.lastHigh = false
	g.autoStart = false
	g.tossing = false
	g.risking = false
	g.riskEnable = false
	g.riskHalf = false
	g.gameIn = 0
	g.gameOut = 0

	for i := range g.reels {
		g.reels[i].pos = 0
		g.reels[i].dir = reelDirForw
	}
	g.dispReels()
	g.dispBank()
	g.dispWin()
	g.dispBet()
	g.dispCredit()
}

// play is a game: the reels turn, a win blinks and can be risked.
func (g *Game) play() {
	if !g.autoStart {
		g.KeyFlush()
	}
	if g.betVal > 0 {
		lines := g.winLines
		g.getReelPos(lines)
		g.lastHigh = g.betBank > 0
		g.subBet()

		// the bonus turns back
		dir := reelDirForw
		if g.bonus != 0 {
			dir = reelDirBack
		}
		for i := range g.reels {
			g.reels[i].dir = dir
		}
		g.turnReel(true)

		winsym := testSymbol(g.reels[0].pos/256, g.reels[1].pos/256, g.reels[2].pos/256)
		if prize := getPrize(lines, winsym); prize > 0 {
			// the high game wins 4 times
			if g.bonus == 0 && g.lastHigh {
				prize *= 4
			}
			// above the max. win: the rest is a bonus
			if prize > maxWin {
				g.bonus += prize
				prize = maxWin
				g.dispBet()
			}
			g.winInd(prize)
			if g.bonus > 0 {
				g.bonus = max(g.bonus-prize, 0)
				g.dispBet()
			}
			g.gameOut += prize
			g.doRisk(prize)
		}
	}
	if !g.autoStart {
		g.KeyFlush()
	}
}

// Run plays until Y (main of the C source, which then resets to the boot
// loader).
func (g *Game) Run() {
	g.newGame()
	g.DispUpdate()
	for {
		ch := g.KeyGet()
		if g.autoStart {
			if ch == picopad.NoKey && g.creditVal > 0 {
				ch = picopad.KeyA
			} else {
				// PLAYSOUND(AutoOffSnd)
				g.autoStart = false
				ch = picopad.NoKey
				g.KeyFlush()
			}
		}

		switch ch {
		case picopad.KeyA:
			g.play()
		case picopad.KeyB: // the bet
			g.KeyFlush()
			if g.bonus == 0 && g.creditVal > 0 {
				bet := g.userBetVal + 1
				if bet > betHigh {
					bet = 1
				}
				oldbet := g.betVal
				g.setBet(bet)
				if oldbet == g.betVal {
					g.setBet(1)
				}
				g.userBetVal = max(g.betVal, 1)
			}
			g.KeyFlush()
		case picopad.KeyUp: // 10 credits
			g.KeyFlush()
			// PLAYSOUND(CashSnd)
			g.addCredit(10)
			g.KeyFlush()
		case picopad.KeyDown: // cash
			g.KeyFlush()
			g.doCash()
			g.KeyFlush()
		case picopad.KeyLeft: // toss
			g.KeyFlush()
			g.doToss()
			g.KeyFlush()
		case picopad.KeyX: // autostart
			// PLAYSOUND(AutoOnSnd)
			g.autoStart = true
			g.KeyFlush()
		case picopad.KeyY:
			return
		}
		g.DispUpdate()
	}
}
