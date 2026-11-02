// Package mahjong is riichi mahjong for four: the player and three CPUs, on a
// 480x272 or 320x240 touch screen. It depends only on hal: examples/34_mahjong
// runs it alone, cmd/games from a launcher.
//
// The rules are in table.go, score.go and shanten.go, with no screen; the CPU
// is in cpu.go.
//
// The screen is in Japanese (jptext.go) but for the "< GAMES" button.
//
// Tap a tile to lift it, tap it again to discard it. With the mini joystick
// module the stick moves between the tiles and the buttons, A discards the
// tile or presses the button, B passes.
package mahjong

import (
	"image/color"
	"strconv"
	"time"

	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/pad"
	"github.com/sat0ken/tinygo-cyd/romfont/proggy"
	"tinygo.org/x/tinyfont"
)

const (
	tapGap     = 0.15 // seconds; ignore taps closer than this (resistive touch bounces)
	drawDelay  = 0.25 // seconds before a CPU draws
	turnDelay  = 0.45 // seconds a CPU seems to think
	claimDelay = 0.35 // seconds before the CPUs answer to a discard
	overDelay  = 0.8  // seconds the table stays after the end of a hand
	lockDelay  = 0.6  // seconds before a tile is discarded for a player in riichi
	armTime    = 3.0  // seconds QUIT? waits for the second press
)

var (
	feltColor  = color.RGBA{14, 84, 60, 255}
	bgColor    = color.RGBA{18, 22, 32, 255}
	centerBg   = color.RGBA{10, 58, 42, 255}
	textColor  = color.RGBA{255, 255, 255, 255}
	dimText    = color.RGBA{170, 190, 180, 255}
	turnColor  = color.RGBA{190, 140, 30, 255}
	panelColor = color.RGBA{10, 58, 42, 255}
	btnColor   = color.RGBA{70, 78, 96, 255}
	winColor   = color.RGBA{190, 50, 50, 255}
	goColor    = color.RGBA{60, 140, 210, 255}
	focusColor = color.RGBA{255, 210, 40, 255}
	lastColor  = color.RGBA{240, 70, 50, 255}
	riichiTag  = color.RGBA{40, 200, 240, 255}
	plusColor  = color.RGBA{110, 230, 130, 255}
	minusColor = color.RGBA{250, 120, 110, 255}
)

const human = 0 // the player's seat; 1 right, 2 across, 3 left

var seatNames = [4]string{"自分", "下家", "対面", "上家"}

type rect struct{ x, y, w, h int16 }

func (r rect) has(x, y int16) bool { return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h }

// layout is where everything is. The discards of the four seats make a cross
// around the middle, the hand is at the bottom.
type layout struct {
	tw, th     int16 // a tile of the hand
	rw, rh     int16 // a discarded or called tile
	handY      int16
	lift       int16
	gap        int16 // between the hand and the tile just drawn
	river      [4][2]int16
	melds      [4][2]int16 // the called tiles of the others
	panel      [4]rect
	center     rect
	buttons    rect
	btnW, btnH int16
	btnCols    int
	quit       rect

	titleTop, titleScale, aboutTop int16
	short, long                    rect  // the buttons of the menu
	stickLong, stickThick          int16 // a riichi stick

	smallLines bool // the result in fewer lines
}

func layoutFor(w, h int16) layout {
	if w >= 480 && h >= 272 {
		return layout{
			tw: 30, th: 40, rw: 15, rh: 20, handY: 232, lift: 4, gap: 6,
			river:  [4][2]int16{{195, 144}, {290, 73}, {195, 2}, {97, 73}},
			melds:  [4][2]int16{{0, 0}, {386, 73}, {292, 2}, {2, 73}},
			panel:  [4]rect{{100, 155, 92, 30}, {382, 44, 96, 27}, {100, 2, 92, 30}, {2, 44, 96, 27}},
			center: rect{192, 83, 96, 60}, buttons: rect{290, 156, 189, 70}, btnW: 60, btnH: 33, btnCols: 3,
			quit:     rect{100, 192, 88, 30},
			titleTop: 44, titleScale: 6, aboutTop: 104, stickLong: 56, stickThick: 4,
			short: rect{50, 140, 180, 70}, long: rect{250, 140, 180, 70},
		}
	}
	return layout{
		tw: 21, th: 28, rw: 11, rh: 14, handY: 212, lift: 3, gap: 4,
		river:  [4][2]int16{{125, 107}, {194, 39}, {125, 1}, {56, 39}},
		melds:  [4][2]int16{{0, 0}, {264, 39}, {194, 1}, {1, 39}},
		panel:  [4]rect{{60, 97, 66, 22}, {262, 97, 57, 22}, {60, 1, 66, 22}, {1, 97, 58, 22}},
		center: rect{127, 58, 66, 48}, buttons: rect{2, 166, 316, 40}, btnW: 60, btnH: 38, btnCols: 5,
		quit:     rect{194, 98, 60, 24},
		titleTop: 50, titleScale: 4, aboutTop: 92, stickLong: 40, stickThick: 3,
		short: rect{8, 125, 148, 74}, long: rect{162, 125, 150, 74},
		smallLines: true,
	}
}

func backButton() rect { return rect{8, 8, 84, 28} }

// The screens.
const (
	scrMenu = iota
	scrTable
	scrResult
	scrOver
)

// What the buttons of the table do.
const (
	actNone = iota
	actTsumo
	actRiichi
	actKan
	actAbort
	actRon
	actPon
	actChi
	actPass
	actCancel
	actChiLow  // choose a run; tile: the lowest
	actKanTile // choose a kan; tile: which
)

type button struct {
	label string
	act   uint8
	tile  Tile
}

// What the player is choosing.
const (
	modeNormal = iota
	modeRiichi
	modeChi
	modeKan
)

// look is how a tile on the screen looks; only what changes is drawn.
type look struct {
	tile   Tile
	flags  uint8
	offset int16
}

const (
	flagDim uint8 = 1 << iota
	flagLast
	flagRiichi // on its side
	flagLift
	flagFocus
)

var noLook = look{tile: noTile, flags: 255}

// Game is the mahjong game. It depends only on hal.
type Game struct {
	d hal.Display
	t hal.Touch
	l layout

	// OnExit, if set, shows a "< GAMES" button on the menu; tapping it calls
	// OnExit (used by the cmd/games launcher).
	OnExit func()

	// Pad, if set, is the game controller.
	Pad hal.Pad
	in  pad.Input

	seed uint32
	tb   *Table
	scr  int

	mode      int
	lifted    int // the slot lifted, -1 if none
	focus     int // the controller's focus: slots, then buttons, then QUIT
	menuFocus int
	buttons   [5]button
	nButtons  int
	drawnBtns string
	riichi    Mask
	kans      Mask
	armed     float32 // > 0: QUIT? is waiting for the second press

	slots  [14]Tile
	nSlots int
	nHand  int // the slots before the tile just drawn

	drawnRiver  [4][24]look
	drawnSlots  [14]look
	drawnHandAt uint32 // the melds and the length of the hand when it was drawn
	drawnMelds  [4]uint32
	drawnPanels [4]string
	drawnCenter string
	drawnQuit   string

	page  int // the win shown on the result screen
	timer float32
	wait  float32 // the result screen ignores taps at first

	slowest int64 // the longest a CPU has thought, in ms (for the log)

	clock    float32
	lastTap  float32
	wasTouch bool
}

// NewGame makes the game; call DrawMenu to show the menu. seed makes the
// walls repeatable.
func NewGame(d hal.Display, t hal.Touch, seed uint32) *Game {
	if seed == 0 {
		seed = 1
	}
	w, h := d.Size()
	return &Game{d: d, t: t, l: layoutFor(w, h), seed: seed, lastTap: -1, lifted: -1}
}

// ---- drawing helpers ----

var smallFont = &proggy.TinySZ8pt7b // "< GAMES"; the rest is in Japanese

func (g *Game) fill(r rect, c color.RGBA) { g.d.FillRectangle(r.x, r.y, r.w, r.h, c) }

func (g *Game) textCentered(r rect, font tinyfont.Fonter, s string, c color.RGBA, baseline int16) {
	_, w := tinyfont.LineWidth(font, s)
	tinyfont.WriteLine(g.d, font, r.x+(r.w-int16(w))/2, r.y+baseline, s, c)
}

// jp writes Japanese with the top at (x, y).
func (g *Game) jp(x, y int16, s string, scale int16, c color.RGBA) {
	jpText(g.d, x, y, s, scale, c)
}

// jpCentered writes Japanese in the middle of r, the top at r.y+top.
func (g *Game) jpCentered(r rect, s string, scale int16, c color.RGBA, top int16) {
	g.jp(r.x+(r.w-jpWidth(s, scale))/2, r.y+top, s, scale, c)
}

// ---- menu ----

// DrawMenu draws the first screen.
func (g *Game) DrawMenu() {
	g.scr = scrMenu
	g.tb = nil
	w, h := g.d.Size()
	g.fill(rect{0, 0, w, h}, bgColor)
	g.jpCentered(rect{0, 0, w, 0}, "麻雀", g.l.titleScale, textColor, g.l.titleTop)
	g.jpCentered(rect{0, 0, w, 0}, "リーチ麻雀  自分とコンピュータ3人", 1, dimText, g.l.aboutTop)
	for k := range g.menuBoxes() {
		g.drawMenuItem(k)
	}
	g.d.Display()
}

func (g *Game) menuBoxes() []pad.Box {
	var b [3]pad.Box
	add := func(n int, r rect) { b[n] = pad.Box{X: r.x, Y: r.y, W: r.w, H: r.h} }
	add(0, g.l.short)
	add(1, g.l.long)
	if g.OnExit == nil {
		return b[:2]
	}
	add(2, backButton())
	return b[:3]
}

func (g *Game) drawMenuItem(k int) {
	var r rect
	switch k {
	case 0, 1:
		r = g.l.short
		name, about := "東風戦", "東場のみ"
		if k == 1 {
			r = g.l.long
			name, about = "半荘戦", "東場と南場"
		}
		g.fill(r, goColor)
		g.jpCentered(r, name, 2, textColor, r.h/2-16)
		g.jpCentered(r, about, 1, textColor, r.h/2+8)
	default:
		r = backButton()
		g.fill(r, btnColor)
		g.textCentered(r, smallFont, "< GAMES", textColor, 18)
	}
	if g.in.Used && k == g.menuFocus {
		pad.Frame(g.d, r.x, r.y, r.w, r.h, 2, focusColor)
	}
}

func (g *Game) updateMenuPad(first bool) {
	in := &g.in
	boxes := g.menuBoxes()
	if first {
		g.drawMenuItem(g.menuFocus)
		g.d.Display()
	}
	if d := in.Direction(); d != 0 {
		if f := pad.Move(boxes, g.menuFocus, d); f != g.menuFocus {
			old := g.menuFocus
			g.menuFocus = f
			g.drawMenuItem(old)
			g.drawMenuItem(f)
			g.d.Display()
		}
	}
	switch {
	case in.Pressed(hal.PadB) && g.OnExit != nil, in.Pressed(hal.PadA|hal.PadOK) && g.menuFocus == 2:
		g.OnExit()
	case in.Pressed(hal.PadA | hal.PadOK):
		g.start(g.menuFocus == 1)
	}
}

func (g *Game) tapMenu(x, y int16) {
	switch {
	case g.OnExit != nil && backButton().has(x, y):
		g.OnExit()
	case g.l.short.has(x, y):
		g.start(false)
	case g.l.long.has(x, y):
		g.start(true)
	}
}

// ---- the flow of the game ----

func (g *Game) start(long bool) {
	g.seed = g.seed*2654435761 + 1
	g.tb = NewTable(g.seed, long)
	g.lastTap = g.clock
	g.showTable()
}

// showTable draws the table from nothing.
func (g *Game) showTable() {
	g.scr = scrTable
	w, h := g.d.Size()
	g.fill(rect{0, 0, w, h}, feltColor)
	for s := range g.drawnRiver {
		for i := range g.drawnRiver[s] {
			g.drawnRiver[s][i] = noLook
		}
		g.drawnMelds[s] = 1 << 31
		g.drawnPanels[s] = ""
	}
	for i := range g.drawnSlots {
		g.drawnSlots[i] = noLook
	}
	g.drawnHandAt = 1 << 31
	g.drawnCenter, g.drawnBtns, g.drawnQuit = "", "\x00", ""
	g.changed()
}

// changed is called after every action: the timer starts again, the
// player's choices are worked out and the screen follows.
func (g *Game) changed() {
	t := g.tb
	g.timer = 0
	g.mode = modeNormal
	g.lifted = -1
	g.riichi, g.kans = 0, 0
	if t.Phase == phaseTurn && t.Turn == human {
		g.riichi = t.RiichiTiles()
		g.kans = t.KanTiles()
	}
	g.setButtons()
	g.buildSlots()
	if t.Phase == phaseTurn && t.Turn == human {
		g.focus = g.nSlots - 1 // the tile just drawn
		if g.in.Used {
			g.lifted = g.focus
		}
	} else if g.nButtons > 0 {
		g.focus = 14
	} else {
		g.focus = max(g.nSlots-1, 0) // not on QUIT: A may be pressed while waiting
	}
	g.refresh()
}

// setButtons works out the buttons from the phase and the mode.
func (g *Game) setButtons() {
	t := g.tb
	n := 0
	add := func(label string, act uint8, k Tile) {
		g.buttons[n] = button{label, act, k}
		n++
	}
	switch {
	case g.mode == modeRiichi:
		add("戻る", actCancel, 0)
	case g.mode == modeKan:
		for k := Tile(0); k < numKinds && n < 4; k++ {
			if g.kans.has(k) {
				add("", actKanTile, k)
			}
		}
		add("戻る", actCancel, 0)
	case g.mode == modeChi:
		o := t.OptsOf(human)
		for i := 0; i < o.NChi; i++ {
			add("", actChiLow, o.Chi[i])
		}
		add("戻る", actCancel, 0)
	case t.Phase == phaseTurn && t.Turn == human:
		if t.CanTsumo() {
			add("ツモ", actTsumo, 0)
		}
		if g.riichi != 0 {
			add("リーチ", actRiichi, 0)
		}
		if g.kans != 0 {
			add("カン", actKan, 0)
		}
		if t.CanAbort() {
			add("九種", actAbort, 0)
		}
	case t.Phase == phaseClaims && t.Waiting(human):
		o := t.OptsOf(human)
		if o.Ron {
			add("ロン", actRon, 0)
		}
		if o.Pon {
			add("ポン", actPon, 0)
		}
		if o.Kan {
			add("カン", actKan, 0)
		}
		if o.NChi > 0 {
			add("チー", actChi, 0)
		}
		add("パス", actPass, 0)
	}
	g.nButtons = n
}

// buildSlots lays out the player's hand: sorted, the tile just drawn last.
func (g *Game) buildSlots() {
	p := &g.tb.P[human]
	h := p.Hand
	if p.Drawn != noTile {
		h[p.Drawn]--
	}
	n := 0
	for k := Tile(0); k < numKinds; k++ {
		for c := uint8(0); c < h[k]; c++ {
			g.slots[n] = k
			n++
		}
	}
	g.nHand = n
	if p.Drawn != noTile {
		g.slots[n] = p.Drawn
		n++
	}
	g.nSlots = n
	if g.lifted >= n {
		g.lifted = -1
	}
}

// slotAllowed reports that the tile of a slot can be discarded now.
func (g *Game) slotAllowed(i int) bool {
	t := g.tb
	if t.Phase != phaseTurn || t.Turn != human || i >= g.nSlots {
		return false
	}
	k := g.slots[i]
	if g.mode == modeRiichi {
		return g.riichi.has(k)
	}
	if g.mode != modeNormal {
		return false
	}
	return t.CanDiscard(k)
}

// discardSlot discards the tile of a slot.
func (g *Game) discardSlot(i int) {
	riichi := g.mode == modeRiichi
	g.tb.DiscardTile(g.slots[i], riichi)
	g.changed()
}

func (g *Game) press(b *button) {
	t := g.tb
	switch b.act {
	case actTsumo:
		t.Tsumo()
	case actRiichi:
		g.mode = modeRiichi
		g.lifted = -1
		g.setButtons()
		g.focus = g.firstAllowed()
		g.refresh()
		return
	case actAbort:
		t.Abort()
	case actKan:
		if t.Phase == phaseClaims {
			t.AnswerClaim(human, Claim{Kind: claimKan})
			break
		}
		if g.kans&(g.kans-1) != 0 { // more than one
			g.mode = modeKan
			g.setButtons()
			g.focus = 14
			g.refresh()
			return
		}
		for k := Tile(0); k < numKinds; k++ {
			if g.kans.has(k) {
				t.Kan(k)
				break
			}
		}
	case actKanTile:
		t.Kan(b.tile)
	case actRon:
		t.AnswerClaim(human, Claim{Kind: claimRon})
	case actPon:
		t.AnswerClaim(human, Claim{Kind: claimPon})
	case actChi:
		o := t.OptsOf(human)
		if o.NChi == 1 {
			t.AnswerClaim(human, Claim{Kind: claimChi, Low: o.Chi[0]})
			break
		}
		g.mode = modeChi
		g.setButtons()
		g.focus = 14
		g.refresh()
		return
	case actChiLow:
		t.AnswerClaim(human, Claim{Kind: claimChi, Low: b.tile})
	case actPass:
		t.AnswerClaim(human, Claim{Kind: claimPass})
	case actCancel:
		g.mode = modeNormal
		g.lifted = -1
		g.setButtons()
		if t.Phase == phaseTurn {
			g.focus = g.nSlots - 1
		} else {
			g.focus = 14
		}
		g.refresh()
		return
	}
	g.changed()
}

func (g *Game) firstAllowed() int {
	for i := g.nSlots - 1; i >= 0; i-- {
		if g.slotAllowed(i) {
			return i
		}
	}
	return 0
}

// step advances the game when it is not the player's move.
func (g *Game) step(dt float32) {
	t := g.tb
	g.timer += dt
	switch t.Phase {
	case phaseDraw:
		delay := float32(drawDelay)
		if t.Turn == human {
			delay = 0.1
		}
		if g.timer >= delay {
			t.Draw()
			g.changed()
		}
	case phaseTurn:
		if t.Turn != human {
			if g.timer >= turnDelay {
				start := time.Now()
				t.CPUTurn()
				if ms := time.Since(start).Milliseconds(); ms > g.slowest {
					g.slowest = ms
					println("mahjong: slowest CPU turn so far:", ms, "ms")
				}
				g.changed()
			}
			return
		}
		// A player in riichi with nothing to decide discards the tile drawn.
		if t.P[human].Riichi && g.nButtons == 0 && g.timer >= lockDelay {
			t.DiscardTile(t.P[human].Drawn, false)
			g.changed()
		}
	case phaseClaims:
		if g.timer < claimDelay {
			return
		}
		answered := false
		for s := range t.P {
			if s != human && t.Waiting(s) {
				t.CPUAnswer(s)
				answered = true
			}
		}
		if answered {
			g.changed()
		}
	case phaseOver:
		if g.timer >= overDelay {
			g.page = 0
			g.showResult()
		}
	}
}

// nextPage goes on from the result screen.
func (g *Game) nextPage() {
	t := g.tb
	if t.Result.Kind == endWin && g.page+1 < t.Result.NWins {
		g.page++
		g.showResult()
		return
	}
	t.NextHand()
	if t.GameOver {
		g.showOver()
		return
	}
	g.showTable()
}

// Update reads the touch panel and the controller and advances the game by
// dt seconds.
func (g *Game) Update(dt float32) {
	g.clock += dt
	used := g.in.Used
	g.in.Update(g.Pad, dt)
	first := g.in.Used && !used

	x, y, pressed := g.t.ReadTouch()
	tap := pressed && !g.wasTouch && g.clock-g.lastTap >= tapGap
	g.wasTouch = pressed
	if tap {
		g.lastTap = g.clock
	}

	switch g.scr {
	case scrMenu:
		if g.in.Used {
			g.updateMenuPad(first)
			if g.scr != scrMenu || g.tb != nil {
				return
			}
		}
		if tap {
			g.tapMenu(x, y)
		}
	case scrTable:
		if g.armed > 0 {
			g.armed -= dt
			if g.armed <= 0 {
				g.refresh()
			}
		}
		if g.in.Used && g.updatePad(first) {
			return
		}
		if tap && g.tapTable(x, y) {
			return
		}
		g.step(dt)
	case scrResult:
		g.wait -= dt
		if g.wait <= 0 && (tap || g.in.Pressed(hal.PadA|hal.PadOK)) {
			g.nextPage()
		}
	case scrOver:
		g.wait -= dt
		if g.wait <= 0 && (tap || g.in.Pressed(hal.PadA|hal.PadOK)) {
			g.DrawMenu()
		}
	}
}

// pressQuit asks first and leaves the game at the second press. It returns
// true when the game has been left.
func (g *Game) pressQuit() bool {
	if g.armed > 0 {
		g.armed = 0
		g.DrawMenu()
		return true
	}
	g.armed = armTime
	g.refresh()
	return false
}

// tapTable handles a tap on the table; true if the table is gone.
func (g *Game) tapTable(x, y int16) bool {
	if g.l.quit.has(x, y) {
		return g.pressQuit()
	}
	for i := 0; i < g.nButtons; i++ {
		if g.buttonRect(i).has(x, y) {
			g.press(&g.buttons[i])
			return false
		}
	}
	for i := 0; i < g.nSlots; i++ {
		r := g.slotRect(i)
		r.y -= g.l.lift
		r.h += g.l.lift
		if !r.has(x, y) || !g.slotAllowed(i) {
			continue
		}
		if g.lifted == i {
			g.discardSlot(i)
			return false
		}
		g.lifted, g.focus = i, i
		g.refresh()
		return false
	}
	return false
}

// updatePad handles the controller; true if the table is gone.
func (g *Game) updatePad(first bool) bool {
	in := &g.in
	if d := in.Direction(); d != 0 {
		var boxes [20]pad.Box
		n := 0
		var index [20]int
		add := func(r rect, id int) {
			boxes[n] = pad.Box{X: r.x, Y: r.y, W: r.w, H: r.h}
			index[n] = id
			n++
		}
		cur := 0
		for i := 0; i < g.nSlots; i++ {
			add(g.slotRect(i), i)
		}
		for i := 0; i < g.nButtons; i++ {
			add(g.buttonRect(i), 14+i)
		}
		add(g.l.quit, 19)
		for i := 0; i < n; i++ {
			if index[i] == g.focus {
				cur = i
			}
		}
		g.focus = index[pad.Move(boxes[:n], cur, d)]
		first = true
	}
	if g.focus >= 14 && g.focus < 19 && g.focus-14 >= g.nButtons {
		g.focus = max(g.nSlots-1, 0)
	}
	if g.focus < 14 && g.focus >= g.nSlots && g.nSlots > 0 {
		g.focus = g.nSlots - 1
	}
	switch {
	case in.Pressed(hal.PadB):
		for i := 0; i < g.nButtons; i++ {
			if a := g.buttons[i].act; a == actPass || a == actCancel {
				g.press(&g.buttons[i])
				return false
			}
		}
	case in.Pressed(hal.PadA | hal.PadOK):
		switch {
		case g.focus == 19:
			return g.pressQuit()
		case g.focus >= 14:
			g.press(&g.buttons[g.focus-14])
			return false
		case g.slotAllowed(g.focus):
			g.discardSlot(g.focus)
			return false
		}
	}
	if first {
		if g.focus < 14 && g.slotAllowed(g.focus) {
			g.lifted = g.focus
		} else {
			g.lifted = -1
		}
		g.refresh()
	}
	return false
}

// ---- the table ----

func (g *Game) slotRect(i int) rect {
	l := &g.l
	x := int16(2) + int16(i)*l.tw
	if i >= g.nHand {
		x += l.gap
	}
	return rect{x, l.handY, l.tw, l.th}
}

func (g *Game) buttonRect(i int) rect {
	l := &g.l
	r := l.buttons
	return rect{r.x + int16(i%l.btnCols)*(l.btnW+3), r.y + int16(i/l.btnCols)*(l.btnH+3), l.btnW, l.btnH}
}

func (g *Game) refresh() {
	for s := range g.tb.P {
		g.drawRiver(s)
		g.drawMelds(s)
		g.drawPanel(s)
	}
	g.drawCenter()
	g.drawHand()
	g.drawButtons()
	g.drawQuit()
	g.d.Display()
}

// riverPos is where discard i of a seat goes; offset is how far a tile on its
// side earlier in the row pushes it.
func (g *Game) riverPos(seat, i int, offset int16) (x, y int16) {
	l := &g.l
	return l.river[seat][0] + int16(i%6)*l.rw + offset, l.river[seat][1] + int16(i/6)*l.rh
}

// riverRect is the room a discard takes.
func (g *Game) riverRect(seat, i int, v look) rect {
	l := &g.l
	x, y := g.riverPos(seat, i, v.offset)
	if v.flags&flagRiichi != 0 {
		return rect{x, y + l.rh - l.rw, l.rh, l.rw} // on its side, at the bottom of the row
	}
	return rect{x, y, l.rw, l.rh}
}

func (g *Game) drawRiver(seat int) {
	t, l := g.tb, &g.l
	p := &t.P[seat]
	var looks [24]look
	for i := range looks {
		looks[i] = look{tile: noTile}
	}
	// With more than fit, the last ones.
	n := 0
	for i := 0; i < p.NRiv; i++ {
		if !p.River[i].Called {
			n++
		}
	}
	skip := max(n-24, 0)
	slot := 0
	side := false // the riichi tile is to come: it lies on its side, or if
	// somebody took it, the next one does
	for i := 0; i < p.NRiv; i++ {
		r := &p.River[i]
		side = side || r.Riichi
		if r.Called {
			continue
		}
		if skip > 0 {
			skip--
			side = false
			continue
		}
		v := look{tile: r.Tile}
		if side {
			v.flags |= flagRiichi
			side = false
		}
		if i == p.NRiv-1 && t.Discarder == seat {
			v.flags |= flagLast
		}
		looks[slot] = v
		slot++
	}
	var offset int16
	for i := range looks {
		if i%6 == 0 {
			offset = 0
		}
		looks[i].offset = offset
		if looks[i].flags&flagRiichi != 0 {
			offset += l.rh - l.rw
		}
	}
	// First clear what changed, then draw: with a tile on its side the old
	// and the new places can overlap.
	for i, v := range looks {
		if old := g.drawnRiver[seat][i]; v != old && old.tile != noTile {
			g.fill(g.riverRect(seat, i, old), feltColor)
		}
	}
	for i, v := range looks {
		old := g.drawnRiver[seat][i]
		if v == old && !g.riverTouched(seat, i, &looks) {
			continue
		}
		g.drawnRiver[seat][i] = v
		if v.tile == noTile {
			continue
		}
		r := g.riverRect(seat, i, v)
		if v.flags&flagRiichi != 0 {
			drawSideways(g.d, v.tile, r.x, r.y, l.rw, l.rh)
		} else {
			drawTile(g.d, v.tile, r.x, r.y, l.rw, l.rh, false)
		}
		if v.flags&flagLast != 0 {
			pad.Frame(g.d, r.x, r.y, r.w, r.h, 1, lastColor)
		}
	}
}

// riverTouched reports that clearing another tile may have erased this one.
func (g *Game) riverTouched(seat, i int, looks *[24]look) bool {
	row := i / 6
	for j := row * 6; j < row*6+6; j++ {
		if j != i && looks[j] != g.drawnRiver[seat][j] && looks[j].offset != looks[i].offset {
			return true
		}
	}
	return false
}

// meldsKey changes when the melds change.
func meldsKey(p *Player) uint32 {
	k := uint32(p.NMeld)
	for i := 0; i < p.NMeld; i++ {
		k = k*5 + uint32(p.Melds[i].Kind)
	}
	return k
}

// drawMeld draws a meld at (x, y) and returns the x after it. The tile taken
// from somebody comes first, with a red edge at the bottom.
func (g *Game) drawMeld(m *Meld, x, y int16) int16 {
	l := &g.l
	var tiles [4]Tile
	n := 0
	switch m.Kind {
	case chiMeld:
		// The tile called comes first.
		tiles[0] = m.Called
		for k := m.Tile; k <= m.Tile+2; k++ {
			if k != m.Called {
				n++
				tiles[n] = k
			}
		}
		n = 3
	case ponMeld:
		tiles[0], tiles[1], tiles[2] = m.Tile, m.Tile, m.Tile
		n = 3
	default:
		tiles[0], tiles[1], tiles[2], tiles[3] = m.Tile, m.Tile, m.Tile, m.Tile
		n = 4
	}
	for i := 0; i < n; i++ {
		tx := x + int16(i)*l.rw
		if m.Kind == closedKan && (i == 0 || i == 3) {
			drawBack(g.d, tx, y, l.rw, l.rh)
			continue
		}
		drawTile(g.d, tiles[i], tx, y, l.rw, l.rh, false)
		if i == 0 && m.Kind != closedKan {
			g.fill(rect{tx, y + l.rh - 2, l.rw, 2}, lastColor)
		}
	}
	return x + int16(n)*l.rw
}

func (g *Game) drawMelds(seat int) {
	p := &g.tb.P[seat]
	key := meldsKey(p)
	if seat == human || key == g.drawnMelds[seat] {
		return // the player's are drawn with the hand
	}
	g.drawnMelds[seat] = key
	l := &g.l
	x0, y0 := l.melds[seat][0], l.melds[seat][1]
	mw := 4*l.rw + 2
	if seat == 2 {
		// Across: two by two, in the corner.
		g.fill(rect{x0, y0, 2 * mw, 2 * l.rh}, feltColor)
		for i := 0; i < p.NMeld; i++ {
			g.drawMeld(&p.Melds[i], x0+int16(i%2)*mw, y0+int16(i/2)*l.rh)
		}
		return
	}
	g.fill(rect{x0, y0, mw, 4 * l.rh}, feltColor)
	for i := 0; i < p.NMeld; i++ {
		g.drawMeld(&p.Melds[i], x0, y0+int16(i)*l.rh)
	}
}

func (g *Game) drawPanel(seat int) {
	t, l := g.tb, &g.l
	p := &t.P[seat]
	r := l.panel[seat]
	score := strconv.Itoa(p.Score)
	tag := p.Last
	if seat == human && !p.Riichi && p.Waits != 0 {
		tag = "テンパイ"
	}
	if p.Riichi {
		tag = "リーチ"
	}
	if seat == human && p.Waits != 0 && p.Furiten() {
		tag = "フリテン"
	}
	wind := t.SeatWind(seat)
	turn := t.Turn == seat && t.Phase != phaseOver
	state := score + "|" + tag + "|" + strconv.Itoa(int(wind)) + "|" + strconv.FormatBool(turn)
	if state == g.drawnPanels[seat] {
		return
	}
	g.drawnPanels[seat] = state
	bg := panelColor
	if turn {
		bg = turnColor
	}
	g.fill(r, bg)
	c := textColor
	if seat == t.Dealer {
		c = focusColor
	}
	glyph(g.d, honors9[wind-east][:], r.x+3, r.y+2, 1, c)
	g.jp(r.x+16, r.y+3, score, 1, c)
	if tag != "" {
		tc := dimText
		if tag == "リーチ" || tag == "フリテン" {
			tc = riichiTag
		}
		g.jp(r.x+3, r.y+13, tag, 1, tc)
	}
}

// drawStick draws a thousand point stick, white with a red dot.
func (g *Game) drawStick(r rect) {
	g.fill(r, textColor)
	d := min(r.w, r.h)
	g.fill(rect{r.x + (r.w-d)/2, r.y + (r.h-d)/2, d, d}, inkRed)
}

// stickRect is where the stick of a seat lies: on the side of the middle
// towards that seat.
func (g *Game) stickRect(seat int) rect {
	l := &g.l
	c := l.center
	n, t := l.stickLong, l.stickThick
	switch seat {
	case 0:
		return rect{c.x + (c.w-n)/2, c.y + c.h - t - 1, n, t}
	case 1:
		return rect{c.x + c.w - t - 1, c.y + (c.h-n*3/4)/2, t, n * 3 / 4}
	case 2:
		return rect{c.x + (c.w-n)/2, c.y + 1, n, t}
	}
	return rect{c.x + 1, c.y + (c.h-n*3/4)/2, t, n * 3 / 4}
}

func (g *Game) drawCenter() {
	t, l := g.tb, &g.l
	r := l.center
	wind := "東"
	if t.RoundWind == south {
		wind = "南"
	}
	round := wind + strconv.Itoa(t.HandNo+1) + "局"
	honba := strconv.Itoa(t.Honba) + "本場"
	lines := [3]string{round, honba + " 供託" + strconv.Itoa(t.Sticks), "残り " + strconv.Itoa(t.Live())}
	if l.smallLines {
		lines = [3]string{round + " " + honba, "残り" + strconv.Itoa(t.Live()) + " 供託" + strconv.Itoa(t.Sticks), ""}
	}
	sticks := ""
	for s := range t.P {
		// A stick is on the table once it is paid.
		sticks += strconv.FormatBool(t.P[s].Riichi && !t.P[s].riichiStick)
	}
	state := lines[0] + "|" + lines[1] + "|" + lines[2] + "|" + strconv.Itoa(t.NDora) + "|" + sticks
	if state == g.drawnCenter {
		return
	}
	g.drawnCenter = state
	g.fill(r, centerBg)
	for s := range t.P {
		if t.P[s].Riichi && !t.P[s].riichiStick {
			g.drawStick(g.stickRect(s))
		}
	}
	x := r.x + l.stickThick + 3
	y := r.y + l.stickThick + 1
	for _, line := range lines {
		if line == "" {
			continue
		}
		g.jp(x, y, line, 1, textColor)
		y += 10
	}
	// The dora indicators: the unknown ones as backs.
	inds := t.Indicators()
	x = r.x + (r.w-5*l.rw)/2
	for i := 0; i < 5; i++ {
		if i < len(inds) {
			drawTile(g.d, inds[i], x+int16(i)*l.rw, y, l.rw, l.rh, false)
		} else {
			drawBack(g.d, x+int16(i)*l.rw, y, l.rw, l.rh)
		}
	}
}

func (g *Game) drawHand() {
	l := &g.l
	p := &g.tb.P[human]
	y0 := l.handY - l.lift
	key := meldsKey(p)<<4 | uint32(g.nHand)
	if key != g.drawnHandAt {
		// The tiles moved: everything again.
		g.drawnHandAt = key
		w, h := g.d.Size()
		g.fill(rect{0, y0, w, h - y0}, feltColor)
		for i := range g.drawnSlots {
			g.drawnSlots[i] = noLook
		}
		x := w - 2
		for i := p.NMeld - 1; i >= 0; i-- {
			m := &p.Melds[i]
			n := int16(3)
			if m.isKan() {
				n = 4
			}
			x -= n*l.rw + 2
			g.drawMeld(m, x, y0+(h-y0-l.rh)/2)
		}
	}
	for i := range g.drawnSlots {
		v := look{tile: noTile}
		if i < g.nSlots {
			v.tile = g.slots[i]
			if g.tb.Phase == phaseTurn && g.tb.Turn == human && !g.slotAllowed(i) {
				v.flags |= flagDim
			}
			if i == g.lifted {
				v.flags |= flagLift
			}
			if g.in.Used && i == g.focus {
				v.flags |= flagFocus
			}
		}
		if v == g.drawnSlots[i] {
			continue
		}
		g.drawnSlots[i] = v
		r := g.slotRect(i)
		g.fill(rect{r.x, y0, r.w, l.th + l.lift}, feltColor)
		if v.tile == noTile {
			continue
		}
		y := r.y
		if v.flags&flagLift != 0 {
			y -= l.lift
		}
		drawTile(g.d, v.tile, r.x, y, r.w, r.h, v.flags&flagDim != 0)
		if v.flags&flagFocus != 0 {
			pad.Frame(g.d, r.x, y, r.w, r.h, 2, focusColor)
		}
	}
}

func (g *Game) drawButtons() {
	l := &g.l
	state := strconv.Itoa(g.focus) + strconv.FormatBool(g.in.Used)
	for i := 0; i < g.nButtons; i++ {
		state += "|" + g.buttons[i].label + string(rune('a'+g.buttons[i].act)) + string(rune('a'+g.buttons[i].tile))
	}
	if state == g.drawnBtns {
		return
	}
	g.drawnBtns = state
	g.fill(l.buttons, feltColor)
	for i := 0; i < g.nButtons; i++ {
		b := &g.buttons[i]
		r := g.buttonRect(i)
		c := btnColor
		switch b.act {
		case actTsumo, actRon:
			c = winColor
		case actRiichi, actPon, actChi, actKan, actAbort:
			c = goColor
		}
		g.fill(r, c)
		switch b.act {
		case actChiLow:
			g.drawChoice(r, b.tile, b.tile+1, b.tile+2)
		case actKanTile:
			g.drawChoice(r, b.tile, b.tile, b.tile)
		default:
			scale := int16(2)
			if jpWidth(b.label, 2) > r.w-4 {
				scale = 1
			}
			g.jpCentered(r, b.label, scale, textColor, (r.h-jpHeight(scale))/2)
		}
		if g.in.Used && g.focus == 14+i {
			pad.Frame(g.d, r.x, r.y, r.w, r.h, 2, focusColor)
		}
	}
}

// drawChoice draws three tiles on a button.
func (g *Game) drawChoice(r rect, a, b, c Tile) {
	l := &g.l
	x := r.x + (r.w-3*l.rw)/2
	y := r.y + (r.h-l.rh)/2
	for i, k := range [3]Tile{a, b, c} {
		drawTile(g.d, k, x+int16(i)*l.rw, y, l.rw, l.rh, false)
	}
}

func (g *Game) drawQuit() {
	label := "終了"
	if g.armed > 0 {
		label = "終了?"
	}
	state := label + strconv.FormatBool(g.in.Used && g.focus == 19)
	if state == g.drawnQuit {
		return
	}
	g.drawnQuit = state
	r := g.l.quit
	g.fill(r, btnColor)
	g.jpCentered(r, label, 2, textColor, (r.h-bigH)/2)
	if g.in.Used && g.focus == 19 {
		pad.Frame(g.d, r.x, r.y, r.w, r.h, 2, focusColor)
	}
}

// ---- the result of a hand, the end of the game ----

func (g *Game) showResult() {
	g.scr = scrResult
	g.wait = 0.3
	w, h := g.d.Size()
	g.fill(rect{0, 0, w, h}, bgColor)
	r := &g.tb.Result
	switch r.Kind {
	case endWin:
		g.drawWin(&r.Wins[g.page])
	case endDraw:
		g.drawDraw()
	default:
		g.jpCentered(rect{0, 0, w, 0}, "途中流局", 2, textColor, 50)
		g.jpCentered(rect{0, 0, w, 0}, r.Reason, 2, dimText, 84)
	}
	g.drawDeltas()
	g.jpCentered(rect{0, 0, w, 0}, "タップで次へ", 1, dimText, h-10)
	g.d.Display()
}

// drawTiles draws a closed hand from (x, y) and the melds after it, and
// returns the x at the end. The winning tile, if any, is apart from the rest.
func (g *Game) drawTiles(c Counts, melds []Meld, win Tile, x, y int16) int16 {
	l := &g.l
	if win != noTile {
		c[win]--
	}
	for k := Tile(0); k < numKinds; k++ {
		for n := uint8(0); n < c[k]; n++ {
			drawTile(g.d, k, x, y, l.rw, l.rh, false)
			x += l.rw
		}
	}
	if win != noTile {
		x += 3
		drawTile(g.d, win, x, y, l.rw, l.rh, false)
		pad.Frame(g.d, x, y, l.rw, l.rh, 1, lastColor)
		x += l.rw
	}
	for i := range melds {
		x = g.drawMeld(&melds[i], x+3, y)
	}
	return x
}

func (g *Game) drawWin(w *WinInfo) {
	l := &g.l
	sw, _ := g.d.Size()
	title := seatNames[w.Seat] + " ツモ"
	if w.From != w.Seat {
		title = seatNames[w.Seat] + " ロン " + seatNames[w.From] + "から"
	}
	g.jpCentered(rect{0, 0, sw, 0}, title, 2, focusColor, 4)
	y := int16(22)
	g.drawTiles(w.Hand, w.Melds[:w.NMeld], w.Tile, 4, y)
	y += l.rh + 3

	// The dora indicators.
	mid := y + (l.rh-smallH)/2
	g.jp(4, mid, "ドラ", 1, dimText)
	x := int16(26)
	for _, ind := range g.tb.Indicators() {
		drawTile(g.d, ind, x, y, l.rw, l.rh, false)
		x += l.rw
	}
	if w.NUra > 0 {
		x += 8
		g.jp(x, mid, "裏", 1, dimText)
		x += 22
		for _, ind := range w.Ura[:w.NUra] {
			drawTile(g.d, ind, x, y, l.rw, l.rh, false)
			x += l.rw
		}
	}
	y += l.rh + 6

	// The yaku, in two columns: large if they fit.
	s := &w.Score
	colW := (sw - 8) / 2
	rows := (s.nYaku + 1) / 2
	scale, rowH := int16(2), int16(18)
	if _, sh := g.d.Size(); y+int16(rows)*rowH+22 > sh-36 {
		scale, rowH = 1, 11
	}
	for i := 0; i < s.nYaku; i++ {
		id := s.yaku[i]
		x := int16(4) + int16(i/rows)*colW
		ly := y + int16(i%rows)*rowH
		g.jp(x, ly, yakuNames[id.id], scale, textColor)
		if s.Yakuman == 0 {
			g.jp(x+colW-12*scale, ly, strconv.Itoa(int(id.han)), scale, dimText)
		}
	}
	y += int16(rows)*rowH + 6

	var line string
	switch {
	case s.Yakuman > 1:
		line = strconv.Itoa(s.Yakuman) + s.Limit()
	case s.Yakuman > 0:
		line = s.Limit()
	case s.Limit() != "":
		line = strconv.Itoa(s.Han) + "翻 " + s.Limit()
	default:
		line = strconv.Itoa(s.Han) + "翻 " + strconv.Itoa(s.Fu) + "符"
	}
	line += " " + strconv.Itoa(w.Points) + "点"
	g.jpCentered(rect{0, 0, sw, 0}, line, 2, textColor, y)
}

func (g *Game) drawDraw() {
	l := &g.l
	r := &g.tb.Result
	sw, _ := g.d.Size()
	g.jpCentered(rect{0, 0, sw, 0}, "流局", 2, focusColor, 4)
	y := int16(26)
	for s := range r.Tenpai {
		mid := y + (l.rh-bigH)/2
		if s != human && !r.Tenpai[s] {
			g.jp(4, mid, seatNames[s]+" ノーテン", 2, dimText)
			y += max(l.rh, bigH) + 4
			continue
		}
		g.jp(4, mid, seatNames[s], 2, dimText)
		g.drawTiles(r.Hands[s], g.tb.P[s].melds(), noTile, 42, y)
		y += max(l.rh, bigH) + 4
	}
}

// drawDeltas shows the points of the four players and what changed.
func (g *Game) drawDeltas() {
	sw, sh := g.d.Size()
	r := &g.tb.Result
	colW := sw / 4
	y := sh - 36
	for s := range g.tb.P {
		cell := rect{int16(s) * colW, y, colW, 24}
		g.jpCentered(cell, seatNames[s], 1, textColor, 0)
		g.jpCentered(cell, strconv.Itoa(g.tb.P[s].Score), 1, textColor, 9)
		if d := r.Delta[s]; d != 0 {
			txt, c := strconv.Itoa(d), minusColor
			if d > 0 {
				txt, c = "+"+txt, plusColor
			}
			g.jpCentered(cell, txt, 1, c, 18)
		}
	}
}

func (g *Game) showOver() {
	g.scr = scrOver
	g.wait = 0.3
	sw, sh := g.d.Size()
	g.fill(rect{0, 0, sw, sh}, bgColor)
	g.jpCentered(rect{0, 0, sw, 0}, "終局", 4, textColor, 20)
	y := int16(66)
	for i, s := range g.tb.Ranking() {
		c := textColor
		if s == human {
			c = focusColor
		}
		g.jp(sw/2-110, y, strconv.Itoa(i+1)+"位 "+seatNames[s], 2, c)
		score := strconv.Itoa(g.tb.P[s].Score)
		g.jp(sw/2+110-jpWidth(score, 2), y, score, 2, c)
		y += 30
	}
	g.jpCentered(rect{0, 0, sw, 0}, "タップで次へ", 1, dimText, sh-10)
	g.d.Display()
}
