package zoom

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

// frameClock is a clock that goes on only when the game sleeps: the time
// stands still in a frame, as FakeTime of testdata/csim.
type frameClock struct{ t time.Duration }

func (c *frameClock) Now() time.Duration    { return c.t }
func (c *frameClock) Sleep(d time.Duration) { c.t += d }

// TestMatchesC plays 30,000 frames of 30 ms with the player of
// testdata/csim/sim.cpp, which plays the C source compiled on the host
// (testdata/csim/run.sh) with the same random numbers, and compares the
// ship, the objects, the shots, the gates and the stars of every frame by
// their checksums: sectors 1 to 7 with all their parts (the mother ship in
// space).
func TestMatchesC(t *testing.T) {
	want := map[int]string{
		999:   "a0864d05",
		1999:  "245494ce",
		2999:  "fe60135d",
		3999:  "a8095000",
		4999:  "a65e33c5",
		5999:  "3a91564c",
		6999:  "b57868e2",
		7999:  "55e656be",
		8999:  "50d658c2",
		9999:  "e94f47f6",
		10999: "33d52769",
		11999: "ab26351b",
		12999: "4944f3b1",
		13999: "6c5c1920",
		14999: "86e28966",
		15999: "6f3b782e",
		16999: "8e2657a4",
		17999: "8cd920a0",
		18999: "6c9d8250",
		19999: "cc65525d",
		20999: "6fb31367",
		21999: "c9f8aeee",
		22999: "95b9c901",
		23999: "c341e8db",
		24999: "c6cce197",
		25999: "2b3c4010",
		26999: "abe564d0",
		27999: "2265ed87",
		28999: "464d92a7",
		29999: "a1b5eadf",
	}
	g := New(memlcd.New(320, 240), nil, 9)
	g.Clock = &frameClock{}
	frame := 0
	var left, right bool
	g.keyHeld = func(key uint8) bool {
		switch key {
		case picopad.KeyA:
			return true
		case picopad.KeyLeft:
			return left
		case picopad.KeyRight:
			return right
		case picopad.KeyUp:
			return frame%300 < 100
		case picopad.KeyDown:
			return frame%300 >= 200 && frame%300 < 260
		}
		return false
	}
	g.keyNext = func() uint8 { return picopad.NoKey }
	// the player: to the nearest gate ahead, or else to the nearest enemy
	decide := func() {
		target, best := -1, 1<<30
		for _, gt := range g.gates {
			if gt.x != gateFree && !gt.def && gt.z > 0 && int(gt.z) < best {
				best, target = int(gt.z), int(gt.x)<<frac
			}
		}
		if target < 0 {
			for _, o := range g.obj {
				if (o.typ == enemyDisk || o.typ == enemyBig || o.typ == enemyBoss) && o.z > 0 && int(o.z) < best {
					best, target = int(o.z), int(o.x)
				}
			}
		}
		left = target >= 0 && target < int(g.shipX)-4<<frac
		right = target >= 0 && target > int(g.shipX)+4<<frac
	}

	g.newGame()
	g.initLevel()
	crc := crc32.NewIEEE()
	for ; frame < 30000; frame++ {
		decide()
		g.dispAll()
		if g.subLevel == 3 {
			g.shiftStars()
			g.newStars()
		}
		g.step(30000)
		g.WaitMs(30)
		if g.lives == 0 {
			g.lives = 3
		}

		fmt.Fprintf(crc, "%d: t%d l%d.%d sc%d li%d f%d g%d s%d,%d c%d gp%d bl%d",
			frame, g.Time(), g.level, g.subLevel, g.score, g.lives, g.fuel, g.goals, g.shipX, g.speed, g.shipInCrash, g.grassPhase, g.bossLives)
		for i, o := range g.obj {
			if o.typ != enemyNo {
				fmt.Fprintf(crc, " o%d:%d,%d,%d,%d,%d,%d,%d", i, o.typ, o.move, o.istep, o.x, o.z, o.sx, o.sz)
			}
		}
		for i, s := range g.shots {
			if s.x != shotFree {
				fmt.Fprintf(crc, " s%d:%d,%d", i, s.x, s.z)
			}
		}
		for i, gt := range g.gates {
			if gt.x != gateFree {
				fmt.Fprintf(crc, " g%d:%d,%d,%d,%d", i, gt.x, gt.z, b2i(gt.def), b2i(gt.bomb))
			}
		}
		for i, s := range g.stars {
			if s.x != starFree {
				fmt.Fprintf(crc, " *%d:%d,%d,%d,%d", i, s.x, s.y, s.dx, s.dy)
			}
		}
		crc.Write([]byte("\n"))
		if w, ok := want[frame]; ok {
			if got := fmt.Sprintf("%08x", crc.Sum32()); got != w {
				t.Fatalf("frame %d: checksum %s, want %s (sector %d.%d, score %d)", frame, got, w, g.level+1, g.subLevel, g.score)
			}
		}
	}
	if g.level != 6 || g.subLevel != 2 || g.score != 13290247 {
		t.Errorf("the end: sector %d.%d, score %d", g.level+1, g.subLevel, g.score)
	}
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func newTest(t *testing.T, seed uint32) (*Game, *memlcd.Display, *picotest.Clock, *picotest.Pad) {
	d := memlcd.New(320, 240)
	g := New(d, nil, seed)
	clock := picotest.NewClock(t, 30*time.Minute)
	clock.Step = 20 * time.Microsecond
	g.Clock = clock
	p := &picotest.Pad{Clock: clock}
	g.Pad = p
	return g, d, clock, p
}

// TestPlay goes through the title, the start of sector 1, flying (LEFT,
// UP, shooting) and back to the title with Y, which shows the score.
func TestPlay(t *testing.T) {
	g, d, clock, p := newTest(t, 1)
	clock.At(300*ms, func() { goldentest.Check(t, "zoom_title", d.Image()) })
	p.Tap(hal.PadA, 400*ms)
	clock.At(1500*ms, func() {
		goldentest.Check(t, "zoom_enter", d.Image())
	})
	var x0, v0 int16
	clock.At(4100*ms, func() {
		x0, v0 = g.shipX, g.speed
		if g.level != 0 || g.subLevel != 0 || g.lives != 3 || g.goals != 10 {
			t.Errorf("start: sector %d.%d, %d lives, %d goals", g.level+1, g.subLevel, g.lives, g.goals)
		}
	})
	// 1/128 of 1/64 pixel a µs: 122 pixels a second (and 50 ms until it
	// counts as released)
	p.Hold(hal.PadLeft, 4200*ms, 500*ms)
	p.Hold(hal.PadDown, 4800*ms, 200*ms)
	p.Hold(hal.PadA, 5400*ms, 300*ms)
	clock.At(5600*ms, func() {
		if dx := int(x0-g.shipX) >> frac; dx < 60 || dx > 75 {
			t.Errorf("LEFT for 0.5 s: %d pixels", dx)
		}
		if dv := int(v0-g.speed) >> frac; dv < 25 || dv > 36 {
			t.Errorf("DOWN for 0.2 s: speed -%d", dv)
		}
		n := 0
		for _, s := range g.shots {
			if s.x != shotFree {
				n++
			}
		}
		if n != 1 {
			t.Errorf("%d shots", n)
		}
	})
	clock.At(7*time.Second, func() {
		goldentest.Check(t, "zoom_game", d.Image())
		if g.score <= 0 || g.fuel >= fuelMax<<frac {
			t.Errorf("flying: score %d, fuel %d", g.score, g.fuel)
		}
	})
	p.Tap(hal.PadY, 7100*ms)
	clock.At(7500*ms, func() {
		if g.maxScore != g.score || g.maxScore == 0 {
			t.Errorf("max. score %d, score %d", g.maxScore, g.score)
		}
	})
	p.Tap(hal.PadY, 7600*ms)
	g.Run()
}

// scene prepares part sub of sector level (without the entering screen)
// in a clock that stands still.
func scene(t *testing.T, level, sub int) (*Game, *memlcd.Display) {
	d := memlcd.New(320, 240)
	g := New(d, nil, 3)
	g.Clock = &frameClock{t: 10 * time.Second}
	g.newGame()
	g.level, g.subLevel = level, sub
	g.fuel = 100 << frac
	g.goals = goalsInit[sub]
	g.initStars()
	g.resetShots()
	g.resetGates()
	g.initObj()
	g.shipX = shipDefX - 20<<frac
	return g, d
}

// put puts an object at x (pixels), z.
func (g *Game) put(typ int, x, z, istep int) {
	o := g.getFreeObj()
	*o = obj{typ: typ, istep: istep, x: int16(x << frac), z: int16(z)}
}

// TestScenes draws the new pictures in the game: sector 3 (part 2) with
// the city, the gates, the saucers, a walker, a bomb, an explosion and a
// shot; space with the mother ship and the stars.
func TestScenes(t *testing.T) {
	g, d := scene(t, 2, 2)
	g.gates[0] = gate{x: 200, z: 12 << frac}
	g.gates[1] = gate{x: 120, z: 90 << frac}
	g.put(enemyDisk, 80, 30<<frac, 0)
	g.put(enemyDisk, 130, 60<<frac, 0)
	g.put(enemyBig, 250, 70<<frac, 0)
	g.put(enemyBomb, 220, 20<<frac, 0)
	g.put(enemyCrash, 60, 50<<frac, 2)
	g.shots[0] = shot{x: g.shipX, z: 25 << frac}
	g.lives = 4
	g.keyHeld = func(key uint8) bool { return key == picopad.KeyRight }
	g.dispAll()
	goldentest.Check(t, "zoom_sector3", d.Image())

	g, d = scene(t, 0, 3)
	g.goals = 1
	g.fuel = 8 << frac // LOW FUEL
	for i := 0; i < 40; i++ {
		g.WaitMs(30)
		g.newStars()
		g.shiftStars()
	}
	g.put(enemyBoss, 160, 40<<frac, 0)
	g.put(enemyBomb, 150, 25<<frac, 0)
	g.dispAll()
	goldentest.Check(t, "zoom_space", d.Image())
}

// TestOneGame: with OneGame, Run returns after the last ship, and on Y in
// the game (not to the title).
func TestOneGame(t *testing.T) {
	g, _, clock, p := newTest(t, 1)
	g.OneGame = true
	p.Tap(hal.PadA, 400*ms)
	clock.At(7*time.Second, func() { // after the protection of the new ship
		g.lives = 0
		g.shipCrash()
	})
	g.Run()
	if clock.T < 10*time.Second || clock.T > 12*time.Second {
		t.Errorf("Run returned at %v", clock.T)
	}

	g, _, clock, p = newTest(t, 1)
	g.OneGame = true
	p.Tap(hal.PadA, 400*ms)
	p.Tap(hal.PadY, 5*time.Second)
	g.Run()
	if clock.T > 5200*ms {
		t.Errorf("Y: Run returned at %v", clock.T)
	}
}
