package raptor

import (
	"fmt"
	"hash/crc32"
	"testing"
	"time"

	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/internal/goldentest"
	"github.com/sat0ken/tinygo-cyd/internal/picotest"
	"github.com/sat0ken/tinygo-cyd/memlcd"
	"github.com/sat0ken/tinygo-cyd/picopad"
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

// TestMatchesC plays 10,000 frames with the script of testdata/csim/sim.cpp,
// which plays the C source compiled on the host (testdata/csim/run.sh)
// with the same random numbers, and compares the ship, the enemies and the
// missiles of every frame by their checksums. The game goes through all
// the weapons (level 7 at frame 5,500). The trigonometric functions of the
// sonar bomb are float64 rounded to float32 in both (the float versions of
// the C library round some of them the other way: from frame 5,085 on, the
// bomb would be off by the last bit).
func TestMatchesC(t *testing.T) {
	want := map[int]string{
		499:  "3b6f64ee",
		999:  "78cb24f7",
		1499: "c3a9a7b2",
		1999: "8e1f714a",
		2499: "30649f12",
		2999: "952bd09b",
		3499: "6a67d1c0",
		3999: "07d1c8ee",
		4499: "e11eb17a",
		4999: "9d9721ee",
		5499: "973d6207",
		5999: "921bb68d",
		6499: "ece2c944",
		6999: "1826f413",
		7499: "e494da32",
		7999: "8041477f",
		8499: "fdd3363d",
		8999: "c34912e1",
		9499: "76eefa08",
		9999: "d3d2f21a",
	}
	g, _, _, _ := newTest(t, 4)
	frame := 0
	g.keys = func(key uint8) bool {
		switch key {
		case picopad.KeyA:
			return true
		case picopad.KeyLeft:
			return frame%40 < 10
		case picopad.KeyRight:
			return frame%40 >= 20 && frame%40 < 30
		case picopad.KeyUp:
			return frame%120 >= 60 && frame%120 < 70
		case picopad.KeyDown:
			return frame%120 >= 100 && frame%120 < 110
		}
		return false
	}
	g.newGame()
	g.shields = 50
	crc := crc32.NewIEEE()
	for ; frame < 10000; frame++ {
		g.doLandscape()
		g.doShip()
		g.doShooting()
		g.moveMissile()
		g.doEnemy()
		g.doEnemyShooting()
		g.moveEnemyMissile()
		g.display()
		g.gameFrame++

		fmt.Fprintf(crc, "%d: ship %d,%d sh %d en %d sc %d lv %d w %d/%d", frame, g.shipX, g.shipY, g.shields, g.energy, g.score, g.level, g.weapon, g.weaponMax)
		for i, e := range g.enemy {
			if e.typ >= 0 {
				fmt.Fprintf(crc, " e%d:%d,%.3f,%.3f,%d", i, e.typ, e.x, e.y, e.energy)
			}
		}
		for i, m := range g.myMissile {
			if m.typ >= 0 {
				fmt.Fprintf(crc, " m%d:%d,%.3f,%.3f", i, m.typ, m.x, m.y)
			}
		}
		for i, m := range g.enMissile {
			if m.typ >= 0 {
				fmt.Fprintf(crc, " n%d:%d,%.3f,%.3f", i, m.typ, m.x, m.y)
			}
		}
		crc.Write([]byte("\n"))
		if w, ok := want[frame]; ok {
			if got := fmt.Sprintf("%08x", crc.Sum32()); got != w {
				t.Fatalf("frame %d: checksum %s, want %s (score %d, level %d, shields %d)", frame, got, w, g.score, g.level, g.shields)
			}
		}
	}
	if g.score != 120060 || g.level != 12 || g.shields != 41 {
		t.Errorf("the end: score %d, level %d, shields %d", g.score, g.level, g.shields)
	}
}

func TestScreens(t *testing.T) {
	g, d, clock, p := newTest(t, 1)
	clock.At(150*ms, func() {
		goldentest.Check(t, "raptor_start", d.Image())
		if g.shipX != 144 || g.shipY != 208 || g.shields != 5 || g.energy != energyMax {
			t.Errorf("start: ship at %d,%d, %d shields, energy %d", g.shipX, g.shipY, g.shields, g.energy)
		}
	})
	p.Hold(hal.PadUp, 300*ms, 300*ms) // 10 pixels a frame of 100ms
	clock.At(time.Second, func() {
		if g.shipY > 208-30 || g.shipY < 208-40 {
			t.Errorf("UP: ship at %d,%d", g.shipX, g.shipY)
		}
	})
	p.Hold(hal.PadA, 1100*ms, 500*ms)
	clock.At(1400*ms, func() {
		goldentest.Check(t, "raptor_shooting", d.Image())
		n := 0
		for _, m := range g.myMissile {
			if m.typ == 0 {
				n++
			}
		}
		if n < 1 {
			t.Error("no missile of the machine gun")
		}
	})
	p.Tap(hal.PadY, 2*time.Second)
	g.Run()
}

// TestOneGame: with OneGame, Run returns 3 s after the ship is lost, and
// on Y.
func TestOneGame(t *testing.T) {
	g, _, clock, _ := newTest(t, 1)
	g.OneGame = true
	clock.At(time.Second, func() { g.shields = -1 })
	g.Run()
	if clock.T < 3900*ms || clock.T > 4300*ms {
		t.Errorf("Run returned at %v", clock.T)
	}

	g, _, clock, p := newTest(t, 1)
	g.OneGame = true
	p.Tap(hal.PadY, time.Second)
	g.Run()
	if clock.T > 1300*ms {
		t.Errorf("Y: Run returned at %v", clock.T)
	}
}
