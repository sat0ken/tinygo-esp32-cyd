//go:build esp32_2432s028r || esp32_3248s035c || esp32s3

package platform

import (
	"machine"

	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/minijoy"
	"github.com/sat0ken/tinygo-cyd/pad"
)

// initJoystick looks for the mini joystick module (package minijoy) on bus
// with the given pins and returns it as a game controller, or nil if it
// does not answer. It is looked for once: plug it in before the start.
func initJoystick(bus *machine.I2C, sda, scl machine.Pin) hal.Pad {
	if err := bus.Configure(machine.I2CConfig{SDA: sda, SCL: scl, Frequency: 100_000}); err != nil {
		println("joystick: I2C:", err.Error())
		return nil
	}
	// A read, not CheckDevice: the ESP32-S3's machine.I2C does not have it.
	var one [1]byte
	if bus.Tx(minijoy.Address, nil, one[:]) != nil {
		println("joystick: none (I2C SDA", sda, "SCL", scl, ")")
		return nil
	}
	println("joystick: found (I2C SDA", sda, "SCL", scl, ")")
	return pad.NewJoystick(bus)
}
