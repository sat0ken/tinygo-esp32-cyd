// Package ants is the Ants of PicoLibSDK (PicoPad/GAME/ANTS, by Miroslav
// Nemecek, Unlicense), a card game of the black and the red ants building
// castles, ported from C to package picopad. It follows the C source
// function by function (src/main.cpp, src/game.cpp); the C globals are
// the fields of Game.
//
// The open screen chooses with LEFT/RIGHT and A (or DOWN): two players,
// a player against the computer (black or red), or the computer against
// itself; X changes the sound mode, Y ends Run (on the PicoPad: back to
// the boot loader). In the game LEFT/RIGHT choose a card, A or DOWN plays
// it, B or UP discards it, X shows the computer's choice, Y goes back. A
// castle of 100 or the other castle down wins. The two USB multiplayer
// choices of the open screen (two PicoPads on a cable) are not here, and
// there is no sound yet.
package ants

import (
	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/picopad"
)

//go:generate go run ../../tools/picoimg -pkg ants -root $PICOLIBSDK -o img.go PicoPad/GAME/ANTS/img/background.cpp PicoPad/GAME/ANTS/img/blacksreds.cpp PicoPad/GAME/ANTS/img/cards.cpp PicoPad/GAME/ANTS/img/castle.cpp PicoPad/GAME/ANTS/img/cloud.cpp PicoPad/GAME/ANTS/img/cursor.cpp PicoPad/GAME/ANTS/img/fence.cpp PicoPad/GAME/ANTS/img/flags.cpp PicoPad/GAME/ANTS/img/grass.cpp PicoPad/GAME/ANTS/img/open.cpp PicoPad/GAME/ANTS/img/open_sel.cpp PicoPad/GAME/ANTS/img/open_sel2.cpp PicoPad/GAME/ANTS/img/open_sel3.cpp PicoPad/GAME/ANTS/img/shadow.cpp PicoPad/GAME/ANTS/img/state.cpp PicoPad/GAME/ANTS/img/title.cpp PicoPad/GAME/ANTS/img/title2.cpp PicoPad/GAME/ANTS/img/trumpet.cpp PicoPad/GAME/ANTS/img/win.cpp

const (
	width  = picopad.Width
	height = picopad.Height

	blacksRedsAllW = 128
	blacksRedsW    = 64
	blacksRedsH    = 32
	cardsAllW      = 1320
	cardW          = 40 // card width
	cardH          = 60 // card height
	castleAllW     = 128
	castleW        = 64
	castleH        = 116
	cloudW         = 32
	cloudH         = 21
	cursorW        = 16
	cursorH        = 30
	fenceW         = 8
	fenceH         = 108
	flagsAllW      = 192
	flagsW         = 24
	flagsH         = 20
	grassW         = 80
	grassH         = 16
	grassY         = height - cardH - grassH
	stateW         = 41
	stateH         = 144
	stateY         = 36 - 2
	stateDY        = stateH / 8 // 18
	trumpetAllW    = 38
	trumpetW       = 19
	trumpetH       = 32
	winAllW        = 1024
	winW           = 128
	winH           = 32

	openSelNum = 4 // the choices of the open screen without the 2 of USB

	verText   = "v1.2"
	soundText = "X:Sound "
)

var (
	openText = [...]string{
		"None of the black or red ants remember",
		"who and when started their age-old war",
		"for the anthill \"At the Two Spruces\".",
		"In order to put an end to the eternal",
		"battles, they decided to build castles,",
		"with the help of which they would gain",
		"dominance over the whole territory.",
		"The aim of the game is to build a castle with a height of 100",
		"or destroy the opponent's castle. Each card consumes certain",
		"amount of resources (number at the top right). The rate of",
		"resource generation depends on the amount of workers.",
	}
	soundTextSel = [3]string{"OFF ", "SOME", "ALL "}
)

// Game is Ants on a console.
type Game struct {
	*picopad.Console

	openSelInx int // the choice of the open screen
	soundMode  int // sound*

	players     [2]player
	demo        bool // the computer against itself
	player      int  // the active player (0 or 1)
	last        int  // the last laid card (-1: none)
	lastDiscard bool // the last card was discarded
	selCard     int  // the selected card (cursor)
	freePos     int  // the free card position
	endGame     int  // the winner (-1: none)

	// the screen under the moving card and its shadow, the clouds and the
	// winners (C: u16 from FrameBuf)
	saveCardBuf  [cardW * cardH]uint8
	saveCardBuf2 [cardW * cardH]uint8
	saveCloudBuf [cloudW * cloudH * parNum * 2]uint8
	saveWinBuf   [winW * winH]uint8
}

// New returns the game on d (at least 320x240) with the touch panel t;
// seed starts the random numbers. Set Pad to use a game controller.
func New(d hal.Display, t hal.Touch, seed uint32) *Game {
	return &Game{Console: picopad.New(d, t, seed), soundMode: soundAll, endGame: -1}
}

// dispOpen displays the open screen.
func (g *Game) dispOpen() {
	g.DrawClear()

	// the ants and the title
	g.DrawImg4Pal(openImg, openImgPal, 0, 0, 0, 0, 120, 120, 240)
	g.DrawImg4Pal(openImg, openImgPal, 120, 0, width-120, 53, 120, 120, 240)
	g.DrawImg4Pal(titleImg, titleImgPal, 0, 0, 120+20, 0, 140, 53, 140)
	g.DrawImg4Pal(title2Img, title2ImgPal, 0, 0, 120+10, 53, 60, 50, 60)

	g.SelFont5x8()
	for i, s := range openText {
		g.DrawText(s, 0, 110+i*8, picopad.ColWhite)
	}

	g.SelFont8x16()
	g.DrawText(verText, 280, 40, picopad.ColWhite)
	g.DrawText(soundText, 0, 0, picopad.ColGray)
	g.DrawText(soundTextSel[g.soundMode], 8*8, 0, picopad.ColGray)
}

// dispOpenSel displays the choices of the open screen.
func (g *Game) dispOpenSel() {
	g.DrawImg4Pal(openSelImg, openSelImgPal, 0, 0, 0, height-40, width, 40, width)
	if g.openSelInx < 4 {
		g.DrawBlit4Pal(openSel2Img, openSel2ImgPal, 0, 0, g.openSelInx*60, height-40, 60, 40, 60, picopad.ColBlack)
	} else {
		g.DrawBlit4Pal(openSel3Img, openSel3ImgPal, 0, 0, (g.openSelInx-4)*40+4*60, height-40, 40, 40, 40, picopad.ColBlack)
	}
}

// Run plays until Y on the open screen (main of the C source, which then
// resets to the boot loader). With OneGame it returns after a game (the
// winner, or Y).
func (g *Game) Run() {
	g.dispOpen()
	g.dispOpenSel()
	for {
		switch g.KeyGet() {
		case picopad.KeyLeft:
			g.openSelInx--
			if g.openSelInx < 0 {
				g.openSelInx = openSelNum - 1
			}
			g.dispOpenSel()
			g.KeyFlush()
		case picopad.KeyRight:
			g.openSelInx++
			if g.openSelInx >= openSelNum {
				g.openSelInx = 0
			}
			g.dispOpenSel()
			g.KeyFlush()
		case picopad.KeyDown, picopad.KeyA:
			switch g.openSelInx {
			case 0:
				g.game(playerHuman, playerHuman)
			case 1:
				g.game(playerHuman, playerComp)
			case 2:
				g.game(playerComp, playerHuman)
			case 3:
				g.game(playerComp, playerComp)
			}
			if g.OneGame {
				return
			}
			g.dispOpen()
			g.dispOpenSel()
			g.KeyFlush()
		case picopad.KeyY:
			return
		case picopad.KeyX: // the sound mode
			g.soundMode--
			if g.soundMode < 0 {
				g.soundMode = soundAll
			}
			g.dispOpen()
			g.dispOpenSel()
		case picopad.NoKey:
		default:
			g.KeyFlush()
		}
		g.DispUpdate()
	}
}
