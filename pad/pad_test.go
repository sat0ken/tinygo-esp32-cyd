package pad

import (
	"testing"

	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/minijoy"
)

const dt = float32(1.0 / 60)

func TestPressedOnceAndRepeat(t *testing.T) {
	f := &Fake{B: hal.PadA}
	var in Input
	in.Update(f, dt) // A was already down: not pressed
	if in.Pressed(hal.PadA) || !in.Held(hal.PadA) || in.Used {
		t.Fatal("a button held from before counted as pressed")
	}
	f.B = 0
	in.Update(f, dt)
	f.B = hal.PadA | hal.PadRight
	in.Update(f, dt)
	if !in.Pressed(hal.PadA) || in.Direction() != hal.PadRight || !in.Used {
		t.Fatal("A and right not pressed")
	}
	// Held for one second: right repeats after 0.35s, then every 0.1s; A
	// does not repeat.
	rights, as := 0, 0
	for i := 0; i < 60; i++ {
		in.Update(f, dt)
		if in.Pressed(hal.PadRight) {
			rights++
		}
		if in.Pressed(hal.PadA) {
			as++
		}
	}
	if as != 0 || rights < 6 || rights > 8 {
		t.Fatalf("in one second: %d repeats of right, %d of A", rights, as)
	}
	// Another direction starts over.
	f.B = hal.PadDown
	in.Update(f, dt)
	if in.Direction() != hal.PadDown {
		t.Fatal("down not pressed")
	}
	in.Update(f, dt)
	if in.Direction() != 0 {
		t.Fatal("down repeated at once")
	}
}

func TestNoPad(t *testing.T) {
	var in Input
	for i := 0; i < 3; i++ {
		in.Update(nil, dt)
	}
	if in.Used || in.Held(^hal.Buttons(0)) {
		t.Fatal("a nil pad pressed something")
	}
}

func TestMove(t *testing.T) {
	// A 3x3 grid of 20x20 boxes and a column of two wide buttons on the
	// right, level with the first two rows.
	var boxes []Box
	for i := 0; i < 9; i++ {
		boxes = append(boxes, Box{int16(i%3) * 20, int16(i/3) * 20, 20, 20})
	}
	boxes = append(boxes, Box{70, 0, 40, 18}, Box{70, 22, 40, 18})
	for _, c := range []struct {
		from int
		dir  hal.Buttons
		want int
	}{
		{4, hal.PadRight, 5}, {4, hal.PadLeft, 3}, {4, hal.PadUp, 1}, {4, hal.PadDown, 7},
		{0, hal.PadLeft, 0}, {0, hal.PadUp, 0}, // nothing there
		{2, hal.PadRight, 9}, {5, hal.PadRight, 10}, {8, hal.PadRight, 10},
		{9, hal.PadLeft, 2}, {10, hal.PadLeft, 5}, {9, hal.PadDown, 10},
		{10, hal.PadDown, 8}, // the nearest box below, though not in line
	} {
		if got := Move(boxes, c.from, c.dir); got != c.want {
			t.Errorf("from %d dir %v: %d, want %d", c.from, c.dir, got, c.want)
		}
	}
}

// module answers like the joystick module: block reads move on to the
// next register.
type module struct {
	regs [0x30]byte
	ptr  byte
}

func (m *module) Tx(addr uint16, w, r []byte) error {
	if len(w) > 0 {
		m.ptr = w[0]
	}
	for i := range r {
		r[i] = m.regs[m.ptr]
		m.ptr++
	}
	return nil
}

func TestJoystick(t *testing.T) {
	m := &module{}
	m.regs[minijoy.RegX], m.regs[minijoy.RegY] = 250, 128 // right
	for r := 0x20; r <= 0x24; r++ {
		m.regs[r] = byte(minijoy.NoPress)
	}
	m.regs[0x21] = byte(minijoy.PressDown)     // A
	m.regs[0x22] = byte(minijoy.LongPressHold) // Y
	m.regs[0x24] = byte(minijoy.SingleClick)   // X: released already
	j := NewJoystick(m)
	if got, want := j.ReadPad(), hal.PadRight|hal.PadA|hal.PadY; got != want {
		t.Fatalf("buttons %b, want %b", got, want)
	}
	m.regs[minijoy.RegX], m.regs[minijoy.RegY] = 100, 20 // a little left: down only
	m.regs[0x21], m.regs[0x22] = byte(minijoy.PressUp), byte(minijoy.NoPress)
	m.regs[0x20] = byte(minijoy.PressDown) // OK
	if got, want := j.ReadPad(), hal.PadDown|hal.PadOK; got != want {
		t.Fatalf("buttons %b, want %b", got, want)
	}
}
