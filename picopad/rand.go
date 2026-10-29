package picopad

// The random numbers of PicoLibSDK (_lib/src/lib_rand.c): a 64-bit linear
// congruential generator, the high 32 bits of which are the result.

// randInit is RandInit with seed in place of the ring oscillator.
func (c *Console) randInit(seed uint32) {
	c.rand = 123456789123456789 + 198765432198765432 + uint64(seed)
	c.RandShift()
	c.RandShift()
	c.RandShift()
}

// RandShift returns a random 32-bit number.
func (c *Console) RandShift() uint32 {
	c.rand = c.rand*214013 + 2531011
	return uint32(c.rand >> 32)
}

// RandU8 returns a random 8-bit number.
func (c *Console) RandU8() uint8 { return uint8(c.RandShift() >> 24) }

// RandU16 returns a random 16-bit number.
func (c *Console) RandU16() uint16 { return uint16(c.RandShift() >> 16) }

// RandU8Max returns a random number from 0 to max.
func (c *Console) RandU8Max(max uint8) uint8 {
	if max == 0 {
		return 0
	}
	m := uint8(mask(uint32(max)))
	for {
		if r := c.RandU8() & m; r <= max {
			return r
		}
	}
}

// RandU16Max returns a random number from 0 to max.
func (c *Console) RandU16Max(max uint16) uint16 {
	if max == 0 {
		return 0
	}
	m := uint16(mask(uint32(max)))
	for {
		if r := c.RandU16() & m; r <= max {
			return r
		}
	}
}

// RandS8MinMax returns a random number from min to max.
func (c *Console) RandS8MinMax(min, max int8) int8 { return int8(c.RandU8Max(uint8(max-min))) + min }

// RandU8MinMax returns a random number from min to max.
func (c *Console) RandU8MinMax(min, max uint8) uint8 { return c.RandU8Max(max-min) + min }

// RandU16MinMax returns a random number from min to max.
func (c *Console) RandU16MinMax(min, max uint16) uint16 { return c.RandU16Max(max-min) + min }

// RandFloat returns a random number from 0 up to 1 (not included).
func (c *Console) RandFloat() float32 {
	return float32(c.RandShift()>>(32-24)) * (1.0 / 0x1000000)
}

// RandFloatMax returns a random number from 0 up to max.
func (c *Console) RandFloatMax(max float32) float32 { return c.RandFloat() * max }

// RandFloatMinMax returns a random number from min up to max.
func (c *Console) RandFloatMinMax(min, max float32) float32 {
	return c.RandFloatMax(max-min) + min
}

// RandS16Max returns a random number from 0 to max (max can be negative).
func (c *Console) RandS16Max(max int16) int16 {
	if max >= 0 {
		return int16(c.RandU16Max(uint16(max)))
	}
	return -int16(c.RandU16Max(uint16(-max)))
}

// mask returns v with all the bits below its highest bit set (Mask).
func mask(v uint32) uint32 {
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	return v
}
