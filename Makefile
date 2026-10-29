TARGET    ?= ./targets/esp32-4827s043.json
PKG       ?= ./cmd/app
PORT      ?=
ADDR      ?= :8080
GOLDEN    ?= app_initial
DUMP_MS   ?= 1500
TINYGOROOT := $(shell tinygo env TINYGOROOT)

PORTFLAG := $(if $(PORT),-port $(PORT),)
EXAMPLES := 01_backlight 02_colorbars 03_tinydraw 04_touch 05_slideshow 06_sdslideshow 07_breakout 08_invaders 09_sudoku 10_othello 11_solitaire 12_poker

# ESP32-3248S035C (3.5" ST7796 SPI panel, GT911 touch) and the programs that
# run on it (the others use the RGB panel, PWM or the SD bus of the S043).
S035      := ./targets/esp32-3248s035c.json
S035_PKGS := ./examples/s035_check ./examples/03_tinydraw ./examples/07_breakout \
	./examples/08_invaders ./examples/09_sudoku ./examples/10_othello \
	./examples/11_solitaire ./examples/12_poker ./cmd/app ./cmd/games \
	./examples/joystick_check ./examples/13_kart ./examples/14_starguard \
	./examples/15_tetris ./examples/16_maze ./examples/17_fifteen ./examples/18_life \
	./examples/19_tvtennis ./examples/20_atoms ./examples/21_chess ./examples/22_eggs \
	./examples/23_train ./examples/24_picoinvaders ./examples/25_pacman ./examples/26_sokoban \
	./examples/27_vegaslot ./examples/28_raptor ./examples/29_ants ./examples/30_ghostrac \
	./examples/31_fifteen2 ./examples/32_pictor

# ESP32-2432S028R (2.8" ILI9341 320x240, XPT2046 touch): only the programs
# that adapt to the screen size (the games need 480x272).
S028      := ./targets/esp32-2432s028r.json
S028_PKGS := ./examples/s028_check ./examples/08_invaders ./examples/09_sudoku \
	./examples/10_othello ./examples/11_solitaire ./examples/12_poker ./cmd/app \
	./cmd/games ./examples/joystick_check ./examples/13_kart ./examples/14_starguard \
	./examples/15_tetris ./examples/16_maze ./examples/17_fifteen ./examples/18_life \
	./examples/19_tvtennis ./examples/20_atoms ./examples/21_chess ./examples/22_eggs \
	./examples/23_train ./examples/24_picoinvaders ./examples/25_pacman ./examples/26_sokoban \
	./examples/27_vegaslot ./examples/28_raptor ./examples/29_ants ./examples/30_ghostrac \
	./examples/31_fifteen2 ./examples/32_pictor

.PHONY: flash flash-noerase flash-slides flash-s035 flash-s028 monitor build examples examples-s035 examples-s028 wasm serve test test-browser update-golden slides clean

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

## ESP32-3248S035C ---------------------------------------------------------

# make flash-s035 PKG=./examples/s035_check
# make flash-s035 PKG=./cmd/games PORT=/dev/ttyUSB0
flash-s035:
	tinygo flash -target=$(S035) $(PORTFLAG) -monitor $(PKG)

# Build every program that runs on the ESP32-3248S035C (compile check).
examples-s035:
	@$(call esp32-check,$(S035),$(S035_PKGS))

# esp32-check builds every program in $(2) for the ESP32 target $(1), and
# checks that machine._UART0 is initialised at compile time (in .data, "d"):
# if a package initializer is left for run time, TinyGo may leave UART0's
# for run time too, and the ESP32 then hangs at boot because the runtime
# uses the UART before running package initializers (see
# docs/ARCHITECTURE.md, chapter 15).
define esp32-check
mkdir -p out; \
for p in $(2); do \
	echo "== $$p"; \
	tinygo build -target=$(1) -size short -o out/esp32-check.elf $$p || exit 1; \
	go tool nm out/esp32-check.elf | grep ' d machine._UART0$$' >/dev/null || \
		{ echo "$$p: machine._UART0 is not initialised at compile time: it would hang at boot"; exit 1; }; \
done
endef

## ESP32-2432S028R ---------------------------------------------------------

# make flash-s028 PKG=./examples/s028_check
# make flash-s028 PKG=./cmd/app PORT=/dev/ttyUSB0
# TAGS=e32r28t selects the E32R28T wiring in examples/joystick_check.
flash-s028:
	tinygo flash -target=$(S028) $(if $(TAGS),-tags $(TAGS)) $(PORTFLAG) -monitor $(PKG)

# Build every program that runs on the ESP32-2432S028R (compile check).
examples-s028:
	@$(call esp32-check,$(S028),$(S028_PKGS))

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
