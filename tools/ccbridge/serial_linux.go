//go:build linux

package main

import (
	"os"
	"syscall"
	"unsafe"
)

// openSerial opens a serial port at 115200 baud, 8N1, raw. DTR and RTS
// stay as they are when the port is closed (no HUPCL), so closing it does
// not reset the board.
func openSerial(name string) (*os.File, error) {
	fd, err := syscall.Open(name, syscall.O_RDWR|syscall.O_NOCTTY|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: name, Err: err}
	}
	var t syscall.Termios
	if err := ioctl(fd, syscall.TCGETS, &t); err != nil {
		syscall.Close(fd)
		return nil, &os.PathError{Op: "TCGETS", Path: name, Err: err}
	}
	t.Iflag, t.Oflag, t.Lflag = 0, 0, 0
	t.Cflag = syscall.CS8 | syscall.CREAD | syscall.CLOCAL | syscall.B115200
	t.Ispeed, t.Ospeed = syscall.B115200, syscall.B115200
	t.Cc[syscall.VMIN], t.Cc[syscall.VTIME] = 1, 0
	if err := ioctl(fd, syscall.TCSETS, &t); err != nil {
		syscall.Close(fd)
		return nil, &os.PathError{Op: "TCSETS", Path: name, Err: err}
	}
	// Non-blocking: os.File then waits in the poller, and Close stops a
	// Read that waits.
	return os.NewFile(uintptr(fd), name), nil
}

func ioctl(fd int, req uintptr, t *syscall.Termios) error {
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), req, uintptr(unsafe.Pointer(t))); e != 0 {
		return e
	}
	return nil
}
