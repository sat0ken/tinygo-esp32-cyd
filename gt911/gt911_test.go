package gt911

import (
	"errors"
	"testing"

	"github.com/sat0ken/tinygo-cyd/touchcal"
)

// chip simulates the GT911 registers.
type chip struct {
	addr uint16
	regs map[uint16]byte
}

func newChip(addr uint16) *chip {
	c := &chip{addr: addr, regs: map[uint16]byte{}}
	for i, b := range []byte{'9', '1', '1', 0, 0x60, 0x10, 0x40, 0x01, 0xE0, 0x01} {
		c.regs[regProduct+uint16(i)] = b
	}
	c.regs[regConfig] = 0x41
	return c
}

func (c *chip) Tx(addr uint16, w, r []byte) error {
	if addr != c.addr {
		return errors.New("nack")
	}
	reg := uint16(w[0])<<8 | uint16(w[1])
	for i, b := range w[2:] {
		c.regs[reg+uint16(i)] = b
	}
	for i := range r {
		r[i] = c.regs[reg+uint16(i)]
	}
	return nil
}

// floating is a bus with nothing on it that does not report errors: every
// byte reads 0xFF.
type floating struct{}

func (floating) Tx(addr uint16, w, r []byte) error {
	for i := range r {
		r[i] = 0xFF
	}
	return nil
}

// touch puts a measurement in the registers.
func (c *chip) touch(n int, x, y uint16) {
	c.regs[regStatus] = 0x80 | byte(n)
	c.regs[0x814F] = 0
	c.regs[0x8150], c.regs[0x8151] = byte(x), byte(x>>8)
	c.regs[0x8152], c.regs[0x8153] = byte(y), byte(y>>8)
}

func TestProbeFindsEitherAddress(t *testing.T) {
	for _, a := range []uint16{Address, AddressAlt} {
		d := New(newChip(a))
		info, ok := d.Probe()
		if !ok || d.Addr != a {
			t.Fatalf("address %x: ok %v addr %x", a, ok, d.Addr)
		}
		if info.ProductID != "911" || info.Firmware != 0x1060 || info.XRes != 320 || info.YRes != 480 || info.Config != 0x41 {
			t.Errorf("info %+v", info)
		}
	}
	if _, ok := New(newChip(0x33)).Probe(); ok {
		t.Error("found a controller that is not there")
	}
	if _, ok := New(floating{}).Probe(); ok {
		t.Error("took a bus that reads 0xFF for a controller")
	}
}

func TestTouchKeepsStateBetweenMeasurements(t *testing.T) {
	c := newChip(Address)
	d := New(c)
	d.Probe()
	cal := touchcal.Calibration{Width: 480, Height: 320, RawXLeft: 0, RawXRight: 479, RawYTop: 319, RawYBottom: 0, SwapXY: true}
	tt := NewTouch(d, cal)

	if _, _, p := tt.ReadTouch(); p {
		t.Fatal("pressed before any touch")
	}
	c.touch(1, 100, 400) // portrait x 100, y 400
	x, y, p := tt.ReadTouch()
	if !p || x != 400 || y != 219 {
		t.Fatalf("got %d,%d %v", x, y, p)
	}
	if c.regs[regStatus] != 0 {
		t.Error("status not cleared")
	}
	// No new measurement yet: still pressed at the same place.
	if x2, y2, p := tt.ReadTouch(); !p || x2 != x || y2 != y {
		t.Fatalf("lost the touch between measurements: %d,%d %v", x2, y2, p)
	}
	if tt.Raw().X != 100 || tt.Raw().Y != 400 {
		t.Errorf("raw %+v", tt.Raw())
	}
	// Released: a measurement with no points.
	c.touch(0, 0, 0)
	if _, _, p := tt.ReadTouch(); p {
		t.Fatal("still pressed after release")
	}
}
