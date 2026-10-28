package minijoy

import (
	"errors"
	"testing"
)

// module simulates the CH32V003 firmware as the Arduino library expects
// it: a write sets the register pointer, a read returns bytes from it
// (moving on after each byte when autoInc is set).
type module struct {
	regs    [0x30]byte
	ptr     byte
	autoInc bool
	txs     int
}

func (m *module) Tx(addr uint16, w, r []byte) error {
	m.txs++
	if addr != Address {
		return errors.New("nack")
	}
	if len(w) > 0 {
		m.ptr = w[0]
	}
	for i := range r {
		r[i] = m.regs[m.ptr]
		if m.autoInc {
			m.ptr++
		}
	}
	return nil
}

func newModule(autoInc bool) *module {
	m := &module{autoInc: autoInc}
	m.regs[RegX], m.regs[RegY] = 200, 30
	m.regs[0x20] = byte(NoPress)       // OK
	m.regs[0x21] = byte(DoubleClick)   // A
	m.regs[0x22] = byte(PressDown)     // Y
	m.regs[0x23] = byte(LongPressHold) // B
	m.regs[0x24] = byte(SingleClick)   // X
	return m
}

func check(t *testing.T, s State, what string) {
	t.Helper()
	want := State{X: 200, Y: 30, Buttons: [numButtons]Event{
		A: DoubleClick, B: LongPressHold, X: SingleClick, Y: PressDown, OK: NoPress,
	}}
	if s != want {
		t.Errorf("%s: %+v, want %+v", what, s, want)
	}
}

func TestRead(t *testing.T) {
	m := newModule(false)
	d := New(m)
	var s State
	if err := d.Read(&s); err != nil {
		t.Fatal(err)
	}
	check(t, s, "Read")
	if m.txs != 14 {
		t.Errorf("%d transactions, want 14", m.txs)
	}
}

func TestReadBlock(t *testing.T) {
	m := newModule(true)
	d := New(m)
	var s State
	if err := d.ReadBlock(&s); err != nil {
		t.Fatal(err)
	}
	check(t, s, "ReadBlock")
	if m.txs != 4 {
		t.Errorf("%d transactions, want 4", m.txs)
	}
}

func TestNotThere(t *testing.T) {
	d := New(newModule(false))
	d.Address = 0x33
	var s State
	if err := d.Read(&s); err == nil {
		t.Error("no error without a module")
	}
}

func TestDown(t *testing.T) {
	for e := Event(0); e <= NoPress; e++ {
		want := e == PressDown || e == PressRepeat || e == LongPressStart || e == LongPressHold
		if e.Down() != want {
			t.Errorf("%v.Down() = %v", e, e.Down())
		}
	}
}

func TestNames(t *testing.T) {
	if A.String() != "A" || Y.String() != "Y" || OK.String() != "OK" || PressDown.String() != "press down" ||
		NoPress.String() != "none" || Event(200).String() != "?" {
		t.Error("names")
	}
}
