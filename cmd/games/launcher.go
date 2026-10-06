package main

import (
	"image/color"
	"time"

	"github.com/sat0ken/tinygo-cyd/games/cards"
	"github.com/sat0ken/tinygo-cyd/games/invaders"
	"github.com/sat0ken/tinygo-cyd/games/othello"
	"github.com/sat0ken/tinygo-cyd/games/poker"
	"github.com/sat0ken/tinygo-cyd/games/solitaire"
	"github.com/sat0ken/tinygo-cyd/games/sudoku"
	"github.com/sat0ken/tinygo-cyd/hal"
	"tinygo.org/x/tinydraw"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/freesans"
	"tinygo.org/x/tinyfont/proggy"
)

// game is what the launcher needs from a game: one step per frame.
type game interface {
	Update(dt float32)
}

var (
	bgColor    = color.RGBA{18, 22, 32, 255}
	tileColor  = color.RGBA{40, 48, 70, 255}
	textColor  = color.RGBA{255, 255, 255, 255}
	dimText    = color.RGBA{150, 160, 180, 255}
	greenBoard = color.RGBA{0, 120, 70, 255}
	invColor   = color.RGBA{80, 220, 255, 255}
	gridColor  = color.RGBA{170, 180, 200, 255}
	digitColor = color.RGBA{120, 200, 255, 255}
)

type rect struct{ x, y, w, h int16 }

func (r rect) has(x, y int16) bool { return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h }

type entry struct {
	name, about string
	icon        func(l *Launcher, r rect)
	start       func(l *Launcher) game
}

var entries = []entry{
	{"INVADERS", "hold to fire", drawInvaderIcon, func(l *Launcher) game {
		g := invaders.NewGame(l.d, l.t, l.Seed())
		g.OnExit = l.exit
		g.DrawAll()
		return g
	}},
	{"SUDOKU", "3 levels", drawSudokuIcon, func(l *Launcher) game {
		g := sudoku.NewGame(l.d, l.t, l.Seed)
		g.OnExit = l.exit
		g.DrawMenu()
		return g
	}},
	{"OTHELLO", "vs the CPU", drawOthelloIcon, func(l *Launcher) game {
		g := othello.NewGame(l.d, l.t, l.Seed())
		g.OnExit = l.exit
		g.DrawMenu()
		return g
	}},
	{"SOLITAIRE", "draw 1 or 3", drawSolitaireIcon, func(l *Launcher) game {
		g := solitaire.NewGame(l.d, l.t, l.Seed)
		g.OnExit = l.exit
		g.Start()
		return g
	}},
	{"POKER", "jacks or better", drawPokerIcon, func(l *Launcher) game {
		g := poker.NewGame(l.d, l.t, l.Seed)
		g.OnExit = l.exit
		g.Start()
		return g
	}},
}

// tileRect is tile k of a 3x2 grid (one place is still free).
func tileRect(k int) rect { return rect{10 + int16(k%3)*156, 32 + int16(k/3)*108, 148, 100} }

// Fonts of the tiles (TestTilesFit checks that the texts fit).
var (
	nameFont  = &freesans.Bold9pt7b
	aboutFont = &proggy.TinySZ8pt7b
)

// Launcher shows the list of games and runs the chosen one. Each game gets
// an OnExit callback, which shows a "< GAMES" button on its first screen.
type Launcher struct {
	d hal.Display
	t hal.Touch

	// Seed returns a new random seed for a game.
	Seed func() uint32

	current     game
	leaving     bool // the current game asked to exit
	waitRelease bool // ignore the touch until the finger is lifted
	wasTouch    bool
}

func NewLauncher(d hal.Display, t hal.Touch, seed func() uint32) *Launcher {
	return &Launcher{d: d, t: t, Seed: seed}
}

func (l *Launcher) exit() { l.leaving = true }

// Playing returns the name of the game being played, or "".
func (l *Launcher) Playing() string {
	if l.current == nil {
		return ""
	}
	switch l.current.(type) {
	case *invaders.Game:
		return "INVADERS"
	case *sudoku.Game:
		return "SUDOKU"
	case *othello.Game:
		return "OTHELLO"
	case *solitaire.Game:
		return "SOLITAIRE"
	case *poker.Game:
		return "POKER"
	}
	return "?"
}

// DrawMenu draws the list of games.
func (l *Launcher) DrawMenu() {
	w, h := l.d.Size()
	l.d.FillRectangle(0, 0, w, h, bgColor)
	center(l.d, rect{0, 0, w, 0}, &freesans.Bold12pt7b, "TINYGO GAMES", textColor, 23)
	for k, e := range entries {
		r := tileRect(k)
		l.d.FillRectangle(r.x, r.y, r.w, r.h, tileColor)
		e.icon(l, rect{r.x + 8, r.y + 5, r.w - 16, 56})
		center(l.d, r, nameFont, e.name, textColor, 78)
		center(l.d, r, aboutFont, e.about, dimText, 93)
	}
	center(l.d, rect{0, 0, w, 0}, &proggy.TinySZ8pt7b, "tap a game", dimText, 262)
	l.d.Display()
}

// Update runs one frame of the launcher or of the current game.
func (l *Launcher) Update(dt float32) {
	if l.waitRelease {
		if _, _, pressed := l.t.ReadTouch(); pressed {
			return
		}
		l.waitRelease = false
		l.wasTouch = false
	}
	if l.current != nil {
		l.current.Update(dt)
		if l.leaving {
			l.leaving = false
			l.current = nil // the game is garbage now
			l.DrawMenu()
			l.waitRelease = true // the finger is still on the "< GAMES" button
		}
		return
	}
	x, y, pressed := l.t.ReadTouch()
	tap := pressed && !l.wasTouch
	l.wasTouch = pressed
	if !tap {
		return
	}
	for k, e := range entries {
		if tileRect(k).has(x, y) {
			println("games: start", e.name)
			l.current = e.start(l)
			// The finger is still on the tile: do not let the game see it
			// as a tap on its first screen.
			l.waitRelease = true
			return
		}
	}
}

func center(d hal.Display, r rect, font tinyfont.Fonter, s string, c color.RGBA, baseline int16) {
	_, w := tinyfont.LineWidth(font, s)
	tinyfont.WriteLine(d, font, r.x+(r.w-int16(w))/2, r.y+baseline, s, c)
}

// ---- icons (drawn here, so the games do not need to export anything) ----

func drawInvaderIcon(l *Launcher, r rect) {
	art := []string{
		"..#.....#..",
		"...#...#...",
		"..#######..",
		".##.###.##.",
		"###########",
		"#.#######.#",
		"#.#.....#.#",
		"...##.##...",
	}
	const px = 6
	x0 := r.x + (r.w-int16(len(art[0]))*px)/2
	y0 := r.y + (r.h-int16(len(art))*px)/2
	for y, row := range art {
		for x, c := range row {
			if c == '#' {
				l.d.FillRectangle(x0+int16(x)*px, y0+int16(y)*px, px, px, invColor)
			}
		}
	}
}

func drawSudokuIcon(l *Launcher, r rect) {
	const cell = 18
	x0 := r.x + (r.w-3*cell)/2
	y0 := r.y + (r.h-3*cell)/2
	for i := int16(0); i <= 3; i++ {
		l.d.FillRectangle(x0+i*cell, y0, 2, 3*cell+2, gridColor)
		l.d.FillRectangle(x0, y0+i*cell, 3*cell+2, 2, gridColor)
	}
	for _, d := range []struct {
		c, r int16
		s    string
	}{{0, 0, "5"}, {2, 0, "3"}, {1, 1, "9"}, {0, 2, "7"}, {2, 2, "1"}} {
		tinyfont.WriteLine(l.d, &freesans.Bold9pt7b, x0+d.c*cell+5, y0+d.r*cell+16, d.s, digitColor)
	}
}

func drawOthelloIcon(l *Launcher, r rect) {
	const cell = 27
	x0 := r.x + (r.w-2*cell)/2
	y0 := r.y + (r.h-2*cell)/2
	l.d.FillRectangle(x0, y0, 2*cell, 2*cell, greenBoard)
	for k := int16(0); k < 4; k++ {
		cx := x0 + (k%2)*cell + cell/2
		cy := y0 + (k/2)*cell + cell/2
		c := color.RGBA{240, 240, 240, 255}
		if k == 1 || k == 2 {
			c = color.RGBA{25, 25, 25, 255}
		}
		tinydraw.FilledCircle(l.d, cx, cy, 11, c)
	}
}

func drawSolitaireIcon(l *Launcher, r rect) {
	// Three fanned cards: a back and two faces.
	const w, h, step = 46, 54, 16
	x0 := r.x + (r.w-w-2*step)/2
	y0 := r.y + (r.h-h)/2
	cards.DrawBack(l.d, x0, y0, w, h)
	cards.DrawFace(l.d, cards.New(cards.Hearts, 1), x0+step, y0, w, h, false)
	cards.DrawFace(l.d, cards.New(cards.Spades, 13), x0+2*step, y0, w, h, false)
}

func drawPokerIcon(l *Launcher, r rect) {
	// A royal flush, fanned; only the corners of the first four show.
	const w, h, step = 38, 54, 16
	x0 := r.x + (r.w-w-4*step)/2
	y0 := r.y + (r.h-h)/2
	for k, rank := range []int{10, 11, 12, 13, 1} {
		cards.DrawFaceSmall(l.d, cards.New(cards.Diamonds, rank), x0+int16(k)*step, y0, w, h)
	}
}

// vsyncWaiter is implemented by rgblcd.Device.
type vsyncWaiter interface {
	WaitVSync(timeout time.Duration) bool
}
