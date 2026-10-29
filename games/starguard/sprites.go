package starguard

import "image/color"

// sprite is a small picture with transparent pixels.
type sprite struct {
	w, h int16
	pix  []uint16 // RGB565, bytes swapped (see bigEndian)
	on   []bool
}

func (s *sprite) at(x, y int16) (uint16, bool) {
	i := int(y)*int(s.w) + int(x)
	return s.pix[i], s.on[i]
}

// parseSprite turns rows of letters (see spriteColor) into a sprite.
func parseSprite(rows []string) *sprite {
	s := &sprite{w: int16(len(rows[0])), h: int16(len(rows))}
	for _, r := range rows {
		for i := 0; i < len(r); i++ {
			c, on := spriteColor(r[i])
			s.pix = append(s.pix, bigEndian(rgb565(c)))
			s.on = append(s.on, on)
		}
	}
	return s
}

// spriteColor is the colour of a letter of the sprites; '.' is
// transparent.
func spriteColor(c byte) (color.RGBA, bool) {
	switch c {
	case 'a':
		return color.RGBA{70, 60, 55, 255}, true // rock, dark
	case 'b':
		return color.RGBA{120, 105, 95, 255}, true // rock
	case 'c':
		return color.RGBA{175, 160, 145, 255}, true // rock, lit
	case 'k':
		return color.RGBA{40, 44, 60, 255}, true // robot, dark
	case 'w':
		return color.RGBA{200, 205, 220, 255}, true // robot, silver
	case 'r':
		return color.RGBA{240, 50, 50, 255}, true
	case 'y':
		return color.RGBA{255, 220, 60, 255}, true
	case 'o':
		return color.RGBA{255, 140, 30, 255}, true
	case 'W':
		return color.RGBA{255, 255, 240, 255}, true
	}
	return color.RGBA{}, false
}

// rockArt is an asteroid.
var rockArt = [...]string{
	".....bbbbbb.....",
	"...bbbccbbbbb...",
	"..bbccccbbabbb..",
	".bbccbbbbaabbbb.",
	".bcccbbbbbbbbab.",
	"bbccbbbaabbbbbab",
	"bbcbbbaaaabbbbab",
	"bbbbbbaaabbbbaab",
	"bbbabbbbbbbbbaab",
	"bbaaabbbbbccbbab",
	".bbabbbbbccccbb.",
	".bbbbbbbbccbbaa.",
	"..bbbaabbbbbaa..",
	"...bbaaabbbaa...",
	".....baaaaaa....",
	"................",
}

// robotArt is a robot ship.
var robotArt = [...]string{
	"......kkkk......",
	".....kwwwwk.....",
	"....kwwwwwwk....",
	"..kkwrwwwwrwkk..",
	".kwwwwwwwwwwwwk.",
	"kwwwwwwwwwwwwwwk",
	"kyykyykyykyykyyk",
	".kwwwwwwwwwwwwk.",
	"..kkkwwwwwwkkk..",
	"....kk....kk....",
	"...kk......kk...",
	"................",
}

// boomArt are the frames of an explosion.
var boomArt = [3][16]string{
	{
		"................",
		"................",
		"................",
		"................",
		"......y..y......",
		".....yoyyoy.....",
		"......yWWy......",
		".....oyWWyo.....",
		"......yWWy......",
		".....yoyyoy.....",
		"......y..y......",
		"................",
		"................",
		"................",
		"................",
		"................",
	},
	{
		"................",
		"....y......y....",
		".....y.oo.y.....",
		"...o..yyyy..o...",
		"....oyyWWyyo....",
		"..y.yWWWWWWy.y..",
		"...oyWWWWWWyo...",
		"...yyWWWWWWyy...",
		"...oyWWWWWWyo...",
		"..y.yWWWWWWy.y..",
		"....oyyWWyyo....",
		"...o..yyyy..o...",
		".....y.oo.y.....",
		"....y......y....",
		"................",
		"................",
	},
	{
		"..o..........o..",
		"................",
		"....r..o..r.....",
		"..........o.....",
		".o..r........r..",
		"......o..o......",
		"...r........o...",
		"..o....rr....r..",
		"...o........o...",
		"......o..r......",
		".r...........o..",
		"....o..r..o.....",
		"..........r.....",
		".o...r.......o..",
		"................",
		"...o.........r..",
	},
}
