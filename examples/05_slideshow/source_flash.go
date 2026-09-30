//go:build esp32s3

package main

import (
	"bytes"
	"errors"

	"github.com/sat0ken/tinygo-cyd/board"
	"github.com/sat0ken/tinygo-cyd/flashmap"
	"github.com/sat0ken/tinygo-cyd/slidepack"
)

// `tinygo flash` (make flash) erases the whole flash, slides included.
const packHelp = "run make flash-slides (make flash erases them; use make flash-noerase)"

// openPack maps the flash region written by `make flash-slides`: first the
// header, then, once its size is known, the whole pack.
func openPack() (*slidepack.Pack, error) {
	hdr, err := flashmap.Map(board.SlidesFlashOffset, slidepack.HeaderSize)
	if err != nil {
		return nil, err
	}
	println("slides: flash", hex(board.SlidesFlashOffset), "header", hex(uint32(hdr[0])<<24|uint32(hdr[1])<<16|uint32(hdr[2])<<8|uint32(hdr[3])))
	p, err := slidepack.Open(bytes.NewReader(hdr))
	if err != nil {
		return nil, err
	}
	if p.Size() > board.SlidesFlashSize {
		return nil, errors.New("slides: pack larger than the flash region")
	}
	all, err := flashmap.Map(board.SlidesFlashOffset, uint32(p.Size()))
	if err != nil {
		return nil, err
	}
	return slidepack.Open(bytes.NewReader(all))
}

func hex(v uint32) string {
	const digits = "0123456789abcdef"
	var b [10]byte
	b[0], b[1] = '0', 'x'
	for i := 9; i >= 2; i-- {
		b[i] = digits[v&0xF]
		v >>= 4
	}
	return string(b[:])
}
