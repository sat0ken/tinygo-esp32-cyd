package invaders

// sprite is a 1-bit bitmap, drawn at 2x (every bit is 2x2 pixels).
type sprite struct {
	w, h int16 // in bits
	bits []bool
}

const scale = 2

func parseSprite(rows ...string) sprite {
	s := sprite{w: int16(len(rows[0])), h: int16(len(rows))}
	for _, r := range rows {
		for _, c := range r {
			s.bits = append(s.bits, c == '#')
		}
	}
	return s
}

func (s sprite) on(x, y int16) bool { return s.bits[int(y)*int(s.w)+int(x)] }

// Pixel size on screen.
func (s sprite) pw() int16 { return s.w * scale }
func (s sprite) ph() int16 { return s.h * scale }

// The three invaders, two animation frames each.
var invaderSprites = [3][2]sprite{
	{ // squid, top row, 30 points
		parseSprite(
			"...##...",
			"..####..",
			".######.",
			"##.##.##",
			"########",
			"..#..#..",
			".#.##.#.",
			"#.#..#.#",
		),
		parseSprite(
			"...##...",
			"..####..",
			".######.",
			"##.##.##",
			"########",
			".#.##.#.",
			"#......#",
			".#....#.",
		),
	},
	{ // crab, 20 points
		parseSprite(
			"..#.....#..",
			"...#...#...",
			"..#######..",
			".##.###.##.",
			"###########",
			"#.#######.#",
			"#.#.....#.#",
			"...##.##...",
		),
		parseSprite(
			"..#.....#..",
			"#..#...#..#",
			"#.#######.#",
			"###.###.###",
			"###########",
			".#########.",
			"..#.....#..",
			".#.......#.",
		),
	},
	{ // octopus, 10 points
		parseSprite(
			"....####....",
			".##########.",
			"############",
			"###..##..###",
			"############",
			"...##..##...",
			"..##.##.##..",
			"##........##",
		),
		parseSprite(
			"....####....",
			".##########.",
			"############",
			"###..##..###",
			"############",
			"..###..###..",
			".##..##..##.",
			"..##....##..",
		),
	},
}

var cannonSprite = parseSprite(
	"......#......",
	".....###.....",
	".....###.....",
	".###########.",
	"#############",
	"#############",
	"#############",
	"#############",
)

var explosionSprite = parseSprite(
	"....#...#....",
	".#...#.#...#.",
	"..#.......#..",
	"...#.....#...",
	"##.........##",
	"...#.....#...",
	"..#.#...#.#..",
	".#...#.#...#.",
)

var ufoSprite = parseSprite(
	".....######.....",
	"...##########...",
	"..############..",
	".##.##.##.##.##.",
	"################",
	"..###..##..###..",
	"...#........#...",
)

// Bunker shape in cells (each cell is 4x4 pixels).
var bunkerShape = []string{
	"..#######..",
	".#########.",
	"###########",
	"###########",
	"###########",
	"####...####",
	"###.....###",
	"###.....###",
}
