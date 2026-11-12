//go:build baremetal && !esp32s3

package platform

import "machine"

// SerialInput returns the serial console's input: machine.Serial.
func SerialInput() SerialReader { return machine.Serial }
