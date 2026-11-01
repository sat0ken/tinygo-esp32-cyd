//go:build esp32s3 || esp32_2432s028r

package xpttouch

import (
	"machine"
	"runtime/volatile"

	"tinygo.org/x/drivers/xpt2046"
)

// Touch is an XPT2046 touch panel returning screen coordinates.
type Touch struct {
	dev  xpt2046.Device
	poll *poller // instead of dev, when the IRQ pin is not used
	Cal  Calibration
}

// New configures the XPT2046 on the given pins (bit-banged SPI, as done by
// the tinygo drivers xpt2046 package).
func New(sck, cs, mosi, miso, irq machine.Pin, cal Calibration) *Touch {
	t := &Touch{dev: xpt2046.New(sck, cs, mosi, miso, irq), Cal: cal}
	t.dev.Configure(&xpt2046.Config{Precision: 10})
	return t
}

// NewPolled configures the XPT2046 without its IRQ pin, for a board that
// uses that pin for something else: the controller's PENIRQ output is
// turned off between reads and the pressure tells whether the panel is
// pressed. The readings are the same as New's.
func NewPolled(sck, cs, mosi, miso machine.Pin, cal Calibration) *Touch {
	p := &poller{clk: sck, cs: cs, din: mosi, dout: miso}
	sck.Configure(machine.PinConfig{Mode: machine.PinOutput})
	cs.Configure(machine.PinConfig{Mode: machine.PinOutput})
	mosi.Configure(machine.PinConfig{Mode: machine.PinOutput})
	miso.Configure(machine.PinConfig{Mode: machine.PinInput})
	sck.Low()
	cs.High()
	mosi.Low()
	p.read() // the first command turns PENIRQ off
	return &Touch{poll: p, Cal: cal}
}

// ReadRaw returns the averaged raw 12-bit readings. ok is false when the
// panel is not pressed.
//
// The xpt2046 driver scales X to 16 bits (x << 4) and inverts Y
// ((4096 - y) << 4); this undoes the scaling but keeps the inversion.
func (t *Touch) ReadRaw() (x, y int32, ok bool) {
	if t.poll != nil {
		return t.poll.read()
	}
	if !t.dev.Touched() {
		return 0, 0, false
	}
	p := t.dev.ReadTouchPoint()
	if p.X == 0 && p.Y == 0 {
		return 0, 0, false
	}
	return int32(p.X >> 4), int32(p.Y >> 4), true
}

// ReadTouch implements hal.Touch.
func (t *Touch) ReadTouch() (x, y int16, pressed bool) {
	rx, ry, ok := t.ReadRaw()
	if !ok {
		return 0, 0, false
	}
	x, y = t.Cal.Map(rx, ry)
	return x, y, true
}

// XPT2046 commands: start bit, channel, 12 bits, differential, and the power
// mode. Measuring is done with PD = 00, as in the xpt2046 driver (on the
// board, with PD = 01 Z2 reads 7 instead of 4090 when not pressed). PENIRQ is
// on then, but only while the CPU is here, not using anything else on that
// line. The last command, with PD = 01, turns PENIRQ off until the next read.
const (
	cmdY    = 0x90
	cmdX    = 0xD0
	cmdZ1   = 0xB0
	cmdZ2   = 0xC0
	cmdIdle = 0xD1
)

// samples is how many readings are averaged, as New's Precision.
const samples = 10

// poller reads the XPT2046 by bit-banging, with PENIRQ off.
type poller struct {
	clk, cs, din, dout machine.Pin
}

var delayReg volatile.Register8

// halfClock waits about 1us: the XPT2046 clock can go up to 2MHz, and the
// panel settles better at a slower one.
func halfClock() {
	for i := 0; i < 30; i++ {
		delayReg.Get()
	}
}

func (p *poller) pulse() {
	p.clk.High()
	halfClock()
	p.clk.Low()
	halfClock()
}

// convert sends a command and returns the 12-bit result.
func (p *poller) convert(cmd uint8) int32 {
	for i := 0; i < 8; i++ {
		p.din.Set(cmd&0x80 != 0)
		cmd <<= 1
		p.pulse()
	}
	v := int32(0)
	for i := 0; i < 12; i++ {
		v <<= 1
		p.pulse()
		if p.dout.Get() {
			v |= 1
		}
	}
	for i := 0; i < 4; i++ {
		p.pulse()
	}
	return v
}

func (p *poller) pressed() bool {
	z1 := p.convert(cmdZ1)
	return pressed(z1, p.convert(cmdZ2))
}

// read returns the averaged readings as ReadRaw does. A reading is dropped
// if the panel was released before it ended (the last samples would be off).
func (p *poller) read() (x, y int32, ok bool) {
	p.cs.Low()
	if p.pressed() {
		p.convert(cmdY) // the first one after switching the panel is off
		for i := 0; i < samples; i++ {
			y += p.convert(cmdY)
			x += p.convert(cmdX)
		}
		ok = p.pressed()
	}
	p.convert(cmdIdle)
	p.cs.High()
	if !ok {
		return 0, 0, false
	}
	return x / samples, 4096 - y/samples, true
}
