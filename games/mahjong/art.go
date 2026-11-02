package mahjong

import (
	"image/color"

	"github.com/sat0ken/tinygo-cyd/hal"
)

// The faces of the tiles are drawn with rectangles at any size, so that they
// stay sharp on the 30 x 40 tiles of the hand and on the 11 x 14 tiles of the
// discards of the small screen. The characters are pixel art drawn for this
// game.

var (
	tileBody   = color.RGBA{244, 240, 226, 255}
	tileDim    = color.RGBA{158, 156, 148, 255}
	tileEdge   = color.RGBA{96, 92, 80, 255}
	tileShadow = color.RGBA{196, 186, 160, 255}
	tileBack   = color.RGBA{36, 96, 170, 255}
	tileBack2  = color.RGBA{24, 66, 124, 255}

	inkBlack = color.RGBA{28, 28, 40, 255}
	inkRed   = color.RGBA{204, 34, 34, 255}
	inkGreen = color.RGBA{18, 130, 66, 255}
	inkBlue  = color.RGBA{28, 70, 170, 255}
)

// The characters, 13 x 13 and 9 x 9: E S W N, green, red; and "ten thousand"
// for the characters suit, 9 x 9 and 7 x 7.
var honors13 = [...][13]string{
	{ // east
		"......#......",
		"#############",
		"......#......",
		"..#########..",
		"..#...#...#..",
		"..#########..",
		"..#...#...#..",
		"..#########..",
		"......#......",
		".....###.....",
		"....#.#.#....",
		"..##..#..##..",
		".#....#....#.",
	},
	{ // south
		"......#......",
		"#############",
		"......#......",
		".###########.",
		".#..#...#..#.",
		".#...#.#...#.",
		".#.#######.#.",
		".#....#....#.",
		".#.#######.#.",
		".#....#....#.",
		".#....#....#.",
		".#....#..#.#.",
		".#........##.",
	},
	{ // west
		"#############",
		"....#...#....",
		"....#...#....",
		".###########.",
		".#..#...#..#.",
		".#..#...#..#.",
		".#.#....#..#.",
		".##......###.",
		".#.........#.",
		".#.........#.",
		".#.........#.",
		".###########.",
		".#.........#.",
	},
	{ // north
		"....#...#....",
		"....#...#....",
		"....#...#..#.",
		"....#...#.#..",
		"....#...##...",
		"#####...#....",
		"....#...#....",
		"....#...#....",
		"....#...#....",
		"....#...#....",
		"..###...#...#",
		".#..#...#...#",
		"....#....####",
	},
	{}, // white has no character
	{ // green
		"..##....#..#.",
		"...#....#.#..",
		"...#.#..##...",
		".#.##...#..#.",
		"#.#.....####.",
		"..#..........",
		".###########.",
		"...#.....#...",
		"...#.....#...",
		"#############",
		"...#.....#...",
		"..#......#...",
		".#.......#...",
	},
	{ // red
		"......#......",
		"......#......",
		".###########.",
		".#....#....#.",
		".#....#....#.",
		".#....#....#.",
		".#....#....#.",
		".###########.",
		"......#......",
		"......#......",
		"......#......",
		"......#......",
		"......#......",
	},
}

var honors9 = [...][9]string{
	{ // east
		"....#....",
		"#########",
		"..#####..",
		"..#.#.#..",
		"..#####..",
		"..#.#.#..",
		"..#####..",
		"...###...",
		".##.#.##.",
	},
	{ // south
		"....#....",
		"#########",
		".#######.",
		".#.#.#.#.",
		".#.###.#.",
		".#..#..#.",
		".#.###.#.",
		".#..#..#.",
		".#....##.",
	},
	{ // west
		"#########",
		"...#.#...",
		".#######.",
		".#.#.#.#.",
		".##...##.",
		".#.....#.",
		".#.....#.",
		".#######.",
		".#.....#.",
	},
	{ // north
		"...#.#...",
		"...#.#.#.",
		"...#.##..",
		"####.#...",
		"...#.#...",
		"...#.#...",
		"...#.#...",
		".###.#..#",
		"...#..###",
	},
	{}, // white
	{ // green
		".#...#.#.",
		"..#.#.#..",
		".#.#..##.",
		"#.......#",
		".#######.",
		"..#...#..",
		"#########",
		"..#...#..",
		".#....#..",
	},
	{ // red
		"....#....",
		"....#....",
		".#######.",
		".#..#..#.",
		".#..#..#.",
		".#######.",
		"....#....",
		"....#....",
		"....#....",
	},
}

var man9 = [9]string{
	"..#...#..",
	"#########",
	"..#...#..",
	".#######.",
	".#..#..#.",
	".#######.",
	"#.#####.#",
	"#..#.#..#",
	"#.#...#.#",
}

var man7 = [7]string{
	".#...#.",
	"#######",
	".#...#.",
	".#####.",
	".#.#.#.",
	"#.###.#",
	"#.#.#.#",
}

var digits5 = [9][7]string{
	{"..#..", ".##..", "..#..", "..#..", "..#..", "..#..", ".###."},
	{".###.", "#...#", "....#", "...#.", "..#..", ".#...", "#####"},
	{".###.", "#...#", "....#", "..##.", "....#", "#...#", ".###."},
	{"...#.", "..##.", ".#.#.", "#..#.", "#####", "...#.", "...#."},
	{"#####", "#....", "####.", "....#", "....#", "#...#", ".###."},
	{".###.", "#....", "#....", "####.", "#...#", "#...#", ".###."},
	{"#####", "....#", "...#.", "..#..", "..#..", "..#..", "..#.."},
	{".###.", "#...#", "#...#", ".###.", "#...#", "#...#", ".###."},
	{".###.", "#...#", "#...#", ".####", "....#", "....#", ".###."},
}

var digits3 = [9][5]string{
	{".#.", "##.", ".#.", ".#.", "###"},
	{"##.", "..#", ".#.", "#..", "###"},
	{"##.", "..#", ".#.", "..#", "##."},
	{"#.#", "#.#", "###", "..#", "..#"},
	{"###", "#..", "##.", "..#", "##."},
	{".##", "#..", "###", "#.#", "###"},
	{"###", "..#", ".#.", ".#.", ".#."},
	{"###", "#.#", "###", "#.#", "###"},
	{"###", "#.#", "###", "..#", "##."},
}

// The bird of the one of bamboo: g green, r red.
var bird = [11]string{
	"....rr...",
	"...rrrr..",
	"....rr...",
	"...ggg...",
	"..ggggg..",
	".ggrgrgg.",
	"gggrgrggg",
	".ggrgrgg.",
	"..ggggg..",
	"...g.g...",
	"..gg.gg..",
}

// glyph draws rows of pixel art.
func glyph(d hal.Display, rows []string, x, y, scale int16, c color.RGBA) {
	for j, row := range rows {
		i := 0
		for i < len(row) {
			if row[i] != '#' {
				i++
				continue
			}
			start := i
			for i < len(row) && row[i] == '#' {
				i++
			}
			d.FillRectangle(x+int16(start)*scale, y+int16(j)*scale, int16(i-start)*scale, scale, c)
		}
	}
}

// disc is a filled circle in horizontal runs.
func disc(d hal.Display, cx, cy, r int16, c color.RGBA) {
	if r <= 1 {
		d.FillRectangle(cx-r, cy-r, 2*r+1, 2*r+1, c)
		return
	}
	for dy := -r; dy <= r; dy++ {
		dx := r
		for dx*dx+dy*dy > r*r+r/2 {
			dx--
		}
		d.FillRectangle(cx-dx, cy+dy, 2*dx+1, 1, c)
	}
}

// Where the circles and sticks are, in twelfths of the face.
type spot struct {
	x, y int16
	c    uint8 // the colour of a circle: 0 blue, 1 red, 2 green
}

var pinSpots = [9][]spot{
	{{6, 6, 1}},
	{{6, 3, 2}, {6, 9, 0}},
	{{3, 2, 0}, {6, 6, 1}, {9, 10, 2}},
	{{3, 3, 0}, {9, 3, 2}, {3, 9, 2}, {9, 9, 0}},
	{{3, 2, 0}, {9, 2, 2}, {6, 6, 1}, {3, 10, 2}, {9, 10, 0}},
	{{3, 2, 2}, {9, 2, 2}, {3, 6, 1}, {9, 6, 1}, {3, 10, 1}, {9, 10, 1}},
	{{2, 1, 2}, {6, 1, 2}, {10, 1, 2}, {3, 6, 1}, {9, 6, 1}, {3, 11, 1}, {9, 11, 1}},
	{{3, 1, 0}, {9, 1, 0}, {3, 4, 0}, {9, 4, 0}, {3, 8, 0}, {9, 8, 0}, {3, 11, 0}, {9, 11, 0}},
	{{2, 2, 0}, {6, 2, 0}, {10, 2, 0}, {2, 6, 1}, {6, 6, 1}, {10, 6, 1}, {2, 10, 2}, {6, 10, 2}, {10, 10, 2}},
}

// The sticks: the colour is 1 for red; rows is how many fit one above the
// other.
var souSpots = [9]struct {
	s    []spot
	rows int16
}{
	{},
	{[]spot{{6, 3, 0}, {6, 9, 0}}, 2},
	{[]spot{{6, 3, 0}, {3, 9, 0}, {9, 9, 0}}, 2},
	{[]spot{{3, 3, 0}, {9, 3, 0}, {3, 9, 0}, {9, 9, 0}}, 2},
	{[]spot{{2, 2, 0}, {10, 2, 0}, {6, 6, 1}, {2, 10, 0}, {10, 10, 0}}, 3},
	{[]spot{{2, 3, 0}, {6, 3, 0}, {10, 3, 0}, {2, 9, 0}, {6, 9, 0}, {10, 9, 0}}, 2},
	{[]spot{{6, 2, 1}, {2, 6, 0}, {6, 6, 0}, {10, 6, 0}, {2, 10, 0}, {6, 10, 0}, {10, 10, 0}}, 3},
	{[]spot{{2, 2, 0}, {6, 2, 0}, {10, 2, 0}, {3, 6, 0}, {9, 6, 0}, {2, 10, 0}, {6, 10, 0}, {10, 10, 0}}, 3},
	{[]spot{{2, 2, 0}, {6, 2, 1}, {10, 2, 0}, {2, 6, 0}, {6, 6, 1}, {10, 6, 0}, {2, 10, 0}, {6, 10, 1}, {10, 10, 0}}, 3},
}

// drawFace draws what is on tile k inside the rectangle.
func drawFace(d hal.Display, k Tile, x, y, w, h int16) {
	switch k.suit() {
	case 0:
		drawMan(d, k.number(), x, y, w, h)
	case 1:
		drawPin(d, k.number(), x, y, w, h)
	case 2:
		drawSou(d, k.number(), x, y, w, h)
	default:
		drawHonor(d, k, x, y, w, h)
	}
}

func drawMan(d hal.Display, n int, x, y, w, h int16) {
	switch {
	case w >= 18:
		// digit, gap, character
		scaleD, scaleK := int16(2), int16(1)
		if w >= 24 {
			scaleK = 2
		}
		kh := 9 * scaleK
		y0 := y + (h-14-1-kh)/2
		glyph(d, digits5[n][:], x+(w-10)/2, y0, scaleD, inkBlack)
		glyph(d, man9[:], x+(w-9*scaleK)/2, y0+15, scaleK, inkRed)
	case w >= 12:
		y0 := y + (h-7-1-9)/2
		glyph(d, digits5[n][:], x+(w-5)/2, y0, 1, inkBlack)
		glyph(d, man9[:], x+(w-9)/2, y0+8, 1, inkRed)
	default:
		y0 := y + (h-5-7)/2
		glyph(d, digits3[n][:], x+(w-3)/2, y0, 1, inkBlack)
		glyph(d, man7[:], x+(w-7)/2, y0+5, 1, inkRed)
	}
}

func spotColor(c uint8) color.RGBA {
	switch c {
	case 1:
		return inkRed
	case 2:
		return inkGreen
	}
	return inkBlue
}

func drawPin(d hal.Display, n int, x, y, w, h int16) {
	spots := pinSpots[n]
	if n == 0 { // one big circle with a ring
		r := min(w, h) * 2 / 5
		cx, cy := x+w/2, y+h/2
		disc(d, cx, cy, r, inkGreen)
		if r >= 3 {
			disc(d, cx, cy, r-r/4-1, tileBody)
		}
		disc(d, cx, cy, r/3, inkRed)
		return
	}
	// The size: what fits between the nearest neighbours.
	pos := func(p spot) (int16, int16) { return x + ((w-1)*p.x+6)/12, y + ((h-1)*p.y+6)/12 }
	size := min(w, h)
	for i, a := range spots {
		ax, ay := pos(a)
		size = min(size, 2*min(ax-x, x+w-1-ax)+1, 2*min(ay-y, y+h-1-ay)+1)
		for _, b := range spots[i+1:] {
			bx, by := pos(b)
			size = min(size, max(abs(ax-bx), abs(ay-by))-1)
		}
	}
	size = max(size, 1)
	for _, p := range spots {
		cx, cy := pos(p)
		if size <= 3 {
			d.FillRectangle(cx-size/2, cy-size/2, size, size, spotColor(p.c))
		} else {
			disc(d, cx, cy, (size-1)/2, spotColor(p.c))
		}
	}
}

func abs(v int16) int16 {
	if v < 0 {
		return -v
	}
	return v
}

func drawSou(d hal.Display, n int, x, y, w, h int16) {
	if n == 0 {
		drawBird(d, x, y, w, h)
		return
	}
	sp := &souSpots[n]
	sw := max(w/6, 1)
	gap := max(h/12, 1)
	sh := (h - (sp.rows-1)*gap) / sp.rows
	y0 := y + (h-sp.rows*sh-(sp.rows-1)*gap)/2
	for _, s := range sp.s {
		cx := x + (w-sw)*s.x/12
		cy := y0 + s.y*sp.rows/12*(sh+gap)
		c := inkGreen
		if s.c == 1 {
			c = inkRed
		}
		d.FillRectangle(cx, cy, sw, sh, c)
		if sh >= 8 { // a joint in the middle
			d.FillRectangle(cx, cy+sh/2, sw, 1, tileBody)
		}
	}
}

func drawBird(d hal.Display, x, y, w, h int16) {
	scale := min(w/9, h/11)
	if scale < 1 {
		// A face too small for the bird: one stick.
		d.FillRectangle(x+w/2-1, y+1, 2, h-2, inkGreen)
		d.FillRectangle(x+w/2-1, y+h/2, 2, 1, inkRed)
		return
	}
	x0, y0 := x+(w-9*scale)/2, y+(h-11*scale)/2
	for j, row := range bird {
		for i := 0; i < len(row); i++ {
			switch row[i] {
			case 'g':
				d.FillRectangle(x0+int16(i)*scale, y0+int16(j)*scale, scale, scale, inkGreen)
			case 'r':
				d.FillRectangle(x0+int16(i)*scale, y0+int16(j)*scale, scale, scale, inkRed)
			}
		}
	}
}

func drawHonor(d hal.Display, k Tile, x, y, w, h int16) {
	i := int(k - east)
	if k == white {
		// A frame.
		fw, fh := w*2/3, h*2/3
		if fw%2 != w%2 {
			fw++
		}
		fx, fy := x+(w-fw)/2, y+(h-fh)/2
		t := int16(1)
		if w >= 24 {
			t = 2
		}
		d.FillRectangle(fx, fy, fw, t, inkBlue)
		d.FillRectangle(fx, fy+fh-t, fw, t, inkBlue)
		d.FillRectangle(fx, fy, t, fh, inkBlue)
		d.FillRectangle(fx+fw-t, fy, t, fh, inkBlue)
		return
	}
	c := inkBlack
	switch k {
	case green:
		c = inkGreen
	case red:
		c = inkRed
	}
	switch {
	case w >= 26:
		glyph(d, honors13[i][:], x+(w-26)/2, y+(h-26)/2, 2, c)
	case w >= 18:
		glyph(d, honors9[i][:], x+(w-18)/2, y+(h-18)/2, 2, c)
	case w >= 13:
		glyph(d, honors13[i][:], x+(w-13)/2, y+(h-13)/2, 1, c)
	default:
		glyph(d, honors9[i][:], x+(w-9)/2, y+(h-9)/2, 1, c)
	}
}

// drawTile draws a tile lying face up, with its frame and its thickness at
// the bottom.
func drawTile(d hal.Display, k Tile, x, y, w, h int16, dim bool) {
	body := tileBody
	if dim {
		body = tileDim
	}
	d.FillRectangle(x, y, w, h, tileEdge)
	d.FillRectangle(x+1, y+1, w-2, h-3, body)
	d.FillRectangle(x+1, y+h-2, w-2, 1, tileShadow)
	drawFace(d, k, x+1, y+1, w-2, h-3)
}

// drawBack draws a tile lying face down.
func drawBack(d hal.Display, x, y, w, h int16) {
	d.FillRectangle(x, y, w, h, tileEdge)
	d.FillRectangle(x+1, y+1, w-2, h-3, tileBack)
	d.FillRectangle(x+1, y+h-2, w-2, 1, tileBack2)
}

// sideways draws on a display turned a quarter to the left: what is drawn at
// (u, v) in a tile w wide goes to the screen with its top to the left.
type sideways struct {
	d       hal.Display
	x, y, w int16
}

func (s *sideways) Size() (x, y int16) { return s.d.Size() }

func (s *sideways) SetPixel(u, v int16, c color.RGBA) { s.d.SetPixel(s.x+v, s.y+s.w-1-u, c) }

func (s *sideways) Display() error { return s.d.Display() }

func (s *sideways) FillRectangle(u, v, w, h int16, c color.RGBA) error {
	return s.d.FillRectangle(s.x+v, s.y+s.w-u-w, h, w, c)
}

// drawSideways draws tile k lying on its side at (x, y): h wide and w high.
func drawSideways(d hal.Display, k Tile, x, y, w, h int16) {
	s := sideways{d, x, y, w}
	drawTile(&s, k, 0, 0, w, h, false)
}

// DrawIcon draws three tiles for the launcher, h pixels high, centered in a
// rectangle w wide.
func DrawIcon(d hal.Display, x, y, w, h int16) {
	th := h - 2
	tw := th * 3 / 4
	x0 := x + (w-3*tw-4)/2
	y0 := y + 1
	drawTile(d, man+4, x0, y0, tw, th, false)
	drawTile(d, pin+2, x0+tw+2, y0, tw, th, false)
	drawTile(d, red, x0+2*tw+4, y0, tw, th, false)
}
