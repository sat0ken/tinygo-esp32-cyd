//go:build !linux

package main

import (
	"errors"
	"os"
)

func openSerial(name string) (*os.File, error) {
	return nil, errors.New("ccbridge: serial ports are only supported on Linux for now")
}
