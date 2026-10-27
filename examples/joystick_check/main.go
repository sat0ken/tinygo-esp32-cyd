//go:build esp32_2432s028r || esp32_3248s035c

// joystick_check checks the mini joystick module (package minijoy) on an
// ESP32 CYD board. Power the module with 3.3V, not 5V: its I2C lines are
// pulled up to its supply. The wiring for each board is in pins_*.go.
//
// At start the log shows:
//
//   - the I2C devices found (the module is 0x5A);
//   - every register read one at a time, as the module's Arduino library
//     does;
//   - whether block reads and repeated-start reads give the same values
//     (keep the stick in the middle and the buttons up while it starts);
//   - how long one reading of the whole module takes.
//
// Then the screen shows the stick position and the event of each button,
// and the log prints every change. Which events mean "held down" is not
// documented: press, hold and release each button once and look at the
// log.
//
//	make flash-s028 PKG=./examples/joystick_check                # ESP32-2432S028R
//	make flash-s028 PKG=./examples/joystick_check TAGS=e32r28t   # E32R28T
//	make flash-s035 PKG=./examples/joystick_check                # ESP32-3248S035C
package main

import (
	"image/color"
	"machine"
	"strconv"
	"time"

	"github.com/sat0ken/tinygo-cyd/minijoy"
	"github.com/sat0ken/tinygo-cyd/platform"
	"github.com/sat0ken/tinygo-cyd/romfont/freesans"
	"github.com/sat0ken/tinygo-cyd/romfont/proggy"
	"github.com/sat0ken/tinygo-cyd/spilcd"
	"tinygo.org/x/tinyfont"
)

var (
	bg     = color.RGBA{16, 24, 48, 255}
	box    = color.RGBA{32, 40, 72, 255}
	line   = color.RGBA{80, 90, 130, 255}
	white  = color.RGBA{255, 255, 255, 255}
	dim    = color.RGBA{150, 160, 180, 255}
	yellow = color.RGBA{255, 220, 60, 255}
	green  = color.RGBA{60, 220, 90, 255}
	red    = color.RGBA{230, 60, 60, 255}
)

func main() {
	time.Sleep(time.Second)
	println("\n=== joystick_check ===")
	d, err := platform.InitScreen()
	if err != nil {
		println("InitScreen:", err.Error())
		platform.Halt()
	}
	println("I2C SDA", sdaPin, "SCL", sclPin, "100kHz,", wiring)
	println("power the module with 3.3V, not 5V")
	if err := bus.Configure(machine.I2CConfig{SDA: sdaPin, SCL: sclPin, Frequency: 100_000}); err != nil {
		println("I2C:", err.Error())
	}

	s := newScreen(d)
	s.drawAll()
	j := minijoy.New(bus)
	for !found(j) {
		s.message("module not found at 0x5A - wiring? 3.3V?", red)
		time.Sleep(2 * time.Second)
	}
	block := probe(j)
	s.message("found 0x5A, "+mode(block)+" reads", green)

	var st, last minijoy.State
	last.X, last.Y = 128, 128
	for b := range last.Buttons {
		last.Buttons[b] = 255 // draw everything once
	}
	read := j.Read
	if block {
		read = j.ReadBlock
	}
	var lastLog time.Time
	errors := 0
	for {
		if err := read(&st); err != nil {
			errors++
			if errors%30 == 1 {
				println("read error:", err.Error(), "(", errors, "so far)")
				s.message("read error: "+err.Error(), red)
			}
			time.Sleep(100 * time.Millisecond)
			continue
		}
		for b := minijoy.A; b <= minijoy.OK; b++ {
			if st.Buttons[b] != last.Buttons[b] {
				if last.Buttons[b] != 255 {
					println("button", b.String()+":", last.Buttons[b].String(), "->", st.Buttons[b].String(), "(", uint8(st.Buttons[b]), ")")
				}
				s.button(b, st.Buttons[b])
			}
		}
		if st.X != last.X || st.Y != last.Y {
			s.stick(last.X, last.Y, st.X, st.Y)
			if time.Since(lastLog) > 200*time.Millisecond && (diff(st.X, last.X) > 3 || diff(st.Y, last.Y) > 3) {
				println("stick x", st.X, "y", st.Y)
				lastLog = time.Now()
			}
		}
		last = st
		d.Display()
		time.Sleep(30 * time.Millisecond)
	}
}

func diff(a, b uint8) uint8 {
	if a > b {
		return a - b
	}
	return b - a
}

func mode(block bool) string {
	if block {
		return "block"
	}
	return "single"
}

// found scans the bus and reports whether the module answers.
func found(j *minijoy.Device) bool {
	println("scanning I2C 0x08..0x77")
	ok := false
	for a := uint16(0x08); a < 0x78; a++ {
		if bus.CheckDevice(a) {
			println("  found", "0x"+strconv.FormatUint(uint64(a), 16))
			if a == j.Address {
				ok = true
			}
		}
	}
	if !ok {
		println("  no module at 0x5a")
	}
	return ok
}

// probe dumps the registers, checks block and repeated-start reads against
// single reads, and times a whole reading. It returns whether block reads
// can be used.
func probe(j *minijoy.Device) bool {
	var one [1]byte
	println("registers, one at a time (as the Arduino library):")
	for _, r := range []byte{0x10, 0x11, 0x12, 0x13, 0x20, 0x21, 0x22, 0x23, 0x24} {
		if err := j.ReadRegisters(r, one[:]); err != nil {
			println("  reg", "0x"+strconv.FormatUint(uint64(r), 16), "error", err.Error())
			continue
		}
		println("  reg", "0x"+strconv.FormatUint(uint64(r), 16), "=", one[0])
	}

	var single, blk minijoy.State
	errS := j.Read(&single)
	errB := j.ReadBlock(&blk)
	blockOK := errS == nil && errB == nil && same(single, blk)
	println("single read: x", single.X, "y", single.Y, "buttons", events(single))
	println("block read:  x", blk.X, "y", blk.Y, "buttons", events(blk), "->", okText(blockOK))

	var two [2]byte
	errR := j.ReadRegistersRestart(minijoy.RegX, two[:])
	println("repeated start read of x,y:", two[0], two[1], "->",
		okText(errR == nil && diff(two[0], single.X) <= 3 && diff(two[1], single.Y) <= 3))

	for _, block := range []bool{false, true} {
		if block && !blockOK {
			continue
		}
		var s minijoy.State
		start := time.Now()
		for i := 0; i < 20; i++ {
			if block {
				j.ReadBlock(&s)
			} else {
				j.Read(&s)
			}
		}
		println(mode(block), "read of the whole module:", time.Since(start).Microseconds()/20, "us")
	}
	return blockOK
}

// same compares two readings taken one after the other (the stick may move
// a little between them).
func same(a, b minijoy.State) bool {
	return diff(a.X, b.X) <= 3 && diff(a.Y, b.Y) <= 3 && a.Buttons == b.Buttons
}

func okText(ok bool) string {
	if ok {
		return "same as single reads"
	}
	return "DIFFERENT (or error): not usable"
}

func events(s minijoy.State) string {
	t := ""
	for b := minijoy.A; b <= minijoy.OK; b++ {
		t += " " + b.String() + "=" + strconv.Itoa(int(s.Buttons[b]))
	}
	return t
}

// ---- screen ----

// screen draws the stick in a square on the left and the buttons on the
// right; it adapts to 320x240 and 480x320.
type screen struct {
	d          *spilcd.Device
	w, h       int16
	bx, by, bs int16 // stick box
	tx         int16 // left of the text column
}

func newScreen(d *spilcd.Device) *screen {
	w, h := d.Size()
	bs := h - 70
	if bs > 200 {
		bs = 200
	}
	return &screen{d: d, w: w, h: h, bx: 10, by: 32, bs: bs, tx: 10 + bs + 16}
}

func (s *screen) drawAll() {
	d := s.d
	d.FillRectangle(0, 0, s.w, s.h, bg)
	tinyfont.WriteLine(d, &freesans.Bold9pt7b, 8, 20, "mini joystick (I2C 0x5A)", white)
	tinyfont.WriteLine(d, &proggy.TinySZ8pt7b, 8, s.h-22, wiring, dim)
	d.FillRectangle(s.bx, s.by, s.bs, s.bs, box)
	s.cross()
	d.Display()
}

// cross draws the centre lines of the stick box.
func (s *screen) cross() {
	s.d.FillRectangle(s.bx, s.by+s.bs/2, s.bs, 1, line)
	s.d.FillRectangle(s.bx+s.bs/2, s.by, 1, s.bs, line)
}

func (s *screen) pos(x, y uint8) (int16, int16) {
	return s.bx + int16(x)*(s.bs-1)/255, s.by + int16(y)*(s.bs-1)/255
}

// stick moves the dot and rewrites the numbers.
func (s *screen) stick(ox, oy, x, y uint8) {
	d := s.d
	px, py := s.pos(ox, oy)
	s.fillInBox(px-3, py-3, 7, 7, box)
	s.cross()
	px, py = s.pos(x, y)
	s.fillInBox(px-3, py-3, 7, 7, yellow)
	d.FillRectangle(s.tx, 40, s.w-s.tx, 40, bg)
	tinyfont.WriteLine(d, &freesans.Bold9pt7b, s.tx, 56, "X "+strconv.Itoa(int(x)), white)
	tinyfont.WriteLine(d, &freesans.Bold9pt7b, s.tx+70, 56, "Y "+strconv.Itoa(int(y)), white)
}

// fillInBox fills a rectangle clipped to the stick box.
func (s *screen) fillInBox(x, y, w, h int16, c color.RGBA) {
	if x < s.bx {
		w -= s.bx - x
		x = s.bx
	}
	if y < s.by {
		h -= s.by - y
		y = s.by
	}
	if x+w > s.bx+s.bs {
		w = s.bx + s.bs - x
	}
	if y+h > s.by+s.bs {
		h = s.by + s.bs - y
	}
	if w > 0 && h > 0 {
		s.d.FillRectangle(x, y, w, h, c)
	}
}

// held guesses which events mean that the button is down (not documented;
// the log shows what the module really reports).
func held(e minijoy.Event) bool {
	switch e {
	case minijoy.PressDown, minijoy.PressRepeat, minijoy.LongPressStart, minijoy.LongPressHold:
		return true
	}
	return false
}

func (s *screen) button(b minijoy.Button, e minijoy.Event) {
	d := s.d
	y := 76 + int16(b)*24
	d.FillRectangle(s.tx, y, s.w-s.tx, 22, bg)
	c := box
	if held(e) {
		c = green
	}
	d.FillRectangle(s.tx, y+2, 18, 18, c)
	tinyfont.WriteLine(d, &freesans.Bold9pt7b, s.tx+24, y+16, b.String(), white)
	tinyfont.WriteLine(d, &proggy.TinySZ8pt7b, s.tx+44, y+15, short(e)+" ("+strconv.Itoa(int(e))+")", dim)
}

// short names of the events, to fit next to the buttons on 320x240.
func short(e minijoy.Event) string {
	switch e {
	case minijoy.PressDown:
		return "down"
	case minijoy.PressUp:
		return "up"
	case minijoy.PressRepeat:
		return "repeat"
	case minijoy.SingleClick:
		return "click"
	case minijoy.DoubleClick:
		return "double"
	case minijoy.LongPressStart:
		return "long"
	case minijoy.LongPressHold:
		return "hold"
	case minijoy.NoPress:
		return "none"
	}
	return "?"
}

func (s *screen) message(text string, c color.RGBA) {
	s.d.FillRectangle(0, s.h-14, s.w, 14, bg)
	tinyfont.WriteLine(s.d, &proggy.TinySZ8pt7b, 8, s.h-3, text, c)
	s.d.Display()
}
