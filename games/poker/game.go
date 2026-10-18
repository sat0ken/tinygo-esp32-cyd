// Package poker is "Jacks or Better" video poker for a 480x272 touch
// screen. It depends only on hal: examples/12_poker runs it alone,
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

// Layout for 480x272: the pay table, the five cards with their HELD
// labels, and a button bar.
const (
	tableY   = 2
	rowH     = 11
	nameX    = 12
	betColX  = 172 // x of the bet-1 column of the pay table
	betColW  = 60
	cardW    = 84
	cardH    = 100
	cardGap  = 8
	leftX    = (480 - (5*cardW + 4*cardGap)) / 2
	cardY    = tableY + 9*rowH + 7
	labelY   = cardY + cardH + 2 // the HELD labels / the result line
	labelH   = 22
	barY     = 234
	btnH     = 34
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

func payRect() rect { return rect{4, tableY, 472, 9 * rowH} }

// cardRect includes the HELD label under the card, so a tap there holds
// the card too.
func cardRect(i int) rect {
	return rect{leftX + int16(i)*(cardW+cardGap), cardY, cardW, cardH + 2 + labelH}
}

func backButton() rect { return rect{4, barY + 2, 58, btnH} }
func betButton() rect  { return rect{66, barY + 2, 90, btnH} }
func maxButton() rect  { return rect{160, barY + 2, 90, btnH} }
func infoRect() rect   { return rect{254, barY + 2, 122, btnH} }
func dealButton() rect { return rect{380, barY + 2, 96, btnH} }

// Fonts of the buttons (TestTextsFit checks that the texts fit).
var (
	btnFont  = &freesans.Bold9pt7b
	backFont = &proggy.TinySZ8pt7b
)

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
	return &Game{d: d, t: t, NewSeed: newSeed, lastTap: -1, credits: startCr, bet: 1}
}

// Start draws the whole screen, ready for the first deal.
func (g *Game) Start() {
	g.rand = cards.NewRand(g.NewSeed())
	w, h := g.d.Size()
	g.d.FillRectangle(0, 0, w, h, tableColor)
	g.d.FillRectangle(0, barY, w, h-barY, barColor)
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
	if g.OnExit != nil && backButton().has(x, y) {
		g.OnExit()
		return
	}
	switch g.phase {
	case idle:
		switch {
		case betButton().has(x, y):
			g.bet = g.bet%MaxBet + 1
			g.dirty = true
		case maxButton().has(x, y):
			g.bet = MaxBet
			g.deal()
		case dealButton().has(x, y):
			g.deal()
		}
	case holding:
		if dealButton().has(x, y) {
			g.draw()
			return
		}
		for i := range g.round.Held {
			if cardRect(i).has(x, y) {
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
	r := payRect()
	h, hc := g.lit()
	key := strconv.Itoa(g.bet) + "/" + strconv.Itoa(int(h)) + "/" + strconv.Itoa(int(hc.R))
	g.region(r, payColor, key, func() {
		col := rect{betColX + int16(g.bet-1)*betColW, r.y, betColW, r.h}
		g.d.FillRectangle(col.x, col.y, col.w, col.h, betColumn)
		for k := RoyalFlush; k >= JacksOrBetter; k-- {
			y := r.y + int16(RoyalFlush-k)*rowH + 9
			name, pay := textColor, payText
			if k == h {
				name, pay = hc, hc
			}
			g.text(&proggy.TinySZ8pt7b, nameX, y, k.String(), name)
			for b := 1; b <= MaxBet; b++ {
				s := strconv.Itoa(Payout(k, b))
				_, w := tinyfont.LineWidth(&proggy.TinySZ8pt7b, s)
				g.text(&proggy.TinySZ8pt7b, betColX+int16(b)*betColW-10-int16(w), y, s, pay)
			}
		}
	})
}

func (g *Game) drawCard(i int) {
	r := cardRect(i)
	r.h = cardH
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
	r := rect{0, labelY, 480, labelH}
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
					g.centered(cardRect(i), &freesans.Bold9pt7b, "HELD", winColor, cardH+18)
				}
			}
			return
		}
		g.centered(r, &freesans.Bold9pt7b, msg, msgColor, 17)
	})
}

func (g *Game) drawButtons() {
	type button struct {
		r     rect
		name  string
		on    bool // usable now
		color color.RGBA
		font  tinyfont.Fonter
		base  int16
	}
	deal := "DEAL"
	if g.phase == holding || g.phase == drawing {
		deal = "DRAW"
	}
	buttons := []button{
		{betButton(), "BET ONE", g.phase == idle, btnColor, btnFont, 23},
		{maxButton(), "MAX BET", g.phase == idle, btnColor, btnFont, 23},
		{dealButton(), deal, g.phase == idle || g.phase == holding, dealColor, btnFont, 23},
	}
	if g.OnExit != nil {
		buttons = append(buttons, button{backButton(), "< GAMES", true, btnColor, backFont, 21})
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

	r := infoRect()
	credit := "CREDIT " + strconv.Itoa(g.credits)
	bet := "BET " + strconv.Itoa(g.bet)
	g.region(r, barColor, credit+"/"+bet, func() {
		g.centered(r, btnFont, credit, textColor, 15)
		g.centered(r, &proggy.TinySZ8pt7b, bet, payText, 30)
	})
}
