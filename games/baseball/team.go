package baseball

import "github.com/sat0ken/tinygo-cyd/picopad"

// The abilities are grades from 1 (G) to 8 (S), as on a scouting report.
const gradeLetters = "GFEDCBAS"

// player is a batter and fielder.
type player struct {
	name    string
	pos     int  // fielding position
	left    bool // bats left-handed
	contact int8 // the size of the cursor, the timing window
	power   int8 // how hard the ball comes off the bat
	speed   int8 // running
	arm     int8 // throwing
	field   int8 // range
}

// pitcher is the team's pitcher (he bats ninth).
type pitcher struct {
	velocity float32 // the fastball, km/h
	control  int8
	pitches  [3]pitchLevel // breaking balls (level 0: none)
}

// pitchLevel is a breaking ball and how much it breaks (1..7).
type pitchLevel struct {
	kind  int8
	level int8
}

// team is a team.
type team struct {
	name, short string
	color       picopad.Color // the cap and the trim
	dark        picopad.Color
	order       [9]player
	pitcher     pitcher
}

// runSpeed is how fast a player runs the bases (ft/s).
func (p *player) runSpeed() float32 { return 23 + float32(p.speed)*0.9 }

// rangeSpeed is how fast he gets to a ball in the field (ft/s).
func (p *player) rangeSpeed() float32 { return 20 + float32(p.field+p.speed)*0.3 }

// throwSpeed is how fast he throws (ft/s).
func (p *player) throwSpeed() float32 { return 95 + float32(p.arm)*6 }

// teams are the two teams: the player's, and the computer's.
var teams = [2]team{
	{
		name: "BLUE STARS", short: "BLU",
		color: picopad.RGB(40, 90, 220), dark: picopad.RGB(20, 40, 120),
		order: [9]player{
			{"AOKI", posCF, true, 6, 3, 8, 5, 6},
			{"MIZUNO", posSS, false, 5, 3, 6, 6, 7},
			{"SORANO", pos1B, true, 6, 6, 3, 4, 4},
			{"KAIDO", posLF, false, 4, 8, 3, 5, 3},
			{"HOSHIDA", pos3B, false, 5, 6, 4, 6, 5},
			{"KUROKI", posRF, true, 5, 5, 5, 7, 5},
			{"YANAGI", pos2B, false, 6, 2, 6, 4, 6},
			{"ISHIDA", posC, false, 3, 5, 2, 6, 5},
			{"TAKA", posP, false, 2, 2, 3, 6, 4},
		},
		pitcher: pitcher{148, 5, [3]pitchLevel{{ptSlider, 5}, {ptCurve, 3}, {ptFork, 4}}},
	},
	{
		name: "RED COMETS", short: "RED",
		color: picopad.RGB(210, 40, 40), dark: picopad.RGB(110, 20, 20),
		order: [9]player{
			{"HIBINO", pos2B, false, 6, 2, 7, 4, 6},
			{"AKAGI", posCF, true, 5, 3, 7, 6, 7},
			{"MINE", posRF, true, 6, 5, 4, 7, 5},
			{"GODA", pos1B, false, 4, 8, 2, 4, 3},
			{"KIRYU", pos3B, false, 5, 6, 4, 6, 5},
			{"SENA", posLF, false, 5, 4, 5, 5, 4},
			{"OKITA", posSS, true, 5, 3, 6, 6, 7},
			{"NUMATA", posC, false, 3, 4, 2, 7, 6},
			{"HAGANE", posP, true, 2, 3, 2, 6, 4},
		},
		pitcher: pitcher{145, 6, [3]pitchLevel{{ptShoot, 4}, {ptCurve, 5}, {ptSlider, 3}}},
	},
}
