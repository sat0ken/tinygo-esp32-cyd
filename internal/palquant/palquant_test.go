package palquant

import (
	"testing"

	"github.com/sat0ken/tinygo-cyd/picopad"
)

func TestMake(t *testing.T) {
	// two clusters: reds (10 times as many) and blues
	var cs []Weighted
	for i := 0; i < 8; i++ {
		cs = append(cs, Weighted{picopad.RGB(uint8(128+i*16), 0, 0), 10})
		cs = append(cs, Weighted{picopad.RGB(0, 0, uint8(128+i*16)), 1})
	}
	got := Make(cs, 2)
	if len(got) != 2 {
		t.Fatalf("%d colours", len(got))
	}
	// one red, one blue
	if got[0]&0x1F == 0 || got[1]>>11 == 0 {
		t.Errorf("colours %#04x %#04x", got[0], got[1])
	}
	// as many colours as there are: each its own
	if got := Make(cs, 100); len(got) != 16 {
		t.Errorf("%d colours for 16", len(got))
	}
}

func TestCount(t *testing.T) {
	pal := "\x00\xf8" + "\x1f\xf8" + "\xe0\x07" // red, magenta (transparent), green
	imgs := []Image{{Img: "\x00\x01\x01\x02", Pal: pal, Bits: 8, Weight: 3, Trans: 0xF81F, HasTrans: true}}
	cs := Count(imgs)
	if len(cs) != 2 || cs[0] != (Weighted{0xF800, 3}) || cs[1] != (Weighted{0x07E0, 3}) {
		t.Errorf("%v", cs)
	}
	if p := Palette([]picopad.Color{picopad.ColBlack}, imgs); len(p) != 6 || p[:2] != "\x00\x00" {
		t.Errorf("palette %q", p)
	}
}
