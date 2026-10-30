package main

import (
	"github.com/sat0ken/tinygo-cyd/games/ants"
	"github.com/sat0ken/tinygo-cyd/games/atoms"
	"github.com/sat0ken/tinygo-cyd/games/chess"
	"github.com/sat0ken/tinygo-cyd/games/eggs"
	"github.com/sat0ken/tinygo-cyd/games/fifteen"
	"github.com/sat0ken/tinygo-cyd/games/ghostrac"
	"github.com/sat0ken/tinygo-cyd/games/life"
	"github.com/sat0ken/tinygo-cyd/games/maze"
	"github.com/sat0ken/tinygo-cyd/games/pacman"
	"github.com/sat0ken/tinygo-cyd/games/picoinvaders"
	"github.com/sat0ken/tinygo-cyd/games/pictor"
	"github.com/sat0ken/tinygo-cyd/games/raptor"
	"github.com/sat0ken/tinygo-cyd/games/sokoban"
	"github.com/sat0ken/tinygo-cyd/games/tetris"
	"github.com/sat0ken/tinygo-cyd/games/train"
	"github.com/sat0ken/tinygo-cyd/games/tvtennis"
	"github.com/sat0ken/tinygo-cyd/games/vegaslot"
	"github.com/sat0ken/tinygo-cyd/games/zoom"
	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/picopad"
)

// starter makes a game: its console and its Run.
type starter func(d hal.Display, t hal.Touch, seed uint32) (*picopad.Console, func())

// entry is a game of the menu.
type entry struct {
	name  string // up to 18 characters
	about string // up to 40
	back  string // how the game comes back here (up to 40)
	start starter
}

// The ways back to the menu.
const (
	backY     = "Y: back to this menu"
	backOver  = "Game over or Y: back to this menu"
	backMatch = "End of a game or Y: back to this menu"
)

// entries are the games, in the order of the examples (15 to 33).
var entries = []entry{
	{"Tetris", "Falling bricks. UP/X turn, B pause", backOver, func(d hal.Display, t hal.Touch, seed uint32) (*picopad.Console, func()) {
		g := tetris.New(d, t, seed)
		return g.Console, g.Run
	}},
	{"Maze", "Find the way out. A shows the door", backY, func(d hal.Display, t hal.Touch, seed uint32) (*picopad.Console, func()) {
		g := maze.New(d, t, seed)
		return g.Console, g.Run
	}},
	{"Fifteen", "Slide the tiles in order. X solves", backY, func(d hal.Display, t hal.Touch, seed uint32) (*picopad.Console, func()) {
		g := fifteen.New(d, t, seed)
		return g.Console, g.Run
	}},
	{"Life", "Game of Life. A cell, B run, X next", backY, func(d hal.Display, t hal.Touch, seed uint32) (*picopad.Console, func()) {
		g := life.New(d, t)
		return g.Console, g.Run
	}},
	{"TV Tennis", "LEFT/RIGHT: against the CPU, DOWN: 2", backY, func(d hal.Display, t hal.Touch, seed uint32) (*picopad.Console, func()) {
		g := tvtennis.New(d, t, seed)
		return g.Console, g.Run
	}},
	{"Exploding Atoms", "Chain reactions for 2 to 5 players", backMatch, func(d hal.Display, t hal.Touch, seed uint32) (*picopad.Console, func()) {
		g := atoms.New(d, t, seed)
		return g.Console, g.Run
	}},
	{"Chess", "Chess against the computer", "Checkmate or Y: back to this menu", func(d hal.Display, t hal.Touch, seed uint32) (*picopad.Console, func()) {
		g := chess.New(d, t, seed)
		return g.Console, g.Run
	}},
	{"Eggs", "Reversi with eggs and hens", backMatch, func(d hal.Display, t hal.Touch, seed uint32) (*picopad.Console, func()) {
		g := eggs.New(d, t, seed)
		return g.Console, g.Run
	}},
	{"Train", "Collect all and reach the gate", backY, func(d hal.Display, t hal.Touch, seed uint32) (*picopad.Console, func()) {
		g := train.New(d, t, seed)
		return g.Console, g.Run
	}},
	{"Space Invaders", "Shoot the aliens. A fires", backOver, func(d hal.Display, t hal.Touch, seed uint32) (*picopad.Console, func()) {
		g := picoinvaders.New(d, t, seed)
		return g.Console, g.Run
	}},
	{"PacMan", "Eat the dots, flee the ghosts", backOver, func(d hal.Display, t hal.Touch, seed uint32) (*picopad.Console, func()) {
		g := pacman.New(d, t, seed)
		return g.Console, g.Run
	}},
	{"Sokoban", "Push the boxes: 2,020 levels", "Y on the authors: back to this menu", func(d hal.Display, t hal.Touch, seed uint32) (*picopad.Console, func()) {
		g := sokoban.New(d, t, seed)
		return g.Console, g.Run
	}},
	{"Vegaslot", "Slot machine. A spins, B bets", backY, func(d hal.Display, t hal.Touch, seed uint32) (*picopad.Console, func()) {
		g := vegaslot.New(d, t, seed)
		return g.Console, g.Run
	}},
	{"Raptor", "Vertical shooter. A fires, B weapon", backOver, func(d hal.Display, t hal.Touch, seed uint32) (*picopad.Console, func()) {
		g := raptor.New(d, t, seed)
		return g.Console, g.Run
	}},
	{"Ants", "Card game: build up your castle", backMatch, func(d hal.Display, t hal.Touch, seed uint32) (*picopad.Console, func()) {
		g := ants.New(d, t, seed)
		return g.Console, g.Run
	}},
	{"Ghost Racing", "Race your best lap. UP/X faster", backY, func(d hal.Display, t hal.Touch, seed uint32) (*picopad.Console, func()) {
		g := ghostrac.New(d, t)
		return g.Console, g.Run
	}},
	{fifteen2Name, "The 15 puzzle with 13 pictures", "Y and Y on the list: back to this menu", startFifteen2},
	{"Pictor", "Side shooter: collect PicoPad parts", "Game over, or Y and X: back here", func(d hal.Display, t hal.Touch, seed uint32) (*picopad.Console, func()) {
		g := pictor.New(d, t, seed)
		return g.Console, g.Run
	}},
	{"Gate Runner", "Fly through the gates (ZOOM)", backOver, func(d hal.Display, t hal.Touch, seed uint32) (*picopad.Console, func()) {
		g := zoom.New(d, t, seed)
		return g.Console, g.Run
	}},
}

const fifteen2Name = "Fifteen Pictures"

// The menu: two columns of rows names under the title, the about and back
// lines of the chosen game at the bottom.
const (
	rows    = 10
	listY   = 40
	rowH    = 16
	colX    = 4
	colW    = 156
	aboutY  = 206
	backLnY = 222
)

// Launcher shows the menu of the PicoPad games and plays them one after
// another: a game comes back to the menu at its end (picopad's OneGame).
type Launcher struct {
	d     hal.Display
	t     hal.Touch
	Pad   hal.Pad       // the game controller (nil: none)
	Clock picopad.Clock // nil: the real time
	Seed  func() uint32 // the random seed of a game
	games []entry       // the games of this board
	sel   int           // the chosen game
}

// NewLauncher returns the launcher on d with the touch panel t (nil:
// none).
func NewLauncher(d hal.Display, t hal.Touch, seed func() uint32) *Launcher {
	l := &Launcher{d: d, t: t, Seed: seed}
	for _, e := range entries {
		if e.name != fifteen2Name || hasFifteen2 {
			l.games = append(l.games, e)
		}
	}
	return l
}

// Run shows the menu and plays the games chosen, for ever.
func (l *Launcher) Run() {
	for {
		l.play(l.menu())
	}
}

// console makes a console for the menu or a game.
func (l *Launcher) console(c *picopad.Console) *picopad.Console {
	c.Pad = l.Pad
	if l.Clock != nil {
		c.Clock = l.Clock
	}
	return c
}

// play plays game i until it comes back.
func (l *Launcher) play(i int) {
	c, run := l.games[i].start(l.d, l.t, l.Seed())
	l.console(c)
	c.OneGame = true
	run()
	c.Close() // its frame buffer for the next console
}

// menu shows the games until one is chosen: the arrows move, A plays; a
// tap chooses a game, a tap on the chosen one plays it.
func (l *Launcher) menu() int {
	c := l.console(picopad.New(l.d, nil, 0)) // the touch panel is read here
	defer c.Close()
	l.draw(c)
	n := len(l.games)
	touched := true // a touch still there from the game does not count
	for {
		sel := l.sel
		switch c.KeyGet() {
		case picopad.KeyUp:
			sel = (sel + n - 1) % n
		case picopad.KeyDown:
			sel = (sel + 1) % n
		case picopad.KeyLeft:
			sel = max(sel-rows, 0)
		case picopad.KeyRight:
			sel = min(sel+rows, n-1)
		case picopad.KeyA:
			return l.sel
		}
		if l.t != nil {
			x, y, pressed := l.t.ReadTouch()
			if pressed && !touched {
				if i := l.itemAt(x, y); i == l.sel {
					return i
				} else if i >= 0 {
					sel = i
				}
			}
			touched = pressed
		}
		if sel != l.sel {
			l.sel = sel
			l.draw(c)
		}
		c.WaitMs(20)
	}
}

// itemAt returns the game at x, y of the display (-1: none).
func (l *Launcher) itemAt(x, y int16) int {
	w, h := l.d.Size()
	px, py := int(x)-int(w-picopad.Width)/2, int(y)-int(h-picopad.Height)/2
	col, row := (px-colX)/colW, (py-listY)/rowH
	if px < colX || py < listY || col > 1 || row >= rows {
		return -1
	}
	if i := col*rows + row; i < len(l.games) {
		return i
	}
	return -1
}

// draw draws the menu.
func (l *Launcher) draw(c *picopad.Console) {
	c.DrawClear()
	c.SelFont8x16()
	c.DrawText2("PICOPAD GAMES", (picopad.Width-13*16)/2, 0, picopad.ColYellow)
	c.SelFont8x8()
	c.DrawText("ported from PicoLibSDK", (picopad.Width-22*8)/2, 30, picopad.ColGray)
	c.SelFont8x16()
	for i, e := range l.games {
		x, y := colX+i/rows*colW, listY+i%rows*rowH
		fg, bg := picopad.ColWhite, picopad.ColBlack
		if i == l.sel {
			fg, bg = picopad.ColBlack, picopad.ColYellow
			c.DrawRect(x, y, colW-4, rowH, bg)
		}
		c.DrawTextBg(e.name, x+4, y, fg, bg)
	}
	e := &l.games[l.sel]
	c.DrawText(e.about, 0, aboutY, picopad.ColCyan)
	c.DrawText(e.back, 0, backLnY, picopad.ColGreen)
	c.DispUpdate()
}
