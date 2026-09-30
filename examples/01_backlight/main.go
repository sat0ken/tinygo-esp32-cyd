//go:build esp32s3

// 01_backlight fades the backlight (GPIO2) with PWM and logs once a second.
//
// Flash: tinygo flash -target=./targets/esp32-4827s043.json -monitor ./examples/01_backlight
package main

import (
	"machine"
	"time"

	"github.com/sat0ken/tinygo-cyd/board"
)

func main() {
	time.Sleep(500 * time.Millisecond)
	println("01_backlight: start")

	pwm := machine.PWM0
	if err := pwm.Configure(machine.PWMConfig{Period: 1e9 / 5000}); err != nil { // 5kHz
		println("pwm configure:", err.Error())
		return
	}
	ch, err := pwm.Channel(machine.Pin(board.PinBacklight))
	if err != nil {
		println("pwm channel:", err.Error())
		return
	}
	top := pwm.Top()
	println("pwm top:", top)

	// 0% -> 100% -> 0% in 10% steps, one step per second.
	levels := []uint32{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 90, 80, 70, 60, 50, 40, 30, 20, 10}
	for i := 0; ; i++ {
		pct := levels[i%len(levels)]
		pwm.Set(ch, top*pct/100)
		println("tick", i, "backlight", pct, "%")
		time.Sleep(time.Second)
	}
}
