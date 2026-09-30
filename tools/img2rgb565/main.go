// Command img2rgb565 converts PNG/JPEG/GIF images to raw big endian RGB565
// files sized for the panel, to be embedded with go:embed.
//
//	go run ./tools/img2rgb565 [-w 480] [-h 272] [-fit cover|contain] [-dither=true] -o outdir inputs...
//
// Inputs may be files or directories (every .png/.jpg/.jpeg/.gif inside).
// Each input a/b/photo.jpg becomes outdir/photo.rgb565: w*h*2 bytes, no
// header, row-major, 2 bytes per pixel big endian (pixel.RGB565BE, the
// format of DrawRGBBitmap8).
package main

import (
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
)

func main() {
	w := flag.Int("w", 480, "output width")
	h := flag.Int("h", 272, "output height")
	fitName := flag.String("fit", "cover", "cover (crop to fill) or contain (letterbox)")
	dither := flag.Bool("dither", true, "Floyd-Steinberg dithering to RGB565")
	outDir := flag.String("o", ".", "output directory")
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
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		log.Fatal(err)
	}
	for _, in := range inputs {
		img, err := load(in)
		if err != nil {
			log.Fatalf("%s: %v", in, err)
		}
		data := Convert(img, *w, *h, fit, *dither)
		name := strings.TrimSuffix(filepath.Base(in), filepath.Ext(in)) + ".rgb565"
		out := filepath.Join(*outDir, name)
		if err := os.WriteFile(out, data, 0o644); err != nil {
			log.Fatal(err)
		}
		b := img.Bounds()
		fmt.Printf("%s (%dx%d) -> %s (%dx%d, %d bytes)\n", in, b.Dx(), b.Dy(), out, *w, *h, len(data))
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
