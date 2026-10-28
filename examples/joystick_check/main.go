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
//   - how long one reading of the whole module takes, one register at a
//     time and in blocks.
//
// Then the screen shows the stick position and the event of each button
// (green while minijoy's Event.Down says it is held), and the log prints
// every change with the time in milliseconds.
//
// Whether block reads and repeated-start reads work is checked while the
// module is used: with the stick in the middle and no button down every
// register holds the same value, so a module that does not move to the
// next register after each byte would still look right. Move the stick
// to a corner and press a button; the log says when it is clear.
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
	probe(j)
	s.message("found 0x5A", green)

	var st, last minijoy.State
	last.X, last.Y = 128, 128
	for b := range last.Buttons {
		last.Buttons[b] = 255 // draw everything once
	}
	blockCheck := readCheck{name: "block"}
	restartCheck := readCheck{name: "repeated start"}
	var lastLog time.Time
	t0 := time.Now()
	errors := 0
	for {
		if err := j.Read(&st); err != nil {
			errors++
			if errors%30 == 1 {
				println("read error:", err.Error(), "(", errors, "so far)")
				s.message("read error: "+err.Error(), red)
			}
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if !blockCheck.done() && distinct(st) {
			var b minijoy.State
			err := j.ReadBlock(&b)
			if blockCheck.add(err == nil && same(st, b), b, st) {
				s.message("block reads: "+blockCheck.verdict(), green)
			}
		}
		if !restartCheck.done() && diff(st.X, st.Y) > 40 {
			var r minijoy.State
			var two [2]byte
			err := j.ReadRegistersRestart(minijoy.RegX, two[:])
			r.X, r.Y, r.Buttons = two[0], two[1], st.Buttons
			restartCheck.add(err == nil && same(st, r), r, st)
		}
		for b := minijoy.A; b <= minijoy.OK; b++ {
			if st.Buttons[b] != last.Buttons[b] {
				if last.Buttons[b] != 255 {
					println(time.Since(t0).Milliseconds(), "button", b.String()+":", last.Buttons[b].String(), "->", st.Buttons[b].String(), "(", uint8(st.Buttons[b]), ")")
				}
				s.button(b, st.Buttons[b])
			}
		}
		if st.X != last.X || st.Y != last.Y {
			s.stick(last.X, last.Y, st.X, st.Y)
			if time.Since(lastLog) > 200*time.Millisecond && (diff(st.X, last.X) > 3 || diff(st.Y, last.Y) > 3) {
				println(time.Since(t0).Milliseconds(), "stick x", st.X, "y", st.Y)
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

// probe dumps the registers and times a whole reading.
func probe(j *minijoy.Device) {
	var one [1]byte
	println("registers, one at a time (as the Arduino library):")
	for _, r := range []byte{0x10, 0x11, 0x12, 0x13, 0x20, 0x21, 0x22, 0x23, 0x24} {
		if err := j.ReadRegisters(r, one[:]); err != nil {
			println("  reg", "0x"+strconv.FormatUint(uint64(r), 16), "error", err.Error())
			continue
		}
		println("  reg", "0x"+strconv.FormatUint(uint64(r), 16), "=", one[0])
	}

	for _, block := range []bool{false, true} {
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
	println("block and repeated-start reads are checked when the stick is off the diagonal or a button is down")
}

// distinct reports whether the registers hold different values, so that a
// read that stays on the first register would give a different result.
func distinct(s minijoy.State) bool {
	if diff(s.X, s.Y) > 40 {
		return true
	}
	for _, e := range s.Buttons {
		if e != s.Buttons[0] {
			return true
		}
	}
	return false
}

// same compares two readings taken a few milliseconds apart (the stick may
// move a little between them).
func same(a, b minijoy.State) bool {
	return diff(a.X, b.X) <= 10 && diff(a.Y, b.Y) <= 10 && a.Buttons == b.Buttons
}

// readCheck compares another way of reading with single reads until the
// answer is clear: 5 times the same, or 3 times different.
type readCheck struct {
	name    string
	ok, bad int
}

func (c *readCheck) done() bool { return c.ok >= 5 || c.bad >= 3 }

func (c *readCheck) verdict() string {
	if c.ok >= 5 {
		return "work"
	}
	return "do NOT work"
}

// add records one comparison and reports whether it settled the answer.
func (c *readCheck) add(same bool, got, want minijoy.State) bool {
	if same {
		c.ok++
	} else {
		c.bad++
		println(c.name, "read differs:", text(got))
		println("  single reads:", text(want))
	}
	if c.done() {
		println(c.name, "reads", c.verdict(), "(", c.ok, "same,", c.bad, "different )")
		return true
	}
	return false
}

func text(s minijoy.State) string {
	t := "x " + strconv.Itoa(int(s.X)) + " y " + strconv.Itoa(int(s.Y))
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

// pos is the dot's position for a reading: Y grows upwards on the module.
func (s *screen) pos(x, y uint8) (int16, int16) {
	return s.bx + int16(x)*(s.bs-1)/255, s.by + int16(255-y)*(s.bs-1)/255
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

func (s *screen) button(b minijoy.Button, e minijoy.Event) {
	d := s.d
	y := 76 + int16(b)*24
	d.FillRectangle(s.tx, y, s.w-s.tx, 22, bg)
	c := box
	if e.Down() {
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
