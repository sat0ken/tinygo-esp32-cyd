package main

import (
	"image/color"
	"strconv"

	"github.com/sat0ken/tinygo-cyd/hal"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/freesans"
	"tinygo.org/x/tinyfont/proggy"
)

// Layout for 480x272: the grid on the left, the controls on the right.
const (
	gridX = 8
	gridY = 9
	pitch = 28 // cell pitch; lines are drawn on the pitch, thick ones every 3 cells

	panelX  = 282
	padY    = 40
	padW    = 58
	padH    = 42
	padGap  = 6
	btnY    = 188
	btnH    = 36
	statusY = 244

	tapGap = 0.15 // seconds; ignore taps closer than this (resistive touch bounces)
)

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

// look is how a cell is drawn; cells are redrawn only when it changes.
type look struct {
	bg, fg color.RGBA
	digit  uint8
}

// Game is the sudoku game. It depends only on hal.
type Game struct {
	d hal.Display
	t hal.Touch

	// NewSeed returns the seed for the next puzzle.
	NewSeed func() uint32

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
}

// NewGame shows the difficulty menu.
func NewGame(d hal.Display, t hal.Touch, newSeed func() uint32) *Game {
	g := &Game{d: d, t: t, NewSeed: newSeed, sel: -1, lastTap: -1}
	return g
}

// ---- geometry ----

// cellRect is the inside of cell i (without the grid lines).
func cellRect(i int) rect {
	r, c := i/9, i%9
	x0 := gridX + int16(c)*pitch
	y0 := gridY + int16(r)*pitch
	left, top := int16(1), int16(1)
	if c%3 == 0 {
		left = 2 // after a thick line
	}
	if r%3 == 0 {
		top = 2
	}
	return rect{x0 + left, y0 + top, pitch - left, pitch - top}
}

func gridRect() rect { return rect{gridX, gridY, 9*pitch + 2, 9*pitch + 2} }

func padRect(k int) rect { // k = 0..8 for digits 1..9
	return rect{panelX + int16(k%3)*(padW+padGap), padY + int16(k/3)*(padH+padGap), padW, padH}
}

func buttonRect(k int) rect { // 0 = ERASE, 1 = HINT, 2 = NEW
	return rect{panelX + int16(k)*(padW+padGap), btnY, padW, btnH}
}

func menuButton(k int) rect { return rect{40 + int16(k)*140, 160, 120, 56} }

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
	g.textCentered(rect{0, 0, w, 0}, &freesans.Bold24pt7b, "SUDOKU", textColor, 70)
	g.textCentered(rect{0, 0, w, 0}, &proggy.TinySZ8pt7b, "choose a difficulty", dimText, 120)
	for k, df := range difficulties {
		r := menuButton(k)
		g.fill(r, df.Color)
		g.textCentered(r, &freesans.Bold12pt7b, df.Name, textColor, 36)
		g.textCentered(rect{r.x, r.y + r.h, r.w, 0}, &proggy.TinySZ8pt7b, strconv.Itoa(df.Clues)+" clues", dimText, 18)
	}
	g.d.Display()
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
	g.drawPlay()
}

func (g *Game) drawPlay() {
	w, h := g.d.Size()
	g.fill(rect{0, 0, w, h}, bgColor)
	// Grid lines: the cells are drawn on top, between the lines.
	gr := gridRect()
	for i := int16(0); i <= 9; i++ {
		c, t := thinColor, int16(1)
		if i%3 == 0 {
			c, t = thickColor, 2
		}
		g.fill(rect{gr.x + i*pitch, gr.y, t, gr.h}, c)
		g.fill(rect{gr.x, gr.y + i*pitch, gr.w, t}, c)
	}
	g.drawn = [81]look{}
	g.drawnPad = [9]look{}
	g.drawnState = ""
	for k, name := range []string{"ERASE", "HINT", "NEW"} {
		r := buttonRect(k)
		g.fill(r, btnColor)
		g.textCentered(r, &proggy.TinySZ8pt7b, name, textColor, 22)
	}
	g.refresh()
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
		r := cellRect(i)
		g.fill(r, l.bg)
		if l.digit != 0 {
			g.textCentered(r, &freesans.Bold12pt7b, string(rune('0'+l.digit)), l.fg, 21)
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
		if l == g.drawnPad[k] {
			continue
		}
		r := padRect(k)
		g.fill(r, l.bg)
		g.textCentered(r, &freesans.Bold12pt7b, string(rune('0'+d)), l.fg, 29)
		g.drawnPad[k] = l
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
	head := difficulties[g.diff].Name + "   " + clock(g.elapsed)
	tail := "LEFT " + strconv.Itoa(g.emptyCount()) + "   MISS " + strconv.Itoa(g.mistakes)
	state := head + "|" + tail
	if state == g.drawnState {
		return
	}
	w, _ := g.d.Size()
	g.fill(rect{panelX, 8, w - panelX - 4, 24}, bgColor)
	tinyfont.WriteLine(g.d, &freesans.Bold9pt7b, panelX, 26, head, textColor)
	g.fill(rect{panelX, statusY - 2, w - panelX - 4, 22}, bgColor)
	tinyfont.WriteLine(g.d, &proggy.TinySZ8pt7b, panelX, statusY+12, tail, dimText)
	g.drawnState = state
}

func (g *Game) drawSolved() {
	r := rect{gridX + 30, gridY + 80, 9*pitch - 58, 94}
	g.fill(r, bgColor)
	g.fill(rect{r.x, r.y, r.w, 3}, winColor)
	g.fill(rect{r.x, r.y + r.h - 3, r.w, 3}, winColor)
	g.textCentered(r, &freesans.Bold18pt7b, "SOLVED!", winColor, 40)
	g.textCentered(r, &proggy.TinySZ8pt7b, "TIME "+clock(g.elapsed)+"   MISS "+strconv.Itoa(g.mistakes), textColor, 62)
	g.textCentered(r, &proggy.TinySZ8pt7b, "tap for a new game", dimText, 80)
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
	}

	switch g.screen {
	case menuScreen:
		if tap {
			for k := range difficulties {
				if menuButton(k).has(x, y) {
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
	if gridRect().has(x, y) {
		c := int((x - gridX) / pitch)
		r := int((y - gridY) / pitch)
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
		if padRect(k).has(x, y) {
			g.enter(uint8(k + 1))
			return
		}
	}
	switch {
	case buttonRect(0).has(x, y):
		g.erase()
	case buttonRect(1).has(x, y):
		g.hint()
	case buttonRect(2).has(x, y):
		g.DrawMenu()
	}
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
