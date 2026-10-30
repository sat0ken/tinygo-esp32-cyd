package main

import (
	"testing"
	"time"

	"github.com/sat0ken/tinygo-cyd/ccproto"
	"github.com/sat0ken/tinygo-cyd/internal/goldentest"
	"github.com/sat0ken/tinygo-cyd/memlcd"
)

// sizes are the screens the meter has a layout for.
var sizes = []struct {
	w, h   int16
	golden string
}{
	{480, 320, "ccmeter_480x320"}, // ESP32-3248S035C
	{480, 272, "ccmeter_480x272"}, // ESP32-4827S043, the browser
	{320, 240, "ccmeter_320x240"}, // ESP32-2432S028R
}

var t0 = time.Date(2026, 10, 30, 14, 10, 0, 0, time.UTC)

func newTestMeter(w, h int16) (*Meter, *memlcd.Display) {
	d := memlcd.New(w, h)
	m := NewMeter(d)
	m.Draw()
	return m, d
}

// send feeds the lines of u to m at now.
func send(m *Meter, u *ccproto.Usage, now time.Time) {
	for _, b := range ccproto.AppendTokens(ccproto.AppendStatus(nil, u), u) {
		m.Feed(b, now)
	}
}

func TestScreens(t *testing.T) {
	u := sampleUsage()
	for _, s := range sizes {
		m, d := newTestMeter(s.w, s.h)
		send(m, &u, t0)
		m.Update(t0.Add(500 * time.Millisecond))
		goldentest.Check(t, s.golden, d.Image())

		// Drawing everything again gives the same screen.
		m.Draw()
		goldentest.Check(t, s.golden, d.Image())
	}
}

func TestWaiting(t *testing.T) {
	m, d := newTestMeter(480, 320)
	m.Update(t0)
	goldentest.Check(t, "ccmeter_waiting", d.Image())
}

func TestTimePasses(t *testing.T) {
	u := sampleUsage()
	m, d := newTestMeter(480, 320)
	send(m, &u, t0)
	m.Update(t0)
	// 10 minutes later, nothing more from the PC: the clock and the
	// countdowns go on, and the footer says the link is lost.
	m.Update(t0.Add(10*time.Minute + 30*time.Second))
	goldentest.Check(t, "ccmeter_lost", d.Image())
	for _, c := range []struct {
		t    *text
		want string
	}{
		{&m.clock, "14:20"},
		{&m.info[rowFiveHour], "resets 17:30 (3h 9m)"},
		{&m.info[rowWeek], "resets Sun 19:10 (2d 4h)"},
		{&m.foot, "No data from ccbridge for 10m"},
	} {
		if got := string(c.t.shown[:c.t.n]); got != c.want {
			t.Errorf("%q, want %q", got, c.want)
		}
	}

	// The PC is back: the values are new, the link is fine again.
	u.Clock += 630
	u.FiveHour = ccproto.Limit{Percent: 0, ResetsIn: -1} // the window has reset
	u.StatusAge = 0
	now := t0.Add(10*time.Minute + 31*time.Second)
	send(m, &u, now)
	m.Update(now)
	if got := string(m.foot.shown[:m.foot.n]); got != "PC link OK, status line 0s ago" {
		t.Errorf("footer %q", got)
	}
	if got := string(m.info[rowFiveHour].shown[:m.info[rowFiveHour].n]); got != "" {
		t.Errorf("5-hour reset %q, want nothing", got)
	}
	if m.bar[rowFiveHour].pct != 0 {
		t.Errorf("5-hour bar %d%%", m.bar[rowFiveHour].pct)
	}
}

func TestLongNames(t *testing.T) {
	u := sampleUsage()
	u.Model = "Claude Opus 5.5 with a long name"
	u.Project = "a-project-with-a-very-long-name"
	for _, s := range sizes {
		m, d := newTestMeter(s.w, s.h)
		send(m, &u, t0)
		m.Update(t0)
		// Nothing is drawn over the title or the labels.
		ref, rd := newTestMeter(s.w, s.h)
		short := sampleUsage()
		short.Model, short.Project = "", ""
		send(ref, &short, t0)
		ref.Update(t0)
		for _, r := range [][4]int16{
			{0, 0, m.model.x, m.l.titleH},                             // "Claude Code"
			{0, m.l.rowY[rowContext], m.info[rowContext].x, m.l.barY}, // "CONTEXT"
		} {
			for y := r[1]; y < r[1]+r[3]; y++ {
				for x := r[0]; x < r[0]+r[2]; x++ {
					if d.Pixel(x, y) != rd.Pixel(x, y) {
						t.Fatalf("%dx%d: pixel %d,%d drawn over", s.w, s.h, x, y)
					}
				}
			}
		}
	}
}

func TestBadLines(t *testing.T) {
	u := sampleUsage()
	m, _ := newTestMeter(480, 320)
	// The boot messages of the ESP32, a line too long for the buffer, then
	// the real lines.
	for _, b := range []byte("ets Jun  8 2016 00:22:57\r\nrst:0x1 (POWERON_RESET)\r\n") {
		m.Feed(b, t0)
	}
	for i := 0; i < 300; i++ {
		m.Feed('U', t0)
	}
	m.Feed('\n', t0)
	if m.got {
		t.Fatal("got data from lines that are not ours")
	}
	send(m, &u, t0)
	if !m.got || m.u != u {
		t.Errorf("got %+v\nwant %+v", m.u, u)
	}
}

func TestFormat(t *testing.T) {
	for _, c := range []struct {
		n    int64
		want string
	}{
		{0, "0"}, {999, "999"}, {1000, "1.0k"}, {1234, "1.2k"}, {9949, "9.9k"}, {9950, "10k"},
		{999_499, "999k"}, {999_500, "1.0M"}, {34_560_000, "35M"}, {123_456_789, "123M"},
		{1_234_567_890, "1.2G"}, {5_000_000_000_000, "5.0T"},
	} {
		if got := string(appendTokens(nil, c.n)); got != c.want {
			t.Errorf("appendTokens(%d) = %q, want %q", c.n, got, c.want)
		}
	}
	for _, c := range []struct {
		s        int
		dur, ago string
	}{
		{5, "<1m", "5s"}, {60, "1m", "1m"}, {3599, "59m", "59m"}, {3600, "1h 0m", "1h"},
		{5*3600 - 1, "4h 59m", "4h"}, {86400, "1d 0h", "1d"}, {7*86400 - 1, "6d 23h", "6d"},
	} {
		if got := string(appendDuration(nil, c.s)); got != c.dur {
			t.Errorf("appendDuration(%d) = %q, want %q", c.s, got, c.dur)
		}
		if got := string(appendAgo(nil, c.s)); got != c.ago {
			t.Errorf("appendAgo(%d) = %q, want %q", c.s, got, c.ago)
		}
	}
	if got := string(appendClock(nil, 1793369400)); got != "14:10" {
		t.Errorf("appendClock = %q", got)
	}
}

// TestNoAllocs: receiving and showing new values does not allocate (with
// the same names). TinyGo 0.44's testing.AllocsPerRun always returns 0:
// run it with go test.
func TestNoAllocs(t *testing.T) {
	u := sampleUsage()
	m, _ := newTestMeter(480, 320)
	now := t0
	send(m, &u, now)
	m.Update(now)
	status := ccproto.AppendStatus(nil, &u)
	tokens := ccproto.AppendTokens(nil, &u)
	if n := testing.AllocsPerRun(20, func() {
		now = now.Add(61 * time.Second)
		u.FiveHour.Percent = (u.FiveHour.Percent + 7) % 101
		status = ccproto.AppendStatus(status[:0], &u)
		tokens[len(tokens)-3]++ // another level for the current hour
		for _, b := range status {
			m.Feed(b, now)
		}
		for _, b := range tokens {
			m.Feed(b, now)
		}
		m.Update(now)
	}); n != 0 {
		t.Errorf("%v allocations per update", n)
	}
}
