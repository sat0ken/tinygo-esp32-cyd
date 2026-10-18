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

// The sprites are parsed by parseSprites when the first game is created,
// not by package initializers: TinyGo cannot run parseSprite at compile
// time, and a package initializer left for run time made TinyGo also leave
// the initializer of machine's UART0 for run time. On the ESP32 the runtime
// sets up the UART before any package initializer runs, so the board hung
// at boot as soon as this package was linked (cmd/games on the
// ESP32-3248S035C).
var (
	invaderSprites                           [3][2]sprite
	cannonSprite, explosionSprite, ufoSprite sprite
	spritesParsed                            bool
)

func parseSprites() {
	if spritesParsed {
		return
	}
	for k := range invaderArt {
		for f := range invaderArt[k] {
			invaderSprites[k][f] = parseSprite(invaderArt[k][f]...)
		}
	}
	cannonSprite = parseSprite(cannonArt...)
	explosionSprite = parseSprite(explosionArt...)
	ufoSprite = parseSprite(ufoArt...)
	spritesParsed = true
}

// The three invaders, two animation frames each.
var invaderArt = [3][2][]string{
	{ // squid, top row, 30 points
		{
			"...##...",
			"..####..",
			".######.",
			"##.##.##",
			"########",
			"..#..#..",
			".#.##.#.",
			"#.#..#.#",
		},
		{
			"...##...",
			"..####..",
			".######.",
			"##.##.##",
			"########",
			".#.##.#.",
			"#......#",
			".#....#.",
		},
	},
	{ // crab, 20 points
		{
			"..#.....#..",
			"...#...#...",
			"..#######..",
			".##.###.##.",
			"###########",
			"#.#######.#",
			"#.#.....#.#",
			"...##.##...",
		},
		{
			"..#.....#..",
			"#..#...#..#",
			"#.#######.#",
			"###.###.###",
			"###########",
			".#########.",
			"..#.....#..",
			".#.......#.",
		},
	},
	{ // octopus, 10 points
		{
			"....####....",
			".##########.",
			"############",
			"###..##..###",
			"############",
			"...##..##...",
			"..##.##.##..",
			"##........##",
		},
		{
			"....####....",
			".##########.",
			"############",
			"###..##..###",
			"############",
			"..###..###..",
			".##..##..##.",
			"..##....##..",
		},
	},
}

var cannonArt = []string{
	"......#......",
	".....###.....",
	".....###.....",
	".###########.",
	"#############",
	"#############",
	"#############",
	"#############",
}

var explosionArt = []string{
	"....#...#....",
	".#...#.#...#.",
	"..#.......#..",
	"...#.....#...",
	"##.........##",
	"...#.....#...",
	"..#.#...#.#..",
	".#...#.#...#.",
}

var ufoArt = []string{
	".....######.....",
	"...##########...",
	"..############..",
	".##.##.##.##.##.",
	"################",
	"..###..##..###..",
	"...#........#...",
}

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
