package slideshow

import (
	"io"

	"github.com/sat0ken/tinygo-cyd/slidepack"
)

// Source provides the slides: full-screen images as big endian RGB565
// (pixel.RGB565BE), row-major, width*height*2 bytes.
type Source interface {
	Len() int
	Name(i int) string
	// Open returns the pixels of slide i. The reader must stay valid until
	// the next call to Open (Show reads the slide on screen again to remove
	// the caption), so a source may keep only one file open.
	Open(i int) (io.ReaderAt, error)
}

// PackSource serves the slides of a slidepack.
type PackSource struct{ Pack *slidepack.Pack }

func (p PackSource) Len() int                        { return p.Pack.Len() }
func (p PackSource) Name(i int) string               { return p.Pack.Names[i] }
func (p PackSource) Open(i int) (io.ReaderAt, error) { return p.Pack.Slide(i), nil }
