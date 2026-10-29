package pictor

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

// TestMatchesC plays 9,000 frames with the script of testdata/csim/sim.cpp,
// which plays the C source compiled on the host (testdata/csim/run.sh)
// with the same random numbers, and compares the actor, the enemies, the
// bullets, the missiles and the points of every frame by their checksums:
// all the scenes and all the helpers (B every 90 frames, X every 250),
// and the first scene again.
func TestMatchesC(t *testing.T) {
	want := map[int]string{
		499:  "74ac5a37",
		999:  "48a3877f",
		1499: "96bfe78a",
		1999: "23738c83",
		2499: "819f177e",
		2999: "9470e3bc",
		3499: "5d6f3c06",
		3999: "cd3d6047",
		4499: "ade9bd13",
		4999: "8d1fc6c0",
		5499: "a87991bd",
		5999: "7ab05b14",
		6499: "3437019f",
		6999: "cdef0c7d",
		7499: "87daa5b8",
		7999: "16bbf7b8",
		8499: "adc65dab",
		8999: "baa585f2",
	}
	g, _, _, _ := newTest(t, 5)
	frame := 0
	g.keyHeld = func(key uint8) bool {
		switch key {
		case picopad.KeyA:
			return frame%200 < 150
		case picopad.KeyLeft:
			return frame%40 < 10
		case picopad.KeyRight:
			return frame%40 >= 20 && frame%40 < 30
		case picopad.KeyUp:
			return frame%60 >= 30 && frame%60 < 45
		case picopad.KeyDown:
			return frame%60 >= 45 && frame%60 < 55
		}
		return false
	}
	g.keyNext = func() uint8 {
		switch {
		case frame%90 == 45:
			return picopad.KeyB
		case frame%250 == 100:
			return picopad.KeyX
		}
		return picopad.NoKey
	}
	g.score, g.bonusMask, g.life, g.actInx = 0, 0, lifeInit, 0
	g.initScene(0)
	g.dispGameStart(false)
	crc := crc32.NewIEEE()
	for ; frame < 9000; frame++ {
		g.step()
		if g.currentFrame >= g.maxFrame {
			g.initScene(g.level + 1)
			g.dispGameStart(false)
		}
		if g.life < 50 {
			g.life = 300
		}

		fmt.Fprintf(crc, "%d: f%d lv%d sc%d life%d a%d %d,%d ph%d j%d r%d bm%d b%d %d,%d,%d s%d bp%d,%d,%d bl%d %d,%d",
			frame, g.currentFrame, g.level, g.score, g.life, g.actInx, g.actorX, g.actorY, g.actorPhase, g.actorJumpSpeed, g.actorReload,
			g.bonusMask, g.bonusInx, g.bonusX, g.bonusY, g.bonusPhase, g.enemyRandSeed, g.backPhase1, g.backPhase2, g.backPhase3,
			g.bloodTime, g.bloodX, g.bloodY)
		for i, e := range g.enemy {
			if e.temp != nil {
				fmt.Fprintf(crc, " e%d:%d,%d,%d,%d,%d", i, e.inx, e.x, e.y, e.anim, e.step)
			}
		}
		for i, x := range g.explosion {
			if x.anim < explosionPhases {
				fmt.Fprintf(crc, " x%d:%d,%d,%d", i, x.anim, x.x, x.y)
			}
		}
		for i, b := range g.bullet {
			if b.temp != nil {
				fmt.Fprintf(crc, " b%d:%d,%d,%d", i, bulletIndex(b.temp), b.x, b.y)
			}
		}
		for i, m := range g.missile {
			if m.temp != nil {
				fmt.Fprintf(crc, " m%d:%d,%d,%d,%d,%d,%d,%d,%d,%d", i, m.actInx, m.x, m.y, m.dx, m.dy, m.phase,
					m.lifetime, m.hits, g.enemyIndex(m.target))
			}
		}
		for i, p := range g.point {
			if p.text != "" {
				fmt.Fprintf(crc, " p%d:%s,%d,%d,%d", i, p.text, p.x, p.y, p.frames)
			}
		}
		crc.Write([]byte("\n"))
		if w, ok := want[frame]; ok {
			if got := fmt.Sprintf("%08x", crc.Sum32()); got != w {
				t.Fatalf("frame %d: checksum %s, want %s (score %d, level %d, life %d)", frame, got, w, g.score, g.level, g.life)
			}
		}
	}
	if g.score != 51310 || g.level != 12 || g.life != 222 {
		t.Errorf("the end: score %d, level %d, life %d", g.score, g.level, g.life)
	}
}

func bulletIndex(b *bulletTemp) int {
	for i := range bulletTemps {
		if &bulletTemps[i] == b {
			return i
		}
	}
	return -1
}

func (g *Game) enemyIndex(e *enemy) int {
	for i := range g.enemy {
		if &g.enemy[i] == e {
			return i
		}
	}
	return -1
}

// TestPlay goes through the title, the start of the scene (B and X
// transform Bara), the game, the menu and the end of the game with a new
// max. score.
func TestPlay(t *testing.T) {
	g, d, clock, p := newTest(t, 1)
	clock.At(100*ms, func() { goldentest.Check(t, "pictor_title", d.Image()) })
	p.Tap(hal.PadA, 200*ms)
	clock.At(1600*ms, func() {
		goldentest.Check(t, "pictor_start", d.Image())
		if g.level != 0 || g.actInx != actJill || g.currentFrame != 0 {
			t.Errorf("start: level %d, actor %d, frame %d", g.level, g.actInx, g.currentFrame)
		}
	})
	p.Tap(hal.PadB, 1700*ms)
	clock.At(1900*ms, func() {
		if g.actInx != 1 || g.actor.name != "Bird" {
			t.Errorf("B: actor %d %s", g.actInx, g.actor.name)
		}
	})
	p.Tap(hal.PadX, 2000*ms)
	clock.At(2200*ms, func() {
		if g.actInx != actJill {
			t.Errorf("X: actor %d", g.actInx)
		}
	})
	p.Tap(hal.PadA, 2300*ms)
	p.Hold(hal.PadA, 2600*ms, 2*time.Second)
	p.Hold(hal.PadRight, 2600*ms, 500*ms)
	clock.At(3*time.Second, func() {
		n := 0
		for _, m := range g.missile {
			if m.temp != nil {
				n++
			}
		}
		if n < 3 || g.actorX <= g.actorMinX {
			t.Errorf("shooting: %d shurikens, x %d", n, g.actorX)
		}
	})
	clock.At(5*time.Second, func() {
		goldentest.Check(t, "pictor_game", d.Image())
		// a frame at least 86 ms
		if g.currentFrame < 25 || g.currentFrame > 31 {
			t.Errorf("frames after 2.7s: %d", g.currentFrame)
		}
	})
	p.Tap(hal.PadY, 5100*ms)
	clock.At(5400*ms, func() { goldentest.Check(t, "pictor_menu", d.Image()) })
	p.Tap(hal.PadY, 5500*ms) // continue
	frame := 0
	clock.At(5600*ms, func() { frame = g.currentFrame })
	clock.At(6*time.Second, func() {
		if g.currentFrame <= frame {
			t.Errorf("the game does not continue: frame %d", g.currentFrame)
		}
		g.score = 1230 // a new max. score
	})
	p.Tap(hal.PadY, 6100*ms)
	p.Tap(hal.PadX, 6300*ms) // the end of the game
	clock.At(7600*ms, func() {
		goldentest.Check(t, "pictor_end", d.Image())
	})
	// the name: UP 0 1 2 ..., RIGHT, DOWN Z Y ...
	p.Tap(hal.PadUp, 7700*ms)
	p.Hold(hal.PadUp, 7800*ms, 1100*ms)
	p.Tap(hal.PadRight, 9000*ms)
	p.Hold(hal.PadDown, 9100*ms, 900*ms)
	p.Tap(hal.PadA, 10100*ms)
	clock.At(10300*ms, func() {
		if g.maxScore != 1230 || g.top[0].score != 1230 {
			t.Errorf("top list: %+v, max score %d", g.top[0], g.maxScore)
		}
		if name := string(g.top[0].name[:]); name != "8U      " {
			t.Errorf("name %q", name)
		}
	})
	p.Tap(hal.PadY, 10500*ms)
	p.Tap(hal.PadX, 10700*ms)
	g.Run()
}

// TestNewArt draws the scenes with the new pictures: the fire ant on the
// beach, the robot as the new helper of the spacecraft, the star cruiser,
// the warship and the space mine in the galaxy, with the UFO.
func TestNewArt(t *testing.T) {
	g, d, _, _ := newTest(t, 1)
	scene := func(level int) {
		g.actInx = 0
		g.initScene(level)
		for j, e := range g.enemyNow {
			g.addEnemy(e, 120+j*70, 70+j*45, move1, j)
		}
	}
	draw := func() {
		g.dispBack()
		g.dispActor()
		g.enemyDisp()
		g.dispHeader()
		g.dispFoot()
		g.DispUpdate()
	}
	scene(2)
	g.actorX = 60
	draw()
	goldentest.Check(t, "pictor_beach", d.Image())

	g.initScene(7)
	g.dispGameStart(false)
	goldentest.Check(t, "pictor_robot", d.Image())

	scene(8)
	g.setActor(9) // the UFO
	g.actorX, g.actorY = 50, 120
	draw()
	goldentest.Check(t, "pictor_galaxy", d.Image())
}
