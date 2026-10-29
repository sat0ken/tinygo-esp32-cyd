//go:build esp32_2432s028r && e32r28t

package platform

import (
	"machine"

	"github.com/sat0ken/tinygo-cyd/board/s028"
	"github.com/sat0ken/tinygo-cyd/hal"
)

// InitPad returns the mini joystick module (SDA IO27, SCL IO18 on the SPI
// connector) as a game controller, or nil if it is not there. Build with
// -tags e32r28t (make flash-s028 TAGS=e32r28t).
func InitPad() hal.Pad {
	return initJoystick(machine.I2C0, machine.Pin(s028.PinJoySDA), machine.Pin(s028.E32R28TPinJoySCL))
}
