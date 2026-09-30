// Command img2rgb565 converts PNG/JPEG/GIF images to big endian RGB565
// sized for the panel.
//
//	go run ./tools/img2rgb565 [-w 480] [-h 272] [-fit cover|contain] [-dither=true] -pack out.pack inputs...
//	go run ./tools/img2rgb565 [...] -o outdir inputs...
//
// Inputs may be files or directories (every .png/.jpg/.jpeg/.gif inside,
// in name order).
//
// With -pack, all images go into one slidepack file (see package
// slidepack), which examples/05_slideshow embeds (browser) or reads from
// flash (board). With -o, each input a/b/photo.jpg becomes
// outdir/photo.rgb565: w*h*2 bytes, no header, row-major, 2 bytes per pixel
// big endian (pixel.RGB565BE, the format of DrawRGBBitmap8).
package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sat0ken/tinygo-cyd/slidepack"
)

func main() {
	w := flag.Int("w", 480, "output width")
	h := flag.Int("h", 272, "output height")
	fitName := flag.String("fit", "cover", "cover (crop to fill) or contain (letterbox)")
	dither := flag.Bool("dither", true, "Floyd-Steinberg dithering to RGB565")
	outDir := flag.String("o", "", "output directory for one .rgb565 file per image")
	packFile := flag.String("pack", "", "write all images into this slidepack file")
	flag.Parse()

	var fit Fit
	switch *fitName {
	case "cover":
		fit = FitCover
	case "contain":
		fit = FitContain
	default:
		log.Fatalf("unknown -fit %q", *fitName)
	}

	inputs, err := expand(flag.Args())
	if err != nil {
		log.Fatal(err)
	}
	if len(inputs) == 0 {
		log.Fatal("no input images")
	}
	if (*outDir == "") == (*packFile == "") {
		log.Fatal("give exactly one of -o and -pack")
	}
	if *outDir != "" {
		if err := os.MkdirAll(*outDir, 0o755); err != nil {
			log.Fatal(err)
		}
	}
	var names []string
	var frames [][]byte
	for _, in := range inputs {
		img, err := load(in)
		if err != nil {
			log.Fatalf("%s: %v", in, err)
		}
		data := Convert(img, *w, *h, fit, *dither)
		base := strings.TrimSuffix(filepath.Base(in), filepath.Ext(in))
		b := img.Bounds()
		if *packFile != "" {
			names = append(names, base)
			frames = append(frames, data)
			fmt.Printf("%s (%dx%d) -> %dx%d\n", in, b.Dx(), b.Dy(), *w, *h)
			continue
		}
		out := filepath.Join(*outDir, base+".rgb565")
		if err := os.WriteFile(out, data, 0o644); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s (%dx%d) -> %s (%dx%d, %d bytes)\n", in, b.Dx(), b.Dy(), out, *w, *h, len(data))
	}
	if *packFile != "" {
		var buf bytes.Buffer
		if err := slidepack.Write(&buf, *w, *h, names, frames); err != nil {
			log.Fatal(err)
		}
		if err := os.WriteFile(*packFile, buf.Bytes(), 0o644); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s: %d slides, %d bytes\n", *packFile, len(frames), buf.Len())
	}
}

// expand replaces directories with the images they contain, sorted by name.
func expand(args []string) ([]string, error) {
	var files []string
	for _, a := range args {
		st, err := os.Stat(a)
		if err != nil {
			return nil, err
		}
		if !st.IsDir() {
			files = append(files, a)
			continue
		}
		entries, err := os.ReadDir(a)
		if err != nil {
			return nil, err
		}
		var found []string
		for _, e := range entries {
			switch strings.ToLower(filepath.Ext(e.Name())) {
			case ".png", ".jpg", ".jpeg", ".gif":
				found = append(found, filepath.Join(a, e.Name()))
			}
		}
		sort.Strings(found)
		files = append(files, found...)
	}
	return files, nil
}

func load(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}
