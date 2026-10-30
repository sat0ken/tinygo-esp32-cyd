// Command ccmeter shows the Claude Code usage on the board: the 5-hour and
// weekly limits of the plan, the context window of the latest session and
// the tokens per hour. tools/ccbridge on the PC sends them over the USB
// serial port (package ccproto); see README "Claude Code の使用量".
//
//	make flash-s035 PKG=./cmd/ccmeter     # ESP32-3248S035C (480x320)
//	make flash-noerase PKG=./cmd/ccmeter  # ESP32-4827S043 (480x272)
//	make flash-s028 PKG=./cmd/ccmeter     # ESP32-2432S028R (320x240)
//	make wasm PKG=./cmd/ccmeter && make serve   # sample data in the browser
//
// Stop ccbridge before flashing: it keeps the serial port open.
package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/platform"
)

// input is where the lines from the PC come from (machine.Serial).
type input interface {
	Buffered() int
	ReadByte() (byte, error)
}

func main() {
	d, _, err := platform.InitFull() // on the S035, the whole 480x320 screen
	if err != nil {
		println("platform.InitFull:", err.Error())
		platform.Halt()
	}
	m := NewMeter(d)
	m.Draw()
	d.Display()
	in := openInput()
	println("ccmeter: started, waiting for ccbridge")
	for {
		now := time.Now()
		for in.Buffered() > 0 {
			b, err := in.ReadByte()
			if err != nil {
				break
			}
			m.Feed(b, now)
		}
		m.Update(now)
		d.Display()
		time.Sleep(20 * time.Millisecond)
	}
}
