// Package goldentest compares screens rendered on the host with the golden
// images in testdata/golden.
//
//	go test ./... -update   # rewrite the golden images (review them before committing)
package goldentest

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden images in testdata/golden")

// Dir returns testdata/golden of the module, found by walking up from the
// test's working directory to go.mod.
func Dir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Join(dir, "testdata", "golden")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

// Check compares img with testdata/golden/<name>.png pixel by pixel. On a
// mismatch the actual image is written to <name>.actual.png (ignored by
// git) for inspection. With -update the golden image is rewritten instead.
func Check(t *testing.T, name string, img *image.RGBA) {
	t.Helper()
	dir := Dir(t)
	path := filepath.Join(dir, name+".png")
	actualPath := filepath.Join(dir, name+".actual.png")
	if *update {
		if err := writePNG(path, img); err != nil {
			t.Fatal(err)
		}
		os.Remove(actualPath)
		t.Logf("updated %s", path)
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run the test with -update to create it)", err)
	}
	want, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if diff := Compare(want, img); diff != "" {
		writePNG(actualPath, img)
		t.Fatalf("%s: %s (actual written to %s)", name, diff, actualPath)
	}
	os.Remove(actualPath)
}

// Compare returns "" if the images are equal, or a description of the
// difference.
func Compare(want image.Image, got *image.RGBA) string {
	if want.Bounds() != got.Bounds() {
		return fmt.Sprintf("size %v, want %v", got.Bounds(), want.Bounds())
	}
	n := 0
	first := ""
	b := want.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			wr, wg, wb, _ := want.At(x, y).RGBA()
			gr, gg, gb, _ := got.At(x, y).RGBA()
			if wr != gr || wg != gg || wb != gb {
				if n == 0 {
					first = fmt.Sprintf("first at (%d,%d): got %02x%02x%02x want %02x%02x%02x",
						x, y, gr>>8, gg>>8, gb>>8, wr>>8, wg>>8, wb>>8)
				}
				n++
			}
		}
	}
	if n == 0 {
		return ""
	}
	return fmt.Sprintf("%d pixels differ, %s", n, first)
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
