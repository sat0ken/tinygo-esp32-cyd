// Package cards has playing cards for the card games: the card type, a
// shuffled deck and drawing of faces, backs and empty slots. It depends
// only on hal.
package cards

import (
	"image/color"

	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/romfont/freesans"
	"github.com/sat0ken/tinygo-cyd/romfont/proggy"
	"tinygo.org/x/tinyfont"
)

// Suit of a card. Spades and clubs are black, hearts and diamonds red.
type Suit uint8

const (
	Spades Suit = iota
	Hearts
	Diamonds
	Clubs
)

// Red reports whether the suit is red.
func (s Suit) Red() bool { return s == Hearts || s == Diamonds }

// Card is one of the 52 cards: Suit*13 + Rank-1.
type Card uint8

// New returns the card of a suit and rank (1 = ace ... 13 = king).
func New(s Suit, rank int) Card { return Card(int(s)*13 + rank - 1) }

func (c Card) Suit() Suit { return Suit(c / 13) }
func (c Card) Rank() int  { return int(c%13) + 1 }
func (c Card) Red() bool  { return c.Suit().Red() }

var rankNames = [14]string{"", "A", "2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K"}

// String returns e.g. "10H" (for logs and tests).
func (c Card) String() string { return rankNames[c.Rank()] + "SHDC"[c.Suit():c.Suit()+1] }

// Rand is a small xorshift32 generator, so deals are repeatable from a seed.
type Rand struct{ s uint32 }

func NewRand(seed uint32) *Rand {
	if seed == 0 {
		seed = 0x9E3779B9
	}
	return &Rand{s: seed}
}

func (r *Rand) Next() uint32 {
	r.s ^= r.s << 13
	r.s ^= r.s >> 17
	r.s ^= r.s << 5
	return r.s
}

// Intn returns 0..n-1.
func (r *Rand) Intn(n int) int { return int(r.Next() % uint32(n)) }

// Shuffled returns the 52 cards in random order (Fisher-Yates).
func Shuffled(r *Rand) [52]Card {
	var d [52]Card
	for i := range d {
		d[i] = Card(i)
	}
	for i := 51; i > 0; i-- {
		j := r.Intn(i + 1)
		d[i], d[j] = d[j], d[i]
	}
	return d
}

// ---- drawing ----

var (
	faceColor   = color.RGBA{250, 250, 245, 255}
	edgeColor   = color.RGBA{120, 120, 120, 255}
	redInk      = color.RGBA{210, 30, 40, 255}
	blackInk    = color.RGBA{20, 20, 20, 255}
	backColor   = color.RGBA{40, 80, 170, 255}
	backPattern = color.RGBA{70, 120, 220, 255}
	slotColor   = color.RGBA{0, 80, 45, 255}
	slotEdge    = color.RGBA{60, 150, 100, 255}
)

// Suit symbols, 7x7.
var suitArt = [4][7]string{
	{ // spades
		"...#...",
		"..###..",
		".#####.",
		"#######",
		"##.#.##",
		"...#...",
		"..###..",
	},
	{ // hearts
		".##.##.",
		"#######",
		"#######",
		"#######",
		".#####.",
		"..###..",
		"...#...",
	},
	{ // diamonds
		"...#...",
		"..###..",
		".#####.",
		"#######",
		".#####.",
		"..###..",
		"...#...",
	},
	{ // clubs
		"..###..",
		"..###..",
		"#.###.#",
		"#######",
		"#.#.#.#",
		"...#...",
		"..###..",
	},
}

// DrawSuit draws the suit symbol with its top-left at (x, y), each dot
// scale x scale pixels.
func DrawSuit(d hal.Display, s Suit, x, y, scale int16, c color.RGBA) {
	for row, line := range suitArt[s] {
		for col, ch := range line {
			if ch == '#' {
				d.FillRectangle(x+int16(col)*scale, y+int16(row)*scale, scale, scale, c)
			}
		}
	}
}

// frame draws a card-sized rectangle with cut corners.
func frame(d hal.Display, x, y, w, h int16, fill, edge color.RGBA) {
	d.FillRectangle(x+1, y, w-2, 1, edge)
	d.FillRectangle(x+1, y+h-1, w-2, 1, edge)
	d.FillRectangle(x, y+1, 1, h-2, edge)
	d.FillRectangle(x+w-1, y+1, 1, h-2, edge)
	d.FillRectangle(x+1, y+1, w-2, h-2, fill)
}

// DrawFace draws card c face up in the w x h rectangle at (x, y). The rank
// and suit are in the top-left corner (visible when cards overlap) and a
// large suit is in the middle.
func DrawFace(d hal.Display, c Card, x, y, w, h int16, highlight bool) {
	drawFace(d, c, x, y, w, h, highlight, false)
}

// SmallCornerBelow is the overlap (in pixels) under which DrawFaceSmall
// should be used: the normal corner is about 14 pixels tall.
const SmallCornerBelow = 14

// DrawFaceSmall is DrawFace with a small corner (about 7 pixels tall), for
// cards that overlap so much that the normal corner would be cut off.
func DrawFaceSmall(d hal.Display, c Card, x, y, w, h int16) {
	drawFace(d, c, x, y, w, h, false, true)
}

func drawFace(d hal.Display, c Card, x, y, w, h int16, highlight, small bool) {
	edge := edgeColor
	if highlight {
		edge = color.RGBA{255, 200, 0, 255}
	}
	frame(d, x, y, w, h, faceColor, edge)
	if highlight {
		frame(d, x+1, y+1, w-2, h-2, faceColor, edge)
	}
	ink := blackInk
	if c.Red() {
		ink = redInk
	}
	name := rankNames[c.Rank()]
	_, tw := tinyfont.LineWidth(&freesans.Bold9pt7b, name)
	if small {
		tinyfont.WriteLine(d, &proggy.TinySZ8pt7b, x+4, y+7, name, ink)
		_, sw := tinyfont.LineWidth(&proggy.TinySZ8pt7b, name)
		DrawSuit(d, c.Suit(), x+6+int16(sw), y+1, 1, ink)
	} else {
		tinyfont.WriteLine(d, &freesans.Bold9pt7b, x+4, y+15, name, ink)
		DrawSuit(d, c.Suit(), x+6+int16(tw), y+5, 1, ink)
	}
	// Big suit in the middle, a bit low so the corner stays readable; even
	// bigger on large cards.
	scale := int16(3)
	if w >= 80 {
		scale = 5
	}
	big := 7 * scale
	DrawSuit(d, c.Suit(), x+(w-big)/2, y+(h-big)/2+8, scale, ink)
	if c.Rank() > 10 && h >= 70 {
		// Face cards: the letter again above the big suit (on small cards
		// it would run into the corner).
		tinyfont.WriteLine(d, &freesans.Bold9pt7b, x+(w-int16(tw))/2, y+(h-big)/2+4, name, ink)
	}
}

// DrawBack draws a face-down card.
func DrawBack(d hal.Display, x, y, w, h int16) {
	frame(d, x, y, w, h, backColor, edgeColor)
	for yy := y + 4; yy < y+h-4; yy += 4 {
		d.FillRectangle(x+4, yy, w-8, 1, backPattern)
	}
}

// DrawSlot draws an empty place for a card, with an optional label.
func DrawSlot(d hal.Display, x, y, w, h int16, label string) {
	frame(d, x, y, w, h, slotColor, slotEdge)
	if label != "" {
		_, tw := tinyfont.LineWidth(&freesans.Bold9pt7b, label)
		tinyfont.WriteLine(d, &freesans.Bold9pt7b, x+(w-int16(tw))/2, y+h/2+6, label, slotEdge)
	}
}
