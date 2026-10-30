// Package chess is the Chess of PicoLibSDK (PicoPad/GAME/CHESS, by
// Miroslav Nemecek, Unlicense), ported from C to package picopad. It
// follows the C source function by function; the C globals are the fields
// of Game.
//
// The first screen chooses: LEFT plays white against the computer, RIGHT
// black, DOWN is two players, UP a demo, A the level (2 to 5 moves ahead).
// The arrows move the cursor, A picks up a piece and puts it down, B puts
// it back, X lets the computer move for you, Y ends the game; Y on the
// first screen ends Run (on the PicoPad: back to the boot loader). As the
// C source says, castling moves are shown on the wrong places, and a pawn
// always becomes a queen.
package chess

import (
	"strconv"

	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/picopad"
)

//go:generate go run ../../tools/picoimg -pkg chess -root $PICOLIBSDK -o pieces.go PicoPad/GAME/CHESS/img/pieces.cpp

const (
	// the text console on the right
	fontW = 8  // font width
	fontH = 16 // font height
	textW = 10 // text width (10 characters, 80 pixels)
	textH = 15 // text height (15 rows, 240 pixels)
	textS = textW * textH

	tileW = 28 // tile width
	tileH = 28 // tile height

	map0W   = 8           // board width in fields, without the border
	map0H   = 8           // board height in fields, without the border
	mapW    = map0W + 2   // board width with a border of 1+1 columns (10)
	mapH    = map0H + 4   // board height with a border of 2+2 rows (12)
	mapSize = mapW * mapH // 120

	frame  = 8                     // frame width
	boardW = 2*frame + map0W*tileW // board total width (240)
	width  = picopad.Width         // WIDTH
	height = picopad.Height        // HEIGHT

	piecesImgW = 196 // width of piecesImg

	// pieces
	empty     = 0 // empty field (or black pawn)
	pawn      = 1 // pawn (or white pawn)
	knight    = 2
	bishop    = 3
	rook      = 4
	queen     = 5
	king      = 6
	border    = 7 // invalid field
	pieceMask = 7

	moveMask = 1 << 4   // flag of a piece that has not moved
	moving   = 0        // the piece has moved
	noMoving = moveMask // the piece has not moved yet

	colorMask = 1 << 5 // mask of the piece color
	white     = 0
	black     = colorMask

	whitePlayer = 0
	blackPlayer = 1

	valWin = 10000 // winning value

	moveFlagNormal = 0 // normal move
	moveFlagQueen  = 1 // pawn becomes the piece in extra
	moveFlagSmall  = 2 // small castle
	moveFlagBig    = 3 // big castle
	moveFlagEP     = 4 // en passant

	stackMax = 500 // max. size of the move stack
	deepMax  = 5   // max. depth of the move stack
)

// player is a player (sPlayer).
type player struct {
	comp   bool  // the computer plays
	curPos uint8 // cursor position
	king   uint8 // king position
}

// move is an entry of the move stack (sMove).
type move struct {
	src, dst           uint8
	srcPiece, dstPiece uint8 // with the NOMOVING flag and the color
	flags              uint8 // moveFlag*
	extra              uint8 // new piece for a pawn, the en passant position
	val                int16 // value of the move
}

// boardTemp is the board of a new game with its border, from A1 to H8.
var boardTemp = [mapSize]uint8{
	border, border, border, border, border, border, border, border, border, border,
	border, border, border, border, border, border, border, border, border, border,
	border, rook + white, knight + white, bishop + white, queen + white, king + white, bishop + white, knight + white, rook + white, border,
	border, pawn + white, pawn + white, pawn + white, pawn + white, pawn + white, pawn + white, pawn + white, pawn + white, border,
	border, empty, empty, empty, empty, empty, empty, empty, empty, border,
	border, empty, empty, empty, empty, empty, empty, empty, empty, border,
	border, empty, empty, empty, empty, empty, empty, empty, empty, border,
	border, empty, empty, empty, empty, empty, empty, empty, empty, border,
	border, pawn + black, pawn + black, pawn + black, pawn + black, pawn + black, pawn + black, pawn + black, pawn + black, border,
	border, rook + black, knight + black, bishop + black, queen + black, king + black, bishop + black, knight + black, rook + black, border,
	border, border, border, border, border, border, border, border, border, border,
	border, border, border, border, border, border, border, border, border, border,
}

// movesTemp are the steps of the pieces (a row is 10 fields).
var movesTemp = [24]int{
	9, 11, -11, -9, // 0: diagonally
	10, -10, 1, -1, // 4: perpendicularly
	-21, -12, 8, 19, // 8: knight to the left
	21, 12, -8, -19, // 12: knight to the right
	10, 20, 11, 9, // 16: white pawn
	-10, -20, -11, -9, // 20: black pawn
}

// pieceMoves are the steps of each piece: offset in movesTemp, number.
var pieceMoves = [14]uint8{
	20, 4, // 0: black pawn
	16, 4, // 1: white pawn
	8, 8, // 2: knight
	0, 4, // 3: bishop
	4, 4, // 4: rook
	0, 8, // 5: queen
	0, 8, // 6: king
}

// pieceRate is the value of each piece.
var pieceRate = [7]int16{0, 1, 3, 3, 5, 9, 15}

// openMovesTab are the white player's opening moves.
var openMovesTab = [4]move{
	{35, 55, pawn + white + noMoving, empty, moveFlagNormal, 0, 0},
	{34, 54, pawn + white + noMoving, empty, moveFlagNormal, 0, 0},
	{22, 43, knight + white + noMoving, empty, moveFlagNormal, 0, 0},
	{27, 46, knight + white + noMoving, empty, moveFlagNormal, 0, 0},
}

// Game is Chess on a console.
type Game struct {
	*picopad.Console

	textBuf  [textS]byte // the text console
	textRows int

	playerView uint8 // the board as seen by whitePlayer or blackPlayer

	board       [mapSize]uint8 // the board with its border
	boardEnable [mapSize]bool  // fields to move from or to

	moveStack    [stackMax]move
	moveStackTop int  // number of entries
	lastMove     move // the last move
	lastMoveOK   bool // false: checkmate (LastMove == NULL)

	moveStackStart   [deepMax]int // start of each level
	moveStackNum     [deepMax]int // moves of each level
	movePassantPiece [deepMax + 1]uint8
	movePassantPos   [deepMax + 1]uint8
	moveDeep         int
	deepMaxN         int // DeepMax

	player    uint8 // current player
	players   [2]player
	moveNum   int    // move counter (1..)
	blinkTime uint32 // cursor blinking time
}

// New returns the game on d (at least 320x240) with the touch panel t;
// seed starts the random numbers. Set Pad to use a game controller.
func New(d hal.Display, t hal.Touch, seed uint32) *Game {
	return &Game{Console: picopad.New(d, t, seed)}
}

// outText prints a row to the text console, scrolling it.
func (g *Game) outText(txt string) {
	if g.textRows >= textH {
		copy(g.textBuf[:], g.textBuf[textW:])
		for i := textS - textW; i < textS; i++ {
			g.textBuf[i] = ' '
		}
		g.textRows--
	}
	copy(g.textBuf[g.textRows*textW:g.textRows*textW+min(len(txt), textW)], txt)

	g.SelFont8x16()
	for y := 0; y < textH; y++ {
		for i := 0; i < textW; i++ {
			g.DrawCharBg(g.textBuf[y*textW+i], width-textW*fontW+i*fontW, y*fontH, picopad.ColWhite, picopad.ColBlack)
		}
	}
	g.DispUpdate()
	g.textRows++
}

// dispField displays a field (turned with the board): w is the width of
// the cursor frame (0: none), usemove and usecatch the colors of a move.
func (g *Game) dispField(inx uint8, w int, color picopad.Color, usemove, usecatch bool) {
	piece := g.board[inx]
	if piece == border {
		return
	}

	row := int(inx) / mapW
	col := int(inx) - row*mapW - 1
	row -= 2

	xs := int(piece&pieceMask) * tileW // piece type
	ys := 0                            // piece color
	if piece&colorMask == white {
		ys = tileH
	}
	switch {
	case usemove:
		ys += 4 * tileH
	case usecatch:
		ys += 6 * tileH
	case (row+col)&1 != 0:
		ys += 2 * tileH // field color
	}

	var x, y int
	if g.playerView != 0 {
		x = frame + (map0W-1-col)*tileW
		y = frame + row*tileH
	} else {
		x = frame + col*tileW
		y = frame + (map0H-1-row)*tileH
	}

	g.DrawImg4Pal(piecesImg, piecesImgPal, xs, ys, x, y, tileW, tileH, piecesImgW)

	if w > 0 {
		g.DrawRect(x, y, tileW, w, color)               // top
		g.DrawRect(x, y+tileH-w, tileW, w, color)       // bottom
		g.DrawRect(x, y+w, w, tileH-2*w, color)         // left
		g.DrawRect(x+tileW-w, y+w, w, tileH-2*w, color) // right
	} else if g.boardEnable[inx] {
		// the mark of a field to move from or to
		color = picopad.RGB(0, 255, 0)
		if (row+col)&1 != 0 {
			color = picopad.RGB(0, 180, 0)
		}
		const ddw = 2
		w = 2
		g.DrawRect(x+ddw, y+ddw, tileW-2*ddw, w, color)
		g.DrawRect(x+ddw, y+tileH-w-ddw, tileW-2*ddw, w, color)
		g.DrawRect(x+ddw, y+w+ddw, w, tileH-2*w-2*ddw, color)
		g.DrawRect(x+tileW-w-ddw, y+w+ddw, w, tileH-2*w-2*ddw, color)
	}
}

// legend are the letters and digits of the board frame.
const legend = "ABCDEFGH12345678"

// dispFrame displays the board frame.
func (g *Game) dispFrame() {
	frameCol := picopad.RGB(200, 0, 0)
	g.SelFont8x8()

	g.DrawRect(0, 0, boardW, frame, frameCol)
	g.DrawRect(0, height-frame, boardW, frame, frameCol)
	g.DrawRect(0, frame, frame, height-2*frame, frameCol)
	g.DrawRect(boardW-frame, frame, frame, height-2*frame, frameCol)

	for i := 0; i < map0W; i++ {
		ch := legend[i] // 'A' + i
		if g.playerView != 0 {
			ch = legend[7-i] // 'H' - i
		}
		g.DrawChar(ch, frame+tileW/2-4+i*tileW, 0, picopad.ColWhite)
		g.DrawChar(ch, frame+tileW/2-4+i*tileW, height-frame, picopad.ColWhite)
	}
	for i := 0; i < map0H; i++ {
		ch := legend[15-i] // '8' - i
		if g.playerView != 0 {
			ch = legend[8+i] // '1' + i
		}
		g.DrawChar(ch, 0, frame+tileH/2-4+i*tileH, picopad.ColWhite)
		g.DrawChar(ch, boardW-frame, frame+tileH/2-4+i*tileH, picopad.ColWhite)
	}
}

// dispBoard displays the board.
func (g *Game) dispBoard() {
	for i := 0; i < mapSize; i++ {
		g.dispField(uint8(i), 0, 0, false, false)
	}
}

// newGame starts a new game.
func (g *Game) newGame() {
	g.DrawClear()

	g.playerView = 0
	g.moveNum = 1
	g.player = whitePlayer
	g.moveStackTop = 0
	g.movePassantPiece[0] = 0
	g.movePassantPos[0] = 0

	for i := range g.textBuf {
		g.textBuf[i] = ' '
	}
	g.textRows = 0

	for i := range g.board {
		g.boardEnable[i] = false
		b := boardTemp[i]
		if b == king+white {
			g.players[whitePlayer].king = uint8(i)
		}
		if b == king+black {
			g.players[blackPlayer].king = uint8(i)
		}
		if b != border && b != empty {
			b |= noMoving
		}
		g.board[i] = b
	}

	for i := range g.players {
		g.players[i].curPos = g.players[i].king
	}
}

// doMove does a move (without showing it).
func (g *Game) doMove(m *move) {
	if m.srcPiece&pieceMask == king {
		g.players[g.player].king = m.dst
	}
	switch m.flags {
	case moveFlagNormal:
		g.board[m.src] = empty
		g.board[m.dst] = m.srcPiece &^ moveMask
	case moveFlagQueen:
		g.board[m.src] = empty
		g.board[m.dst] = m.extra
	case moveFlagEP:
		g.board[m.src] = empty
		g.board[m.dst] = m.srcPiece &^ moveMask
		g.board[m.extra] = empty // the enemy pawn
	case moveFlagSmall:
		g.board[m.src] = empty
		g.board[m.src+1] = m.dstPiece &^ moveMask // the rook
		g.board[m.dst] = m.srcPiece &^ moveMask   // the king
		g.board[m.src+3] = empty
	case moveFlagBig:
		g.board[m.src] = empty
		g.board[m.src-1] = m.dstPiece &^ moveMask
		g.board[m.dst] = m.srcPiece &^ moveMask
		g.board[m.src-4] = empty
	}
}

// undoMove undoes a move (without showing it).
func (g *Game) undoMove(m *move) {
	if m.srcPiece&pieceMask == king {
		g.players[g.player].king = m.src
	}
	switch m.flags {
	case moveFlagEP:
		g.board[m.src] = m.srcPiece
		g.board[m.dst] = empty
		g.board[m.extra] = m.dstPiece
	case moveFlagSmall:
		g.board[m.src] = m.srcPiece | noMoving
		g.board[m.src+1] = empty
		g.board[m.dst] = empty
		g.board[m.src+3] = m.dstPiece | noMoving
	case moveFlagBig:
		g.board[m.src] = m.srcPiece | noMoving
		g.board[m.src-1] = empty
		g.board[m.dst] = empty
		g.board[m.src-4] = m.dstPiece | noMoving
	default:
		g.board[m.src] = m.srcPiece
		g.board[m.dst] = m.dstPiece
	}
}

// doMoveDisp does a move and shows it.
func (g *Game) doMoveDisp(m *move) {
	g.lastMove, g.lastMoveOK = *m, true

	// note a pawn's en passant
	g.movePassantPiece[0] = 0
	g.movePassantPos[0] = 0
	if m.srcPiece&(pieceMask|moveMask) == pawn|noMoving {
		dif := int(m.dst) - int(m.src)
		if dif == 20 || dif == -20 {
			g.movePassantPiece[0] = m.dst
			g.movePassantPos[0] = uint8(int(m.src) + dif/2)
		}
	}

	// the color of the destination
	usemove, usecatch := true, false
	if m.dstPiece != empty && m.flags != moveFlagSmall && m.flags != moveFlagBig {
		usemove, usecatch = false, true
	}

	g.dispField(m.dst, 0, 0, usemove, usecatch)

	// blink the source
	for i := 0; i < 4; i++ {
		g.board[m.src] = m.srcPiece
		g.dispField(m.src, 0, 0, true, false)
		g.DispUpdate()
		g.WaitMs(100)

		g.board[m.src] = empty
		g.dispField(m.src, 0, 0, true, false)
		g.DispUpdate()
		g.WaitMs(100)
	}
	g.board[m.src] = m.srcPiece

	g.doMove(m)
	piece := g.board[m.dst]

	g.dispField(m.src, 0, 0, true, false)

	// blink the destination
	for i := 0; i < 4; i++ {
		g.board[m.dst] = piece
		g.dispField(m.dst, 0, 0, usemove, usecatch)
		g.DispUpdate()
		g.WaitMs(100)

		g.board[m.dst] = empty
		g.dispField(m.dst, 0, 0, usemove, usecatch)
		g.DispUpdate()
		g.WaitMs(100)
	}
	g.board[m.dst] = piece

	g.dispBoard()
	g.DispUpdate()
}

// testCheck reports whether the king of the current player is in check.
func (g *Game) testCheck() bool {
	pos0 := int(g.players[g.player].king)
	kingPiece := g.board[pos0]

	// every direction, with the knight's jumps
	for dir := 0; dir < 16; dir++ {
		jump := movesTemp[dir]
		pos := pos0
		for steps := 1; ; steps++ {
			pos += jump
			piece := g.board[pos]

			if piece == border {
				break
			}

			// empty: go on, but not for a knight's jump
			if piece == empty {
				if dir >= 8 {
					break
				}
				continue
			}

			// a piece of the same color covers the king
			if (piece^kingPiece)&colorMask == 0 {
				break
			}
			piece &= pieceMask

			if piece == knight {
				if dir >= 8 {
					return true
				}
				break
			}

			// only knights jump
			if dir >= 8 {
				break
			}

			if piece == king {
				if steps == 1 {
					return true
				}
				break
			}
			if piece == queen {
				return true
			}
			if piece == bishop && dir <= 3 {
				return true
			}
			if piece == rook && dir >= 4 {
				return true
			}
			if piece == pawn && dir <= 3 && steps == 1 {
				if kingPiece&colorMask == white {
					// a black pawn attacking the white king
					if jump == 11 || jump == 9 {
						return true
					}
				} else if jump == -11 || jump == -9 {
					// a white pawn attacking the black king
					return true
				}
			}

			// the others are safe
			break
		}
	}
	return false
}

// addMove adds a move to the stack (if there is room).
func (g *Game) addMove(n *int, m move) {
	if *n < stackMax {
		g.moveStack[*n] = m
		*n++
	}
}

// tryCheck reports whether moving piece from pos0 to pos (over piece3)
// puts the king of the current player in check.
func (g *Game) tryCheck(pos0, pos int, piece, piece3 uint8) bool {
	g.board[pos0] = empty
	g.board[pos] = piece
	check := g.testCheck()
	g.board[pos0] = piece
	g.board[pos] = piece3
	return check
}

// searchMoves puts all the possible moves of the current player on the
// stack at the current level, with val added to their values.
func (g *Game) searchMoves(val int16) {
	level := g.moveDeep
	n := g.moveStackTop
	g.moveStackStart[level] = n
	col := uint8(white)
	if g.player != whitePlayer {
		col = black
	}
	var extra uint8

	for pos0 := 21; pos0 <= 98; pos0++ {
		piece := g.board[pos0]
		if piece == border || piece == empty || piece&colorMask != col {
			continue // not the player's piece
		}

		// can taking the piece away put the king in check?
		piece2 := piece & pieceMask
		possibleCheck := true
		if piece2 != king {
			g.board[pos0] = empty
			possibleCheck = g.testCheck()
			g.board[pos0] = piece
		}

		// the black pawn has index 0
		if piece2 == pawn && col == black {
			piece2 = 0
		}
		dir := int(pieceMoves[piece2*2])
		for dirNum := int(pieceMoves[piece2*2+1]); dirNum > 0; dirNum-- {
			jump := movesTemp[dir]
			dir++

			pos := pos0
			for {
				pos += jump
				piece3 := g.board[pos]
				if piece3 == border {
					break
				}
				if piece3 != empty && piece3&colorMask == col {
					break // own piece
				}

				flags := uint8(moveFlagNormal)
				if piece2 <= 1 { // a pawn
					// 2 fields: the first move, to an empty field
					if jump == 20 || jump == -20 {
						if piece&moveMask == moving || piece3 != empty || g.board[pos-jump/2] != empty {
							break
						}
					}
					// 1 step straight: to an empty field
					if jump == 10 || jump == -10 {
						if piece3 != empty {
							break
						}
					}
					// 1 step diagonally: must take an enemy
					if jump == -11 || jump == -9 || jump == 11 || jump == 9 {
						if piece3 == empty {
							// en passant
							if uint8(pos) != g.movePassantPos[level] {
								break
							}
							extra = g.movePassantPiece[level]
							piece3 = g.board[extra]
							flags = moveFlagEP
						}
					}

					// the last row: the pawn becomes another piece
					if pos >= 21 && pos <= 28 || pos >= 91 && pos <= 98 {
						check := false
						if possibleCheck {
							check = g.tryCheck(pos0, pos, piece, piece3)
						}
						if !check {
							for piece4 := uint8(knight); piece4 <= queen; piece4++ {
								val2 := val
								if piece3 != empty {
									val2 += pieceRate[piece3&pieceMask]
								}
								val2 += pieceRate[piece4] - 1
								g.addMove(&n, move{uint8(pos0), uint8(pos), piece, piece3, moveFlagQueen, piece4 | col, val2})
							}
						}
						break
					}
				}

				// would the move put the king in check?
				check := false
				if piece2 == king {
					g.players[g.player].king = uint8(pos)
					check = g.tryCheck(pos0, pos, piece, piece3)
					g.players[g.player].king = uint8(pos0)
				} else if possibleCheck {
					check = g.tryCheck(pos0, pos, piece, piece3)
				}

				if !check {
					val2 := val
					if piece3 != empty {
						val2 += pieceRate[piece3&pieceMask]
					}
					g.addMove(&n, move{uint8(pos0), uint8(pos), piece, piece3, flags, extra, val2})
				}

				// an enemy stops the steps
				if piece3 != empty {
					break
				}

				// the king, the pawns and the knights go 1 step
				if piece2 == knight || piece2 <= 1 || piece2 == king {
					break
				}
			}
		}
	}

	// castling
	pos0 := int(g.players[g.player].king)
	piece := g.board[pos0]
	if piece&moveMask == noMoving && !g.testCheck() {
		// small castle
		piece3 := g.board[pos0+3]
		if piece3&(pieceMask|moveMask) == rook|noMoving && g.board[pos0+1] == empty && g.board[pos0+2] == empty {
			g.players[g.player].king = uint8(pos0 + 1)
			check := g.testCheck()
			g.players[g.player].king = uint8(pos0 + 2)
			check = g.testCheck() || check
			g.players[g.player].king = uint8(pos0)
			if !check {
				g.addMove(&n, move{uint8(pos0), uint8(pos0 + 2), piece, piece3, moveFlagSmall, extra, val})
			}
		}

		// big castle
		piece3 = g.board[pos0-4]
		if piece3&(pieceMask|moveMask) == rook|noMoving && g.board[pos0-1] == empty && g.board[pos0-2] == empty && g.board[pos0-3] == empty {
			g.players[g.player].king = uint8(pos0 - 1)
			check := g.testCheck()
			g.players[g.player].king = uint8(pos0 - 2)
			check = g.testCheck() || check
			g.players[g.player].king = uint8(pos0)
			if !check {
				g.addMove(&n, move{uint8(pos0), uint8(pos0 - 2), piece, piece3, moveFlagBig, extra, val})
			}
		}
	}

	g.moveStackNum[level] = n - g.moveStackTop
	g.moveStackTop = n
}

// moveCompLevel finds the computer's best move at the current level: its
// index in the stack, or -1 if there is none.
func (g *Game) moveCompLevel() int {
	deep := g.moveDeep

	// SearchMoves(deep) in the C source: the level is the start value
	g.searchMoves(int16(deep))

	if g.moveStackNum[deep] == 0 {
		return -1
	}

	// the first moves look less far ahead
	dm := min(g.moveNum, g.deepMaxN-1)

	if deep < dm {
		g.moveDeep = deep + 1
		start := g.moveStackStart[deep]
		for k := 0; k < g.moveStackNum[deep]; k++ {
			m := &g.moveStack[start+k]

			// note a pawn's en passant
			g.movePassantPiece[deep+1] = 0
			g.movePassantPos[deep+1] = 0
			if m.srcPiece&(pieceMask|moveMask) == pawn|noMoving {
				dif := int(m.dst) - int(m.src)
				if dif == 20 || dif == -20 {
					g.movePassantPiece[deep+1] = m.dst
					g.movePassantPos[deep+1] = uint8(int(m.src) + dif/2)
				}
			}

			g.doMove(m)
			g.player ^= 1
			top := g.moveStackTop

			m2 := g.moveCompLevel()
			if m2 < 0 {
				m.val += valWin // checkmate or stalemate
			} else {
				m.val -= g.moveStack[m2].val
			}

			g.moveStackTop = top
			g.player ^= 1
			g.undoMove(m)
		}
		g.moveDeep = deep
	}

	// the best value, and how many moves have it
	start := g.moveStackStart[deep]
	moves := g.moveStack[start : start+g.moveStackNum[deep]]
	bestNum := 1
	bestVal := moves[0].val
	for _, m := range moves[1:] {
		if m.val == bestVal {
			bestNum++
		} else if m.val > bestVal {
			bestNum = 1
			bestVal = m.val
		}
	}

	// one of them
	i := int(g.RandS16Max(int16(bestNum - 1)))
	for k := range moves {
		if moves[k].val == bestVal {
			i--
			if i < 0 {
				return start + k
			}
		}
	}
	return -1 // not reached
}

// moveComp makes the computer's move.
func (g *Game) moveComp() {
	// white opens
	if g.moveNum == 1 && g.player == whitePlayer {
		m := openMovesTab[g.RandU8Max(3)]
		g.doMoveDisp(&m)
		return
	}

	g.moveDeep = 0
	g.moveStackTop = 0

	m := g.moveCompLevel()
	g.lastMoveOK = m >= 0
	if m >= 0 {
		g.doMoveDisp(&g.moveStack[m])
	}
}

// turnKey turns the arrows when the board is seen by black.
func (g *Game) turnKey(ch uint8) uint8 {
	if g.playerView != 0 {
		if ch == picopad.KeyLeft || ch == picopad.KeyRight {
			ch ^= picopad.KeyLeft ^ picopad.KeyRight
		}
		if ch == picopad.KeyUp || ch == picopad.KeyDown {
			ch ^= picopad.KeyUp ^ picopad.KeyDown
		}
	}
	return ch
}

// moveCursor moves the cursor of player p with an arrow; it reports
// whether ch was an arrow.
func (g *Game) moveCursor(p *player, ch uint8) bool {
	switch ch {
	case picopad.KeyLeft:
		p.curPos--
		if g.board[p.curPos] == border {
			p.curPos += map0W
		}
	case picopad.KeyRight:
		p.curPos++
		if g.board[p.curPos] == border {
			p.curPos -= map0W
		}
	case picopad.KeyDown:
		p.curPos -= mapW
		if g.board[p.curPos] == border {
			p.curPos += map0H * mapW
		}
	case picopad.KeyUp:
		p.curPos += mapW
		if g.board[p.curPos] == border {
			p.curPos -= map0H * mapW
		}
	default:
		return false
	}
	return true
}

// moveHuman lets the human player move; it returns true to quit the game.
func (g *Game) moveHuman() bool {
	p := &g.players[g.player]

	g.playerView = g.player
	g.dispFrame()

	g.moveDeep = 0
	g.moveStackTop = 0
	g.searchMoves(0)

	// checkmate
	if g.moveStackNum[0] == 0 {
		g.lastMoveOK = false
		return false
	}
	moves := g.moveStack[g.moveStackStart[0] : g.moveStackStart[0]+g.moveStackNum[0]]

	for { // MOVE_AGAIN
		// the pieces that can move
		g.boardEnable = [mapSize]bool{}
		for i := range moves {
			g.boardEnable[moves[i].src] = true
		}
		g.dispBoard()
		g.blinkTime = g.Time()
		g.KeyFlush()

		// select a piece
		for {
			col := picopad.RGB(255, 0, 0)
			ok := false
			if g.boardEnable[p.curPos] {
				col = picopad.RGB(0, 64, 255)
				ok = true
			}

			// the blinking cursor
			w := 4
			if ((g.Time()-g.blinkTime)>>16)&3 == 3 {
				w = 0
			}
			g.dispField(p.curPos, w, col, false, false)

			ch := g.turnKey(g.KeyGet())
			sel := false
			switch ch {
			case picopad.KeyA:
				sel = ok
			case picopad.KeyLeft, picopad.KeyRight, picopad.KeyDown, picopad.KeyUp:
				g.dispField(p.curPos, 0, 0, false, false)
				g.moveCursor(p, ch)
				g.blinkTime = g.Time()
			case picopad.KeyX: // the computer moves
				g.boardEnable = [mapSize]bool{}
				g.dispBoard()
				g.DispUpdate()
				g.moveComp()
				return false
			case picopad.KeyY:
				return true
			}
			if sel {
				break
			}
			g.DispUpdate()
		}

		// the fields the piece can go to
		piece := g.board[p.curPos]
		oldPos := p.curPos
		g.boardEnable = [mapSize]bool{}
		g.boardEnable[oldPos] = true // back to where it was
		for i := range moves {
			if moves[i].src == oldPos {
				g.boardEnable[moves[i].dst] = true
			}
		}
		g.dispBoard()

		// pick the piece up
		g.board[oldPos] = empty
		piece2 := uint8(empty)
		g.blinkTime = g.Time()
		g.KeyFlush()

		// move it
		sel, cancel := false, false
		for {
			col := picopad.RGB(255, 0, 0)
			ok := false
			if g.boardEnable[p.curPos] {
				col = picopad.RGB(0, 64, 255)
				ok = true
			}

			// blink between the field and the piece over it
			if ((g.Time()-g.blinkTime)>>17)&1 == 1 {
				g.board[p.curPos] = piece2
				g.dispField(p.curPos, 0, 0, false, false)
			} else {
				g.board[p.curPos] = piece
				g.dispField(p.curPos, 4, col, false, false)
			}

			ch := g.turnKey(g.KeyGet())
			switch ch {
			case picopad.KeyA:
				sel = ok
			case picopad.KeyB:
				cancel = true
			case picopad.KeyLeft, picopad.KeyRight, picopad.KeyDown, picopad.KeyUp:
				g.board[p.curPos] = piece2
				g.dispField(p.curPos, 0, 0, false, false)
				g.moveCursor(p, ch)
				g.blinkTime = g.Time()
				piece2 = g.board[p.curPos]
			case picopad.KeyX:
				g.boardEnable = [mapSize]bool{}
				g.dispBoard()
				g.DispUpdate()
				g.moveComp()
				return false
			case picopad.KeyY:
				return true
			}
			if sel || cancel {
				break
			}
			g.DispUpdate()
		}

		// put the pieces back
		g.board[p.curPos] = piece2
		g.board[oldPos] = piece
		g.boardEnable = [mapSize]bool{}
		g.dispBoard()
		g.DispUpdate()

		if cancel {
			p.curPos = oldPos
			continue
		}
		if p.curPos == oldPos {
			continue // put back where it was
		}

		// the move (a pawn always becomes a queen)
		for i := range moves {
			m := &moves[i]
			if m.src == oldPos && m.dst == p.curPos && (m.flags != moveFlagQueen || m.extra&pieceMask == queen) {
				g.doMoveDisp(m)
				break
			}
		}
		return false
	}
}

// open is the first screen; it returns false to quit.
func (g *Game) open() bool {
	g.SelFont8x16()
	g.DrawClear()
	g.DrawText2("CHESS", (width-5*16)/2, 20, picopad.ColYellow)

	const menuX = 30
	g.DrawText("Select:", menuX, 70, picopad.ColWhite)
	g.DrawText("LEFT .... Play with white", menuX, 105, picopad.ColWhite)
	g.DrawText("RIGHT ... Play with black", menuX, 125, picopad.ColWhite)
	g.DrawText("DOWN .... Play 2 players", menuX, 145, picopad.ColWhite)
	g.DrawText("UP ...... Demo", menuX, 165, picopad.ColWhite)
	g.DrawText("A ....... Select level:", menuX, 195, picopad.ColWhite)

	g.KeyFlush()
	for {
		switch g.KeyGet() {
		case picopad.KeyLeft:
			g.players[0].comp, g.players[1].comp = false, true
			return true
		case picopad.KeyRight:
			g.players[0].comp, g.players[1].comp = true, false
			return true
		case picopad.KeyDown:
			g.players[0].comp, g.players[1].comp = false, false
			return true
		case picopad.KeyUp:
			g.players[0].comp, g.players[1].comp = true, true
			return true
		case picopad.KeyA:
			g.deepMaxN++
			if g.deepMaxN > deepMax {
				g.deepMaxN = 2
			}
		case picopad.KeyY:
			return false
		}

		level := "SUPERPRO"
		switch g.deepMaxN {
		case 2:
			level = "BEGINNER"
		case 3:
			level = "ADVANCED"
		case 4:
			level = "PRO     "
		}
		g.DrawTextBg(level, menuX+24*8, 195, picopad.ColWhite, picopad.ColBlack)
		g.DispUpdate()
	}
}

// field returns the name of a field ("e2").
func field(pos uint8) string {
	row := int(pos) / 10
	col := int(pos) - row*10
	return string([]byte{byte(col + 'a' - 1), byte(row + '1' - 2)})
}

// Run plays until Y on the first screen (main of the C source, which then
// resets to the boot loader). With OneGame it returns after a game
// (checkmate and a key, or Y).
func (g *Game) Run() {
	g.deepMaxN = 4
	for {
		if !g.open() {
			return
		}
		g.newGame()
		g.dispFrame()
		g.dispBoard()

		for {
			if g.players[g.player].comp {
				g.moveComp()
			} else if g.moveHuman() {
				break
			}

			if g.KeyGet() == picopad.KeyY {
				break
			}

			if !g.lastMoveOK {
				g.outText("CHECKMATE")
				g.WaitMs(500)
				g.KeyFlush()
				for g.KeyGet() == picopad.NoKey {
				}
				break
			}

			// the move counter (4 characters), the move
			txt := "    "
			if g.player != blackPlayer {
				txt = strconv.Itoa(g.moveNum) + ":"
				for len(txt) < 4 {
					txt = " " + txt
				}
			}
			m := &g.lastMove
			switch m.flags {
			case moveFlagSmall:
				txt += "O-O"
			case moveFlagBig:
				txt += "O-O-O"
			default:
				txt += field(m.src) + "-" + field(m.dst)
				if m.dstPiece != empty {
					txt += "x"
				}
				if m.flags == moveFlagEP {
					txt += "e"
				}
			}
			g.outText(txt)

			g.player ^= 1
			if g.player == whitePlayer {
				g.moveNum++
			}
		}
		if g.OneGame { // checkmate or Y
			return
		}
	}
}
