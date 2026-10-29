package sokoban

import (
	"strconv"

	"github.com/sat0ken/tinygo-cyd/picopad"
)

// The selection of the author, the collection and the level
// (src/select.cpp).

const (
	selTitle      = "Select Author"
	selTitleColl  = "Collections"
	selTitleLev   = "Levels"
	selTitleTotal = "Total"
	selAuthor     = "Author:"
	selCollect    = "Select Collection"
	selColl       = "Collection:"
	selLevel      = "Select Level"

	autNameW = 21 * 8 // width of the author's name column
	autCollW = 12 * 8 // width of the column with the number of collections
	autLevW  = 7 * 8  // width of the column with the number of levels
	colNameW = 30 * 8 // width of the collection name column
	colLevW  = 10 * 8 // width of the column with the number of levels
	levW     = 4 * 8  // width of one level
	rowLev   = width / levW
)

// waitChar waits for a key.
func (g *Game) waitChar() uint8 {
	for {
		if c := g.KeyGet(); c != picopad.NoKey {
			return c
		}
	}
}

// colors returns the colors of a row, inverted when it is selected.
func colors(sel bool) (col, bgcol picopad.Color) {
	if sel {
		return picopad.ColBlack, picopad.ColWhite
	}
	return picopad.ColWhite, picopad.ColBlack
}

// authorSelect1 draws the row of an author.
func (g *Game) authorSelect1(inx int) {
	y := inx*fontH + fontH + 6
	aut := &authors[inx]
	col, bgcol := colors(inx == g.autInx)

	g.DrawRect(1, y, autNameW-1, fontH, bgcol)
	g.DrawRect(autNameW+1, y, autCollW-1, fontH, bgcol)
	g.DrawRect(autNameW+autCollW+1, y, autLevW-2, fontH, bgcol)
	g.DrawText(aut.name, 2, y, col)

	g.collTotal += len(aut.collect)
	g.DrawText(strconv.Itoa(len(aut.collect)), autNameW+2, y, col)

	j := 0
	for _, c := range aut.collect {
		j += c.num
	}
	g.levelTotal += j
	g.DrawText(strconv.Itoa(j), autNameW+autCollW+2, y, col)
}

// authorSelect chooses the author; it returns false on Esc.
func (g *Game) authorSelect() bool {
	g.setTileMode(0)

	// the title bar and the frame
	g.DrawRect(0, 0, width, fontH, picopad.ColYellow)
	g.DrawRect(0, fontH, 1, height-fontH-3, picopad.ColYellow)
	g.DrawRect(autNameW, fontH, 1, height-fontH-3, picopad.ColYellow)
	g.DrawRect(autNameW+autCollW, fontH, 1, height-fontH-3, picopad.ColYellow)
	g.DrawRect(width-1, fontH, 1, height-fontH-3, picopad.ColYellow)
	g.DrawRect(0, height-3, width, 1, picopad.ColYellow)

	g.SelFont8x16()
	g.DrawText(selTitle, (autNameW-len(selTitle)*fontW)/2, 0, picopad.ColBlack)
	g.DrawText(selTitleColl, autNameW+(autCollW-len(selTitleColl)*fontW)/2, 0, picopad.ColBlack)
	g.DrawText(selTitleLev, autNameW+autCollW+(autLevW-len(selTitleLev)*fontW)/2, 0, picopad.ColBlack)

	// the authors
	g.collTotal, g.levelTotal = 0, 0
	for i := range authors {
		g.authorSelect1(i)
	}

	// the total
	g.SelFont8x8()
	g.DrawTextBg(selTitleTotal, 6, height-7, picopad.ColYellow, picopad.ColBlack)
	s := strconv.Itoa(len(authors))
	g.DrawTextBg(s, (autNameW-len(s)*8)/2, height-7, picopad.ColYellow, picopad.ColBlack)
	s = strconv.Itoa(g.collTotal)
	g.DrawTextBg(s, autNameW+(autCollW-len(s)*8)/2, height-7, picopad.ColYellow, picopad.ColBlack)
	s = strconv.Itoa(g.levelTotal)
	g.DrawTextBg(s, autNameW+autCollW+(autLevW-len(s)*8)/2, height-7, picopad.ColYellow, picopad.ColBlack)
	g.SelFont8x16()

	for {
		g.DispUpdate()
		switch g.waitChar() {
		case picopad.KeyLeft, picopad.KeyUp:
			i := g.autInx
			g.autInx = -1
			g.authorSelect1(i) // clear the old selection
			i--
			if i < 0 {
				i = len(authors) - 1
			}
			g.autInx = i
			g.authorSelect1(i)
			g.selectAuthor(i)
			g.collInx = 0
		case picopad.KeyRight, picopad.KeyDown:
			i := g.autInx
			g.autInx = -1
			g.authorSelect1(i)
			i++
			if i > len(authors)-1 {
				i = 0
			}
			g.autInx = i
			g.authorSelect1(i)
			g.selectAuthor(i)
			g.collInx = 0
		case picopad.KeyA:
			g.selectAuthor(g.autInx)
			g.collInx = 0
			return true
		case picopad.KeyX, picopad.KeyY, picopad.KeyB:
			return false
		}
	}
}

// collFont selects the font of the rows of the collections.
func (g *Game) collFont() {
	g.SelFont8x16()
	if g.collNum > 12 {
		g.SelFont8x14()
		if g.collNum > 14 {
			g.SelFont8x8()
		}
	}
}

// collSelect1 draws the row of a collection.
func (g *Game) collSelect1(inx int) {
	fh := g.DrawFontHeight()
	y := inx*fh + 2*16 + 6
	coll := &g.collect[inx]
	col, bgcol := colors(inx == g.collInx)

	g.DrawRect(1, y, colNameW-1, fh, bgcol)
	g.DrawRect(colNameW+1, y, colLevW-2, fh, bgcol)
	g.DrawText(coll.name, 2, y, col)
	g.levelTotal += coll.num
	g.DrawText(strconv.Itoa(coll.num), colNameW+2, y, col)
}

// collSelect chooses the collection; it returns false on Esc.
func (g *Game) collSelect() bool {
	g.setTileMode(0)
	g.SelFont8x16()
	g.DrawText(selAuthor, 0, 0, picopad.ColWhite)
	g.DrawText(g.autName, (len(selAuthor)+1)*8, 0, picopad.ColGreen)

	// the title bar and the frame
	g.DrawRect(0, fontH, width, fontH, picopad.ColYellow)
	g.DrawRect(0, 2*fontH, 1, height-2*fontH-3, picopad.ColYellow)
	g.DrawRect(colNameW, 2*fontH, 1, height-2*fontH-3, picopad.ColYellow)
	g.DrawRect(width-1, 2*fontH, 1, height-2*fontH-3, picopad.ColYellow)
	g.DrawRect(0, height-3, width, 1, picopad.ColYellow)
	g.DrawText(selCollect, (colNameW-len(selCollect)*fontW)/2, fontH, picopad.ColBlack)
	g.DrawText(selTitleLev, colNameW+(colLevW-len(selTitleLev)*fontW)/2, fontH, picopad.ColBlack)

	// the collections (a smaller font for many)
	g.collFont()
	g.levelTotal = 0
	for i := 0; i < g.collNum; i++ {
		g.collSelect1(i)
	}

	// the total
	g.SelFont8x8()
	g.DrawTextBg(selTitleTotal, 6, height-7, picopad.ColYellow, picopad.ColBlack)
	s := strconv.Itoa(g.collNum)
	g.DrawTextBg(s, (colNameW-len(s)*8)/2, height-7, picopad.ColYellow, picopad.ColBlack)
	s = strconv.Itoa(g.levelTotal)
	g.DrawTextBg(s, colNameW+(colLevW-len(s)*8)/2, height-7, picopad.ColYellow, picopad.ColBlack)
	g.collFont()

	for {
		g.DispUpdate()
		switch g.waitChar() {
		case picopad.KeyLeft, picopad.KeyUp:
			i := g.collInx
			g.collInx = -1
			g.collSelect1(i)
			i--
			if i < 0 {
				i = g.collNum - 1
			}
			g.collInx = i
			g.collSelect1(i)
			g.level = 0
			g.selectColl(i)
		case picopad.KeyRight, picopad.KeyDown:
			i := g.collInx
			g.collInx = -1
			g.collSelect1(i)
			i++
			if i > g.collNum-1 {
				i = 0
			}
			g.collInx = i
			g.collSelect1(i)
			g.level = 0
			g.selectColl(i)
		case picopad.KeyA:
			g.level = 0
			g.selectColl(g.collInx)
			return true
		case picopad.KeyX, picopad.KeyY, picopad.KeyB:
			return false
		}
	}
}

// levSelect1 draws the number of a level.
func (g *Game) levSelect1(inx int) {
	fh := g.DrawFontHeight()
	y := inx / rowLev
	x := (inx - y*rowLev) * levW
	y = y*fh + 3*16 + 6
	col, bgcol := colors(inx == g.level)
	g.DrawRect(x, y, levW, fh, bgcol)
	s := strconv.Itoa(inx + 1)
	g.DrawText(s, x+(levW-len(s)*8)/2, y, col)
}

// levSelect chooses the level; it returns false on Esc.
func (g *Game) levSelect() bool {
	g.setTileMode(0)
	g.SelFont8x16()
	g.DrawText(selAuthor, 0, 0, picopad.ColWhite)
	g.DrawText(g.autName, (len(selAuthor)+1)*8, 0, picopad.ColGreen)
	g.DrawText(selColl, 0, fontH, picopad.ColWhite)
	g.DrawText(g.collName, (len(selColl)+1)*8, fontH, picopad.ColGreen)
	g.DrawRect(0, 2*fontH, width, fontH, picopad.ColYellow)
	g.DrawText(selLevel, (width-len(selLevel)*fontW)/2, 2*fontH, picopad.ColBlack)

	// a smaller font for many levels
	if g.levNum > 11*rowLev {
		g.SelFont8x14()
		if g.levNum > 13*rowLev {
			g.SelFont8x8()
		}
	}
	for i := 0; i < g.levNum; i++ {
		g.levSelect1(i)
	}

	for {
		g.DispUpdate()
		i := g.level
		switch g.waitChar() {
		case picopad.KeyUp:
			g.level = -1
			g.levSelect1(i)
			i -= rowLev
			if i < 0 { // to the last row
				for i < g.levNum {
					i += rowLev
				}
				i -= rowLev
				if i < 0 {
					i += rowLev
				}
			}
			g.level = i
			g.levSelect1(i)
		case picopad.KeyLeft:
			g.level = -1
			g.levSelect1(i)
			i--
			if i < 0 {
				i = g.levNum - 1
			}
			g.level = i
			g.levSelect1(i)
		case picopad.KeyDown:
			g.level = -1
			g.levSelect1(i)
			i += rowLev
			if i >= g.levNum { // to the first row
				for i >= 0 {
					i -= rowLev
				}
				i += rowLev
				if i >= g.levNum {
					i -= rowLev
				}
			}
			g.level = i
			g.levSelect1(i)
		case picopad.KeyRight:
			g.level = -1
			g.levSelect1(i)
			i++
			if i > g.levNum-1 {
				i = 0
			}
			g.level = i
			g.levSelect1(i)
		case picopad.KeyA:
			return true
		case picopad.KeyX, picopad.KeyY, picopad.KeyB:
			return false
		}
	}
}
