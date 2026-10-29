// Package atoms is the Exploding Atoms of PicoLibSDK (PicoPad/GAME/ATOMS,
// by Miroslav Nemecek, Unlicense), ported from C to package picopad. It
// follows the C source function by function; the C globals are the fields
// of Game.
//
// Up to five players, human or computer, put atoms on a 10 x 10 board in
// turn: a place with as many atoms as neighbours explodes into them and
// takes them over. The first screen sets the players (LEFT, UP, RIGHT,
// DOWN, B) and A starts; in the game the arrows move the cursor, A puts
// an atom, Y goes back to the first screen; Y there ends the game and Run
// returns (on the PicoPad: back to the boot loader). There is no sound
// yet: the PLAYSOUND calls of the C source are comments.
package atoms

import (
	"strconv"

	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/picopad"
)

//go:generate go run ../../tools/picoimg -pkg atoms -root $PICOLIBSDK -o atoms_img.go PicoPad/GAME/ATOMS/img/atoms.cpp

const (
	tileW   = 24                     // tile width
	tileH   = 24                     // tile height
	mapW    = 10                     // map width
	mapH    = 10                     // map height
	mapSize = mapW * mapH            // map size (100 tiles)
	boardW  = mapW * tileW           // board width in pixels (240)
	infoW   = picopad.Width - boardW // width of the info box (80)
	infoX   = boardW                 // X coordinate of the info box
	width   = picopad.Width
	height  = picopad.Height

	playerOff   = 0 // the player does not play
	playerHuman = 1 // the player is human
	playerComp  = 2 // the player is the computer

	playerNum = 5 // number of players
)

// player is a player (sPlayer).
type player struct {
	typ    uint8         // player*
	start  int           // index of the start place
	scoreY int           // score Y coordinate
	cursor int           // current cursor
	color  picopad.Color // player color
	atoms  int           // number of atoms (0 = lost)
}

// playerText are the names of the players (PlayerText in the C source,
// where the digit is changed in place).
var playerText = [playerNum]string{"Player 1", "Player 2", "Player 3", "Player 4", "Player 5"}

// playerTypeTxt are the names of the player types.
var playerTypeTxt = [3]string{
	"not playing ",
	"HUMAN       ",
	"COMPUTER    ",
}

// Game is Exploding Atoms on a console.
type Game struct {
	*picopad.Console

	players     [playerNum]player
	player      int    // the player on turn
	playerNum   int    // number of players in the game
	curTime     uint32 // cursor blinking time
	playerHuman int    // number of human players

	atoms [mapSize]uint8 // number of atoms
	owner [mapSize]uint8 // owner of the place
}

// New returns the game on d (at least 320x240) with the touch panel t;
// seed starts the random numbers. Set Pad to use a game controller.
func New(d hal.Display, t hal.Touch, seed uint32) *Game {
	g := &Game{Console: picopad.New(d, t, seed)}
	g.players = [playerNum]player{
		{typ: playerHuman, start: 3*mapW + 3, color: picopad.ColRed},
		{typ: playerComp, start: 3*mapW + 6, color: picopad.ColYellow},
		{typ: playerOff, start: 6*mapW + 3, color: picopad.ColGreen},
		{typ: playerOff, start: 6*mapW + 6, color: picopad.RGB(0, 160, 255)},
		{typ: playerOff, start: 9*mapW + 3, color: picopad.ColMagenta},
	}
	return g
}

// infoBg is the background of the info box (INFOBG).
func infoBg() picopad.Color { return picopad.RGB(16, 16, 16) }

// open is the first screen; it returns false to quit the game.
func (g *Game) open() bool {
	g.player = 0
	g.KeyFlush()

	g.DrawClear()
	g.SelFont8x16()

	const (
		menuX  = 10
		menuY  = 70
		menuDY = 20
	)
	for {
		g.DrawText2("Exploding Atoms", (width-15*16)/2, 15, picopad.ColYellow)

		g.DrawText("Setup players:", menuX, menuY-10, picopad.ColWhite)

		g.DrawText("LEFT ..... Player 1 is", menuX, menuY+1*menuDY, g.players[0].color)
		g.DrawText("UP ....... Player 2 is", menuX, menuY+2*menuDY, g.players[1].color)
		g.DrawText("RIGHT .... Player 3 is", menuX, menuY+3*menuDY, g.players[2].color)
		g.DrawText("DOWN ..... Player 4 is", menuX, menuY+4*menuDY, g.players[3].color)
		g.DrawText("B ........ Player 5 is", menuX, menuY+5*menuDY, g.players[4].color)

		g.DrawText("A ........ Start The Game", menuX, menuY+6*menuDY+10, picopad.ColWhite)

		for i := range g.players {
			g.DrawTextBg(playerTypeTxt[g.players[i].typ], menuX+23*8, menuY+(i+1)*menuDY, g.players[i].color, picopad.ColBlack)
		}

		switch g.KeyGet() {
		case picopad.KeyLeft:
			g.nextType(0)
		case picopad.KeyUp:
			g.nextType(1)
		case picopad.KeyRight:
			g.nextType(2)
		case picopad.KeyDown:
			g.nextType(3)
		case picopad.KeyB:
			g.nextType(4)
		case picopad.KeyA:
			return true
		case picopad.KeyY:
			return false
		}

		g.DispUpdate()
	}
}

// nextType changes the type of player i (off, human, computer).
func (g *Game) nextType(i int) {
	g.players[i].typ++
	if g.players[i].typ > 2 {
		g.players[i].typ = 0
	}
}

// dispScore displays the score of a player.
func (g *Game) dispScore(pl int) {
	p := &g.players[pl]
	g.DrawRect(infoX+3, p.scoreY+16, infoW-7, 15, infoBg())

	switch {
	case p.atoms == 0: // lost
		g.DrawText("-lost-", infoX+(infoW-6*8)/2, p.scoreY+16, p.color)
	case g.playerNum == 1: // the winner
		g.DrawText("WINNER", infoX+(infoW-6*8)/2, p.scoreY+16, p.color)
	default:
		s := strconv.Itoa(p.atoms)
		g.DrawText(s, infoX+(infoW-len(s)*8)/2, p.scoreY+16, p.color)
	}
}

// dispScoreSel displays the score table and frames the player on turn
// (blinking at the end of the game).
func (g *Game) dispScoreSel() {
	for i := range g.players {
		p := &g.players[i]
		if p.typ == playerOff {
			continue
		}
		g.DrawRect(infoX+2, p.scoreY-2, infoW-5, 36, infoBg())
		g.DrawText(playerText[i], infoX+(infoW-8*8)/2, p.scoreY, p.color)
		g.dispScore(i)
		if i == g.player && (g.playerNum > 1 || (g.Time()>>17)&1 == 1) {
			pp := &g.players[g.player]
			g.DrawFrame(infoX+2, pp.scoreY-2, infoW-5, 36, pp.color)
		}
	}
}

// dispTile displays a place of the board, with a cursor frame.
func (g *Game) dispTile(inx int, cursor bool) {
	n := int(min(g.atoms[inx], 4))
	owner := int(g.owner[inx])

	x := inx % mapW * tileW
	y := inx / mapW * tileH

	tile := 0
	if n > 0 {
		tile = n + owner*4
	}
	g.DrawImg4Pal(atomsImg, atomsImgPal, 0, tile*tileH, x, y, tileW, tileH, tileW)

	if cursor {
		g.DrawFrame(x+1, y+1, tileW-2, tileH-2, g.players[g.player].color)
	}
}

// dispBoard displays the board.
func (g *Game) dispBoard() {
	for i := 0; i < mapSize; i++ {
		g.dispTile(i, false)
	}
}

// dispCur displays the cursor of the player on turn (blinking).
func (g *Game) dispCur() {
	g.dispTile(g.players[g.player].cursor, ((g.Time()-g.curTime)>>17)&3 < 3)
}

// dispCurOff hides the cursor of the player on turn.
func (g *Game) dispCurOff() {
	g.dispTile(g.players[g.player].cursor, false)
}

// newGame starts a new game.
func (g *Game) newGame() {
	g.KeyFlush()
	g.DrawClear()

	g.atoms = [mapSize]uint8{}
	g.owner = [mapSize]uint8{}

	g.playerNum = 0
	g.playerHuman = 0
	g.player = -1
	for i := range g.players {
		p := &g.players[i]
		if p.typ == playerOff {
			p.atoms = 0
			continue
		}
		if g.player < 0 {
			g.player = i // the first player
		}
		p.atoms = 1
		p.scoreY = g.playerNum*48 + 8
		g.playerNum++
		g.atoms[p.start] = 1
		g.owner[p.start] = uint8(i)
		p.cursor = p.start
		if p.typ == playerHuman {
			g.playerHuman++
		}
	}

	g.DrawRect(infoX, 0, infoW, height, infoBg())
	g.dispBoard()
	g.dispScoreSel()
	g.DispUpdate()
}

// catch adds an atom of the player on turn to a neighbour, taking it over.
func (g *Game) catch(inx int) {
	n := int(g.atoms[inx])
	own := int(g.owner[inx])

	// atoms of another player
	if n > 0 && own != g.player {
		g.players[own].atoms -= n
		g.dispScore(own)

		g.players[g.player].atoms += n
		g.dispScore(g.player)

		if g.players[own].atoms == 0 {
			g.playerNum-- // lost
		}
	}

	g.atoms[inx] = uint8(n + 1)
	g.owner[inx] = uint8(g.player)
	g.dispTile(inx, false)
}

// maxAtoms returns how many atoms place x, y holds before it explodes
// (its number of neighbours).
func maxAtoms(x, y int) uint8 {
	mx := uint8(4)
	if x == 0 || x == mapW-1 {
		mx--
	}
	if y == 0 || y == mapH-1 {
		mx--
	}
	return mx
}

// explo explodes the full places, a random one at a time.
func (g *Game) explo() {
	for {
		// count the exploding places (only the player on turn can have
		// some now)
		n := 0
		for i := range g.atoms {
			if g.atoms[i] >= maxAtoms(i%mapW, i/mapW) {
				n++
			}
		}
		if n == 0 {
			break
		}

		// explode a random one
		n = int(g.RandS16Max(int16(n - 1)))
		for i := range g.atoms {
			x, y := i%mapW, i/mapW
			mx := maxAtoms(x, y)
			if g.atoms[i] < mx {
				continue
			}
			n--
			if n >= 0 {
				continue
			}
			g.atoms[i] -= mx
			g.dispTile(i, false)

			if x > 0 {
				g.catch(i - 1)
			}
			if x < mapW-1 {
				g.catch(i + 1)
			}
			if y > 0 {
				g.catch(i - mapW)
			}
			if y < mapH-1 {
				g.catch(i + mapW)
			}

			// PLAYSOUND(CrashSnd)
			g.DispUpdate()
			g.WaitMs(80)
			break
		}

		// with one player left and the board full, the explosions might
		// never end
		if g.playerNum <= 1 {
			break
		}
	}
}

// auto finds the place of the computer and puts the cursor there.
func (g *Game) auto() {
	bestSize := uint8(0)
	bestNum := 0
	g.dispCurOff()

	// the largest of the player's places, and how many
	for i := range g.atoms {
		if int(g.owner[i]) == g.player {
			if g.atoms[i] > bestSize {
				bestSize = g.atoms[i]
				bestNum = 1
			} else if g.atoms[i] == bestSize {
				bestNum++
			}
		}
	}

	// one of them
	bestNum = int(g.RandU8Max(uint8(bestNum - 1)))
	i := 0
	for ; ; i++ {
		if int(g.owner[i]) == g.player && g.atoms[i] == bestSize {
			bestNum--
			if bestNum < 0 {
				break
			}
		}
	}

	g.players[g.player].cursor = i
	g.dispTile(i, true)
	g.DispUpdate()
	g.WaitMs(200)
}

// game plays a game.
func (g *Game) game() {
	g.curTime = g.Time()

	for {
		// any key stops the demo
		if g.playerHuman == 0 && g.KeyGet() != picopad.NoKey {
			return
		}

		var ch uint8
		if g.players[g.player].typ == playerComp {
			g.auto()
			ch = picopad.KeyA
		} else {
			ch = g.KeyGet()
		}

		cur := g.players[g.player].cursor

		switch ch {
		case picopad.KeyLeft:
			g.dispCurOff()
			cur--
			if cur < 0 || cur%mapW == mapW-1 {
				cur += mapW
			}
			g.curTime = g.Time()

		case picopad.KeyRight:
			g.dispCurOff()
			cur++
			if cur%mapW == 0 {
				cur -= mapW
			}
			g.curTime = g.Time()

		case picopad.KeyUp:
			g.dispCurOff()
			cur -= mapW
			if cur < 0 {
				cur += mapSize
			}
			g.curTime = g.Time()

		case picopad.KeyDown:
			g.dispCurOff()
			cur += mapW
			if cur >= mapSize {
				cur -= mapSize
			}
			g.curTime = g.Time()

		case picopad.KeyA: // put an atom
			if g.atoms[cur] != 0 && int(g.owner[cur]) != g.player {
				break // another player's
			}
			g.owner[cur] = uint8(g.player)
			g.atoms[cur]++
			g.players[g.player].atoms++
			g.dispTile(cur, false)
			g.dispScore(g.player)

			g.explo()

			// the end
			if g.playerNum < 2 {
				g.dispCurOff()
				g.DispUpdate()

				g.KeyFlush()
				for g.KeyGet() == picopad.NoKey {
					g.dispScoreSel() // the winner blinks
					g.DispUpdate()
				}
				return
			}

			// the next player
			for {
				g.player++
				if g.player >= playerNum {
					g.player = 0
				}
				if g.players[g.player].atoms != 0 {
					break
				}
			}
			cur = g.players[g.player].cursor
			g.dispScoreSel()
			g.curTime = g.Time()

		case picopad.KeyY:
			return
		}
		g.players[g.player].cursor = cur

		g.dispCur()
		g.DispUpdate()
	}
}

// Run plays until Y on the first screen (main of the C source, which then
// resets to the boot loader).
func (g *Game) Run() {
	for {
		if !g.open() {
			return
		}
		g.newGame()
		if g.playerNum > 1 {
			g.game()
		}
	}
}
