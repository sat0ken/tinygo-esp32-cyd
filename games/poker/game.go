// Package poker is "Jacks or Better" video poker for a 480x272 or 320x240
// touch screen. It depends only on hal: examples/12_poker runs it alone,
// cmd/games from a launcher.
package poker

import (
	"image/color"
	"strconv"

	"github.com/sat0ken/tinygo-cyd/games/cards"
	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/romfont/freesans"
	"github.com/sat0ken/tinygo-cyd/romfont/proggy"
	"tinygo.org/x/tinyfont"
)

const (
	startCr  = 100
	tapGap   = 0.15 // seconds; ignore taps closer than this (touch bounce)
	flipStep = 0.09 // seconds between cards turning over
)

var (
	tableColor = color.RGBA{0, 30, 110, 255}
	payColor   = color.RGBA{0, 18, 70, 255}
	betColumn  = color.RGBA{150, 20, 30, 255}
	barColor   = color.RGBA{0, 18, 70, 255}
	btnColor   = color.RGBA{40, 70, 160, 255}
	offColor   = color.RGBA{25, 40, 95, 255} // a button that cannot be used now
	dealColor  = color.RGBA{200, 140, 0, 255}
	textColor  = color.RGBA{255, 255, 255, 255}
	dimText    = color.RGBA{160, 180, 220, 255}
	payText    = color.RGBA{255, 230, 120, 255}
	hintText   = color.RGBA{120, 230, 255, 255}
	winColor   = color.RGBA{255, 220, 60, 255}
)

type rect struct{ x, y, w, h int16 }

func (r rect) has(x, y int16) bool { return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h }

// button is a button of the bar: where it is and how its name is written
// (TestTextsFit checks that the names fit).
type button struct {
	r    rect
	font tinyfont.Fonter
	base int16 // baseline from the top of the button
}

// layout is where everything is: the pay table, the five cards with their
// HELD labels, and a button bar.
type layout struct {
	pay              rect
	rowH, rowBase    int16 // a pay table row and the baseline in it
	nameX            int16
	betColX, betColW int16 // the bet-1 column of the pay table and the column width
	numRight         int16 // space right of the numbers in a column

	cardW, cardH, cardGap, leftX, cardY int16
	labels                              rect // the HELD labels / the result line

	barY                 int16
	back, bet, max, deal button
	info                 rect // credit and bet
	creditFont           tinyfont.Fonter
	creditBase, betBase  int16
}

// layoutFor returns the layout for a w x h screen: the 480x272 one, or the
// 320x240 one (ESP32-2432S028R) on smaller screens.
func layoutFor(w, h int16) layout {
	big, small := &freesans.Bold9pt7b, &proggy.TinySZ8pt7b
	if w >= 480 && h >= 272 {
		const cardW, cardH, cardY = 84, 100, 2 + 9*11 + 7
		const barY, btnY, btnH = 234, 236, 34
		return layout{
			pay: rect{4, 2, 472, 9 * 11}, rowH: 11, rowBase: 9, nameX: 12,
			betColX: 172, betColW: 60, numRight: 10,
			cardW: cardW, cardH: cardH, cardGap: 8, leftX: (480 - (5*cardW + 4*8)) / 2, cardY: cardY,
			labels: rect{0, cardY + cardH + 2, w, 22},
			barY:   barY,
			back:   button{rect{4, btnY, 58, btnH}, small, 21},
			bet:    button{rect{66, btnY, 90, btnH}, big, 23},
			max:    button{rect{160, btnY, 90, btnH}, big, 23},
			deal:   button{rect{380, btnY, 96, btnH}, big, 23},
			info:   rect{254, btnY, 122, btnH}, creditFont: big, creditBase: 15, betBase: 30,
		}
	}
	// Cards of the solitaire's size; the bar's small buttons use the small
	// font.
	const cardW, cardH, cardY = 56, 72, 2 + 9*10 + 5
	const barY, btnY, btnH = 196, 198, 38
	return layout{
		pay: rect{4, 2, w - 8, 9 * 10}, rowH: 10, rowBase: 8, nameX: 6,
		betColX: 100, betColW: 43, numRight: 7, // "JACKS OR BETTER" is 90 pixels
		cardW: cardW, cardH: cardH, cardGap: 6, leftX: (w - (5*cardW + 4*6)) / 2, cardY: cardY,
		labels: rect{0, cardY + cardH + 2, w, 20},
		barY:   barY,
		back:   button{rect{2, btnY, 50, btnH}, small, 22},
		bet:    button{rect{56, btnY, 52, btnH}, small, 22},
		max:    button{rect{112, btnY, 52, btnH}, small, 22},
		deal:   button{rect{244, btnY, 74, btnH}, big, 25},
		info:   rect{168, btnY, 72, btnH}, creditFont: small, creditBase: 15, betBase: 29,
	}
}

// cardRect includes the HELD label under the card, so a tap there holds
// the card too.
func (l *layout) cardRect(i int) rect {
	return rect{l.leftX + int16(i)*(l.cardW+l.cardGap), l.cardY, l.cardW, l.cardH + 2 + l.labels.h}
}

// msgFont returns the font and baseline for a message on the label line:
// the bold font, or the small one if s does not fit.
func (l *layout) msgFont(s string) (tinyfont.Fonter, int16) {
	if _, w := tinyfont.LineWidth(&freesans.Bold9pt7b, s); int16(w) <= l.labels.w-8 {
		return &freesans.Bold9pt7b, 17
	}
	return &proggy.TinySZ8pt7b, 14
}

type phase int

const (
	idle    phase = iota // before the first deal or after a draw
	dealing              // the five new cards are turning over
	holding              // choosing the cards to hold
	drawing              // the new cards are turning over
)

// Game is video poker.
type Game struct {
	d hal.Display
	t hal.Touch
	l layout

	// NewSeed returns the seed for the shuffles.
	NewSeed func() uint32
	// OnExit, if set, shows a "< GAMES" button; tapping it calls OnExit.
	OnExit func()

	rand    *cards.Rand
	round   Round
	faceUp  [5]bool
	dealt   bool // a hand has been dealt (before that the cards are backs)
	phase   phase
	timer   float32
	credits int
	bet     int
	result  Hand // of the last draw
	won     int  // credits won by the last draw
	refill  bool // credits ran out and were filled up again

	clock    float32
	lastTap  float32
	wasTouch bool

	drawn map[rect]string // what each region shows, to redraw only changes
	dirty bool            // something changed since the last refresh
}

// NewGame creates a game; call Start to draw it.
func NewGame(d hal.Display, t hal.Touch, newSeed func() uint32) *Game {
	w, h := d.Size()
	return &Game{d: d, t: t, l: layoutFor(w, h), NewSeed: newSeed, lastTap: -1, credits: startCr, bet: 1}
}

// Start draws the whole screen, ready for the first deal.
func (g *Game) Start() {
	g.rand = cards.NewRand(g.NewSeed())
	w, h := g.d.Size()
	g.d.FillRectangle(0, 0, w, h, tableColor)
	g.d.FillRectangle(0, g.l.barY, w, h-g.l.barY, barColor)
	g.drawn = map[rect]string{}
	g.refresh()
}

// ---- input ----

// Update reads the touch panel and advances the clock by dt seconds.
func (g *Game) Update(dt float32) {
	g.clock += dt
	x, y, pressed := g.t.ReadTouch()
	tap := pressed && !g.wasTouch && g.clock-g.lastTap >= tapGap
	g.wasTouch = pressed
	if tap {
		g.lastTap = g.clock
		g.tapAt(x, y)
	}
	if g.phase == dealing || g.phase == drawing {
		g.timer += dt
		for g.timer >= flipStep && (g.phase == dealing || g.phase == drawing) {
			g.timer -= flipStep
			g.flipNext()
		}
	}
	if g.dirty {
		g.refresh()
	}
}

func (g *Game) tapAt(x, y int16) {
	l := &g.l
	if g.OnExit != nil && l.back.r.has(x, y) {
		g.OnExit()
		return
	}
	switch g.phase {
	case idle:
		switch {
		case l.bet.r.has(x, y):
			g.bet = g.bet%MaxBet + 1
			g.dirty = true
		case l.max.r.has(x, y):
			g.bet = MaxBet
			g.deal()
		case l.deal.r.has(x, y):
			g.deal()
		}
	case holding:
		if l.deal.r.has(x, y) {
			g.draw()
			return
		}
		for i := range g.round.Held {
			if l.cardRect(i).has(x, y) {
				g.round.Held[i] = !g.round.Held[i]
				g.dirty = true
			}
		}
	}
	// While cards are turning over, taps are ignored.
}

// deal takes the bet and deals five new cards face down; they turn over
// one by one.
func (g *Game) deal() {
	g.refill = false
	if g.credits == 0 {
		g.credits = startCr
		g.refill = true
		g.dirty = true
		return
	}
	if g.bet > g.credits {
		g.bet = g.credits
	}
	g.credits -= g.bet
	g.round = NewRound(g.rand)
	g.faceUp = [5]bool{}
	g.dealt = true
	g.won = 0
	g.phase, g.timer = dealing, 0
	g.dirty = true
}

// draw replaces the cards that are not held; they turn over one by one
// and then the hand is paid.
func (g *Game) draw() {
	g.result = g.round.Draw()
	g.faceUp = g.round.Held
	g.phase, g.timer = drawing, 0
	g.dirty = true
	g.flipNext() // all held: done at once
}

// flipNext turns over the next card; when all are face up it ends the
// deal or the draw.
func (g *Game) flipNext() {
	for i, up := range g.faceUp {
		if !up {
			g.faceUp[i] = true
			g.dirty = true
			return
		}
	}
	if g.phase == dealing {
		g.phase = holding
	} else {
		g.won = Payout(g.result, g.bet)
		g.credits += g.won
		g.phase = idle
	}
	g.dirty = true
}

// ---- drawing ----

// region redraws r with draw() if its key changed since the last time.
func (g *Game) region(r rect, bg color.RGBA, key string, draw func()) {
	if g.drawn[r] == key {
		return
	}
	g.d.FillRectangle(r.x, r.y, r.w, r.h, bg)
	draw()
	g.drawn[r] = key
}

func (g *Game) text(font tinyfont.Fonter, x, y int16, s string, c color.RGBA) {
	tinyfont.WriteLine(g.d, font, x, y, s, c)
}

// centered writes s centered in r at baseline r.y+base.
func (g *Game) centered(r rect, font tinyfont.Fonter, s string, c color.RGBA, base int16) {
	_, w := tinyfont.LineWidth(font, s)
	tinyfont.WriteLine(g.d, font, r.x+(r.w-int16(w))/2, r.y+base, s, c)
}

func (g *Game) refresh() {
	g.dirty = false
	g.drawPayTable()
	for i := range g.round.Hand {
		g.drawCard(i)
	}
	g.drawLabels()
	g.drawButtons()
	g.d.Display()
}

// lit returns the pay table row to light up, if any: the hand already
// made while holding (a hint) or the winning hand after a draw.
func (g *Game) lit() (Hand, color.RGBA) {
	switch g.phase {
	case holding:
		return Evaluate(g.round.Hand), hintText
	case idle:
		if g.dealt {
			return g.result, winColor
		}
	}
	return Nothing, textColor
}

func (g *Game) drawPayTable() {
	l := &g.l
	r := l.pay
	h, hc := g.lit()
	key := strconv.Itoa(g.bet) + "/" + strconv.Itoa(int(h)) + "/" + strconv.Itoa(int(hc.R))
	g.region(r, payColor, key, func() {
		col := rect{l.betColX + int16(g.bet-1)*l.betColW, r.y, l.betColW, r.h}
		g.d.FillRectangle(col.x, col.y, col.w, col.h, betColumn)
		for k := RoyalFlush; k >= JacksOrBetter; k-- {
			y := r.y + int16(RoyalFlush-k)*l.rowH + l.rowBase
			name, pay := textColor, payText
			if k == h {
				name, pay = hc, hc
			}
			g.text(&proggy.TinySZ8pt7b, l.nameX, y, k.String(), name)
			for b := 1; b <= MaxBet; b++ {
				s := strconv.Itoa(Payout(k, b))
				_, w := tinyfont.LineWidth(&proggy.TinySZ8pt7b, s)
				g.text(&proggy.TinySZ8pt7b, l.betColX+int16(b)*l.betColW-l.numRight-int16(w), y, s, pay)
			}
		}
	})
}

func (g *Game) drawCard(i int) {
	r := g.l.cardRect(i)
	r.h = g.l.cardH
	c := g.round.Hand[i]
	up := g.dealt && g.faceUp[i]
	held := up && g.round.Held[i] && g.phase == holding
	key := "back"
	if up {
		key = c.String()
		if held {
			key += "*"
		}
	}
	g.region(r, tableColor, key, func() {
		if up {
			cards.DrawFace(g.d, c, r.x, r.y, r.w, r.h, held)
		} else {
			cards.DrawBack(g.d, r.x, r.y, r.w, r.h)
		}
	})
}

// drawLabels draws the HELD labels while holding, otherwise one line
// across the cards: the result or what to do.
func (g *Game) drawLabels() {
	r := g.l.labels
	var key, msg string
	msgColor := dimText
	switch {
	case g.phase == holding:
		key = "held"
		for _, h := range g.round.Held {
			if h {
				key += "1"
			} else {
				key += "0"
			}
		}
	case g.phase != idle:
		key = "" // cards turning over: an empty line
	case g.refill:
		msg = "100 NEW CREDITS - TAP DEAL"
	case g.credits == 0:
		msg = "OUT OF CREDITS - TAP DEAL FOR 100 MORE"
	case !g.dealt:
		msg = "CHOOSE THE BET AND TAP DEAL"
	case g.won > 0:
		msg = g.result.String() + "   WIN " + strconv.Itoa(g.won)
		msgColor = winColor
	default:
		msg = "NO WIN - TAP DEAL"
	}
	if msg != "" {
		key = msg
	}
	g.region(r, tableColor, key, func() {
		if g.phase == holding {
			for i, h := range g.round.Held {
				if h {
					g.centered(g.l.cardRect(i), &freesans.Bold9pt7b, "HELD", winColor, g.l.cardH+18)
				}
			}
			return
		}
		font, base := g.l.msgFont(msg)
		g.centered(r, font, msg, msgColor, base)
	})
}

func (g *Game) drawButtons() {
	l := &g.l
	deal := "DEAL"
	if g.phase == holding || g.phase == drawing {
		deal = "DRAW"
	}
	buttons := []struct {
		button
		name  string
		on    bool // usable now
		color color.RGBA
	}{
		{l.bet, "BET ONE", g.phase == idle, btnColor},
		{l.max, "MAX BET", g.phase == idle, btnColor},
		{l.deal, deal, g.phase == idle || g.phase == holding, dealColor},
		{l.back, "< GAMES", true, btnColor},
	}
	if g.OnExit == nil {
		buttons = buttons[:3]
	}
	for _, b := range buttons {
		key := b.name
		if b.on {
			key += "+"
		}
		g.region(b.r, barColor, key, func() {
			c, tc := b.color, textColor
			if !b.on {
				c, tc = offColor, dimText
			}
			g.d.FillRectangle(b.r.x, b.r.y, b.r.w, b.r.h, c)
			g.centered(b.r, b.font, b.name, tc, b.base)
		})
	}

	r := l.info
	credit := "CREDIT " + strconv.Itoa(g.credits)
	bet := "BET " + strconv.Itoa(g.bet)
	g.region(r, barColor, credit+"/"+bet, func() {
		g.centered(r, l.creditFont, credit, textColor, l.creditBase)
		g.centered(r, &proggy.TinySZ8pt7b, bet, payText, l.betBase)
	})
}
