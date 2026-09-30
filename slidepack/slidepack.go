// Package slidepack defines a simple container for full-screen RGB565
// images: one file that can be embedded with go:embed or written to a flash
// region and read from memory-mapped flash.
//
// Layout (little endian):
//
//	0x0000  "SLD1"                magic
//	0x0004  uint16 width
//	0x0006  uint16 height
//	0x0008  uint32 count
//	0x000C  uint32 frame offset   (always HeaderSize)
//	0x0010  uint32 frame size     (width * height * 2)
//	0x0014  count x [32]byte name (NUL padded)
//	0x1000  frames, big endian RGB565 (pixel.RGB565BE), row-major
package slidepack

import (
	"encoding/binary"
	"errors"
	"io"
	"strings"
)

const (
	Magic      = "SLD1"
	HeaderSize = 4096
	NameSize   = 32
	MaxSlides  = (HeaderSize - 0x14) / NameSize // 127
)

var (
	ErrMagic  = errors.New("slidepack: bad magic (not a slide pack, or not written to flash)")
	ErrHeader = errors.New("slidepack: bad header")
)

// Pack is an opened slide pack.
type Pack struct {
	r         io.ReaderAt
	Width     int
	Height    int
	Names     []string
	frameOff  int64
	frameSize int64
}

// Open reads the header of a pack.
func Open(r io.ReaderAt) (*Pack, error) {
	var h [HeaderSize]byte
	if _, err := r.ReadAt(h[:], 0); err != nil {
		return nil, err
	}
	if string(h[0:4]) != Magic {
		return nil, ErrMagic
	}
	p := &Pack{
		r:         r,
		Width:     int(binary.LittleEndian.Uint16(h[4:])),
		Height:    int(binary.LittleEndian.Uint16(h[6:])),
		frameOff:  int64(binary.LittleEndian.Uint32(h[12:])),
		frameSize: int64(binary.LittleEndian.Uint32(h[16:])),
	}
	count := int(binary.LittleEndian.Uint32(h[8:]))
	if count <= 0 || count > MaxSlides || p.Width <= 0 || p.Height <= 0 ||
		p.frameSize != int64(p.Width*p.Height*2) || p.frameOff != HeaderSize {
		return nil, ErrHeader
	}
	for i := 0; i < count; i++ {
		b := h[0x14+i*NameSize : 0x14+(i+1)*NameSize]
		p.Names = append(p.Names, strings.TrimRight(string(b), "\x00"))
	}
	return p, nil
}

// Len returns the number of slides.
func (p *Pack) Len() int { return len(p.Names) }

// Size returns the total size of the pack in bytes.
func (p *Pack) Size() int64 { return p.frameOff + int64(len(p.Names))*p.frameSize }

// Slide returns a reader for the pixels of slide i.
func (p *Pack) Slide(i int) *io.SectionReader {
	return io.NewSectionReader(p.r, p.frameOff+int64(i)*p.frameSize, p.frameSize)
}

// Write writes a pack. frames[i] must be width*height*2 bytes.
func Write(w io.Writer, width, height int, names []string, frames [][]byte) error {
	if len(names) != len(frames) || len(frames) == 0 || len(frames) > MaxSlides {
		return ErrHeader
	}
	var h [HeaderSize]byte
	copy(h[0:4], Magic)
	binary.LittleEndian.PutUint16(h[4:], uint16(width))
	binary.LittleEndian.PutUint16(h[6:], uint16(height))
	binary.LittleEndian.PutUint32(h[8:], uint32(len(frames)))
	binary.LittleEndian.PutUint32(h[12:], HeaderSize)
	binary.LittleEndian.PutUint32(h[16:], uint32(width*height*2))
	for i, n := range names {
		if len(n) > NameSize {
			n = n[:NameSize]
		}
		copy(h[0x14+i*NameSize:], n)
	}
	if _, err := w.Write(h[:]); err != nil {
		return err
	}
	for _, f := range frames {
		if len(f) != width*height*2 {
			return ErrHeader
		}
		if _, err := w.Write(f); err != nil {
			return err
		}
	}
	return nil
}
