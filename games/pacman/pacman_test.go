package pacman

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
	clock.Step = 100 * time.Microsecond // the game waits for its frames in a loop
	g.Clock = clock
	p := &picotest.Pad{Clock: clock}
	g.Pad = p
	return g, d, clock, p
}

// TestMatchesC plays without a player as testdata/csim/sim.cpp does with
// the C source compiled on the host (testdata/csim/run.sh), and compares
// the positions of the characters, the score, the lives, the dots and the
// timers of every frame by their checksums.
func TestMatchesC(t *testing.T) {
	want := map[int]string{
		499:  "10166c29",
		999:  "63d897d3",
		1499: "d29f4f15",
		1999: "a7ffaa5d",
		2499: "3218b28c",
		2999: "32dae00d",
		3499: "44dac726",
		3999: "a52b846e",
	}
	const wantEnd = "3999: 255,165,0,0 275,30,1,0 70,35,2,0 180,75,2,0 175,100,3,0 s=4600 l=0 d=52 fr=0 lv=1\n"

	g, _, _, _ := newTest(t, 5)
	g.score, g.level, g.lives = 0, 0, 30
	g.newGame(false)
	crc := crc32.NewIEEE()
	var line string
	frighten, eaten, fruit := 0, 0, 0
	for f := 0; f < 4000; f++ {
		// the player of sim.cpp
		xp, yp := xToMap(g.chars[chPacman].x), yToMap(g.chars[chPacman].y)
		best := -1
		for k := 0; k < 4; k++ {
			d := (k + f/50) & 3
			if b := g.getBoardDir(xp, yp, d); b == tileDot || b == tilePellet || b == tilePelletOff {
				best = d
				break
			}
		}
		if g.frightenCnt > 0 {
			bestdist := 1 << 30
			for d := 0; d < 4; d++ {
				if !g.checkBoard(xp, yp, d, false) {
					continue
				}
				x2, y2 := shiftX(xp, d), shiftY(yp, d)
				for i := 1; i < 5; i++ {
					if g.chars[i].state != sFrighten {
						continue
					}
					dx, dy := xToMap(g.chars[i].x)-x2, yToMap(g.chars[i].y)-y2
					if dx*dx+dy*dy < bestdist {
						bestdist, best = dx*dx+dy*dy, d
					}
				}
			}
		}
		if best >= 0 {
			g.nextDir = uint8(best)
		} else if f%6 == 0 {
			g.nextDir = uint8(((f/6)*7 + (f/6)/3) % 4)
		}
		if f == 1500 {
			g.dotCnt = 1
		}

		if !g.step() {
			t.Fatalf("game over at frame %d", f)
		}

		line = fmt.Sprintf("%d:", f)
		for _, c := range g.chars {
			line += fmt.Sprintf(" %d,%d,%d,%d", c.x, c.y, c.dir, c.state)
		}
		line += fmt.Sprintf(" s=%d l=%d d=%d fr=%d lv=%d\n", g.score, g.lives, g.dotCnt, g.frightenCnt, g.level)
		crc.Write([]byte(line))
		for _, c := range g.chars[1:] {
			if c.state == sFrighten {
				frighten++
			}
			if c.state == sEaten {
				eaten++
			}
		}
		if g.fruitSpriteImg >= 0 {
			fruit++
		}
		if w, ok := want[f]; ok {
			if got := fmt.Sprintf("%08x", crc.Sum32()); got != w {
				t.Fatalf("frame %d: checksum %s, want %s; the frame: %s", f, got, w, line)
			}
		}
	}
	if line != wantEnd {
		t.Errorf("the last frame:\n got %s\nwant %s", line, wantEnd)
	}
	// what sim.cpp prints: the game went through these
	if frighten != 1283 || eaten != 124 || fruit != 300 {
		t.Errorf("frames with: frightened %d, eaten %d, fruit %d", frighten, eaten, fruit)
	}
}

func TestScreens(t *testing.T) {
	g, d, clock, p := newTest(t, 1)
	clock.At(5300*ms, func() { goldentest.Check(t, "pacman_open", d.Image()) })
	var started time.Duration
	for s := 10; s < 40; s++ {
		clock.At(time.Duration(s)*time.Second, func() {
			if started == 0 && g.lives == 4 && g.chars[chPacman].x != width {
				started = clock.T
			}
		})
	}
	// the open screen takes about 19.5 seconds, then READY for 4
	clock.At(22*time.Second, func() {
		goldentest.Check(t, "pacman_ready", d.Image())
	})
	clock.At(28*time.Second, func() {
		if g.score == 0 || g.dotCnt == dotNum || g.lives != 4 {
			t.Errorf("playing: score %d, %d dots, %d lives", g.score, g.dotCnt, g.lives)
		}
	})
	p.Tap(hal.PadY, 28500*ms) // back to the open screen
	p.Tap(hal.PadY, 30*time.Second)
	g.Run()
	if started == 0 {
		t.Error("the game did not start")
	}
}

// TestPelletAndGhost eats a power pellet, a frightened ghost, and then
// meets a ghost that is not frightened.
func TestPelletAndGhost(t *testing.T) {
	g, _, _, _ := newTest(t, 1)
	g.score, g.level, g.lives = 0, 0, 4
	g.newGame(false)
	// the pacman goes left to the pellet at 1, 17
	pac := &g.chars[chPacman]
	pac.x, pac.y, pac.dx, pac.dy, pac.steps, pac.dir = mapToX(2), mapToY(17), -charSpeed, 0, 2, dirL
	g.nextDir = dirL
	g.move()
	g.move()
	if g.score != 50 || g.frightenCnt != 9*1000/speed || g.eaten != 200 || g.dotCnt != dotNum-1 || g.getTile(1, 17) != tileEmpty {
		t.Fatalf("pellet: score %d, frighten %d, eaten %d, %d dots", g.score, g.frightenCnt, g.eaten, g.dotCnt)
	}
	for i := 1; i < 5; i++ {
		if g.chars[i].state != sFrighten {
			t.Errorf("ghost %d: state %d", i, g.chars[i].state)
		}
	}
	// the wall at 0, 17 stops the pacman; Blinky meets it
	g.move()
	blinky := &g.chars[chBlinky]
	blinky.x, blinky.y, blinky.dx, blinky.dy, blinky.steps = pac.x, pac.y, 0, 0, 5
	g.move()
	if blinky.state != sEaten || g.score != 250 || g.eaten != 400 || pac.dx != 0 {
		t.Fatalf("frightened ghost: state %d, score %d, eaten %d, pacman dx %d", blinky.state, g.score, g.eaten, pac.dx)
	}
	// Pinky is not frightened any more
	pinky := &g.chars[chPinky]
	pinky.state = sChase
	pinky.x, pinky.y, pinky.dx, pinky.dy, pinky.steps = pac.x, pac.y, 0, 0, 5
	if g.move() || g.lives != 3 {
		t.Errorf("a ghost not frightened: %d lives", g.lives)
	}
}

func TestFruit(t *testing.T) {
	g, _, _, _ := newTest(t, 1)
	g.level = 2
	g.dotCnt = dotFruit1 + 1
	g.eatDot()
	if g.fruitSpriteImg != spriteFruit+2 || g.fruitCnt != 100 || g.fruitSpriteY != mapToY(13) {
		t.Errorf("fruit %d, %d frames", g.fruitSpriteImg, g.fruitCnt)
	}
}

// TestOneGame: with OneGame, Run returns after the last life, and on Y in
// the game (the open screen takes about 19.5 seconds, READY 4).
func TestOneGame(t *testing.T) {
	g, _, clock, _ := newTest(t, 1)
	g.OneGame = true
	var playing bool
	clock.At(25*time.Second, func() {
		playing = g.lives == 4 && g.chars[chPacman].x != width
		g.lives = 0 // the next ghost ends the game
	})
	g.Run()
	if !playing || g.lives > 0 || g.quit || clock.T > 3*time.Minute {
		t.Errorf("Run returned at %v (playing %v, %d lives)", clock.T, playing, g.lives)
	}

	g, _, clock, p := newTest(t, 1)
	g.OneGame = true
	p.Tap(hal.PadY, 26*time.Second)
	g.Run()
	if g.quit || clock.T < 26*time.Second || clock.T > 27*time.Second {
		t.Errorf("Y: Run returned at %v (quit on the open screen %v)", clock.T, g.quit)
	}
}
