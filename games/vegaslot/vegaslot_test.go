package vegaslot

import (
	"fmt"
	"hash/crc32"
	"testing"
	"time"

	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/internal/goldentest"
	"github.com/sat0ken/tinygo-cyd/internal/picotest"
	"github.com/sat0ken/tinygo-cyd/memlcd"
)

const ms = time.Millisecond

func newTest(t *testing.T, seed uint32) (*Game, *memlcd.Display, *picotest.Clock, *picotest.Pad) {
	d := memlcd.New(320, 240)
	g := New(d, nil, seed)
	clock := picotest.NewClock(t, 30*time.Minute)
	clock.Step = 100 * time.Microsecond
	g.Clock = clock
	p := &picotest.Pad{Clock: clock}
	g.Pad = p
	return g, d, clock, p
}

// TestMatchesC chooses the reels of 3000 games as testdata/csim/sim.cpp
// does with the C source compiled on the host (testdata/csim/run.sh), with
// the same random numbers, and compares the positions and the prizes by
// their checksums.
func TestMatchesC(t *testing.T) {
	want := map[int]string{
		499:  "164f8c5e",
		999:  "39335e4e",
		1499: "7afd0266",
		1999: "72fc0437",
		2499: "f5c9fb03",
		2999: "d49317c1",
	}
	g := New(memlcd.New(320, 240), nil, 3)
	crc := crc32.NewIEEE()
	bets, prizes := 0, 0
	for n := 0; n < 3000; n++ {
		lines := n%5 + 1
		g.bonus = 0
		if n%7 == 6 {
			g.bonus = 1000
		}
		g.getReelPos(lines)
		winsym := testSymbol(g.reels[0].dest/256, g.reels[1].dest/256, g.reels[2].dest/256)
		prize := getPrize(lines, winsym)
		if g.bonus == 0 {
			bets += lines
			prizes += prize
		}
		fmt.Fprintf(crc, "%d: %d %d %d %d\n", n, g.reels[0].dest, g.reels[1].dest, g.reels[2].dest, prize)
		if w, ok := want[n]; ok {
			if got := fmt.Sprintf("%08x", crc.Sum32()); got != w {
				t.Fatalf("game %d: checksum %s, want %s", n, got, w)
			}
		}
	}
	if bets != 7718 || prizes != 8610 {
		t.Errorf("bets %d, prizes %d", bets, prizes)
	}
}

// TestTables: the wins of each table are where findReel finds them, and
// the tables return 94% of the bets. A random number below the
// randomness of the first entry (no win) wins: the randomness of the
// others shares it.
func TestTables(t *testing.T) {
	g := New(memlcd.New(320, 240), nil, 1)
	for lines := 1; lines <= winLines; lines++ {
		tab := winTabs[lines-1]
		sum, ret := 0, 0
		for k := range tab {
			it := &tab[k]
			if k > 0 {
				sum += int(it.rand)
				ret += int(it.value) * int(it.rand)
			}
			g.findReel(lines, it)
			winsym := testSymbol(g.reels[0].dest/256, g.reels[1].dest/256, g.reels[2].dest/256)
			if prize := getPrize(lines, winsym); prize != int(it.value) || (prize > 0 && getTypes(lines, winsym) != int(it.typ)) {
				t.Errorf("%d lines, win %d (type %d): the reels win %d", lines, it.value, it.typ, prize)
			}
		}
		if sum != int(tab[0].rand) {
			t.Errorf("%d lines: the wins share %d, the table %d", lines, sum, tab[0].rand)
		}
		if r := float64(ret) / 65536 / float64(lines); r < 0.935 || r > 0.945 {
			t.Errorf("%d lines: return %.3f", lines, r)
		}
	}
}

func TestScreens(t *testing.T) {
	g, d, clock, p := newTest(t, 1)
	clock.At(500*ms, func() {
		goldentest.Check(t, "vegaslot_start", d.Image())
		if g.creditVal != 100 || g.betVal != 3 || g.winLines != 3 {
			t.Errorf("start: credit %d, bet %d, lines %d", g.creditVal, g.betVal, g.winLines)
		}
	})
	p.Tap(hal.PadB, time.Second) // bet 4
	p.Tap(hal.PadB, 1300*ms)     // bet 5
	p.Tap(hal.PadB, 1600*ms)     // the high game: 5 credits and 15 from the bank, but the bank is empty: 1
	clock.At(1900*ms, func() {
		if g.betVal != 1 || g.winLines != 1 {
			t.Errorf("B three times: bet %d, lines %d", g.betVal, g.winLines)
		}
		goldentest.Check(t, "vegaslot_bet1", d.Image())
	})
	p.Tap(hal.PadY, 2200*ms)
	g.Run()
}

// riskPad plays (A) every 10 seconds (a game ends before) until a win;
// then it moves half of the win to the bank (LEFT) and the rest to the
// credit (RIGHT).
type riskPad struct {
	picotest.Pad
	g              *Game
	won            time.Duration
	credit0, bank0 int
}

func (p *riskPad) ReadPad() hal.Buttons {
	t := p.Clock.T
	if p.won == 0 {
		if p.g.risking {
			p.won, p.credit0, p.bank0 = t, p.g.creditVal, p.g.bankVal
		} else if t%(10*time.Second) < 50*ms {
			return hal.PadA
		}
		return 0
	}
	switch d := t - p.won; {
	case d > 200*ms && d < 250*ms:
		return hal.PadLeft
	case d > 2*time.Second && d < 2050*ms:
		return hal.PadRight
	}
	return p.Pad.ReadPad()
}

func TestRisk(t *testing.T) {
	g, _, clock, _ := newTest(t, 1)
	p := &riskPad{Pad: picotest.Pad{Clock: clock}, g: g}
	g.Pad = p
	var games int
	for s := 1; s < 1200; s++ {
		clock.At(time.Duration(s)*time.Second, func() {
			if p.won != 0 && games == 0 && clock.T > p.won+5*time.Second {
				games = g.gameIn / 3
				prize := g.gameOut
				half := prize / 2
				if prize <= 10 {
					half = prize
				}
				if g.risking || g.bankVal != p.bank0+half || g.creditVal != p.credit0+prize-half {
					t.Errorf("a win of %d: bank %d (was %d), credit %d (was %d)", prize, g.bankVal, p.bank0, g.creditVal, p.credit0)
				}
				p.Tap(hal.PadY, clock.T+time.Second)
			}
		})
	}
	g.Run()
	if games == 0 {
		t.Fatal("no win in 20 minutes")
	}
	t.Logf("a win after %d games", games)
}

// TestAutoStart plays until a key; the wins go to the bank.
func TestAutoStart(t *testing.T) {
	g, _, clock, p := newTest(t, 1)
	p.Tap(hal.PadX, time.Second)
	p.Tap(hal.PadB, 60*time.Second)
	clock.At(65*time.Second, func() {
		if g.autoStart || g.gameIn < 30 || g.creditVal != 100-g.gameIn || g.bankVal != g.gameOut {
			t.Errorf("autostart: %v, %d in, %d out, credit %d, bank %d", g.autoStart, g.gameIn, g.gameOut, g.creditVal, g.bankVal)
		}
		t.Logf("%d games, %d won", g.gameIn/3, g.gameOut)
	})
	p.Tap(hal.PadY, 66*time.Second)
	g.Run()
}
