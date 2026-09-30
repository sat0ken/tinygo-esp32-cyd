TARGET    ?= ./targets/esp32-4827s043.json
PKG       ?= ./cmd/app
PORT      ?=
ADDR      ?= :8080
TINYGOROOT := $(shell tinygo env TINYGOROOT)

PORTFLAG := $(if $(PORT),-port $(PORT),)
EXAMPLES := 01_backlight 02_colorbars 03_tinydraw 04_touch

.PHONY: flash monitor build examples wasm serve test test-browser update-golden clean

## Board ------------------------------------------------------------------

# make flash                       -> cmd/app
# make flash PKG=./examples/02_colorbars PORT=/dev/ttyUSB0
flash:
	tinygo flash -target=$(TARGET) $(PORTFLAG) -monitor $(PKG)

monitor:
	tinygo monitor $(PORTFLAG) -baudrate 115200

build:
	tinygo build -target=$(TARGET) -size short -o /dev/null $(PKG)

# Build every example and cmd/app for the board (compile check).
examples:
	@for e in $(EXAMPLES); do \
		echo "== $$e"; \
		tinygo build -target=$(TARGET) -size short -o /dev/null ./examples/$$e || exit 1; \
	done
	@echo "== cmd/app"; tinygo build -target=$(TARGET) -size short -o /dev/null ./cmd/app

## Browser ----------------------------------------------------------------

# wasm_exec.js must come from TinyGo (the one in Go's GOROOT is not
# compatible). It is copied on every build so it follows TinyGo updates.
wasm:
	cp $(TINYGOROOT)/targets/wasm_exec.js web/wasm_exec.js
	tinygo build -target=wasm -o web/app.wasm $(PKG)

serve:
	go run ./tools/serve -addr $(ADDR) -dir web

## Host -------------------------------------------------------------------

test:
	go test ./...

# Optional: render web/index.html in headless Chromium and compare the
# canvas with the same golden image (needs chromium or google-chrome).
test-browser: wasm
	go test -tags browser -count=1 ./tools/wasmcheck

# Rewrite testdata/golden/*.png after an intended UI change. Check the new
# images before committing them.
update-golden:
	go test ./app -update

clean:
	rm -f web/app.wasm web/wasm_exec.js testdata/golden/*.actual.png
