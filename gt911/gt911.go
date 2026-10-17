// Package gt911 reads the Goodix GT911 capacitive touch controller over
// I2C, and adapts it to hal.Touch.
//
// The controller is polled: the status register says whether new
// coordinates are ready (the GT911 measures about 100 times a second), and
// it is cleared after reading. Between two measurements the last state is
// kept, so polling faster than the controller is fine.
//
// The package does not import machine: the I2C bus is an interface
// (machine.I2C), so it can be tested on the host.
package gt911

import "github.com/sat0ken/tinygo-cyd/touchcal"

// Bus is an I2C bus (machine.I2C).
type Bus interface {
	Tx(addr uint16, w, r []byte) error
}

// The two addresses the GT911 can choose at reset.
const (
	Address     = 0x5D
	AddressAlt  = 0x14
	regProduct  = 0x8140 // product ID "911", firmware version, X/Y resolution
	regConfig   = 0x8047 // config version, then X/Y output max
	regStatus   = 0x814E // bit 7: ready, bits 0-3: number of points
	productSize = 10
)

// Device is a GT911.
type Device struct {
	bus  Bus
	Addr uint16
	w    [3]byte
	r    [productSize]byte
}

// New returns a device; call Probe to find its address.
func New(bus Bus) *Device { return &Device{bus: bus, Addr: Address} }

// Info is what the controller says about itself.
type Info struct {
	ProductID  string // "911"
	Firmware   uint16
	XRes, YRes uint16 // coordinate range (portrait)
	Config     uint8  // config version
}

// Probe looks for the controller at both addresses and returns its info.
// The product ID must be digits ("911"), so a bus that reads all 0xFF or
// all 0x00 is not taken for a controller.
func (d *Device) Probe() (Info, bool) {
	for _, a := range []uint16{Address, AddressAlt} {
		d.Addr = a
		if info, err := d.ReadInfo(); err == nil && digits(info.ProductID) {
			return info, true
		}
	}
	d.Addr = Address
	return Info{}, false
}

func digits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return s != ""
}

// ReadInfo reads the product ID, firmware version and resolution.
func (d *Device) ReadInfo() (Info, error) {
	var info Info
	r := d.r[:productSize]
	if err := d.read(regProduct, r); err != nil {
		return info, err
	}
	n := 0
	for n < 4 && r[n] != 0 {
		n++
	}
	info.ProductID = string(r[:n])
	info.Firmware = uint16(r[4]) | uint16(r[5])<<8
	info.XRes = uint16(r[6]) | uint16(r[7])<<8
	info.YRes = uint16(r[8]) | uint16(r[9])<<8
	if err := d.read(regConfig, r[:1]); err != nil {
		return info, err
	}
	info.Config = r[0]
	return info, nil
}

// Point is one touch.
type Point struct {
	ID   uint8
	X, Y uint16 // raw coordinates (portrait)
	Size uint16
}

// Read returns the first touch point and the number of points. ready is
// false if the controller has nothing new since the last read; then the
// other values are meaningless.
func (d *Device) Read() (p Point, n int, ready bool, err error) {
	r := d.r[:9] // status and the first point
	if err = d.read(regStatus, r); err != nil {
		return
	}
	if r[0]&0x80 == 0 {
		return
	}
	n = int(r[0] & 0x0F)
	if n > 5 {
		n = 0 // invalid
	}
	p = Point{
		ID:   r[1],
		X:    uint16(r[2]) | uint16(r[3])<<8,
		Y:    uint16(r[4]) | uint16(r[5])<<8,
		Size: uint16(r[6]) | uint16(r[7])<<8,
	}
	// Clear the status so that the controller reports the next measurement.
	d.w = [3]byte{regStatus >> 8, regStatus & 0xFF, 0}
	err = d.bus.Tx(d.Addr, d.w[:3], nil)
	return p, n, true, err
}

func (d *Device) read(reg uint16, r []byte) error {
	d.w[0], d.w[1] = byte(reg>>8), byte(reg)
	return d.bus.Tx(d.Addr, d.w[:2], r)
}

// Touch adapts a Device to hal.Touch with a calibration.
type Touch struct {
	Dev *Device
	Cal touchcal.Calibration

	x, y    int16
	pressed bool
	raw     Point
}

// NewTouch returns a hal.Touch for dev.
func NewTouch(dev *Device, cal touchcal.Calibration) *Touch {
	return &Touch{Dev: dev, Cal: cal}
}

// ReadTouch returns the touch position in screen coordinates. An I2C error
// counts as not pressed.
func (t *Touch) ReadTouch() (x, y int16, pressed bool) {
	p, n, ready, err := t.Dev.Read()
	switch {
	case err != nil:
		t.pressed = false
	case ready && n > 0:
		t.raw = p
		t.x, t.y = t.Cal.Map(int32(p.X), int32(p.Y))
		t.pressed = true
	case ready:
		t.pressed = false
	}
	return t.x, t.y, t.pressed
}

// Raw returns the raw coordinates of the last touch (for calibration).
func (t *Touch) Raw() Point { return t.raw }
