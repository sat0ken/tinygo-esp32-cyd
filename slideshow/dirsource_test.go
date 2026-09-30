package slideshow

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"io/fs"
	"os"
	"testing"
	"time"

	"github.com/sat0ken/tinygo-cyd/memlcd"
)

// memFS is an in-memory FS with one directory level.
type memFS struct {
	files map[string][]byte // "/dir/name" -> data; a nil value is a directory
	open  int               // files currently open
}

type memInfo struct {
	name string
	size int64
	dir  bool
}

func (i memInfo) Name() string       { return i.name }
func (i memInfo) Size() int64        { return i.size }
func (i memInfo) Mode() fs.FileMode  { return 0 }
func (i memInfo) ModTime() time.Time { return time.Time{} }
func (i memInfo) IsDir() bool        { return i.dir }
func (i memInfo) Sys() any           { return nil }

type memFile struct {
	*bytes.Reader
	fs    *memFS
	infos []os.FileInfo
}

func (f *memFile) Close() error { f.fs.open--; return nil }
func (f *memFile) Readdir(n int) ([]os.FileInfo, error) {
	if f.infos == nil {
		return nil, errors.New("not a directory")
	}
	return f.infos, nil
}

func (m *memFS) Open(path string) (File, error) {
	if path == "/" || m.files[path] == nil && m.isDir(path) {
		var infos []os.FileInfo
		prefix := path
		if prefix != "/" {
			prefix += "/"
		}
		for p, d := range m.files {
			if len(p) > len(prefix) && p[:len(prefix)] == prefix && !bytes.Contains([]byte(p[len(prefix):]), []byte("/")) {
				infos = append(infos, memInfo{name: p[len(prefix):], size: int64(len(d)), dir: d == nil})
			}
		}
		m.open++
		return &memFile{Reader: bytes.NewReader(nil), fs: m, infos: infos}, nil
	}
	d, ok := m.files[path]
	if !ok {
		return nil, fs.ErrNotExist
	}
	m.open++
	return &memFile{Reader: bytes.NewReader(d), fs: m}, nil
}

func (m *memFS) isDir(path string) bool {
	d, ok := m.files[path]
	return ok && d == nil
}

// makeBMP encodes a w x h BMP whose pixel (x, y) has colour c(x, y).
func makeBMP(w, h, bpp int, topDown bool, c func(x, y int) (r, g, b byte)) []byte {
	rowSize := (w*bpp + 31) / 32 * 4
	data := make([]byte, 54+rowSize*h)
	data[0], data[1] = 'B', 'M'
	binary.LittleEndian.PutUint32(data[2:], uint32(len(data)))
	binary.LittleEndian.PutUint32(data[10:], 54)
	binary.LittleEndian.PutUint32(data[14:], 40)
	binary.LittleEndian.PutUint32(data[18:], uint32(w))
	hh := int32(h)
	if topDown {
		hh = -hh
	}
	binary.LittleEndian.PutUint32(data[22:], uint32(hh))
	binary.LittleEndian.PutUint16(data[26:], 1)
	binary.LittleEndian.PutUint16(data[28:], uint16(bpp))
	for y := 0; y < h; y++ {
		fileRow := h - 1 - y
		if topDown {
			fileRow = y
		}
		for x := 0; x < w; x++ {
			r, g, b := c(x, y)
			o := 54 + fileRow*rowSize + x*bpp/8
			data[o], data[o+1], data[o+2] = b, g, r
		}
	}
	return data
}

func pattern(x, y int) (r, g, b byte) { return byte(x * 7), byte(y * 11), byte(x ^ y) }

func rgb565be(r, g, b byte) (byte, byte) {
	v := uint16(r&0xF8)<<8 | uint16(g&0xFC)<<3 | uint16(b)>>3
	return byte(v >> 8), byte(v)
}

func TestBMPReader(t *testing.T) {
	const w, h = 13, 7 // odd width: row padding
	for _, tc := range []struct {
		bpp     int
		topDown bool
	}{{24, false}, {32, true}} {
		br, err := NewBMPReader(bytes.NewReader(makeBMP(w, h, tc.bpp, tc.topDown, pattern)), w, h)
		if err != nil {
			t.Fatal(err)
		}
		got := make([]byte, w*h*2)
		// Read in odd-sized chunks that cross rows.
		for off := 0; off < len(got); off += 18 {
			end := off + 18
			if end > len(got) {
				end = len(got)
			}
			if _, err := br.ReadAt(got[off:end], int64(off)); err != nil {
				t.Fatal(err)
			}
		}
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				hi, lo := rgb565be(pattern(x, y))
				o := (y*w + x) * 2
				if got[o] != hi || got[o+1] != lo {
					t.Fatalf("bpp %d: (%d,%d) = %02x%02x want %02x%02x", tc.bpp, x, y, got[o], got[o+1], hi, lo)
				}
			}
		}
	}
	if _, err := NewBMPReader(bytes.NewReader(makeBMP(4, 4, 24, false, pattern)), 5, 4); err != ErrBMPSize {
		t.Fatalf("size: %v", err)
	}
	if _, err := NewBMPReader(bytes.NewReader([]byte("not a bmp at all, definitely not a bitmap file......")), 5, 4); err != ErrBMPFormat {
		t.Fatalf("format: %v", err)
	}
}

func TestDirSource(t *testing.T) {
	const w, h = 480, 272
	raw := make([]byte, w*h*2)
	for i := range raw {
		raw[i] = byte(i)
	}
	m := &memFS{files: map[string][]byte{
		"/slides":              nil,
		"/slides/b.rgb565":     raw,
		"/slides/a.BMP":        makeBMP(w, h, 24, false, pattern),
		"/slides/small.bmp":    makeBMP(10, 10, 24, false, pattern),
		"/slides/short.rgb565": raw[:100],
		"/slides/._b.rgb565":   raw, // macOS resource fork
		"/slides/notes.txt":    []byte("hello"),
		"/slides/sub":          nil,
		"/other.rgb565":        raw,
	}}
	calls := 0
	src, err := NewDirSource(m, "/slides/", w, h, func() { calls++ })
	if err != nil {
		t.Fatal(err)
	}
	if got := src.Files(); len(got) != 2 || got[0] != "a.BMP" || got[1] != "b.rgb565" {
		t.Fatalf("files %q", got)
	}
	if src.Name(0) != "a" || src.Name(1) != "b" {
		t.Fatalf("names %q %q", src.Name(0), src.Name(1))
	}

	d := memlcd.New(w, h)
	s, err := NewShow(d, &memlcd.Touch{}, src)
	if err != nil {
		t.Fatal(err)
	}
	s.Sleep = func(time.Duration) {}
	s.Log = func(string) {}
	for i, want := range []func(x, y int) (byte, byte){
		func(x, y int) (byte, byte) { return rgb565be(pattern(x, y)) },
		func(x, y int) (byte, byte) { o := (y*w + x) * 2; return raw[o], raw[o+1] },
	} {
		if err := s.Draw(i, WipeDown); err != nil {
			t.Fatal(err)
		}
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				hi, lo := want(x, y)
				if v := d.Pixel(int16(x), int16(y)); v != uint16(hi)<<8|uint16(lo) {
					t.Fatalf("slide %d (%d,%d) = %04x", i, x, y, v)
				}
			}
		}
		if m.open != 1 {
			t.Fatalf("%d files open", m.open)
		}
	}
	if calls == 0 {
		t.Fatal("BeforeIO never called")
	}
}

var _ io.ReaderAt = (*BMPReader)(nil)
