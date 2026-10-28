// Package minijoy reads the nulllab / emakefun mini joystick module
// (https://github.com/nulllaborg/mini-joystick-module): one joystick with
// a push button and four buttons, read by a CH32V003 that answers on I2C
// at address 0x5A. The buttons are named as printed on the module: Y at
// the top, A at the bottom, X on the left, B on the right.
//
// The protocol comes from the module's Arduino library (JoystickHandle):
// write the register number in one transaction, then read the value in a
// second one. Registers: 0x10 X, 0x11 Y (0..255, 128 in the middle; X
// grows to the right and Y upwards, and small movements read 128),
// 0x20..0x24 the button events of OK (stick press), A, Y, B, X. The
// firmware also moves to the next register after each byte, so ReadBlock
// reads everything in two block reads (checked on the module with
// examples/joystick_check: 1.2ms instead of 2.8ms at 100kHz).
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
	A  Button = iota // bottom
	B                // right
	X                // left
	Y                // top
	OK               // pressing the joystick
	numButtons
)

// buttonReg is the register of each button, checked by pressing each one.
// The Arduino library calls them differently (JoystickHandle.h: BUTTON_A
// 0x22, BUTTON_B 0x23, BUTTON_C 0x21, BUTTON_D 0x24), which does not match
// the letters printed on the module.
var buttonReg = [numButtons]byte{A: 0x21, B: 0x23, X: 0x24, Y: 0x22, OK: 0x20}

var buttonNames = [numButtons]string{"A", "B", "X", "Y", "OK"}

func (b Button) String() string { return buttonNames[b] }

// Event is the state the module reports for a button (the event names of
// its firmware's button library). On the module a button reads press down
// while it is held and long press hold once it has been held for 1s; after
// the release it reads press up, then single click or double click 0.3s
// later (only after presses shorter than about 0.3s), then none.
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

// Down reports whether the button is held: press down, press repeat and
// the long press events are reported only while it is held.
func (e Event) Down() bool {
	switch e {
	case PressDown, PressRepeat, LongPressStart, LongPressHold:
		return true
	}
	return false
}

func (e Event) String() string {
	if int(e) < len(eventNames) {
		return eventNames[e]
	}
	return "?"
}

// State is one reading of the whole module. X grows to the right and Y
// upwards (screen coordinates go downwards: use 255-Y).
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
// button registers): 4 I2C transactions.
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
