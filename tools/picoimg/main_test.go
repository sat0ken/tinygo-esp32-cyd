package main

import (
	"bytes"
	"testing"
)

func TestParse(t *testing.T) {
	src := []byte("#include \"../include.h\"\r\n\r\n" +
		"// format: 8-bit paletted pixel graphics\r\n// image width: 2 pixels\r\n" +
		"const u16 TilesImg_Pal[2] __attribute__ ((aligned(4))) = {\r\n\t0x801F, 0x0008, \r\n};\r\n\r\n" +
		"const u8 TilesImg[2] __attribute__ ((aligned(4))) = {\r\n\t0x01, 255\r\n};\r\n")
	cs, err := parse(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 2 {
		t.Fatalf("%d arrays, want 2", len(cs))
	}
	if c := cs[0]; c.name != "tilesImgPal" || !bytes.Equal(c.data, []byte{0x1F, 0x80, 0x08, 0x00}) || len(c.comment) != 2 {
		t.Errorf("palette: %q %x %q", c.name, c.data, c.comment)
	}
	if c := cs[1]; c.name != "tilesImg" || !bytes.Equal(c.data, []byte{1, 255}) || len(c.comment) != 0 {
		t.Errorf("image: %q %x %q", c.name, c.data, c.comment)
	}
}

func TestParseRange(t *testing.T) {
	if _, err := parse([]byte("const u8 X[1] = { 0x100 };")); err == nil {
		t.Error("0x100 in a u8 array: no error")
	}
}
