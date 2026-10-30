//go:build !baremetal

package main

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/ccproto"
)

// openInput returns sample data (in the browser there is no serial port):
// the lines ccbridge would send, again every few seconds.
func openInput() input { return &sample{u: sampleUsage(), next: time.Now()} }

// sampleUsage is the usage the browser and the tests show.
func sampleUsage() ccproto.Usage {
	u := ccproto.Unknown()
	u.Clock = 1793369400 // Fri 2026-10-30 14:10 local time
	u.FiveHour = ccproto.Limit{Percent: 42, ResetsIn: 3*3600 + 20*60}
	u.Week = ccproto.Limit{Percent: 83, ResetsIn: 2*86400 + 5*3600}
	u.Context = 63
	u.StatusAge = 12
	u.Project = "tinygo-cyd"
	u.Model = "Opus 5.5"
	u.Tokens5h = 34_560_000
	u.Tokens24h = 123_400_000
	u.HourMax = 12_000_000
	for i, v := range [ccproto.Hours]int64{
		0, 0, 0, 0, 0, 0, 0, 0, 1_200_000, 4_000_000, 7_500_000, 12_000_000,
		9_000_000, 2_000_000, 0, 6_000_000, 8_800_000, 10_500_000, 11_000_000, 7_000_000,
		3_000_000, 9_700_000, 11_800_000, 5_200_000,
	} {
		u.HourLevel[i] = ccproto.Level(v, u.HourMax)
	}
	return u
}

type sample struct {
	u    ccproto.Usage
	buf  []byte
	i    int
	next time.Time
}

func (s *sample) Buffered() int {
	if s.i == len(s.buf) && !time.Now().Before(s.next) {
		s.buf = ccproto.AppendStatus(s.buf[:0], &s.u)
		s.buf = ccproto.AppendTokens(s.buf, &s.u)
		s.i = 0
		s.next = time.Now().Add(5 * time.Second)
		s.u.Clock += 5
		s.u.FiveHour.ResetsIn -= 5
		s.u.Week.ResetsIn -= 5
	}
	return len(s.buf) - s.i
}

func (s *sample) ReadByte() (byte, error) {
	b := s.buf[s.i]
	s.i++
	return b, nil
}
