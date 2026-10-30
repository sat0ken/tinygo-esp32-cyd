//go:build !esp32_3248s035c

package platform

import "github.com/sat0ken/tinygo-cyd/hal"

// InitFull returns the whole screen: on this board what Init returns
// (only the ESP32-3248S035C has a window in Init).
func InitFull() (hal.Display, hal.Touch, error) { return Init() }
