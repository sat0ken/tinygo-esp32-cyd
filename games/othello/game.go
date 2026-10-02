// Package othello is othello (reversi) against the CPU for a 480x272 touch
// screen. It depends only on hal: examples/10_othello runs it alone,
// cmd/games from a launcher.
package othello

import (
	"image/color"
	"math/bits"
	"strconv"
	"time"

	"github.com/sat0ken/tinygo-cyd/hal"
	"tinygo.org/x/tinydraw"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/freesans"
	"tinygo.org/x/tinyfont/proggy"
)

// Layout for 480x272: the board on the left, the status on the right.
const (
	boardX = 8
	boardY = 8
	cellSz = 32 // the lines are on the 32-pixel grid, the inside is 31x31
	stoneR = 13
	dotR   = 3

	panelX = 282

	tapGap    = 0.15 // seconds; ignore taps closer than this (resistive touch bounces)
	flipStep  = 0.06 // seconds between two stones turning over
	passDelay = 1.2  // seconds the PASS message is shown
)

var (
	bgColor     = color.RGBA{18, 22, 32, 255}
	boardColor  = color.RGBA{0, 120, 70, 255}
	lineColor   = color.RGBA{0, 70, 40, 255}
	hintColor   = color.RGBA{0, 80, 45, 255}
	blackStone  = color.RGBA{25, 25, 25, 255}
	whiteStone  = color.RGBA{240, 240, 240, 255}
	rimColor    = color.RGBA{90, 90, 90, 255}
	lastColor   = color.RGBA{230, 50, 50, 255}
	textColor   = color.RGBA{255, 255, 255, 255}
	dimText     = color.RGBA{150, 160, 180, 255}
	btnColor    = color.RGBA{70, 78, 96, 255}
	selBtnColor = color.RGBA{60, 140, 210, 255}
	turnColor   = color.RGBA{255, 210, 90, 255}
	levelColors = []color.RGBA{{40, 140, 70, 255}, {190, 140, 30, 255}, {190, 50, 50, 255}}
)

type rect struct{ x, y, w, h int16 }

func (r rect) has(x, y int16) bool { return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h }

type phase int

const (
	menu      phase = iota
	humanTurn       // waiting for the player's move
	animating       // stones turning over
	cpuTurn         // "THINKING..." is on screen, the CPU moves on the next update
	passing         // a side has no move; the PASS message is shown for a moment
	gameOver
)

// Stone values of a square.
const (
	empty = iota
	black
	white
)

// sqLook is how a square is drawn; squares are redrawn only when it changes.
type sqLook struct {
	stone int
	hint  bool // a legal move for the player
	last  bool // the last move
}

// snapshot is a position saved for UNDO.
type snapshot struct {
	pos       Position
	blackMove bool
	last      int
}

// Game is the othello game. It depends only on hal.
type Game struct {
	d hal.Display
	t hal.Touch

	// Now is the clock used to measure the CPU's thinking time.
	Now func() time.Time

	// OnExit, if set, shows a "< GAMES" button on the menu; tapping it
	// calls OnExit (used by the cmd/games launcher).
	OnExit func()

	phase      phase
	humanBlack bool // the player has the black stones (moves first)
	level      int
	pos        Position // the side to move is Me
	blackMove  bool     // black is to move
	last       int      // last move, -1 = none
	history    []snapshot
	message    string
	cpuInfo    string

	pending []int // squares still to redraw, in order (the turning animation)
	timer   float32

	clock    float32
	lastTap  float32
	wasTouch bool
	rng      uint32

	drawn      [64]sqLook
	drawnPanel string
}

// NewGame shows the menu. seed makes the EASY level's choices repeatable.
func NewGame(d hal.Display, t hal.Touch, seed uint32) *Game {
	if seed == 0 {
		seed = 1
	}
	g := &Game{d: d, t: t, Now: time.Now, humanBlack: true, level: 1, lastTap: -1, rng: seed}
	return g
}

func (g *Game) noise(n int) int {
	g.rng ^= g.rng << 13
	g.rng ^= g.rng >> 17
	g.rng ^= g.rng << 5
	return int(g.rng%uint32(2*n+1)) - n
}

// ---- geometry ----

func squareRect(sq int) rect {
	r, c := sq/8, sq%8
	return rect{boardX + int16(c)*cellSz + 1, boardY + int16(r)*cellSz + 1, cellSz - 1, cellSz - 1}
}

func boardRect() rect { return rect{boardX, boardY, 8*cellSz + 1, 8*cellSz + 1} }

func colorButton(k int) rect { return rect{90 + int16(k)*160, 100, 140, 44} } // 0 = black, 1 = white
func levelButton(k int) rect { return rect{40 + int16(k)*140, 190, 120, 52} }
func backButton() rect       { return rect{8, 8, 84, 28} }
func undoButton() rect       { return rect{panelX, 214, 88, 40} }
func newButton() rect        { return rect{panelX + 98, 214, 88, 40} }

// ---- drawing helpers ----

func (g *Game) fill(r rect, c color.RGBA) { g.d.FillRectangle(r.x, r.y, r.w, r.h, c) }

func (g *Game) textCentered(r rect, font tinyfont.Fonter, s string, c color.RGBA, baseline int16) {
	_, w := tinyfont.LineWidth(font, s)
	tinyfont.WriteLine(g.d, font, r.x+(r.w-int16(w))/2, r.y+baseline, s, c)
}

func (g *Game) stone(cx, cy int16, black bool) {
	if black {
		tinydraw.FilledCircle(g.d, cx, cy, stoneR, blackStone)
		tinydraw.Circle(g.d, cx, cy, stoneR, rimColor)
	} else {
		tinydraw.FilledCircle(g.d, cx, cy, stoneR, whiteStone)
	}
}

// ---- menu ----

// DrawMenu draws the menu: which colour to play and the CPU level.
func (g *Game) DrawMenu() {
	g.phase = menu
	w, h := g.d.Size()
	g.fill(rect{0, 0, w, h}, bgColor)
	g.textCentered(rect{0, 0, w, 0}, &freesans.Bold24pt7b, "OTHELLO", textColor, 56)
	g.textCentered(rect{0, 0, w, 0}, &proggy.TinySZ8pt7b, "YOU PLAY", dimText, 92)
	for k, name := range []string{"BLACK (1st)", "WHITE (2nd)"} {
		r := colorButton(k)
		c := btnColor
		if (k == 0) == g.humanBlack {
			c = selBtnColor
		}
		g.fill(r, c)
		g.stone(r.x+24, r.y+r.h/2, k == 0)
		tinyfont.WriteLine(g.d, &proggy.TinySZ8pt7b, r.x+44, r.y+26, name, textColor)
	}
	g.textCentered(rect{0, 0, w, 0}, &proggy.TinySZ8pt7b, "CPU LEVEL - TAP TO START", dimText, 180)
	for k, lv := range levels {
		r := levelButton(k)
		g.fill(r, levelColors[k])
		g.textCentered(r, &freesans.Bold12pt7b, lv.Name, textColor, 34)
	}
	if g.OnExit != nil {
		r := backButton()
		g.fill(r, btnColor)
		g.textCentered(r, &proggy.TinySZ8pt7b, "< GAMES", textColor, 18)
	}
	g.d.Display()
}

func (g *Game) tapMenu(x, y int16) {
	if g.OnExit != nil && backButton().has(x, y) {
		g.OnExit()
		return
	}
	for k := 0; k < 2; k++ {
		if colorButton(k).has(x, y) {
			g.humanBlack = k == 0
			g.DrawMenu()
			return
		}
	}
	for k := range levels {
		if levelButton(k).has(x, y) {
			g.level = k
			g.start()
			return
		}
	}
}

// ---- game flow ----

func (g *Game) start() {
	g.pos = Start()
	g.blackMove = true
	g.last = -1
	g.history = g.history[:0]
	g.cpuInfo = ""
	g.pending = g.pending[:0]
	g.drawAll()
	g.nextTurn()
}

func (g *Game) humanToMove() bool { return g.blackMove == g.humanBlack }

// nextTurn decides what happens after a move (or at the start).
func (g *Game) nextTurn() {
	switch {
	case g.pos.Over():
		g.phase = gameOver
		b, w := g.counts()
		mine, theirs := b, w
		if !g.humanBlack {
			mine, theirs = w, b
		}
		score := strconv.Itoa(mine) + "-" + strconv.Itoa(theirs)
		switch {
		case mine > theirs:
			g.message = "YOU WIN " + score
		case mine < theirs:
			g.message = "CPU WINS " + score
		default:
			g.message = "DRAW " + score
		}
	case g.pos.Moves() == 0:
		g.phase = passing
		g.timer = 0
		if g.humanToMove() {
			g.message = "YOU PASS"
		} else {
			g.message = "CPU PASSES"
		}
	case g.humanToMove():
		g.phase = humanTurn
		g.message = "YOUR TURN"
	default:
		g.phase = cpuTurn
		g.message = "THINKING..."
	}
	g.refresh()
}

// play makes a move for the side to move and starts the turning animation.
func (g *Game) play(sq int) {
	flips := g.pos.Flips(sq)
	g.pos = g.pos.Play(sq)
	g.blackMove = !g.blackMove
	g.last = sq
	// Redraw the new stone first, then the turned ones from near to far.
	g.pending = append(g.pending[:0], sq)
	r0, c0 := sq/8, sq%8
	for dist := 1; dist < 8; dist++ {
		for b := flips; b != 0; b &= b - 1 {
			f := bits.TrailingZeros64(b)
			dr, dc := abs(f/8-r0), abs(f%8-c0)
			if max(dr, dc) == dist {
				g.pending = append(g.pending, f)
			}
		}
	}
	g.phase = animating
	g.timer = flipStep // show the new stone right away
	g.refresh()
}

func (g *Game) cpuMove() {
	start := g.Now()
	sq, depth, nodes := BestMove(g.pos, levels[g.level], g.noise)
	ms := g.Now().Sub(start).Milliseconds()
	d := "END"
	if depth > 0 {
		d = strconv.Itoa(depth)
	}
	g.cpuInfo = "depth " + d + "  " + strconv.Itoa(nodes) + " nodes  " + strconv.Itoa(int(ms)) + "ms"
	println("cpu:", levels[g.level].Name, g.cpuInfo)
	g.play(sq)
}

func (g *Game) undo() {
	if len(g.history) == 0 {
		return
	}
	s := g.history[len(g.history)-1]
	g.history = g.history[:len(g.history)-1]
	g.pos, g.blackMove, g.last = s.pos, s.blackMove, s.last
	g.pending = g.pending[:0]
	g.nextTurn()
}

// Update reads the touch panel and advances the game by dt seconds.
func (g *Game) Update(dt float32) {
	g.clock += dt
	x, y, pressed := g.t.ReadTouch()
	tap := pressed && !g.wasTouch && g.clock-g.lastTap >= tapGap
	g.wasTouch = pressed
	if tap {
		g.lastTap = g.clock
	}

	if g.phase == menu {
		if tap {
			g.tapMenu(x, y)
		}
		return
	}
	if tap {
		switch {
		case newButton().has(x, y):
			g.DrawMenu()
			return
		case undoButton().has(x, y) && (g.phase == humanTurn || g.phase == gameOver):
			g.undo()
			return
		case g.phase == humanTurn && boardRect().has(x, y):
			c := int((x - boardX) / cellSz)
			r := int((y - boardY) / cellSz)
			if sq := r*8 + c; r < 8 && c < 8 && g.pos.Moves()&(1<<sq) != 0 {
				g.history = append(g.history, snapshot{g.pos, g.blackMove, g.last})
				g.play(sq)
				return
			}
		}
	}

	switch g.phase {
	case animating:
		g.timer += dt
		for g.timer >= flipStep && len(g.pending) > 0 {
			g.timer -= flipStep
			g.drawSquare(g.pending[0])
			g.pending = g.pending[1:]
		}
		if len(g.pending) == 0 {
			g.nextTurn()
		}
	case cpuTurn:
		g.cpuMove() // "THINKING..." was drawn by the previous update
	case passing:
		g.timer += dt
		if g.timer >= passDelay {
			g.pos = g.pos.Pass()
			g.blackMove = !g.blackMove
			g.nextTurn()
		}
	}
	g.d.Display()
}

// ---- drawing ----

func (g *Game) counts() (b, w int) {
	me, opp := g.pos.Count()
	if g.blackMove {
		return me, opp
	}
	return opp, me
}

func (g *Game) look(sq int) sqLook {
	bit := uint64(1) << sq
	blackBB, whiteBB := g.pos.Me, g.pos.Opp
	if !g.blackMove {
		blackBB, whiteBB = whiteBB, blackBB
	}
	l := sqLook{last: sq == g.last}
	switch {
	case blackBB&bit != 0:
		l.stone = black
	case whiteBB&bit != 0:
		l.stone = white
	case g.phase == humanTurn && g.pos.Moves()&bit != 0:
		l.hint = true
	}
	return l
}

func (g *Game) drawSquare(sq int) {
	l := g.look(sq)
	r := squareRect(sq)
	g.fill(r, boardColor)
	cx, cy := r.x+r.w/2, r.y+r.h/2
	switch l.stone {
	case black, white:
		g.stone(cx, cy, l.stone == black)
		if l.last {
			tinydraw.FilledCircle(g.d, cx, cy, dotR, lastColor)
		}
	default:
		if l.hint {
			tinydraw.FilledCircle(g.d, cx, cy, dotR, hintColor)
		}
	}
	g.drawn[sq] = l
}

func (g *Game) drawAll() {
	w, h := g.d.Size()
	g.fill(rect{0, 0, w, h}, bgColor)
	br := boardRect()
	g.fill(br, boardColor)
	for i := int16(0); i <= 8; i++ {
		g.fill(rect{br.x + i*cellSz, br.y, 1, br.h}, lineColor)
		g.fill(rect{br.x, br.y + i*cellSz, br.w, 1}, lineColor)
	}
	// Small dots at the four classic points.
	for _, p := range [][2]int16{{2, 2}, {6, 2}, {2, 6}, {6, 6}} {
		tinydraw.FilledCircle(g.d, br.x+p[0]*cellSz, br.y+p[1]*cellSz, 2, lineColor)
	}
	for sq := range g.drawn {
		g.drawn[sq] = sqLook{stone: -1} // force a redraw
	}
	g.drawnPanel = ""
	for _, b := range []struct {
		r    rect
		name string
	}{{undoButton(), "UNDO"}, {newButton(), "NEW"}} {
		g.fill(b.r, btnColor)
		g.textCentered(b.r, &proggy.TinySZ8pt7b, b.name, textColor, 24)
	}
	g.refresh()
}

// refresh redraws the squares that changed (except those waiting for the
// animation) and the panel.
func (g *Game) refresh() {
	var waiting uint64
	for _, sq := range g.pending {
		waiting |= 1 << sq
	}
	for sq := 0; sq < 64; sq++ {
		if waiting&(1<<sq) == 0 && g.look(sq) != g.drawn[sq] {
			g.drawSquare(sq)
		}
	}
	g.drawPanel()
	g.d.Display()
}

func (g *Game) drawPanel() {
	b, w := g.counts()
	state := strconv.Itoa(b) + "|" + strconv.Itoa(w) + "|" + strconv.FormatBool(g.blackMove) + "|" +
		g.message + "|" + g.cpuInfo
	if state == g.drawnPanel {
		return
	}
	g.drawnPanel = state
	g.fill(rect{panelX, 0, 480 - panelX, 206}, bgColor)

	// Stone counts; the side to move is underlined.
	you, cpu := "YOU", "CPU"
	nameB, nameW := you, cpu
	if !g.humanBlack {
		nameB, nameW = cpu, you
	}
	for k, side := range []struct {
		black bool
		n     int
		name  string
	}{{true, b, nameB}, {false, w, nameW}} {
		x := panelX + int16(k)*98
		g.stone(x+16, 36, side.black)
		tinyfont.WriteLine(g.d, &freesans.Bold18pt7b, x+36, 48, strconv.Itoa(side.n), textColor)
		tinyfont.WriteLine(g.d, &proggy.TinySZ8pt7b, x+4, 74, side.name, dimText)
		if side.black == g.blackMove && g.phase != gameOver {
			g.fill(rect{x, 80, 88, 3}, turnColor)
		}
	}
	tinyfont.WriteLine(g.d, &proggy.TinySZ8pt7b, panelX, 108, "CPU LEVEL: "+levels[g.level].Name, dimText)
	msgColor := textColor
	if g.phase == gameOver {
		msgColor = turnColor
	}
	tinyfont.WriteLine(g.d, &freesans.Bold12pt7b, panelX, 146, g.message, msgColor)
	if g.cpuInfo != "" {
		tinyfont.WriteLine(g.d, &proggy.TinySZ8pt7b, panelX, 180, g.cpuInfo, dimText)
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
