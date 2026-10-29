package kart

import "image/color"

// sprite is a small picture with transparent pixels, drawn at
// layout.scale pixels per pixel.
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
	case 'b':
		return color.RGBA{40, 90, 220, 255}, true // body
	case 'd':
		return color.RGBA{20, 50, 140, 255}, true // body, shade
	case 'w':
		return color.RGBA{250, 250, 250, 255}, true
	case 'k':
		return color.RGBA{20, 20, 24, 255}, true // tyres
	case 'g':
		return color.RGBA{120, 120, 130, 255}, true
	case 'y':
		return color.RGBA{250, 200, 40, 255}, true // helmet
	case 'r':
		return color.RGBA{230, 40, 40, 255}, true // lights
	}
	return color.RGBA{}, false
}

// kartArt is the player's kart from behind.
var kartArt = [...]string{
	".......yyyyyy.......",
	"......yywwwwyy......",
	"......yyyyyyyy......",
	".......gggggg.......",
	"...bbbbbbbbbbbbbb...",
	"..bbwwbbbbbbbbwwbb..",
	"kkkbbbbbbbbbbbbbbkkk",
	"kkkbdddddddddddddkkk",
	"kkkbdgkkkkkkkkgddkkk",
	"kkkbbbbbbbbbbbbbbkkk",
	"kkk.brrbbbbbbrrb.kkk",
	"kkk..dddddddddd..kkk",
}
