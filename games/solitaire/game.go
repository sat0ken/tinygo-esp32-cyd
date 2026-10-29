// Package solitaire is klondike solitaire for a 480x272 or 320x240 touch
// screen. It depends only on hal: examples/11_solitaire runs it alone,
// cmd/games from a launcher.
package solitaire

import (
	"image/color"
	"strconv"

	"github.com/sat0ken/tinygo-cyd/games/cards"
	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/pad"
	"github.com/sat0ken/tinygo-cyd/romfont/freesans"
	"github.com/sat0ken/tinygo-cyd/romfont/proggy"
	"tinygo.org/x/tinyfont"
)

const (
	minUp = 7 // face-up cards in a column never closer: the rank in the corner must stay readable

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
	focusColor = color.RGBA{255, 210, 40, 255}
)

type rect struct{ x, y, w, h int16 }

func (r rect) has(x, y int16) bool { return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h }

// layout is where everything is: a button bar, the stock/waste/
// foundations row and the seven tableau columns.
type layout struct {
	barH                int16
	cardW, cardH, pitch int16 // pitch: from one column to the next
	leftX, topY, tabY   int16
	fanOff              int16 // waste fan in draw-3 mode
	bottom              int16
	downOff, upOff      int16 // vertical offsets of face-down and face-up cards in a column (squeezed if needed)

	back, new, undo, draw, status rect
	won                           rect // the YOU WIN! box
}

// layoutFor returns the layout for a w x h screen: the 480x272 one, or the
// 320x240 one (ESP32-2432S028R) on smaller screens.
func layoutFor(w, h int16) layout {
	const barH = 22
	if w >= 480 && h >= 272 {
		const cardW, cardH, pitch = 56, 72, 62
		return layout{
			barH: barH, cardW: cardW, cardH: cardH, pitch: pitch,
			leftX: (480 - (7*cardW + 6*(pitch-cardW))) / 2, topY: barH + 4, tabY: barH + 4 + cardH + 4,
			fanOff: 12, bottom: h, downOff: 5, upOff: 16,
			back: rect{2, 1, 70, barH - 2}, new: rect{76, 1, 60, barH - 2},
			undo: rect{140, 1, 60, barH - 2}, draw: rect{204, 1, 70, barH - 2},
			status: rect{284, 1, 480 - 286, barH - 2},
			won:    rect{100, 100, 280, 92},
		}
	}
	// Cards of the same shape, smaller; the longest column (6 face down,
	// 13 face up) still fits with the face-up cards minUp apart.
	const cardW, cardH, pitch = 42, 54, 45
	return layout{
		barH: barH, cardW: cardW, cardH: cardH, pitch: pitch,
		leftX: (w - (7*cardW + 6*(pitch-cardW))) / 2, topY: barH + 4, tabY: barH + 4 + cardH + 4,
		fanOff: 12, bottom: h, downOff: 4, upOff: 14,
		back: rect{2, 1, 50, barH - 2}, new: rect{56, 1, 40, barH - 2},
		undo: rect{100, 1, 44, barH - 2}, draw: rect{148, 1, 48, barH - 2},
		status: rect{200, 1, w - 202, barH - 2},
		won:    rect{30, 80, w - 60, 92},
	}
}

func (l *layout) colX(i int) int16 { return l.leftX + int16(i)*l.pitch }

func (l *layout) stockRect() rect { return rect{l.colX(0), l.topY, l.cardW, l.cardH} }
func (l *layout) wasteRect() rect { return rect{l.colX(1), l.topY, l.cardW + 2*l.fanOff, l.cardH} }

func (l *layout) foundationRect(f int) rect { return rect{l.colX(3 + f), l.topY, l.cardW, l.cardH} }

func (l *layout) columnRect(i int) rect { return rect{l.colX(i), l.tabY, l.cardW, l.bottom - l.tabY} }

// target is something that can have the controller's focus.
type target struct {
	kind  targetKind
	pile  int // the button, foundation or column
	index int // in a column: the face-up card, -1 if the column is empty
}

type targetKind int

const (
	buttonTarget targetKind = iota // pile: 0 "< GAMES", 1 NEW, 2 UNDO, 3 DRAW
	stockTarget
	wasteTarget
	foundationTarget
	columnTarget
)

// maxTargets: four buttons, stock, waste, four foundations and at most 13
// face-up cards in each of the seven columns.
const maxTargets = 4 + 2 + 4 + 7*13

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
	l layout

	// NewSeed returns the seed for the next deal.
	NewSeed func() uint32
	// OnExit, if set, shows a "< GAMES" button; tapping it calls OnExit.
	OnExit func()

	// Pad, if set, is the game controller: the stick moves the focus
	// between the buttons, the stock, the waste, the foundations and the
	// face-up cards, A presses what has the focus (a card moves where it
	// fits, with the cards on it), X is UNDO, Y turns over the stock.
	Pad     hal.Pad
	in      pad.Input
	focus   target
	boxes   [maxTargets]pad.Box
	targets [maxTargets]target
	barShow target // the focus when the bar was drawn

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
	w, h := d.Size()
	return &Game{d: d, t: t, l: layoutFor(w, h), NewSeed: newSeed, draw: 1, lastTap: -1, drawn: map[rect]string{},
		focus: target{kind: stockTarget}}
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
	used := g.in.Used
	g.in.Update(g.Pad, dt)
	if g.in.Used && g.updatePad(!used) {
		return
	}
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
	l := &g.l
	if y >= l.barH {
		return false
	}
	switch {
	case g.OnExit != nil && l.back.has(x, y):
		g.OnExit()
	case l.new.has(x, y):
		g.Start()
	case l.undo.has(x, y):
		g.undo()
		g.dirty = true
	case l.draw.has(x, y):
		g.draw = 4 - g.draw // 1 <-> 3; a new deal in the new mode
		g.Start()
	default:
		return false
	}
	return true
}

func (g *Game) tapTable(x, y int16) {
	l := &g.l
	switch {
	case l.stockRect().has(x, y):
		g.save()
		if !g.s.DrawStock() {
			g.history = g.history[:len(g.history)-1]
		}
		g.dirty = true
		return
	case l.wasteRect().has(x, y) && y < l.topY+l.cardH:
		g.moveFrom(Pos{Kind: WastePile})
		return
	}
	for f := 0; f < 4; f++ {
		if l.foundationRect(f).has(x, y) {
			g.moveFrom(Pos{Kind: FoundationPile, Pile: f})
			return
		}
	}
	for col := 0; col < 7; col++ {
		if l.columnRect(col).has(x, y) {
			if i, ok := g.cardAt(col, y); ok {
				g.moveFrom(Pos{Kind: TableauPile, Pile: col, Index: i})
			}
			return
		}
	}
}

// buttonRect is where bar button k (see target) is.
func (g *Game) buttonRect(k int) rect {
	return [4]rect{g.l.back, g.l.new, g.l.undo, g.l.draw}[k]
}

// press does what a tap on t does. It returns true if the game started
// over or was left.
func (g *Game) press(t target) bool {
	switch t.kind {
	case buttonTarget:
		switch t.pile {
		case 0:
			g.OnExit()
		case 1:
			g.Start()
		case 2:
			g.undo()
			g.dirty = true
			return false
		case 3:
			g.draw = 4 - g.draw
			g.Start()
		}
		return true
	case stockTarget:
		g.save()
		if !g.s.DrawStock() {
			g.history = g.history[:len(g.history)-1]
		}
		g.dirty = true
	case wasteTarget:
		g.moveFrom(Pos{Kind: WastePile})
	case foundationTarget:
		g.moveFrom(Pos{Kind: FoundationPile, Pile: t.pile})
	case columnTarget:
		if t.index >= 0 {
			g.moveFrom(Pos{Kind: TableauPile, Pile: t.pile, Index: t.index})
		}
	}
	return false
}

// fixFocus keeps the focus on something that is there: a card that moved
// away leaves it on the column's last card (or its empty place).
func (g *Game) fixFocus() {
	f := &g.focus
	switch {
	case f.kind == buttonTarget && f.pile == 0 && g.OnExit == nil:
		f.pile = 1
	case f.kind == columnTarget:
		c := &g.s.Tableau[f.pile]
		switch {
		case len(c.Cards) == 0:
			f.index = -1
		case f.index >= len(c.Cards) || f.index < 0:
			f.index = len(c.Cards) - 1
		case f.index < c.Down:
			f.index = c.Down
		}
	}
}

// focusRect is the frame of target t: on a column it goes round the card
// and the cards on it.
func (g *Game) focusRect(t target) rect {
	l := &g.l
	switch t.kind {
	case buttonTarget:
		return g.buttonRect(t.pile)
	case stockTarget:
		return l.stockRect()
	case wasteTarget:
		n := int16(len(g.shownWaste()))
		if n == 0 {
			n = 1
		}
		return rect{l.colX(1) + (n-1)*l.fanOff, l.topY, l.cardW, l.cardH}
	case foundationTarget:
		return l.foundationRect(t.pile)
	}
	if t.index < 0 {
		return rect{l.colX(t.pile), l.tabY, l.cardW, l.cardH}
	}
	ys := g.offsets(t.pile)
	return rect{l.colX(t.pile), ys[t.index], l.cardW, ys[len(ys)-1] + l.cardH - ys[t.index]}
}

// listTargets fills g.targets and g.boxes with what can have the focus
// now; a column's face-up card is the part of it that shows.
func (g *Game) listTargets() int {
	n := 0
	add := func(t target, r rect) {
		g.targets[n], g.boxes[n] = t, pad.Box{X: r.x, Y: r.y, W: r.w, H: r.h}
		n++
	}
	for k := 0; k < 4; k++ {
		if k > 0 || g.OnExit != nil {
			add(target{kind: buttonTarget, pile: k}, g.buttonRect(k))
		}
	}
	add(target{kind: stockTarget}, g.l.stockRect())
	add(target{kind: wasteTarget}, g.focusRect(target{kind: wasteTarget}))
	for f := 0; f < 4; f++ {
		add(target{kind: foundationTarget, pile: f}, g.l.foundationRect(f))
	}
	for col := 0; col < 7; col++ {
		c := &g.s.Tableau[col]
		if len(c.Cards) == 0 {
			t := target{kind: columnTarget, pile: col, index: -1}
			add(t, g.focusRect(t))
			continue
		}
		ys := g.offsets(col)
		for i := c.Down; i < len(c.Cards); i++ {
			h := g.l.cardH
			if i < len(c.Cards)-1 {
				h = ys[i+1] - ys[i]
			}
			add(target{kind: columnTarget, pile: col, index: i}, rect{g.l.colX(col), ys[i], g.l.cardW, h})
		}
	}
	return n
}

// updatePad handles the controller (first: it was used for the first
// time) and returns true if the game started over or was left.
func (g *Game) updatePad(first bool) bool {
	in := &g.in
	if g.phase == won {
		if in.Pressed(hal.PadA | hal.PadOK) {
			g.Start()
			return true
		}
		return false
	}
	g.fixFocus()
	if first {
		g.dirty = true
	}
	if d := in.Direction(); d != 0 {
		n := g.listTargets()
		cur := 0
		for i, t := range g.targets[:n] {
			if t == g.focus {
				cur = i
			}
		}
		if f := g.targets[pad.Move(g.boxes[:n], cur, d)]; f != g.focus {
			g.focus = f
			g.dirty = true
		}
	}
	switch {
	case in.Pressed(hal.PadA | hal.PadOK):
		if g.press(g.focus) {
			return true
		}
	case in.Pressed(hal.PadX):
		g.undo()
		g.dirty = true
	case in.Pressed(hal.PadY):
		g.press(target{kind: stockTarget})
	}
	return false
}

// focused reports whether the frame of the focus is on something of kind
// and pile, and returns the key part for its region.
func (g *Game) focused(kind targetKind, pile int) (bool, string) {
	if !g.in.Used || g.phase == won || g.focus.kind != kind || g.focus.pile != pile {
		return false, ""
	}
	return true, "f" + strconv.Itoa(g.focus.index)
}

func (g *Game) frame(r rect) { pad.Frame(g.d, r.x, r.y, r.w, r.h, 2, focusColor) }

// shownWaste returns the waste cards that show.
func (g *Game) shownWaste() []cards.Card {
	shown := g.s.Waste
	if n := len(shown); n > g.s.Draw {
		shown = shown[n-g.s.Draw:]
	}
	if g.s.Draw == 1 && len(shown) > 1 {
		shown = shown[len(shown)-1:]
	}
	return shown
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
	l := &g.l
	c := &g.s.Tableau[col]
	n := len(c.Cards)
	ys := make([]int16, n)
	down, up := l.downOff, l.upOff
	avail := l.bottom - l.tabY - 2 - l.cardH
	need := func() int16 { return int16(c.Down)*down + int16(n-c.Down-1)*up }
	if n-c.Down > 1 && need() > avail {
		down = 2
		up = (avail - int16(c.Down)*down) / int16(n-c.Down-1)
		if up > l.upOff {
			up = l.upOff
		}
		if up < minUp {
			up = minUp
		}
	}
	y := l.tabY
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
		bottomY := ys[i] + g.l.cardH
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
	g.fixFocus()
	l := &g.l
	cardW, cardH := l.cardW, l.cardH
	if on, _ := g.focused(buttonTarget, g.focus.pile); on != (g.barShow.kind == buttonTarget) || (on && g.barShow != g.focus) {
		g.drawBar() // the focus came to, left or moved in the bar
	}
	sr := l.stockRect()
	on, fkey := g.focused(stockTarget, 0)
	g.region(sr, strconv.Itoa(len(g.s.Stock))+fkey, func() {
		if len(g.s.Stock) > 0 {
			cards.DrawBack(g.d, sr.x, sr.y, cardW, cardH)
		} else {
			cards.DrawSlot(g.d, sr.x, sr.y, cardW, cardH, "O") // tap to turn the waste over
		}
		if on {
			g.frame(sr)
		}
	})

	wr := l.wasteRect()
	shown := g.shownWaste()
	wasteOn, wkey := g.focused(wasteTarget, 0)
	g.region(wr, cardsKey(shown)+wkey, func() {
		if len(shown) == 0 {
			cards.DrawSlot(g.d, wr.x, wr.y, cardW, cardH, "")
		}
		for i, c := range shown {
			cards.DrawFace(g.d, c, wr.x+int16(i)*l.fanOff, wr.y, cardW, cardH, false)
		}
		if wasteOn {
			g.frame(g.focusRect(g.focus))
		}
	})

	for f := 0; f < 4; f++ {
		fr := l.foundationRect(f)
		p := g.s.Foundation[f]
		key := "-"
		if len(p) > 0 {
			key = cardsKey(p[len(p)-1:])
		}
		on, fkey := g.focused(foundationTarget, f)
		g.region(fr, key+fkey, func() {
			if len(p) == 0 {
				cards.DrawSlot(g.d, fr.x, fr.y, cardW, cardH, "A")
			} else {
				cards.DrawFace(g.d, p[len(p)-1], fr.x, fr.y, cardW, cardH, false)
			}
			if on {
				g.frame(fr)
			}
		})
	}

	for col := 0; col < 7; col++ {
		cr := l.columnRect(col)
		c := &g.s.Tableau[col]
		on, fkey := g.focused(columnTarget, col)
		g.region(cr, strconv.Itoa(c.Down)+":"+cardsKey(c.Cards)+fkey, func() {
			if on {
				defer g.frame(g.focusRect(g.focus))
			}
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
	l := &g.l
	g.d.FillRectangle(0, 0, w, l.barH, barColor)
	buttons := []struct {
		r    rect
		name string
	}{{l.new, "NEW"}, {l.undo, "UNDO"}, {l.draw, "DRAW " + strconv.Itoa(g.draw)}, {l.back, "< GAMES"}}
	if g.OnExit == nil {
		buttons = buttons[:3]
	}
	for _, b := range buttons {
		g.d.FillRectangle(b.r.x, b.r.y, b.r.w, b.r.h, btnColor)
		_, tw := tinyfont.LineWidth(&proggy.TinySZ8pt7b, b.name)
		tinyfont.WriteLine(g.d, &proggy.TinySZ8pt7b, b.r.x+(b.r.w-int16(tw))/2, b.r.y+14, b.name, textColor)
	}
	g.barShow = target{kind: -1}
	if on, _ := g.focused(buttonTarget, g.focus.pile); on {
		g.frame(g.buttonRect(g.focus.pile))
		g.barShow = g.focus
	}
	delete(g.drawn, g.l.status) // the bar covers the status
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
	r := g.l.status
	text := "MOVES " + strconv.Itoa(g.s.Moves) + "   " + clock(g.elapsed)
	g.region(r, text, func() {
		g.d.FillRectangle(r.x, r.y, r.w, r.h, barColor)
		tinyfont.WriteLine(g.d, &proggy.TinySZ8pt7b, r.x+4, r.y+14, text, dimText)
	})
}

func (g *Game) drawWon() {
	r := g.l.won
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
