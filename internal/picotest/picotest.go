// Package picotest has a fake clock and a scripted game controller for
// testing games on package picopad, which run their own loop and wait with
// the console's clock.
package picotest

import (
	"testing"
	"time"

	"github.com/sat0ken/tinygo-cyd/hal"
)

// Clock is a picopad.Clock in fake time. Each Now advances it by 1µs (the
// time the game spends), so that a game waiting in a loop goes on; Sleep
// advances it by the time slept. Past Limit the test fails.
//
// The functions of At run in Sleep: a game sleeps after DispUpdate, so
// its screen is up to date then.
type Clock struct {
	T     time.Duration
	Limit time.Duration
	tb    testing.TB
	at    []call
}

type call struct {
	t time.Duration
	f func()
}

// NewClock returns a clock at 0 that fails tb after limit.
func NewClock(tb testing.TB, limit time.Duration) *Clock {
	return &Clock{Limit: limit, tb: tb}
}

// At calls f in the first Sleep that reaches t (calls must be added in
// order).
func (c *Clock) At(t time.Duration, f func()) { c.at = append(c.at, call{t, f}) }

func (c *Clock) Now() time.Duration {
	c.advance(time.Microsecond)
	return c.T
}

func (c *Clock) Sleep(d time.Duration) {
	c.advance(d)
	for len(c.at) > 0 && c.T >= c.at[0].t {
		f := c.at[0].f
		c.at = c.at[1:]
		f()
	}
}

func (c *Clock) advance(d time.Duration) {
	c.T += d
	if c.T > c.Limit {
		c.tb.Fatalf("still running after %v", c.Limit)
	}
}

// Pad is a hal.Pad that holds buttons at the times of its script.
type Pad struct {
	Clock *Clock
	holds []hold
}

type hold struct {
	from, to time.Duration
	b        hal.Buttons
}

// Hold holds b from from for d.
func (p *Pad) Hold(b hal.Buttons, from, d time.Duration) {
	p.holds = append(p.holds, hold{from, from + d, b})
}

// Tap holds b for 50ms from from.
func (p *Pad) Tap(b hal.Buttons, from time.Duration) { p.Hold(b, from, 50*time.Millisecond) }

func (p *Pad) ReadPad() hal.Buttons {
	var b hal.Buttons
	for _, h := range p.holds {
		if p.Clock.T >= h.from && p.Clock.T < h.to {
			b |= h.b
		}
	}
	return b
}
