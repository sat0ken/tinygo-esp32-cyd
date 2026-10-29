package main

import (
	"image/color"
	"time"

	"github.com/sat0ken/tinygo-cyd/games/cards"
	"github.com/sat0ken/tinygo-cyd/games/invaders"
	"github.com/sat0ken/tinygo-cyd/games/kart"
	"github.com/sat0ken/tinygo-cyd/games/othello"
	"github.com/sat0ken/tinygo-cyd/games/poker"
	"github.com/sat0ken/tinygo-cyd/games/solitaire"
	"github.com/sat0ken/tinygo-cyd/games/starguard"
	"github.com/sat0ken/tinygo-cyd/games/sudoku"
	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/pad"
	"github.com/sat0ken/tinygo-cyd/romfont/freesans"
	"github.com/sat0ken/tinygo-cyd/romfont/proggy"
	"tinygo.org/x/tinydraw"
	"tinygo.org/x/tinyfont"
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
	focusColor = color.RGBA{255, 210, 40, 255}
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
		g.OnExit, g.Pad = l.exit, l.Pad
		g.DrawAll()
		return g
	}},
	{"SUDOKU", "3 levels", drawSudokuIcon, func(l *Launcher) game {
		g := sudoku.NewGame(l.d, l.t, l.Seed)
		g.OnExit, g.Pad = l.exit, l.Pad
		g.DrawMenu()
		return g
	}},
	{"OTHELLO", "vs the CPU", drawOthelloIcon, func(l *Launcher) game {
		g := othello.NewGame(l.d, l.t, l.Seed())
		g.OnExit, g.Pad = l.exit, l.Pad
		g.DrawMenu()
		return g
	}},
	{"SOLITAIRE", "draw 1 or 3", drawSolitaireIcon, func(l *Launcher) game {
		g := solitaire.NewGame(l.d, l.t, l.Seed)
		g.OnExit, g.Pad = l.exit, l.Pad
		g.Start()
		return g
	}},
	{"POKER", "jacks or better", drawPokerIcon, func(l *Launcher) game {
		g := poker.NewGame(l.d, l.t, l.Seed)
		g.OnExit, g.Pad = l.exit, l.Pad
		g.Start()
		return g
	}},
	{"KART", "race the CPU", drawKartIcon, func(l *Launcher) game {
		g := kart.NewGame(l.d, l.t)
		g.OnExit, g.Pad = l.exit, l.Pad
		g.DrawMenu()
		return g
	}},
	{"SPACE", "star guard", drawSpaceIcon, func(l *Launcher) game {
		g := starguard.NewGame(l.d, l.t, l.Seed())
		g.OnExit, g.Pad = l.exit, l.Pad
		g.DrawMenu()
		return g
	}},
}

// Fonts of the tiles (TestTilesFit checks that the texts fit).
var (
	nameFont  = &freesans.Bold9pt7b
	aboutFont = &proggy.TinySZ8pt7b
)

// layout is where the tiles are: a grid of cols columns.
type layout struct {
	titleBase           int16
	cols                int
	tile0               rect // the first tile
	stepX, stepY        int16
	icon                rect  // the icon inside a tile; the icons scale with its height
	nameBase, aboutBase int16 // aboutBase 0: no line under the name
	footBase            int16 // "tap a game"
}

// layoutFor returns the layout for a w x h screen: 4x2 tiles on 480x272,
// 3x3 smaller tiles without the line under the name on 320x240
// (ESP32-2432S028R).
func layoutFor(w, h int16) layout {
	if w >= 480 && h >= 272 {
		return layout{
			titleBase: 23, cols: 4, tile0: rect{7, 32, 112, 100}, stepX: 118, stepY: 108,
			icon: rect{4, 5, 112 - 8, 56}, nameBase: 78, aboutBase: 93, footBase: 262,
		}
	}
	// Tiles 104 pixels wide: SOLITAIRE in bold is 99.
	return layout{
		titleBase: 21, cols: 3, tile0: rect{1, 26, 104, 65}, stepX: 106, stepY: 68,
		icon: rect{4, 4, 104 - 8, 38}, nameBase: 59, footBase: h - 3,
	}
}

// tileRect is tile k of the grid.
func (l *layout) tileRect(k int) rect {
	r := l.tile0
	r.x += int16(k%l.cols) * l.stepX
	r.y += int16(k/l.cols) * l.stepY
	return r
}

// Launcher shows the list of games and runs the chosen one. Each game gets
// an OnExit callback, which shows a "< GAMES" button on its first screen.
type Launcher struct {
	d hal.Display
	t hal.Touch
	l layout

	// Seed returns a new random seed for a game.
	Seed func() uint32

	// Pad, if set, is the game controller: the stick moves the focus
	// between the tiles, A (or pressing the stick) starts a game. The
	// games get it too.
	Pad   hal.Pad
	in    pad.Input
	focus int

	current     game
	leaving     bool // the current game asked to exit
	waitRelease bool // ignore the touch until the finger is lifted
	wasTouch    bool
}

func NewLauncher(d hal.Display, t hal.Touch, seed func() uint32) *Launcher {
	w, h := d.Size()
	return &Launcher{d: d, t: t, l: layoutFor(w, h), Seed: seed}
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
	case *kart.Game:
		return "KART"
	case *starguard.Game:
		return "SPACE"
	}
	return "?"
}

// DrawMenu draws the list of games.
func (l *Launcher) DrawMenu() {
	w, h := l.d.Size()
	l.d.FillRectangle(0, 0, w, h, bgColor)
	center(l.d, rect{0, 0, w, 0}, &freesans.Bold12pt7b, "TINYGO GAMES", textColor, l.l.titleBase)
	for k := range entries {
		l.drawTile(k)
	}
	center(l.d, rect{0, 0, w, 0}, &proggy.TinySZ8pt7b, "tap a game", dimText, l.l.footBase)
	l.d.Display()
}

// drawTile draws tile k, with a frame if it has the focus.
func (l *Launcher) drawTile(k int) {
	e, r := &entries[k], l.l.tileRect(k)
	l.d.FillRectangle(r.x, r.y, r.w, r.h, tileColor)
	i := l.l.icon
	e.icon(l, rect{r.x + i.x, r.y + i.y, i.w, i.h})
	center(l.d, r, nameFont, e.name, textColor, l.l.nameBase)
	if l.l.aboutBase > 0 {
		center(l.d, r, aboutFont, e.about, dimText, l.l.aboutBase)
	}
	if l.in.Used && k == l.focus {
		pad.Frame(l.d, r.x, r.y, r.w, r.h, 3, focusColor)
	}
}

// updatePad moves the focus and returns the game to start, or -1.
func (l *Launcher) updatePad(dt float32) int {
	used := l.in.Used
	l.in.Update(l.Pad, dt)
	if !l.in.Used {
		return -1
	}
	if !used {
		l.drawTile(l.focus) // the first use shows the focus
		l.d.Display()
	}
	if d := l.in.Direction(); d != 0 {
		var boxes [9]pad.Box // the grids have up to nine places
		for k := range entries {
			r := l.l.tileRect(k)
			boxes[k] = pad.Box{X: r.x, Y: r.y, W: r.w, H: r.h}
		}
		if f := pad.Move(boxes[:len(entries)], l.focus, d); f != l.focus {
			old := l.focus
			l.focus = f
			l.drawTile(old)
			l.drawTile(f)
			l.d.Display()
		}
	}
	if l.in.Pressed(hal.PadA | hal.PadOK) {
		return l.focus
	}
	return -1
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
			l.in.Reset()    // the game read the controller meanwhile
			l.DrawMenu()
			l.waitRelease = true // the finger is still on the "< GAMES" button
		}
		return
	}
	if k := l.updatePad(dt); k >= 0 {
		l.start(k)
		return
	}
	x, y, pressed := l.t.ReadTouch()
	tap := pressed && !l.wasTouch
	l.wasTouch = pressed
	if !tap {
		return
	}
	for k := range entries {
		if l.l.tileRect(k).has(x, y) {
			l.focus = k
			l.start(k)
			// The finger is still on the tile: do not let the game see it
			// as a tap on its first screen.
			l.waitRelease = true
			return
		}
	}
}

func (l *Launcher) start(k int) {
	println("games: start", entries[k].name)
	l.current = entries[k].start(l)
}

func center(d hal.Display, r rect, font tinyfont.Fonter, s string, c color.RGBA, baseline int16) {
	_, w := tinyfont.LineWidth(font, s)
	tinyfont.WriteLine(d, font, r.x+(r.w-int16(w))/2, r.y+baseline, s, c)
}

// ---- icons (drawn here, so the games do not need to export anything) ----

// The icons are drawn for a 56-pixel high rectangle and scaled with its
// height.

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
	px := r.h / 9
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
	cell := r.h * 18 / 56
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
		if cell < 14 { // small icons: the small font
			tinyfont.WriteLine(l.d, &proggy.TinySZ8pt7b, x0+d.c*cell+(cell-5)/2+1, y0+d.r*cell+(cell+6)/2+1, d.s, digitColor)
			continue
		}
		tinyfont.WriteLine(l.d, &freesans.Bold9pt7b, x0+d.c*cell+(cell-9)/2+1, y0+d.r*cell+(cell+12)/2+1, d.s, digitColor)
	}
}

func drawOthelloIcon(l *Launcher, r rect) {
	cell := r.h * 27 / 56
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
		tinydraw.FilledCircle(l.d, cx, cy, cell*11/27, c)
	}
}

func drawSolitaireIcon(l *Launcher, r rect) {
	// Three fanned cards: a back and two faces.
	h := r.h - 2
	w, step := h*46/54, h*16/54
	x0 := r.x + (r.w-w-2*step)/2
	y0 := r.y + (r.h-h)/2
	cards.DrawBack(l.d, x0, y0, w, h)
	cards.DrawFace(l.d, cards.New(cards.Hearts, 1), x0+step, y0, w, h, false)
	cards.DrawFace(l.d, cards.New(cards.Spades, 13), x0+2*step, y0, w, h, false)
}

func drawPokerIcon(l *Launcher, r rect) {
	// A royal flush, fanned; only the corners of the first four show.
	h := r.h - 2
	w, step := h*38/54, h*16/54
	x0 := r.x + (r.w-w-4*step)/2
	y0 := r.y + (r.h-h)/2
	for k, rank := range []int{10, 11, 12, 13, 1} {
		cards.DrawFaceSmall(l.d, cards.New(cards.Diamonds, rank), x0+int16(k)*step, y0, w, h)
	}
}

func drawKartIcon(l *Launcher, r rect) {
	// A road into the distance and a kart from behind.
	u := r.h / 14 // 4 on 56 pixels
	grass := color.RGBA{60, 150, 60, 255}
	l.d.FillRectangle(r.x+(r.w-12*u)/2, r.y, 12*u, r.h, grass)
	for y := int16(0); y < r.h; y++ {
		half := 2*u + y*4*u/r.h
		l.d.FillRectangle(r.x+r.w/2-half, r.y+y, 2*half, 1, color.RGBA{105, 105, 110, 255})
		if (y/(2*u))%2 == 0 {
			l.d.FillRectangle(r.x+r.w/2-u/4, r.y+y, u/2+1, 1, color.RGBA{240, 240, 240, 255})
		}
	}
	cx, by := r.x+r.w/2, r.y+r.h-u
	blue, black := color.RGBA{40, 90, 220, 255}, color.RGBA{20, 20, 24, 255}
	l.d.FillRectangle(cx-4*u, by-3*u, 2*u, 3*u, black) // tyres
	l.d.FillRectangle(cx+2*u, by-3*u, 2*u, 3*u, black)
	l.d.FillRectangle(cx-3*u, by-4*u, 6*u, 3*u, blue) // body
	l.d.FillRectangle(cx-u, by-6*u, 2*u, 2*u, color.RGBA{250, 200, 40, 255})
}

func drawSpaceIcon(l *Launcher, r rect) {
	// Space: stars, a rock, a robot ship and the crosshair.
	w := r.h * 3 / 2
	x0, y0 := r.x+(r.w-w)/2, r.y
	l.d.FillRectangle(x0, y0, w, r.h, color.RGBA{6, 8, 20, 255})
	white := color.RGBA{200, 200, 230, 255}
	for _, p := range [...][2]int16{{3, 4}, {20, 9}, {37, 3}, {46, 20}, {9, 22}, {28, 25}, {52, 6}, {15, 15}} {
		l.d.FillRectangle(x0+p[0]*w/56, y0+p[1]*r.h/28, 1, 1, white)
	}
	u := r.h / 14 // 4 on 56 pixels
	tinydraw.FilledCircle(l.d, x0+w/4, y0+r.h/2+u, 2*u, color.RGBA{130, 115, 100, 255})
	tinydraw.FilledCircle(l.d, x0+w/4-u/2, y0+r.h/2, u/2+1, color.RGBA{175, 160, 145, 255})
	rx, ry := x0+w*3/4-2*u, y0+r.h/3
	l.d.FillRectangle(rx, ry, 4*u, 2*u, color.RGBA{200, 205, 220, 255})
	l.d.FillRectangle(rx+u/2, ry+u/2, u, u, color.RGBA{240, 50, 50, 255})
	l.d.FillRectangle(rx+5*u/2, ry+u/2, u, u, color.RGBA{240, 50, 50, 255})
	green := color.RGBA{80, 255, 120, 255}
	cx, cy := x0+w*3/4, y0+r.h*2/3
	l.d.FillRectangle(cx-3*u, cy, 2*u, 1, green)
	l.d.FillRectangle(cx+u+1, cy, 2*u, 1, green)
	l.d.FillRectangle(cx, cy-3*u, 1, 2*u, green)
	l.d.FillRectangle(cx, cy+u+1, 1, 2*u, green)
}

// vsyncWaiter is implemented by rgblcd.Device.
type vsyncWaiter interface {
	WaitVSync(timeout time.Duration) bool
}
