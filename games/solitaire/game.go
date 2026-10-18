// Package solitaire is klondike solitaire for a 480x272 touch screen. It
// depends only on hal: examples/11_solitaire runs it alone, cmd/games from
// a launcher.
package solitaire

import (
	"image/color"
	"strconv"

	"github.com/sat0ken/tinygo-cyd/games/cards"
	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/romfont/freesans"
	"github.com/sat0ken/tinygo-cyd/romfont/proggy"
	"tinygo.org/x/tinyfont"
)

// Layout for 480x272: a button bar, the stock/waste/foundations row and
// the seven tableau columns.
const (
	barH   = 22
	cardW  = 56
	cardH  = 72
	pitch  = 62 // column pitch
	leftX  = (480 - (7*cardW + 6*(pitch-cardW))) / 2
	topY   = barH + 4
	tabY   = topY + cardH + 4
	fanOff = 12 // waste fan in draw-3 mode
	bottom = 272

	downOff = 5  // vertical offset of face-down cards in a column
	upOff   = 16 // vertical offset of face-up cards (squeezed if needed)
	minUp   = 7  // never less: the rank in the corner must stay readable

	maxUndo    = 200
	tapGap     = 0.15 // seconds; ignore taps closer than this (touch bounce)
	finishStep = 0.08 // seconds between cards when finishing automatically
)

var (
	tableColor = color.RGBA{0, 100, 60, 255}
	barColor   = color.RGBA{0, 60, 36, 255}
	btnColor   = color.RGBA{30, 110, 75, 255}
	textColor  = color.RGBA{255, 255, 255, 255}
	dimText    = color.RGBA{190, 220, 200, 255}
	winColor   = color.RGBA{255, 220, 60, 255}
)

type rect struct{ x, y, w, h int16 }

func (r rect) has(x, y int16) bool { return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h }

func colX(i int) int16 { return leftX + int16(i)*pitch }

func stockRect() rect           { return rect{colX(0), topY, cardW, cardH} }
func wasteRect() rect           { return rect{colX(1), topY, cardW + 2*fanOff, cardH} }
func foundationRect(f int) rect { return rect{colX(3 + f), topY, cardW, cardH} }
func columnRect(i int) rect     { return rect{colX(i), tabY, cardW, bottom - tabY} }

func backButton() rect { return rect{2, 1, 70, barH - 2} }
func newButton() rect  { return rect{76, 1, 60, barH - 2} }
func undoButton() rect { return rect{140, 1, 60, barH - 2} }
func drawButton() rect { return rect{204, 1, 70, barH - 2} }

type phase int

const (
	playing   phase = iota
	finishing       // moving the remaining cards home by itself
	won
)

// Game is klondike solitaire.
type Game struct {
	d hal.Display
	t hal.Touch

	// NewSeed returns the seed for the next deal.
	NewSeed func() uint32
	// OnExit, if set, shows a "< GAMES" button; tapping it calls OnExit.
	OnExit func()

	s       State
	draw    int // 1 or 3 for the next deal
	phase   phase
	history [][]byte
	elapsed float32
	timer   float32

	clock    float32
	lastTap  float32
	wasTouch bool

	drawn map[rect]string // what each region shows, to redraw only changes
	dirty bool            // the table changed since the last refresh
	shown [2]int          // seconds and moves shown in the status bar
}

// NewGame creates a game; call Start to deal.
func NewGame(d hal.Display, t hal.Touch, newSeed func() uint32) *Game {
	return &Game{d: d, t: t, NewSeed: newSeed, draw: 1, lastTap: -1, drawn: map[rect]string{}}
}

// Start deals a new game and draws the whole screen.
func (g *Game) Start() {
	g.s = Deal(g.NewSeed(), g.draw)
	g.history = g.history[:0]
	g.phase = playing
	g.elapsed = 0
	w, h := g.d.Size()
	g.d.FillRectangle(0, 0, w, h, tableColor)
	g.drawn = map[rect]string{}
	g.drawBar()
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
		if g.tapBar(x, y) {
			return
		}
	}

	switch g.phase {
	case playing:
		g.elapsed += dt
		if tap {
			g.tapTable(x, y)
		}
	case finishing:
		g.elapsed += dt
		g.timer += dt
		for g.timer >= finishStep && g.phase == finishing {
			g.timer -= finishStep
			if p, d, ok := g.s.NextHome(); ok {
				g.s.Move(p, d)
				g.dirty = true
			}
			g.checkEnd()
		}
	case won:
		if tap {
			g.Start()
			return
		}
	}
	// Redraw only when something changed: building the comparison keys
	// every frame would make garbage for the GC at 60 frames a second.
	if g.dirty || g.shown != [2]int{int(g.elapsed), g.s.Moves} {
		g.refresh()
	}
}

// tapBar handles the buttons; it returns true if one was tapped.
func (g *Game) tapBar(x, y int16) bool {
	if y >= barH {
		return false
	}
	switch {
	case g.OnExit != nil && backButton().has(x, y):
		g.OnExit()
	case newButton().has(x, y):
		g.Start()
	case undoButton().has(x, y):
		g.undo()
		g.dirty = true
	case drawButton().has(x, y):
		g.draw = 4 - g.draw // 1 <-> 3; a new deal in the new mode
		g.Start()
	default:
		return false
	}
	return true
}

func (g *Game) tapTable(x, y int16) {
	switch {
	case stockRect().has(x, y):
		g.save()
		if !g.s.DrawStock() {
			g.history = g.history[:len(g.history)-1]
		}
		g.dirty = true
		return
	case wasteRect().has(x, y) && y < topY+cardH:
		g.moveFrom(Pos{Kind: WastePile})
		return
	}
	for f := 0; f < 4; f++ {
		if foundationRect(f).has(x, y) {
			g.moveFrom(Pos{Kind: FoundationPile, Pile: f})
			return
		}
	}
	for col := 0; col < 7; col++ {
		if columnRect(col).has(x, y) {
			if i, ok := g.cardAt(col, y); ok {
				g.moveFrom(Pos{Kind: TableauPile, Pile: col, Index: i})
			}
			return
		}
	}
}

// moveFrom moves the tapped card (and the cards on it) where it fits.
func (g *Game) moveFrom(p Pos) {
	d, ok := g.s.AutoDest(p)
	if !ok {
		return
	}
	g.save()
	g.s.Move(p, d)
	g.dirty = true
	g.checkEnd()
}

func (g *Game) checkEnd() {
	switch {
	case g.s.Won():
		g.phase = won
	case g.s.Solved():
		g.phase = finishing
	}
}

func (g *Game) save() {
	if len(g.history) == maxUndo {
		g.history = append(g.history[:0], g.history[1:]...)
	}
	g.history = append(g.history, g.s.encode())
}

func (g *Game) undo() {
	if len(g.history) == 0 || g.phase != playing {
		return
	}
	draw := g.s.Draw
	g.s.decode(g.history[len(g.history)-1])
	g.s.Draw = draw
	g.history = g.history[:len(g.history)-1]
}

// ---- tableau geometry ----

// offsets returns the y of each card of column col, squeezing the face-up
// spacing (and then the face-down one) when the column is too long.
func (g *Game) offsets(col int) []int16 {
	c := &g.s.Tableau[col]
	n := len(c.Cards)
	ys := make([]int16, n)
	down, up := int16(downOff), int16(upOff)
	avail := int16(bottom - tabY - 2 - cardH)
	need := func() int16 { return int16(c.Down)*down + int16(n-c.Down-1)*up }
	if n-c.Down > 1 && need() > avail {
		down = 2
		up = (avail - int16(c.Down)*down) / int16(n-c.Down-1)
		if up > upOff {
			up = upOff
		}
		if up < minUp {
			up = minUp
		}
	}
	y := int16(tabY)
	for i := 0; i < n; i++ {
		ys[i] = y
		if i < c.Down {
			y += down
		} else {
			y += up
		}
	}
	return ys
}

// cardAt returns the face-up card of column col under y.
func (g *Game) cardAt(col int, y int16) (int, bool) {
	c := &g.s.Tableau[col]
	ys := g.offsets(col)
	for i := len(c.Cards) - 1; i >= 0; i-- {
		bottomY := ys[i] + cardH
		if i < len(c.Cards)-1 {
			bottomY = ys[i+1]
		}
		if y >= ys[i] && y < bottomY {
			return i, i >= c.Down
		}
	}
	return 0, false
}

// ---- drawing ----

// region redraws r with draw() if its key changed since the last time.
func (g *Game) region(r rect, key string, draw func()) {
	if g.drawn[r] == key {
		return
	}
	g.d.FillRectangle(r.x, r.y, r.w, r.h, tableColor)
	draw()
	g.drawn[r] = key
}

func cardsKey(p []cards.Card) string {
	b := make([]byte, len(p))
	for i, c := range p {
		b[i] = byte(c) + 1
	}
	return string(b)
}

func (g *Game) refresh() {
	g.dirty = false
	g.shown = [2]int{int(g.elapsed), g.s.Moves}
	sr := stockRect()
	g.region(sr, strconv.Itoa(len(g.s.Stock)), func() {
		if len(g.s.Stock) > 0 {
			cards.DrawBack(g.d, sr.x, sr.y, cardW, cardH)
		} else {
			cards.DrawSlot(g.d, sr.x, sr.y, cardW, cardH, "O") // tap to turn the waste over
		}
	})

	wr := wasteRect()
	shown := g.s.Waste
	if n := len(shown); n > g.s.Draw {
		shown = shown[n-g.s.Draw:]
	}
	if g.s.Draw == 1 && len(shown) > 1 {
		shown = shown[len(shown)-1:]
	}
	g.region(wr, cardsKey(shown), func() {
		if len(shown) == 0 {
			cards.DrawSlot(g.d, wr.x, wr.y, cardW, cardH, "")
		}
		for i, c := range shown {
			cards.DrawFace(g.d, c, wr.x+int16(i)*fanOff, wr.y, cardW, cardH, false)
		}
	})

	for f := 0; f < 4; f++ {
		fr := foundationRect(f)
		p := g.s.Foundation[f]
		key := "-"
		if len(p) > 0 {
			key = cardsKey(p[len(p)-1:])
		}
		g.region(fr, key, func() {
			if len(p) == 0 {
				cards.DrawSlot(g.d, fr.x, fr.y, cardW, cardH, "A")
			} else {
				cards.DrawFace(g.d, p[len(p)-1], fr.x, fr.y, cardW, cardH, false)
			}
		})
	}

	for col := 0; col < 7; col++ {
		cr := columnRect(col)
		c := &g.s.Tableau[col]
		g.region(cr, strconv.Itoa(c.Down)+":"+cardsKey(c.Cards), func() {
			if len(c.Cards) == 0 {
				cards.DrawSlot(g.d, cr.x, cr.y, cardW, cardH, "K")
				return
			}
			ys := g.offsets(col)
			for i, y := range ys {
				switch {
				case i < c.Down:
					cards.DrawBack(g.d, cr.x, y, cardW, cardH)
				case i < len(ys)-1 && ys[i+1]-y < cards.SmallCornerBelow:
					// Squeezed: only the top few pixels show.
					cards.DrawFaceSmall(g.d, c.Cards[i], cr.x, y, cardW, cardH)
				default:
					cards.DrawFace(g.d, c.Cards[i], cr.x, y, cardW, cardH, false)
				}
			}
		})
	}

	g.drawStatus()
	if g.phase == won {
		g.drawWon()
	}
	g.d.Display()
}

func (g *Game) drawBar() {
	w, _ := g.d.Size()
	g.d.FillRectangle(0, 0, w, barH, barColor)
	buttons := []struct {
		r    rect
		name string
	}{{newButton(), "NEW"}, {undoButton(), "UNDO"}, {drawButton(), "DRAW " + strconv.Itoa(g.draw)}}
	if g.OnExit != nil {
		buttons = append(buttons, struct {
			r    rect
			name string
		}{backButton(), "< GAMES"})
	}
	for _, b := range buttons {
		g.d.FillRectangle(b.r.x, b.r.y, b.r.w, b.r.h, btnColor)
		_, tw := tinyfont.LineWidth(&proggy.TinySZ8pt7b, b.name)
		tinyfont.WriteLine(g.d, &proggy.TinySZ8pt7b, b.r.x+(b.r.w-int16(tw))/2, b.r.y+14, b.name, textColor)
	}
}

func clock(sec float32) string {
	s := int(sec)
	two := func(n int) string {
		if n < 10 {
			return "0" + strconv.Itoa(n)
		}
		return strconv.Itoa(n)
	}
	return two(s/60) + ":" + two(s%60)
}

func (g *Game) drawStatus() {
	r := rect{284, 1, 480 - 286, barH - 2}
	text := "MOVES " + strconv.Itoa(g.s.Moves) + "   " + clock(g.elapsed)
	g.region(r, text, func() {
		g.d.FillRectangle(r.x, r.y, r.w, r.h, barColor)
		tinyfont.WriteLine(g.d, &proggy.TinySZ8pt7b, r.x+4, r.y+14, text, dimText)
	})
}

func (g *Game) drawWon() {
	r := rect{100, 100, 280, 92}
	key := "won " + strconv.Itoa(g.s.Moves)
	g.region(r, key, func() {
		g.d.FillRectangle(r.x, r.y, r.w, r.h, barColor)
		g.d.FillRectangle(r.x, r.y, r.w, 3, winColor)
		g.d.FillRectangle(r.x, r.y+r.h-3, r.w, 3, winColor)
		center := func(font tinyfont.Fonter, s string, c color.RGBA, base int16) {
			_, tw := tinyfont.LineWidth(font, s)
			tinyfont.WriteLine(g.d, font, r.x+(r.w-int16(tw))/2, r.y+base, s, c)
		}
		center(&freesans.Bold18pt7b, "YOU WIN!", winColor, 38)
		center(&proggy.TinySZ8pt7b, "MOVES "+strconv.Itoa(g.s.Moves)+"   TIME "+clock(g.elapsed), textColor, 60)
		center(&proggy.TinySZ8pt7b, "tap for a new game", dimText, 78)
	})
}
