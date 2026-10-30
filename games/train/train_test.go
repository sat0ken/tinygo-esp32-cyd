package train

import (
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
	clock.Step = 100 * time.Microsecond // the game waits for its steps in a loop
	g.Clock = clock
	p := &picotest.Pad{Clock: clock}
	g.Pad = p
	return g, d, clock, p
}

// TestSolutions plays the solutions of all the levels as the AUTOMODE of
// the C source does: each one must bring the train to the gate with
// everything collected.
func TestSolutions(t *testing.T) {
	g, _, _, _ := newTest(t, 1)
	for lv := 1; lv <= levNum; lv++ {
		g.level = uint8(lv)
		g.initLevel()
		items := g.itemNum
		g.state = sHelp
		for k := 0; k < len(solution[lv]) && g.state == sHelp; k++ {
			g.curDir = map[byte]uint8{'R': dirR, 'U': dirU, 'L': dirL, 'D': dirD}[solution[lv][k]]
			for i := 0; i < 3; i++ {
				g.animLevel()
			}
		}
		if g.state != sFinish || g.headX != g.gateX || g.headY != g.gateY || g.itemNum != 0 || g.length != items+1 {
			t.Errorf("level %d: state %d, train at %d,%d (gate %d,%d), %d items left, length %d",
				lv, g.state, g.headX, g.headY, g.gateX, g.gateY, g.itemNum, g.length)
		}
	}
}

// autoPad plays a solution like a player: it presses the direction for
// the next step when the train has moved and the direction changes.
type autoPad struct {
	picotest.Pad
	g     *Game
	sol   string
	steps int // steps of the train seen
	x, y  int // the train
	press hal.Buttons
	until time.Duration
	on    bool
}

var dirButton = map[byte]hal.Buttons{'R': hal.PadRight, 'U': hal.PadUp, 'L': hal.PadLeft, 'D': hal.PadDown}

// start presses the first direction (the game must wait for it).
func (p *autoPad) start() {
	p.on, p.steps = true, 0
	p.x, p.y = p.g.headX, p.g.headY
	p.push(p.sol[0])
}

func (p *autoPad) push(c byte) {
	p.press, p.until = dirButton[c], p.Clock.T+30*ms
}

func (p *autoPad) ReadPad() hal.Buttons {
	if p.on && (p.g.headX != p.x || p.g.headY != p.y) {
		p.x, p.y = p.g.headX, p.g.headY
		p.steps++
		if p.steps >= len(p.sol) {
			p.on = false
		} else if p.sol[p.steps] != p.sol[p.steps-1] {
			p.push(p.sol[p.steps])
		}
	}
	b := p.Pad.ReadPad()
	if p.Clock.T < p.until {
		b |= p.press
	}
	return b
}

func TestScreens(t *testing.T) {
	g, d, clock, p := newTest(t, 1)
	// the demo: the train writes the title (30 steps of half a second)
	clock.At(15500*ms, func() {
		if g.level != 0 {
			t.Errorf("demo: level %d", g.level)
		}
		goldentest.Check(t, "train_demo", d.Image())
	})
	p.Tap(hal.PadB, 16*time.Second) // a key ends the demo: level 1
	clock.At(16500*ms, func() {
		if g.level != 1 || g.state != sWait || g.itemNum != 24 {
			t.Errorf("level 1: level %d, state %d, %d items", g.level, g.state, g.itemNum)
		}
		goldentest.Check(t, "train_level", d.Image())
	})
	// the password of level 41: GOLEM -> HOMER
	at := 17 * time.Second
	for _, b := range []hal.Buttons{hal.PadA, hal.PadUp, hal.PadRight, hal.PadRight, hal.PadUp,
		hal.PadRight, hal.PadRight, hal.PadUp, hal.PadUp, hal.PadUp, hal.PadUp, hal.PadUp, hal.PadA} {
		p.Tap(b, at)
		at += 200 * ms
	}
	clock.At(at+time.Second, func() {
		if g.level != 41 {
			t.Errorf("password HOMER: level %d", g.level)
		}
		goldentest.Check(t, "train_open", d.Image())
	})
	p.Tap(hal.PadB, at+1200*ms)
	clock.At(at+2*time.Second, func() {
		if g.level != 41 || g.state != sWait || g.headX != 10 || g.headY != 10 {
			t.Errorf("level 41: level %d, state %d, train at %d,%d", g.level, g.state, g.headX, g.headY)
		}
	})
	p.Tap(hal.PadY, at+2500*ms)                // back to the demo
	p.Hold(hal.PadY, at+3*time.Second, 400*ms) // held: the demo reads the keys between its steps
	g.Run()
}

// TestPlay plays level 1 with the buttons: the score, the gate, level 2.
func TestPlay(t *testing.T) {
	g, _, clock, _ := newTest(t, 1)
	p := &autoPad{Pad: picotest.Pad{Clock: clock}, g: g, sol: solution[1]}
	g.Pad = p
	p.Tap(hal.PadB, 100*ms) // level 1
	clock.At(500*ms, p.start)
	// 50 steps of half a second, 1.5 seconds at the gate, the walls
	clock.At(28*time.Second, func() {
		if g.level != 2 || g.score != 240 {
			t.Errorf("after level 1: level %d, score %d", g.level, g.score)
		}
		if p.steps != len(solution[1]) {
			t.Errorf("the train made %d steps of %d", p.steps, len(solution[1]))
		}
	})
	p.Tap(hal.PadB, 28500*ms) // the level 2 box
	clock.At(30*time.Second, func() {
		if g.level != 2 || g.state != sWait || g.itemNum != 29 {
			t.Errorf("level 2: level %d, state %d, %d items", g.level, g.state, g.itemNum)
		}
	})
	p.Tap(hal.PadY, 30500*ms)
	p.Hold(hal.PadY, 31*time.Second, 400*ms) // held: the demo reads the keys between its steps
	g.Run()
}

// TestCrash runs into the wall of level 1: 8 crystals on the way.
func TestCrash(t *testing.T) {
	g, _, clock, p := newTest(t, 1)
	p.Tap(hal.PadB, 100*ms)
	p.Tap(hal.PadRight, 500*ms)
	clock.At(12*time.Second, func() {
		if g.state != sCrash || g.score != 80 || g.length != 9 || g.headX != 18 {
			t.Errorf("crash: state %d, score %d, length %d, train at %d", g.state, g.score, g.length, g.headX)
		}
	})
	p.Tap(hal.PadA, 12500*ms) // the level again
	clock.At(13*time.Second, func() {
		if g.state != sWait || g.length != 1 || g.headX != 1 || g.score != 80 {
			t.Errorf("again: state %d, length %d, train at %d, score %d", g.state, g.length, g.headX, g.score)
		}
	})
	p.Tap(hal.PadY, 13500*ms)
	p.Hold(hal.PadY, 14*time.Second, 400*ms) // held: the demo reads the keys between its steps
	g.Run()
}

// TestHelp shows the solution of level 1 with X, then the level waits
// again.
func TestHelp(t *testing.T) {
	g, _, clock, p := newTest(t, 1)
	p.Tap(hal.PadB, 100*ms)
	p.Tap(hal.PadX, 500*ms)
	clock.At(10*time.Second, func() {
		if g.state != sHelp || g.length < 2 {
			t.Errorf("help: state %d, length %d", g.state, g.length)
		}
	})
	// 50 steps, half a second at the gate
	clock.At(27*time.Second, func() {
		if g.level != 1 || g.state != sWait || g.score != 0 || g.length != 1 {
			t.Errorf("after help: level %d, state %d, score %d, length %d", g.level, g.state, g.score, g.length)
		}
	})
	p.Tap(hal.PadY, 27500*ms)
	p.Hold(hal.PadY, 28*time.Second, 400*ms) // held: the demo reads the keys between its steps
	g.Run()
}

// TestOneGame: with OneGame, Y in the game returns from Run (not to the
// demo).
func TestOneGame(t *testing.T) {
	g, _, clock, p := newTest(t, 1)
	g.OneGame = true
	p.Tap(hal.PadB, 100*ms) // level 1
	p.Tap(hal.PadY, time.Second)
	g.Run()
	if clock.T > 1500*ms {
		t.Errorf("Run returned at %v", clock.T)
	}
}
