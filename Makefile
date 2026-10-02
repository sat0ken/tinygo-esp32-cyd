TARGET    ?= ./targets/esp32-4827s043.json
PKG       ?= ./cmd/app
PORT      ?=
ADDR      ?= :8080
GOLDEN    ?= app_initial
DUMP_MS   ?= 1500
TINYGOROOT := $(shell tinygo env TINYGOROOT)

PORTFLAG := $(if $(PORT),-port $(PORT),)
EXAMPLES := 01_backlight 02_colorbars 03_tinydraw 04_touch 05_slideshow 06_sdslideshow 07_breakout 08_invaders 09_sudoku 10_othello

.PHONY: flash flash-noerase flash-slides monitor build examples wasm serve test test-browser update-golden slides clean

## Board ------------------------------------------------------------------

# make flash                       -> cmd/app
# make flash PKG=./examples/02_colorbars PORT=/dev/ttyUSB0
flash:
	tinygo flash -target=$(TARGET) $(PORTFLAG) -monitor $(PKG)

# Like `make flash` but without erasing the whole flash first (tinygo flash
# erases everything, including the slides written by flash-slides): build the
# program image and write it at 0 with espflasher, which also patches the
# flash size into the header.
# make flash-noerase PKG=./examples/05_slideshow && make monitor
flash-noerase:
	@mkdir -p out
	tinygo build -target=$(TARGET) -o out/app.bin $(PKG)
	go run tinygo.org/x/espflasher@v0.8.1 -port $(or $(PORT),/dev/ttyUSB0) -fs 16MB -offset 0x0 out/app.bin

# Write examples/05_slideshow/slides.pack to its own flash region (0x800000,
# board.SlidesFlashOffset). Needed once, again after `make slides`, and after
# every `make flash` (which erases the whole flash); `make flash-noerase`
# keeps it.
# The ROM loader refuses a program image with ~1MB of embedded data, so the
# slides are not part of the program (see package flashmap).
# -fs 16MB: flash size auto-detection only happens for program images; without
# it the flasher stub limits writes to 4MB and fails with status 0xC4.
SLIDES_OFFSET := 0x800000
flash-slides:
	go run tinygo.org/x/espflasher@v0.8.1 -port $(or $(PORT),/dev/ttyUSB0) -fs 16MB -offset $(SLIDES_OFFSET) examples/05_slideshow/slides.pack

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
	@echo "== cmd/games"; tinygo build -target=$(TARGET) -size short -o /dev/null ./cmd/games

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
	GOLDEN=$(GOLDEN) DUMP_MS=$(DUMP_MS) go test -tags browser -count=1 ./tools/wasmcheck

# Rewrite testdata/golden/*.png after an intended UI change. Check the new
# images before committing them.
update-golden:
	go test ./app ./cmd/games ./examples/05_slideshow ./examples/07_breakout ./games/... -update

# Convert examples/05_slideshow/images/* to examples/05_slideshow/slides.pack.
slides:
	go generate ./examples/05_slideshow

clean:
	rm -rf out web/app.wasm web/wasm_exec.js testdata/golden/*.actual.png
