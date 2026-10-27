// Package minijoy reads the nulllab / emakefun mini joystick module
// (https://github.com/nulllaborg/mini-joystick-module): one joystick with
// a push button and four buttons A-D, read by a CH32V003 that answers on
// I2C at address 0x5A.
//
// The protocol comes from the module's Arduino library (JoystickHandle):
// write the register number in one transaction, then read the value in a
// second one. Registers: 0x10 X, 0x11 Y (0..255, 128 in the middle),
// 0x20..0x24 the button events of OK (stick press), C, A, B, D.
//
// Power the module with 3.3V when it is connected to an ESP32: its I2C
// lines are pulled up to its supply (4.7k to VCC, no regulator), so a 5V
// supply puts 5V on the ESP32's pins.
//
// The package does not import machine: the I2C bus is an interface
// (machine.I2C), so it can be tested on the host.
package minijoy

// Bus is an I2C bus (machine.I2C).
type Bus interface {
	Tx(addr uint16, w, r []byte) error
}

// Address is the module's I2C address.
const Address = 0x5A

// Registers.
const (
	RegX       = 0x10
	RegY       = 0x11
	RegButtons = 0x20 // OK, C, A, B, D at 0x20..0x24
)

// Button is one of the five buttons.
type Button uint8

const (
	A Button = iota
	B
	C
	D
	OK // pressing the joystick
	numButtons
)

// buttonReg is the register of each button (JoystickHandle.h).
var buttonReg = [numButtons]byte{A: 0x22, B: 0x23, C: 0x21, D: 0x24, OK: 0x20}

var buttonNames = [numButtons]string{"A", "B", "C", "D", "OK"}

func (b Button) String() string { return buttonNames[b] }

// Event is the state the module reports for a button (the event names of
// its firmware's button library).
type Event uint8

const (
	PressDown      Event = 0
	PressUp        Event = 1
	PressRepeat    Event = 2
	SingleClick    Event = 3
	DoubleClick    Event = 4
	LongPressStart Event = 5
	LongPressHold  Event = 6
	NoPress        Event = 8
)

var eventNames = [...]string{
	PressDown: "press down", PressUp: "press up", PressRepeat: "press repeat",
	SingleClick: "single click", DoubleClick: "double click",
	LongPressStart: "long press start", LongPressHold: "long press hold",
	7: "?", NoPress: "none",
}

func (e Event) String() string {
	if int(e) < len(eventNames) {
		return eventNames[e]
	}
	return "?"
}

// State is one reading of the whole module.
type State struct {
	X, Y    uint8
	Buttons [numButtons]Event
}

// Device is the module.
type Device struct {
	bus     Bus
	Address uint16
	w       [1]byte
	r       [5]byte
}

// New returns the module at Address on bus.
func New(bus Bus) *Device { return &Device{bus: bus, Address: Address} }

// ReadRegisters reads len(buf) (at most 5) bytes from reg the way the
// module's library does: the register number in one transaction, the
// bytes in another.
func (d *Device) ReadRegisters(reg byte, buf []byte) error {
	d.w[0] = reg
	if err := d.bus.Tx(d.Address, d.w[:], nil); err != nil {
		return err
	}
	return d.bus.Tx(d.Address, nil, buf)
}

// ReadRegistersRestart reads len(buf) bytes from reg in one transaction
// with a repeated start (write the register, then read). Whether the
// module answers this way is not documented.
func (d *Device) ReadRegistersRestart(reg byte, buf []byte) error {
	d.w[0] = reg
	return d.bus.Tx(d.Address, d.w[:], buf)
}

// Read reads the joystick and the five buttons one register at a time,
// as the module's library does (14 I2C transactions).
func (d *Device) Read(s *State) error {
	if err := d.ReadRegisters(RegX, d.r[:1]); err != nil {
		return err
	}
	s.X = d.r[0]
	if err := d.ReadRegisters(RegY, d.r[:1]); err != nil {
		return err
	}
	s.Y = d.r[0]
	for b := A; b < numButtons; b++ {
		if err := d.ReadRegisters(buttonReg[b], d.r[:1]); err != nil {
			return err
		}
		s.Buttons[b] = Event(d.r[0])
	}
	return nil
}

// ReadBlock reads the same in two block reads (X and Y, then the five
// button registers), which only works if the module's firmware moves to
// the next register after each byte. examples/joystick_check tells.
func (d *Device) ReadBlock(s *State) error {
	if err := d.ReadRegisters(RegX, d.r[:2]); err != nil {
		return err
	}
	s.X, s.Y = d.r[0], d.r[1]
	if err := d.ReadRegisters(RegButtons, d.r[:5]); err != nil {
		return err
	}
	for b := A; b < numButtons; b++ {
		s.Buttons[b] = Event(d.r[buttonReg[b]-RegButtons])
	}
	return nil
}
