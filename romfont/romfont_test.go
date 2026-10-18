package romfont_test

import (
	"image/color"
	"testing"

	"github.com/sat0ken/tinygo-cyd/memlcd"
	"github.com/sat0ken/tinygo-cyd/romfont/freesans"
	"github.com/sat0ken/tinygo-cyd/romfont/proggy"
	"tinygo.org/x/tinyfont"
	tfreesans "tinygo.org/x/tinyfont/freesans"
	tproggy "tinygo.org/x/tinyfont/proggy"
)

// TestSameAsTinyfont draws every rune from 0 to 0x17F (with ones the fonts
// do not have) with both fonts and compares the pixels and the widths.
func TestSameAsTinyfont(t *testing.T) {
	for _, c := range []struct {
		name string
		rom  tinyfont.Fonter
		orig tinyfont.Fonter
	}{
		{"Bold9pt7b", &freesans.Bold9pt7b, &tfreesans.Bold9pt7b},
		{"Bold12pt7b", &freesans.Bold12pt7b, &tfreesans.Bold12pt7b},
		{"Bold18pt7b", &freesans.Bold18pt7b, &tfreesans.Bold18pt7b},
		{"Bold24pt7b", &freesans.Bold24pt7b, &tfreesans.Bold24pt7b},
		{"Regular9pt7b", &freesans.Regular9pt7b, &tfreesans.Regular9pt7b},
		{"TinySZ8pt7b", &proggy.TinySZ8pt7b, &tproggy.TinySZ8pt7b},
	} {
		if c.rom.GetYAdvance() != c.orig.GetYAdvance() {
			t.Errorf("%s: y advance", c.name)
		}
		white := color.RGBA{255, 255, 255, 255}
		for r := rune(0); r < 0x180; r++ {
			a, b := memlcd.New(80, 80), memlcd.New(80, 80)
			s := string(r) + "A" + string(r)
			tinyfont.WriteLine(a, c.rom, 4, 50, s, white)
			tinyfont.WriteLine(b, c.orig, 4, 50, s, white)
			for y := int16(0); y < 80; y++ {
				for x := int16(0); x < 80; x++ {
					if a.Pixel(x, y) != b.Pixel(x, y) {
						t.Fatalf("%s: rune %#x differs at %d,%d", c.name, r, x, y)
					}
				}
			}
			_, wa := tinyfont.LineWidth(c.rom, s)
			_, wb := tinyfont.LineWidth(c.orig, s)
			if wa != wb {
				t.Fatalf("%s: rune %#x width %d, want %d", c.name, r, wa, wb)
			}
		}
	}
}
