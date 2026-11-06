package futsal

import "github.com/sat0ken/tinygo-cyd/picopad"

// gradeLetters are the grades of an ability 1..8, as on the title screen.
const gradeLetters = "GFEDCBAS"

// pdata is a player of a team: the abilities are 1 (G) .. 8 (S).
type pdata struct {
	name                   string
	speed, kick, tech, def int8 // def: tackling, or saving for a keeper
}

// team is one of the two clubs. Player 0 is the goalkeeper, then the
// back, the left and right wings and the pivot (the forward).
type team struct {
	name, short   string
	color, dark   picopad.Color // the shirts
	gkCol, gkDark picopad.Color // the keeper's shirt
	p             [5]pdata
}

var teams = [2]team{
	{
		name: "SKY HAWKS", short: "HAW",
		color: picopad.RGB(40, 150, 240), dark: picopad.RGB(16, 80, 160),
		gkCol: picopad.RGB(250, 200, 40), gkDark: picopad.RGB(170, 120, 10),
		p: [5]pdata{
			{"MORI", 4, 4, 4, 7},
			{"ISHII", 5, 4, 5, 7},
			{"NAKANO", 7, 5, 6, 4},
			{"TAKEDA", 6, 6, 5, 5},
			{"OGAWA", 6, 8, 6, 3},
		},
	},
	{
		name: "RED FOXES", short: "FOX",
		color: picopad.RGB(228, 50, 40), dark: picopad.RGB(130, 20, 20),
		gkCol: picopad.RGB(60, 200, 90), gkDark: picopad.RGB(20, 110, 40),
		p: [5]pdata{
			{"KONDO", 4, 4, 4, 6},
			{"HARA", 5, 5, 4, 7},
			{"SUGITA", 6, 5, 7, 4},
			{"MAEDA", 7, 6, 5, 4},
			{"ODAGIRI", 6, 7, 7, 3},
		},
	},
}
