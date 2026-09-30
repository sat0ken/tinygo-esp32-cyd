//go:build !esp32s3

package main

import (
	_ "embed"
	"strings"

	"github.com/sat0ken/tinygo-cyd/slidepack"
)

//go:embed slides.pack
var packData string

const packHelp = "slides.pack is embedded; run `make slides` to rebuild it"

func openPack() (*slidepack.Pack, error) {
	return slidepack.Open(strings.NewReader(packData))
}
