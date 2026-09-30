package slideshow

import (
	"encoding/binary"
	"errors"
	"io"
)

var (
	ErrBMPFormat = errors.New("bmp: not an uncompressed 24/32-bit BMP")
	ErrBMPSize   = errors.New("bmp: size differs from the screen")
)

// BMPReader reads an uncompressed 24- or 32-bit BMP (BI_RGB) of exactly
// the screen size and returns its pixels as big endian RGB565, converting
// one BMP row at a time, so a Source can serve BMP files without loading
// them into RAM.
type BMPReader struct {
	r       io.ReaderAt
	w, h    int
	dataOff int64
	rowSize int64 // bytes per BMP row, padded to 4
	bytesPP int
	topDown bool
	row     []byte
	rowY    int // row cached in row, or -1
}

// NewBMPReader parses the BMP header of r.
//
// Header (little endian): "BM" at 0, pixel data offset at 10, DIB header
// size at 14 (>= 40), width at 18, height at 22 (negative = top-down rows),
// bits per pixel at 28, compression at 30 (0 = BI_RGB).
func NewBMPReader(r io.ReaderAt, width, height int) (*BMPReader, error) {
	var h [54]byte
	if _, err := r.ReadAt(h[:], 0); err != nil {
		return nil, ErrBMPFormat
	}
	if h[0] != 'B' || h[1] != 'M' || binary.LittleEndian.Uint32(h[14:]) < 40 {
		return nil, ErrBMPFormat
	}
	bpp := int(binary.LittleEndian.Uint16(h[28:]))
	if (bpp != 24 && bpp != 32) || binary.LittleEndian.Uint32(h[30:]) != 0 {
		return nil, ErrBMPFormat
	}
	w := int(int32(binary.LittleEndian.Uint32(h[18:])))
	ht := int(int32(binary.LittleEndian.Uint32(h[22:])))
	topDown := ht < 0
	if topDown {
		ht = -ht
	}
	if w != width || ht != height {
		return nil, ErrBMPSize
	}
	b := &BMPReader{
		r:       r,
		w:       w,
		h:       ht,
		dataOff: int64(binary.LittleEndian.Uint32(h[10:])),
		rowSize: int64((w*bpp + 31) / 32 * 4),
		bytesPP: bpp / 8,
		topDown: topDown,
		rowY:    -1,
	}
	b.row = make([]byte, b.rowSize)
	return b, nil
}

// loadRow reads screen row y (top = 0) into b.row.
func (b *BMPReader) loadRow(y int) error {
	if y == b.rowY {
		return nil
	}
	fileRow := b.h - 1 - y // rows are stored bottom-up
	if b.topDown {
		fileRow = y
	}
	if _, err := b.r.ReadAt(b.row, b.dataOff+int64(fileRow)*b.rowSize); err != nil {
		b.rowY = -1
		return err
	}
	b.rowY = y
	return nil
}

// ReadAt returns RGB565BE bytes; off and len(p) must be even.
func (b *BMPReader) ReadAt(p []byte, off int64) (int, error) {
	if off%2 != 0 || len(p)%2 != 0 {
		return 0, errors.New("bmp: odd offset or length")
	}
	total := int64(b.w) * int64(b.h) * 2
	n := 0
	for n < len(p) {
		pos := off + int64(n)
		if pos >= total {
			return n, io.EOF
		}
		px := int(pos / 2)
		y, x := px/b.w, px%b.w
		if err := b.loadRow(y); err != nil {
			return n, err
		}
		for ; x < b.w && n < len(p); x++ {
			s := x * b.bytesPP
			blue, green, red := b.row[s], b.row[s+1], b.row[s+2]
			v := uint16(red&0xF8)<<8 | uint16(green&0xFC)<<3 | uint16(blue)>>3
			p[n] = byte(v >> 8)
			p[n+1] = byte(v)
			n += 2
		}
	}
	return n, nil
}
