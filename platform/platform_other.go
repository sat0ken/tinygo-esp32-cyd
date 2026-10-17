//go:build !esp32s3 && !esp32_3248s035c && !(js && wasm)

package platform

import (
	"github.com/sat0ken/tinygo-cyd/board"
	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/memlcd"
)

// Init returns an in-memory display on the host (no screen, no touch).
func Init() (hal.Display, hal.Touch, error) {
	return memlcd.New(board.Width, board.Height), hal.NoTouch{}, nil
}

// SetBacklight does nothing on the host.
func SetBacklight(percent int) {}
