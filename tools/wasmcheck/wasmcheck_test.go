//go:build browser

// Package wasmcheck opens web/index.html in headless Chromium and compares
// the canvas with testdata/golden/app_initial.png, so the wasm build is
// checked against the same golden image as the host test.
//
//	make test-browser   # builds web/app.wasm first
package wasmcheck

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/png"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"regexp"
	"testing"
	"time"
)

func browser(t *testing.T) string {
	if b := os.Getenv("CHROME"); b != "" {
		return b
	}
	for _, name := range []string{"chromium", "chromium-browser", "google-chrome-stable", "google-chrome"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	t.Skip("no Chromium/Chrome found (set CHROME)")
	return ""
}

var dumpRe = regexp.MustCompile(`<pre id="dump">data:image/png;base64,([A-Za-z0-9+/=]+)</pre>`)

func TestCanvasMatchesGolden(t *testing.T) {
	bin := browser(t)
	if _, err := os.Stat("../../web/app.wasm"); err != nil {
		t.Fatal("web/app.wasm missing: run `make wasm` first")
	}
	mime.AddExtensionType(".wasm", "application/wasm")
	srv := httptest.NewServer(http.FileServer(http.Dir("../../web")))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, "--headless=new", "--disable-gpu", "--no-sandbox",
		"--virtual-time-budget=10000", "--dump-dom", srv.URL+"/index.html?dump").Output()
	if err != nil {
		t.Fatalf("%s: %v", bin, err)
	}
	m := dumpRe.FindSubmatch(out)
	if m == nil {
		t.Fatalf("no canvas dump in the page:\n%s", out)
	}
	data, err := base64.StdEncoding.DecodeString(string(m[1]))
	if err != nil {
		t.Fatal(err)
	}
	got, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.ReadFile("../../testdata/golden/app_initial.png")
	if err != nil {
		t.Fatal(err)
	}
	want, err := png.Decode(bytes.NewReader(f))
	if err != nil {
		t.Fatal(err)
	}
	if n := diff(want, got); n != 0 {
		os.WriteFile("../../testdata/golden/app_initial.wasm.actual.png", data, 0o644)
		t.Fatalf("%d pixels differ (canvas written to testdata/golden/app_initial.wasm.actual.png)", n)
	}
}

func diff(a, b image.Image) int {
	if a.Bounds() != b.Bounds() {
		return a.Bounds().Dx() * a.Bounds().Dy()
	}
	n := 0
	r := a.Bounds()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			ar, ag, ab, _ := a.At(x, y).RGBA()
			br, bg, bb, _ := b.At(x, y).RGBA()
			if ar != br || ag != bg || ab != bb {
				n++
			}
		}
	}
	return n
}
