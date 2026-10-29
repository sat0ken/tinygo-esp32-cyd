// Package sudoku is a sudoku game for a 480x272 or 320x240 touch screen.
// It depends only on hal: examples/09_sudoku runs it alone, cmd/games from
// a launcher.
package sudoku

import (
	"image/color"
	"strconv"

	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/pad"
	"github.com/sat0ken/tinygo-cyd/romfont/freesans"
	"github.com/sat0ken/tinygo-cyd/romfont/proggy"
	"tinygo.org/x/tinyfont"
)

const tapGap = 0.15 // seconds; ignore taps closer than this (resistive touch bounces)

var (
	bgColor      = color.RGBA{18, 22, 32, 255}
	cellColor    = color.RGBA{36, 42, 58, 255}
	peerColor    = color.RGBA{46, 55, 78, 255}  // same row, column or box as the selection
	sameColor    = color.RGBA{38, 82, 110, 255} // same digit as the selection
	selColor     = color.RGBA{72, 104, 168, 255}
	thinColor    = color.RGBA{70, 78, 96, 255}
	thickColor   = color.RGBA{170, 180, 200, 255}
	givenColor   = color.RGBA{255, 255, 255, 255}
	playerColor  = color.RGBA{120, 200, 255, 255}
	errorColor   = color.RGBA{255, 90, 90, 255}
	hintColor    = color.RGBA{255, 210, 90, 255}
	padColor     = color.RGBA{50, 58, 80, 255}
	padHiColor   = color.RGBA{120, 200, 255, 255}
	padDoneColor = color.RGBA{30, 34, 46, 255}
	btnColor     = color.RGBA{70, 78, 96, 255}
	textColor    = color.RGBA{255, 255, 255, 255}
	dimText      = color.RGBA{150, 160, 180, 255}
	winColor     = color.RGBA{90, 230, 120, 255}
	focusColor   = color.RGBA{255, 210, 40, 255}
)

// Difficulty is the number of clues left in the puzzle.
type Difficulty struct {
	Name  string
	Clues int
	Color color.RGBA
}

var difficulties = []Difficulty{
	{"EASY", 38, color.RGBA{40, 140, 70, 255}},
	{"MEDIUM", 30, color.RGBA{190, 140, 30, 255}},
	{"HARD", 25, color.RGBA{190, 50, 50, 255}},
}

type screen int

const (
	menuScreen screen = iota
	playScreen
	solvedScreen
)

type rect struct{ x, y, w, h int16 }

func (r rect) has(x, y int16) bool { return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h }

// layout is where everything is: the grid on the left, the controls on the
// right.
type layout struct {
	gridX, gridY int16
	pitch        int16 // cell pitch; lines are drawn on the pitch, thick ones every 3 cells
	cellBase     int16 // baseline of a digit in a cell

	pad0    rect  // the 1 of the number pad; 2..9 follow in rows of three
	padGap  int16 // between the keys
	padBase int16
	buttons [3]rect // ERASE, HINT, NEW
	btnBase int16

	head  rect // difficulty and clock
	tail  rect // cells left and mistakes
	small bool // the status on four short lines instead of two

	solved rect // the SOLVED! box

	titleBase, subBase int16 // the menu's title and the line under it
	menu0              rect  // the first difficulty button
	menuStep           int16 // from one difficulty button to the next
	menuFont           tinyfont.Fonter
	menuBase           int16
}

// layoutFor returns the layout for a w x h screen: the 480x272 one, or the
// 320x240 one (ESP32-2432S028R) on smaller screens.
func layoutFor(w, h int16) layout {
	if w >= 480 && h >= 272 {
		const panelX, padW, padGap = 282, 58, 6
		l := layout{
			gridX: 8, gridY: 9, pitch: 28, cellBase: 21,
			pad0: rect{panelX, 40, padW, 42}, padGap: padGap, padBase: 29,
			btnBase:   22,
			head:      rect{panelX, 8, w - panelX - 4, 24},
			tail:      rect{panelX, 242, w - panelX - 4, 22},
			solved:    rect{38, 89, 9*28 - 58, 94},
			titleBase: 70, subBase: 120,
			menu0: rect{40, 160, 120, 56}, menuStep: 140,
			menuFont: &freesans.Bold12pt7b, menuBase: 36,
		}
		for k := range l.buttons {
			l.buttons[k] = rect{panelX + int16(k)*(padW+padGap), 188, padW, 36}
		}
		return l
	}
	const panelX = 238 // the grid ends at 4+9*25+2 = 231
	return layout{
		gridX: 4, gridY: 6, pitch: 25, cellBase: 20,
		pad0: rect{panelX, 42, 24, 28}, padGap: 3, padBase: 22,
		buttons:   [3]rect{{panelX, 140, 37, 26}, {panelX + 41, 140, 37, 26}, {panelX, 170, 78, 26}},
		btnBase:   17,
		head:      rect{panelX, 2, w - panelX - 4, 36},
		tail:      rect{panelX, 202, w - panelX - 4, 32},
		small:     true,
		solved:    rect{18, 72, 9*25 - 26, 94},
		titleBase: 72, subBase: 98, // the title clears the "< GAMES" button
		menu0: rect{12, 116, 92, 50}, menuStep: 102,
		menuFont: &freesans.Bold9pt7b, menuBase: 31,
	}
}

// cellRect is the inside of cell i (without the grid lines).
func (l *layout) cellRect(i int) rect {
	r, c := i/9, i%9
	x0 := l.gridX + int16(c)*l.pitch
	y0 := l.gridY + int16(r)*l.pitch
	left, top := int16(1), int16(1)
	if c%3 == 0 {
		left = 2 // after a thick line
	}
	if r%3 == 0 {
		top = 2
	}
	return rect{x0 + left, y0 + top, l.pitch - left, l.pitch - top}
}

func (l *layout) gridRect() rect { return rect{l.gridX, l.gridY, 9*l.pitch + 2, 9*l.pitch + 2} }

func (l *layout) padRect(k int) rect { // k = 0..8 for digits 1..9
	p := l.pad0
	p.x += int16(k%3) * (p.w + l.padGap)
	p.y += int16(k/3) * (p.h + l.padGap)
	return p
}

func (l *layout) menuButton(k int) rect {
	r := l.menu0
	r.x += int16(k) * l.menuStep
	return r
}

func backButton() rect { return rect{8, 8, 84, 28} }

// look is how a cell or a key is drawn; they are redrawn only when it
// changes.
type look struct {
	bg, fg color.RGBA
	digit  uint8
	focus  bool // a key with the controller's focus
}

// The controller's focus on the play screen: the cells, the keys of the
// number pad, the buttons.
const (
	focusKeys    = 81
	focusButtons = focusKeys + 9
	focusCount   = focusButtons + 3
)

// Game is the sudoku game. It depends only on hal.
type Game struct {
	d hal.Display
	t hal.Touch
	l layout

	// NewSeed returns the seed for the next puzzle.
	NewSeed func() uint32

	// OnExit, if set, shows a "< GAMES" button on the menu; tapping it
	// calls OnExit (used by the cmd/games launcher).
	OnExit func()

	// Pad, if set, is the game controller. The stick moves the focus: on
	// the menu between the buttons, on the grid it selects the cell, and
	// to the right of the grid it reaches the number pad and the buttons.
	// A presses what has the focus (on a cell: go to the number pad, on a
	// key: enter it and go back to the cell), B goes back to the cell (on
	// the menu: "< GAMES"), X erases, Y gives a hint.
	Pad   hal.Pad
	in    pad.Input
	focus int // on the menu: a difficulty or 3 for "< GAMES"; when playing: see focusKeys
	boxes [focusCount]pad.Box

	screen   screen
	diff     int
	puzzle   Board // clues
	solution Board
	board    Board // what is on the grid now (clues + entries)
	hinted   [81]bool
	sel      int // selected cell, -1 = none
	mistakes int
	elapsed  float32 // seconds playing

	clock    float32
	lastTap  float32
	wasTouch bool

	drawn      [81]look
	drawnPad   [9]look
	drawnState string
	drawnBtns  int // the button with the focus when the buttons were drawn, -1 none, -2 not drawn
}

// NewGame shows the difficulty menu.
func NewGame(d hal.Display, t hal.Touch, newSeed func() uint32) *Game {
	w, h := d.Size()
	g := &Game{d: d, t: t, l: layoutFor(w, h), NewSeed: newSeed, sel: -1, lastTap: -1}
	for i := range g.boxes {
		var r rect
		switch {
		case i < focusKeys:
			r = g.l.cellRect(i)
		case i < focusButtons:
			r = g.l.padRect(i - focusKeys)
		default:
			r = g.l.buttons[i-focusButtons]
		}
		g.boxes[i] = pad.Box{X: r.x, Y: r.y, W: r.w, H: r.h}
	}
	return g
}

func (g *Game) frame(r rect) { pad.Frame(g.d, r.x, r.y, r.w, r.h, 2, focusColor) }

// ---- drawing helpers ----

func (g *Game) fill(r rect, c color.RGBA) { g.d.FillRectangle(r.x, r.y, r.w, r.h, c) }

func (g *Game) textCentered(r rect, font tinyfont.Fonter, s string, c color.RGBA, baseline int16) {
	_, w := tinyfont.LineWidth(font, s)
	tinyfont.WriteLine(g.d, font, r.x+(r.w-int16(w))/2, r.y+baseline, s, c)
}

// ---- screens ----

// DrawMenu draws the difficulty menu.
func (g *Game) DrawMenu() {
	g.screen = menuScreen
	w, h := g.d.Size()
	g.fill(rect{0, 0, w, h}, bgColor)
	g.textCentered(rect{0, 0, w, 0}, &freesans.Bold24pt7b, "SUDOKU", textColor, g.l.titleBase)
	g.textCentered(rect{0, 0, w, 0}, &proggy.TinySZ8pt7b, "choose a difficulty", dimText, g.l.subBase)
	for k, df := range difficulties {
		r := g.l.menuButton(k)
		g.textCentered(rect{r.x, r.y + r.h, r.w, 0}, &proggy.TinySZ8pt7b, strconv.Itoa(df.Clues)+" clues", dimText, 18)
	}
	for k := range g.menuBoxes() {
		g.drawMenuItem(k)
	}
	g.d.Display()
}

// menuBoxes are the buttons of the menu: the difficulties and "< GAMES".
func (g *Game) menuBoxes() []pad.Box {
	var b [4]pad.Box
	n := len(difficulties)
	for k := 0; k < n; k++ {
		r := g.l.menuButton(k)
		b[k] = pad.Box{X: r.x, Y: r.y, W: r.w, H: r.h}
	}
	if g.OnExit != nil {
		r := backButton()
		b[n] = pad.Box{X: r.x, Y: r.y, W: r.w, H: r.h}
		n++
	}
	return b[:n]
}

// drawMenuItem draws button k of the menu (see menuBoxes).
func (g *Game) drawMenuItem(k int) {
	if k == len(difficulties) {
		r := backButton()
		g.fill(r, btnColor)
		g.textCentered(r, &proggy.TinySZ8pt7b, "< GAMES", textColor, 18)
	} else {
		r, df := g.l.menuButton(k), difficulties[k]
		g.fill(r, df.Color)
		g.textCentered(r, g.l.menuFont, df.Name, textColor, g.l.menuBase)
	}
	if g.in.Used && k == g.focus {
		b := g.menuBoxes()[k]
		g.frame(rect{b.X, b.Y, b.W, b.H})
	}
}

// startGame generates a puzzle and draws the play screen.
func (g *Game) startGame(diff int) {
	w, h := g.d.Size()
	g.fill(rect{0, 0, w, h}, bgColor)
	g.textCentered(rect{0, 0, w, 0}, &freesans.Bold12pt7b, "generating...", textColor, h/2)
	g.d.Display()

	g.diff = diff
	g.puzzle, g.solution = Generate(g.NewSeed(), difficulties[diff].Clues)
	g.board = g.puzzle
	g.hinted = [81]bool{}
	g.sel = -1
	g.mistakes = 0
	g.elapsed = 0
	g.screen = playScreen
	g.focus = 40 // the middle cell, selected by the first move of the stick
	g.drawPlay()
}

func (g *Game) drawPlay() {
	w, h := g.d.Size()
	g.fill(rect{0, 0, w, h}, bgColor)
	// Grid lines: the cells are drawn on top, between the lines.
	gr := g.l.gridRect()
	for i := int16(0); i <= 9; i++ {
		c, t := thinColor, int16(1)
		if i%3 == 0 {
			c, t = thickColor, 2
		}
		g.fill(rect{gr.x + i*g.l.pitch, gr.y, t, gr.h}, c)
		g.fill(rect{gr.x, gr.y + i*g.l.pitch, gr.w, t}, c)
	}
	g.drawn = [81]look{}
	g.drawnPad = [9]look{}
	g.drawnState = ""
	g.drawnBtns = -2
	g.refresh()
}

// padFocus returns what has the controller's focus on the play screen
// (see focusKeys), or -1 before the controller is used.
func (g *Game) padFocus() int {
	if !g.in.Used || g.screen != playScreen {
		return -1
	}
	return g.focus
}

// cellLook decides how cell i should look now.
func (g *Game) cellLook(i int) look {
	l := look{bg: cellColor, digit: g.board[i]}
	switch {
	case g.puzzle[i] != 0:
		l.fg = givenColor
	case g.hinted[i]:
		l.fg = hintColor
	case g.board[i] != 0 && g.board[i] != g.solution[i]:
		l.fg = errorColor
	default:
		l.fg = playerColor
	}
	if s := g.sel; s >= 0 {
		sr, sc, r, c := s/9, s%9, i/9, i%9
		switch {
		case i == s:
			l.bg = selColor
		case g.board[s] != 0 && g.board[i] == g.board[s]:
			l.bg = sameColor
		case r == sr || c == sc || (r/3 == sr/3 && c/3 == sc/3):
			l.bg = peerColor
		}
	}
	return l
}

// placed counts how many times digit d is correctly on the board.
func (g *Game) placed(d uint8) int {
	n := 0
	for i, v := range g.board {
		if v == d && v == g.solution[i] {
			n++
		}
	}
	return n
}

func (g *Game) emptyCount() int {
	n := 0
	for i, v := range g.board {
		if v != g.solution[i] {
			n++
		}
	}
	return n
}

// refresh redraws whatever changed: cells, the number pad and the status.
func (g *Game) refresh() {
	for i := range g.board {
		l := g.cellLook(i)
		if l == g.drawn[i] {
			continue
		}
		r := g.l.cellRect(i)
		g.fill(r, l.bg)
		if l.digit != 0 {
			g.textCentered(r, &freesans.Bold12pt7b, string(rune('0'+l.digit)), l.fg, g.l.cellBase)
		}
		g.drawn[i] = l
	}

	selDigit := uint8(0)
	if g.sel >= 0 {
		selDigit = g.board[g.sel]
	}
	for k := 0; k < 9; k++ {
		d := uint8(k + 1)
		l := look{bg: padColor, fg: textColor, digit: d}
		switch {
		case g.placed(d) == 9:
			l.bg, l.fg = padDoneColor, dimText // all nine are in place
		case d == selDigit:
			l.bg, l.fg = padHiColor, bgColor
		}
		l.focus = g.padFocus() == focusKeys+k
		if l == g.drawnPad[k] {
			continue
		}
		r := g.l.padRect(k)
		g.fill(r, l.bg)
		g.textCentered(r, &freesans.Bold12pt7b, string(rune('0'+d)), l.fg, g.l.padBase)
		if l.focus {
			g.frame(r)
		}
		g.drawnPad[k] = l
	}

	if f := g.padFocus() - focusButtons; f != g.drawnBtns && (f >= 0 || g.drawnBtns != -1) {
		if f < 0 {
			f = -1
		}
		for k, name := range []string{"ERASE", "HINT", "NEW"} {
			r := g.l.buttons[k]
			g.fill(r, btnColor)
			g.textCentered(r, &proggy.TinySZ8pt7b, name, textColor, g.l.btnBase)
			if k == f {
				g.frame(r)
			}
		}
		g.drawnBtns = f
	}

	g.drawStatus()
	if g.screen == solvedScreen {
		g.drawSolved()
	}
	g.d.Display()
}

func clock(sec float32) string {
	s := int(sec)
	mm, ss := s/60, s%60
	two := func(n int) string {
		if n < 10 {
			return "0" + strconv.Itoa(n)
		}
		return strconv.Itoa(n)
	}
	return two(mm) + ":" + two(ss)
}

func (g *Game) drawStatus() {
	name, time := difficulties[g.diff].Name, clock(g.elapsed)
	left, miss := "LEFT "+strconv.Itoa(g.emptyCount()), "MISS "+strconv.Itoa(g.mistakes)
	state := name + time + left + miss
	if state == g.drawnState {
		return
	}
	l := &g.l
	g.fill(l.head, bgColor)
	g.fill(l.tail, bgColor)
	if l.small {
		tinyfont.WriteLine(g.d, &proggy.TinySZ8pt7b, l.head.x, l.head.y+12, name, dimText)
		tinyfont.WriteLine(g.d, &freesans.Bold9pt7b, l.head.x, l.head.y+32, time, textColor)
		tinyfont.WriteLine(g.d, &proggy.TinySZ8pt7b, l.tail.x, l.tail.y+12, left, dimText)
		tinyfont.WriteLine(g.d, &proggy.TinySZ8pt7b, l.tail.x, l.tail.y+26, miss, dimText)
	} else {
		tinyfont.WriteLine(g.d, &freesans.Bold9pt7b, l.head.x, l.head.y+18, name+"   "+time, textColor)
		tinyfont.WriteLine(g.d, &proggy.TinySZ8pt7b, l.tail.x, l.tail.y+14, left+"   "+miss, dimText)
	}
	g.drawnState = state
}

func (g *Game) drawSolved() {
	r := g.l.solved
	g.fill(r, bgColor)
	g.fill(rect{r.x, r.y, r.w, 3}, winColor)
	g.fill(rect{r.x, r.y + r.h - 3, r.w, 3}, winColor)
	g.textCentered(r, &freesans.Bold18pt7b, "SOLVED!", winColor, 40)
	g.textCentered(r, &proggy.TinySZ8pt7b, "TIME "+clock(g.elapsed)+"   MISS "+strconv.Itoa(g.mistakes), textColor, 62)
	g.textCentered(r, &proggy.TinySZ8pt7b, "tap for a new game", dimText, 80)
}

// ---- input ----

// Update reads the touch panel and the controller and advances the clock
// by dt seconds.
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
	}

	switch g.screen {
	case menuScreen:
		if tap && g.OnExit != nil && backButton().has(x, y) {
			g.OnExit()
			return
		}
		if tap {
			for k := range difficulties {
				if g.l.menuButton(k).has(x, y) {
					g.startGame(k)
					return
				}
			}
		}
	case playScreen:
		g.elapsed += dt
		if tap {
			g.tapPlay(x, y)
		}
		g.refresh()
	case solvedScreen:
		if tap {
			g.DrawMenu()
		}
	}
}

func (g *Game) tapPlay(x, y int16) {
	l := &g.l
	if l.gridRect().has(x, y) {
		c := int((x - l.gridX) / l.pitch)
		r := int((y - l.gridY) / l.pitch)
		if c > 8 {
			c = 8
		}
		if r > 8 {
			r = 8
		}
		if i := r*9 + c; g.sel == i {
			g.sel = -1 // tap again to deselect
		} else {
			g.sel = i
		}
		return
	}
	for k := 0; k < 9; k++ {
		if l.padRect(k).has(x, y) {
			g.enter(uint8(k + 1))
			return
		}
	}
	switch {
	case l.buttons[0].has(x, y):
		g.erase()
	case l.buttons[1].has(x, y):
		g.hint()
	case l.buttons[2].has(x, y):
		g.DrawMenu()
	}
}

// updatePad handles the controller (first: it was used for the first
// time) and returns true if the screen changed.
func (g *Game) updatePad(first bool) bool {
	in := &g.in
	switch g.screen {
	case menuScreen:
		boxes := g.menuBoxes()
		if g.focus >= len(boxes) {
			g.focus = 0
		}
		if first {
			g.drawMenuItem(g.focus)
			g.d.Display()
		}
		if d := in.Direction(); d != 0 {
			if f := pad.Move(boxes, g.focus, d); f != g.focus {
				old := g.focus
				g.focus = f
				g.drawMenuItem(old)
				g.drawMenuItem(f)
				g.d.Display()
			}
		}
		switch {
		case in.Pressed(hal.PadB) && g.OnExit != nil, in.Pressed(hal.PadA|hal.PadOK) && g.focus == len(difficulties):
			g.OnExit()
			return true
		case in.Pressed(hal.PadA | hal.PadOK):
			g.startGame(g.focus)
			return true
		}
	case playScreen:
		if g.focus < focusKeys && g.sel >= 0 {
			g.focus = g.sel // a tap may have selected another cell
		}
		if d := in.Direction(); d != 0 {
			if g.sel < 0 && g.focus < focusKeys {
				g.focus = 40 // the first move selects the middle cell
			} else {
				g.focus = pad.Move(g.boxes[:], g.focus, d)
			}
			if g.focus < focusKeys {
				g.sel = g.focus
			}
		}
		switch {
		case in.Pressed(hal.PadA | hal.PadOK):
			g.padPress()
		case in.Pressed(hal.PadB):
			g.focusCell()
		case in.Pressed(hal.PadX):
			g.erase()
		case in.Pressed(hal.PadY):
			g.hint()
			g.focusCell()
		}
		switch g.screen {
		case menuScreen:
			return true // NEW
		case solvedScreen:
			g.refresh() // the last digit
			return true
		}
	case solvedScreen:
		if in.Pressed(hal.PadA | hal.PadOK | hal.PadB) {
			g.focus = g.diff
			g.DrawMenu()
			return true
		}
	}
	return false
}

// padPress presses what has the focus on the play screen.
func (g *Game) padPress() {
	switch f := g.focus; {
	case f < focusKeys:
		// Go to the number pad, on the digit of the cell.
		d := 5
		if g.sel >= 0 && g.board[g.sel] != 0 {
			d = int(g.board[g.sel])
		}
		g.sel = f
		g.focus = focusKeys + d - 1
	case f < focusButtons:
		g.enter(uint8(f - focusKeys + 1))
		g.focusCell()
	case f == focusButtons:
		g.erase()
		g.focusCell()
	case f == focusButtons+1:
		g.hint()
		g.focusCell()
	default:
		g.focus = g.diff
		g.DrawMenu()
	}
}

// focusCell moves the focus back to the selected cell.
func (g *Game) focusCell() {
	if g.sel < 0 {
		g.sel = 40
	}
	g.focus = g.sel
}

// editable reports whether the selected cell can be changed.
func (g *Game) editable() bool {
	return g.sel >= 0 && g.puzzle[g.sel] == 0 && !g.hinted[g.sel]
}

func (g *Game) enter(d uint8) {
	if !g.editable() || g.board[g.sel] == d {
		return
	}
	g.board[g.sel] = d
	if d != g.solution[g.sel] {
		g.mistakes++
	}
	g.checkSolved()
}

func (g *Game) erase() {
	if g.editable() {
		g.board[g.sel] = 0
	}
}

// hint fills the selected cell, or a random unsolved cell if the selected
// one is a clue or already right.
func (g *Game) hint() {
	i := g.sel
	if i < 0 || g.board[i] == g.solution[i] {
		var open []int
		for k := range g.board {
			if g.board[k] != g.solution[k] {
				open = append(open, k)
			}
		}
		if len(open) == 0 {
			return
		}
		i = open[int(g.NewSeed()%uint32(len(open)))]
	}
	g.board[i] = g.solution[i]
	g.hinted[i] = true
	g.sel = i
	g.checkSolved()
}

func (g *Game) checkSolved() {
	if g.board == g.solution {
		g.screen = solvedScreen
	}
}
