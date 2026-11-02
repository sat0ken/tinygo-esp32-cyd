package mahjong

// Tile is a kind of tile: 0-8 characters (man), 9-17 circles (pin), 18-26
// bamboo (sou), 27-30 the winds E S W N, 31-33 the dragons white, green, red.
// There are four of each and no red fives.
type Tile uint8

const (
	numKinds = 34

	man   Tile = 0
	pin   Tile = 9
	sou   Tile = 18
	east  Tile = 27
	south Tile = 28
	west  Tile = 29
	north Tile = 30
	white Tile = 31
	green Tile = 32
	red   Tile = 33

	noTile Tile = 255
)

// Counts is how many tiles of each kind a hand has.
type Counts [numKinds]uint8

// Mask is a set of kinds of tiles.
type Mask uint64

func (m Mask) has(t Tile) bool { return m&(1<<t) != 0 }

func (t Tile) isHonor() bool { return t >= east }

// number is 0-8 for the suits.
func (t Tile) number() int { return int(t) % 9 }

// suit is 0-2 for man, pin, sou and 3 for the honors.
func (t Tile) suit() int { return int(t) / 9 }

func (t Tile) isTerminal() bool { return !t.isHonor() && (t.number() == 0 || t.number() == 8) }

// isEdge reports a terminal or an honor.
func (t Tile) isEdge() bool { return t.isHonor() || t.isTerminal() }

// doraOf is the tile a dora indicator points at.
func doraOf(ind Tile) Tile {
	switch {
	case ind.isHonor() && ind <= north:
		return east + (ind-east+1)%4
	case ind.isHonor():
		return white + (ind-white+1)%3
	case ind.number() == 8:
		return ind - 8
	}
	return ind + 1
}

func (c *Counts) total() int {
	n := 0
	for _, v := range c {
		n += int(v)
	}
	return n
}

// parseHand reads "123m456p789s11z" (z: 1-7 are E S W N white green red), for
// the tests.
func parseHand(s string) Counts {
	var c Counts
	start := 0
	for i := 0; i < len(s); i++ {
		var base Tile
		switch s[i] {
		case 'm':
			base = man
		case 'p':
			base = pin
		case 's':
			base = sou
		case 'z':
			base = east
		default:
			continue
		}
		for _, d := range s[start:i] {
			c[base+Tile(d-'1')]++
		}
		start = i + 1
	}
	return c
}
