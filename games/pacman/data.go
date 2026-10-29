package pacman

// The tables of src/data.cpp.

// fruitTab are the bonus fruits of the levels.
var fruitTab = [fruitTabMax]uint8{0, 1, 2, 2, 3, 3, 4, 4, 5, 5, 6, 6, 7, 7, 7, 7, 7, 7, 7}

// bonusTab are the points of the fruits.
var bonusTab = [8]int{100, 300, 500, 700, 1000, 2000, 3000, 5000}

// frightenTab is the frighten time of the levels (seconds).
var frightenTab = [frightenTabMax]uint8{9, 9, 8, 8, 7, 7, 6, 6, 5, 5, 4, 4, 3, 3, 2, 2, 1, 1, 0}

// scatterTab: 5 levels x 4 pairs of scatter and chase times (seconds, 0:
// for ever); every row ends with 0.
var scatterTab = [5 * 4 * 2]uint8{
	7, 20, 7, 20, 5, 20, 5, 0, // level 1
	7, 20, 7, 20, 5, 0, 0, 0, // level 2
	7, 20, 7, 20, 5, 0, 0, 0, // level 3
	7, 20, 7, 20, 5, 0, 0, 0, // level 4
	5, 20, 5, 20, 5, 0, 0, 0, // level 5
}

// boardTemp is the game board (the tile indices of doc/tiles_num.png).
var boardTemp = [mapSize]uint8{
	// 0   1   2   3   4   5   6   7   8   9  10  11  12  13  14  15  16  17  18  19  20  21  22  23  24  25  26  27
	1, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 9, 10, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 3, // 0
	7, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 24, 24, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 7, // 1
	7, 31, 19, 20, 20, 21, 30, 19, 20, 20, 20, 21, 30, 24, 24, 30, 19, 20, 20, 20, 21, 30, 19, 20, 20, 21, 31, 7, // 2
	7, 30, 22, 20, 20, 23, 30, 22, 20, 20, 20, 23, 30, 22, 23, 30, 22, 20, 20, 20, 23, 30, 22, 20, 20, 23, 30, 7, // 3
	7, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 7, // 4
	7, 30, 5, 2, 2, 8, 30, 19, 21, 30, 5, 2, 2, 9, 10, 2, 2, 8, 30, 19, 21, 30, 5, 2, 2, 8, 30, 7, // 5
	7, 30, 30, 30, 30, 30, 30, 24, 24, 30, 30, 30, 30, 24, 24, 30, 30, 30, 30, 24, 24, 30, 30, 30, 30, 30, 30, 7, // 6
	4, 2, 2, 2, 2, 3, 30, 24, 17, 2, 2, 8, 0, 22, 23, 0, 5, 2, 2, 18, 24, 30, 1, 2, 2, 2, 2, 6, // 7
	0, 0, 0, 0, 0, 7, 30, 24, 24, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 24, 24, 30, 7, 0, 0, 0, 0, 0, // 8
	2, 2, 2, 2, 2, 6, 30, 22, 23, 0, 25, 2, 32, 29, 42, 33, 2, 26, 0, 22, 23, 30, 4, 2, 2, 2, 2, 2, // 9
	0, 0, 0, 0, 0, 0, 30, 0, 0, 0, 7, 0, 0, 0, 0, 0, 0, 7, 0, 0, 0, 30, 0, 0, 0, 0, 0, 0, // 10
	2, 2, 2, 2, 2, 3, 30, 19, 21, 0, 7, 0, 0, 0, 0, 0, 0, 7, 0, 19, 21, 30, 1, 2, 2, 2, 2, 2, // 11
	0, 0, 0, 0, 0, 7, 30, 24, 24, 0, 27, 2, 2, 2, 2, 2, 2, 28, 0, 24, 24, 30, 7, 0, 0, 0, 0, 0, // 12
	0, 0, 0, 0, 0, 7, 30, 24, 24, 0, 0, 35, 36, 37, 38, 39, 40, 0, 0, 24, 24, 30, 7, 0, 0, 0, 0, 0, // 13
	1, 2, 2, 2, 2, 6, 30, 22, 23, 0, 5, 2, 2, 9, 10, 2, 2, 8, 0, 22, 23, 30, 4, 2, 2, 2, 2, 3, // 14
	7, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 24, 24, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 7, // 15
	7, 30, 5, 2, 9, 12, 30, 5, 2, 2, 2, 8, 30, 22, 23, 30, 5, 2, 2, 2, 8, 30, 14, 10, 2, 8, 30, 7, // 16
	7, 31, 30, 30, 24, 24, 30, 30, 30, 30, 30, 30, 30, 0, 0, 30, 30, 30, 30, 30, 30, 30, 24, 24, 30, 30, 31, 7, // 17
	11, 2, 8, 30, 22, 23, 30, 19, 21, 30, 5, 2, 2, 9, 10, 2, 2, 8, 30, 19, 21, 30, 22, 23, 30, 5, 2, 13, // 18
	7, 30, 30, 30, 30, 30, 30, 24, 24, 30, 30, 30, 30, 24, 24, 30, 30, 30, 30, 24, 24, 30, 30, 30, 30, 30, 30, 7, // 19
	7, 30, 5, 2, 2, 2, 2, 15, 16, 2, 2, 8, 30, 22, 23, 30, 5, 2, 2, 15, 16, 2, 2, 2, 2, 8, 30, 7, // 20
	7, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 7, // 21
	4, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 6, // 22
}

// the direction vectors: left, down, right, up
var (
	dirX = [4]int{-1, 0, +1, 0}
	dirY = [4]int{0, +1, 0, -1}
)

// spriteInx are the sprites of the characters: 9 characters x 4
// directions x 4 animation phases.
var spriteInx = [charNum * dirNum * phaseNum]uint8{
	// Pacman: left, down, right, up
	spritePacFull, spritePacL, spritePacL2, spritePacL,
	spritePacFull, spritePacD, spritePacD2, spritePacD,
	spritePacFull, spritePacR, spritePacR2, spritePacR,
	spritePacFull, spritePacU, spritePacU2, spritePacU,

	// Blinky
	spriteBlinky, spriteBlinky, spriteBlinky + 1, spriteBlinky + 1,
	spriteBlinky + 2, spriteBlinky + 2, spriteBlinky + 3, spriteBlinky + 3,
	spriteBlinky + 4, spriteBlinky + 4, spriteBlinky + 5, spriteBlinky + 5,
	spriteBlinky + 6, spriteBlinky + 6, spriteBlinky + 7, spriteBlinky + 7,

	// Pinky
	spritePinky, spritePinky, spritePinky + 1, spritePinky + 1,
	spritePinky + 2, spritePinky + 2, spritePinky + 3, spritePinky + 3,
	spritePinky + 4, spritePinky + 4, spritePinky + 5, spritePinky + 5,
	spritePinky + 6, spritePinky + 6, spritePinky + 7, spritePinky + 7,

	// Inky
	spriteInky, spriteInky, spriteInky + 1, spriteInky + 1,
	spriteInky + 2, spriteInky + 2, spriteInky + 3, spriteInky + 3,
	spriteInky + 4, spriteInky + 4, spriteInky + 5, spriteInky + 5,
	spriteInky + 6, spriteInky + 6, spriteInky + 7, spriteInky + 7,

	// Clyde
	spriteClyde, spriteClyde, spriteClyde + 1, spriteClyde + 1,
	spriteClyde + 2, spriteClyde + 2, spriteClyde + 3, spriteClyde + 3,
	spriteClyde + 4, spriteClyde + 4, spriteClyde + 5, spriteClyde + 5,
	spriteClyde + 6, spriteClyde + 6, spriteClyde + 7, spriteClyde + 7,

	// frozen
	spriteFrozen, spriteFrozen, spriteFrozen + 1, spriteFrozen + 1,
	spriteFrozen, spriteFrozen, spriteFrozen + 1, spriteFrozen + 1,
	spriteFrozen, spriteFrozen, spriteFrozen + 1, spriteFrozen + 1,
	spriteFrozen, spriteFrozen, spriteFrozen + 1, spriteFrozen + 1,

	// frightened
	spriteFrigh, spriteFrigh, spriteFrigh + 1, spriteFrigh + 1,
	spriteFrigh, spriteFrigh, spriteFrigh + 1, spriteFrigh + 1,
	spriteFrigh, spriteFrigh, spriteFrigh + 1, spriteFrigh + 1,
	spriteFrigh, spriteFrigh, spriteFrigh + 1, spriteFrigh + 1,

	// eyes
	spriteEye, spriteEye, spriteEye, spriteEye,
	spriteEye + 1, spriteEye + 1, spriteEye + 1, spriteEye + 1,
	spriteEye + 2, spriteEye + 2, spriteEye + 2, spriteEye + 2,
	spriteEye + 3, spriteEye + 3, spriteEye + 3, spriteEye + 3,

	// die (8 phases)
	spriteDie, spriteDie + 1, spriteDie + 2, spriteDie + 3,
	spriteDie + 4, spriteDie + 5, spriteDie + 6, spriteDie + 7,
	0, 0, 0, 0, 0, 0, 0, 0,
}
