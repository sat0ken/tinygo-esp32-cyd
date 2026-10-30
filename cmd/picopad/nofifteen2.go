//go:build esp32_2432s028r || esp32_3248s035c

package main

import (
	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/picopad"
)

// The boards with 4 MB of flash leave out the Fifteen puzzle with pictures
// (fifteen2.go).
const hasFifteen2 = false

func startFifteen2(d hal.Display, t hal.Touch, seed uint32) (*picopad.Console, func()) {
	return nil, nil
}
