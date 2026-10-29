//go:build esp32_2432s028r && !e32r28t

package platform

import (
	"machine"

	"github.com/sat0ken/tinygo-cyd/board/s028"
	"github.com/sat0ken/tinygo-cyd/hal"
)

// InitPad returns the mini joystick module on connector CN1 as a game
// controller, or nil if it is not there.
func InitPad() hal.Pad {
	return initJoystick(machine.I2C0, machine.Pin(s028.PinJoySDA), machine.Pin(s028.PinJoySCL))
}
