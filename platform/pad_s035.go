//go:build esp32_3248s035c

package platform

import (
	"machine"

	"github.com/sat0ken/tinygo-cyd/board/s035"
	"github.com/sat0ken/tinygo-cyd/hal"
)

// InitPad returns the mini joystick module on connector P4 as a game
// controller, or nil if it is not there. It uses I2C1: I2C0 is the GT911's.
func InitPad() hal.Pad {
	return initJoystick(machine.I2C1, machine.Pin(s035.PinJoySDA), machine.Pin(s035.PinJoySCL))
}
