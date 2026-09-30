package slideshow

import (
	"io"
	"os"
	"sort"
	"strings"
)

// FS is the part of a file system DirSource uses. tinyfs/fatfs fits with a
// one-line adapter (its Open returns tinyfs.File).
type FS interface {
	Open(path string) (File, error)
}

// File is an open file or directory.
type File interface {
	io.Reader
	io.Seeker
	io.Closer
	Readdir(n int) ([]os.FileInfo, error)
}

// DirSource serves the images in one directory, in name order:
//   - *.rgb565: raw big endian RGB565 of the screen size (tools/img2rgb565 -o)
//   - *.bmp:    uncompressed 24/32-bit BMP of the screen size
//
// Other files, files of the wrong size and hidden files (names starting
// with ".", e.g. the "._name" files macOS writes to FAT cards) are skipped.
// Only one file is open at a time.
type DirSource struct {
	fs    FS
	dir   string
	w, h  int
	files []string
	cur   File

	// BeforeIO, if set, is called before every access to the file system,
	// e.g. to take a shared SPI bus.
	BeforeIO func()
}

// NewDirSource lists dir. beforeIO may be nil (see DirSource.BeforeIO).
func NewDirSource(fs FS, dir string, width, height int, beforeIO func()) (*DirSource, error) {
	s := &DirSource{fs: fs, dir: strings.TrimSuffix(dir, "/"), w: width, h: height, BeforeIO: beforeIO}
	return s, s.scan()
}

func (s *DirSource) io() {
	if s.BeforeIO != nil {
		s.BeforeIO()
	}
}

func (s *DirSource) path(name string) string { return s.dir + "/" + name }

func (s *DirSource) scan() error {
	s.io()
	d, err := s.fs.Open(s.dirPath())
	if err != nil {
		return err
	}
	infos, err := d.Readdir(0)
	d.Close()
	if err != nil {
		return err
	}
	var names []string
	for _, fi := range infos {
		name := fi.Name()
		if fi.IsDir() || strings.HasPrefix(name, ".") {
			continue
		}
		switch strings.ToLower(ext(name)) {
		case ".rgb565":
			if fi.Size() == int64(s.w*s.h*2) {
				names = append(names, name)
			}
		case ".bmp":
			if s.checkBMP(name) {
				names = append(names, name)
			}
		}
	}
	sort.Strings(names)
	s.files = names
	return nil
}

func (s *DirSource) dirPath() string {
	if s.dir == "" {
		return "/"
	}
	return s.dir
}

func (s *DirSource) checkBMP(name string) bool {
	f, err := s.fs.Open(s.path(name))
	if err != nil {
		return false
	}
	defer f.Close()
	_, err = NewBMPReader(&seekReaderAt{f: f}, s.w, s.h)
	return err == nil
}

func ext(name string) string {
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		return name[i:]
	}
	return ""
}

// Len returns the number of usable images.
func (s *DirSource) Len() int { return len(s.files) }

// Name returns the file name of slide i without the extension.
func (s *DirSource) Name(i int) string {
	n := s.files[i]
	return strings.TrimSuffix(n, ext(n))
}

// Files returns the file names found.
func (s *DirSource) Files() []string { return s.files }

// Open closes the previous file and opens slide i.
func (s *DirSource) Open(i int) (io.ReaderAt, error) {
	s.io()
	if s.cur != nil {
		s.cur.Close()
		s.cur = nil
	}
	f, err := s.fs.Open(s.path(s.files[i]))
	if err != nil {
		return nil, err
	}
	s.cur = f
	r := &seekReaderAt{f: f, before: s.BeforeIO}
	if strings.ToLower(ext(s.files[i])) == ".bmp" {
		return NewBMPReader(r, s.w, s.h)
	}
	return r, nil
}

// seekReaderAt implements io.ReaderAt with Seek and Read, for files that
// have no ReadAt (tinyfs/fatfs).
type seekReaderAt struct {
	f      File
	before func()
}

func (r *seekReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if r.before != nil {
		r.before()
	}
	if _, err := r.f.Seek(off, io.SeekStart); err != nil {
		return 0, err
	}
	return io.ReadFull(r.f, p)
}
