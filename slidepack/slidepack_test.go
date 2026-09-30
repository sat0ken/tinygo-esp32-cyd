package slidepack

import (
	"bytes"
	"io"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	const w, h = 4, 3
	frames := [][]byte{bytes.Repeat([]byte{1}, w*h*2), bytes.Repeat([]byte{2}, w*h*2)}
	var buf bytes.Buffer
	if err := Write(&buf, w, h, []string{"first", "a-very-long-name-that-is-cut-at-32-bytes"}, frames); err != nil {
		t.Fatal(err)
	}
	p, err := Open(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if p.Width != w || p.Height != h || p.Len() != 2 || p.Size() != int64(buf.Len()) {
		t.Fatalf("%+v size %d", p, buf.Len())
	}
	if p.Names[0] != "first" || p.Names[1] != "a-very-long-name-that-is-cut-at-" {
		t.Fatalf("names %q", p.Names)
	}
	data, _ := io.ReadAll(p.Slide(1))
	if !bytes.Equal(data, frames[1]) {
		t.Fatal("frame 1 differs")
	}
}

func TestErasedFlash(t *testing.T) {
	// Flash that was never written reads as 0xFF.
	if _, err := Open(bytes.NewReader(bytes.Repeat([]byte{0xFF}, HeaderSize))); err != ErrMagic {
		t.Fatalf("got %v", err)
	}
}
