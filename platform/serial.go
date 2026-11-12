package platform

// SerialReader is the receiving side of the serial console
// (machine.Serial has it on the ESP32).
type SerialReader interface {
	Buffered() int
	ReadByte() (byte, error)
}
