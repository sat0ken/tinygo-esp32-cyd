package baseball

import (
	"math/rand"
	"testing"
	"time"

	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/internal/goldentest"
	"github.com/sat0ken/tinygo-cyd/internal/picotest"
	"github.com/sat0ken/tinygo-cyd/memlcd"
)

const ms = time.Millisecond

// mrnd is a rnd from math/rand, for the simulations.
type mrnd struct{ r *rand.Rand }

func (m mrnd) RandFloat() float32 { return m.r.Float32() }

// TestFlight: how far balls carry with air resistance, and the fence.
func TestFlight(t *testing.T) {
	for _, c := range []struct {
		b        hitBall
		min, max float32
	}{
		{hitBall{100, 28, 0}, 380, 410},
		{hitBall{90, 25, 0}, 300, 340},
		{hitBall{80, 18, 0}, 200, 240},
		{hitBall{70, 14, 0}, 130, 170},
	} {
		if d := carry(c.b); d < c.min || d > c.max {
			t.Errorf("%+v carries %.0f ft, want %.0f..%.0f", c.b, d, c.min, c.max)
		}
	}
	var f flight
	f.fly(hitBall{106, 28, 0})
	if !f.hr {
		t.Error("106 mph at 28 degrees to center: no home run")
	}
	f.fly(hitBall{100, 26, -44}) // down the line: 330 ft
	if !f.hr {
		t.Error("100 mph at 26 degrees down the line: no home run")
	}
	f.fly(hitBall{95, -5, 3}) // a ground ball rolls and stops
	if f.hr || f.landed < 0 || f.landed > 10 || f.stopped < 0 {
		t.Errorf("ground ball: landed %d, stopped %d", f.landed, f.stopped)
	}
}

// defense is nine average fielders.
func defense() *[numPos]fielder {
	var d [numPos]fielder
	for i := range d {
		d[i] = fielder{23, 125}
	}
	return &d
}

func on(id int8) runner { return runner{true, id, 27} }

// TestPlays: what comes of balls in play in some situations.
func TestPlays(t *testing.T) {
	bat := on(0)
	for _, c := range []struct {
		name   string
		b      hitBall
		sit    situation
		result int
		outs   int
		scored int
		bases  [3]bool
	}{
		{"a grounder to short", hitBall{85, -3, -15}, situation{batter: bat}, resGroundOut, 1, 0, [3]bool{}},
		{"a double play", hitBall{85, -3, -15}, situation{bases: [3]runner{on(1)}, batter: bat}, resDoublePlay, 2, 0, [3]bool{}},
		{"no double play with two outs", hitBall{85, -3, -15}, situation{outs: 2, bases: [3]runner{on(1)}, batter: bat}, resGroundOut, 3, 0, [3]bool{true}},
		{"a run on a ground out", hitBall{85, -3, 15}, situation{bases: [3]runner{{}, {}, on(3)}, batter: bat}, resGroundOut, 1, 1, [3]bool{}},
		{"a sacrifice fly", hitBall{95, 30, 0}, situation{bases: [3]runner{{}, {}, on(3)}, batter: bat}, resSacFly, 1, 1, [3]bool{}},
		{"no run on a third-out fly", hitBall{95, 30, 0}, situation{outs: 2, bases: [3]runner{{}, {}, on(3)}, batter: bat}, resFlyOut, 3, 0, [3]bool{false, false, true}},
		{"a single to left", hitBall{95, 12, -28}, situation{batter: bat}, resSingle, 0, 0, [3]bool{true}},
		{"a hole through short and third", hitBall{88, -4, -28}, situation{batter: bat}, resSingle, 0, 0, [3]bool{true}},
		{"a double into the gap", hitBall{98, 22, 18}, situation{batter: bat}, resDouble, 0, 0, [3]bool{false, true}},
		{"a grand slam", hitBall{106, 28, 0}, situation{bases: [3]runner{on(1), on(2), on(3)}, batter: bat}, resHomeRun, 0, 4, [3]bool{}},
	} {
		var pl play
		pl.resolve(c.b, c.sit, defense())
		b := [3]bool{pl.after.bases[0].on, pl.after.bases[1].on, pl.after.bases[2].on}
		if pl.result != c.result || pl.after.outs != c.outs || pl.scored != c.scored || b != c.bases {
			t.Errorf("%s: %s, %d outs, %d runs, bases %v; want %s, %d, %d, %v", c.name,
				resultText[pl.result], pl.after.outs, pl.scored, b, resultText[c.result], c.outs, c.scored, c.bases)
		}
	}
}

// TestRules: the count, walks pushing runners, the end of a half, a
// walk-off, and the home team not batting when it is ahead.
func TestRules(t *testing.T) {
	var m match
	m.start(9, false, false)
	for i := 0; i < 2; i++ {
		m.pitchDone(pitchFoul)
	}
	if ev := m.pitchDone(pitchFoul); ev != evFoul || m.cnt.strikes != 2 {
		t.Errorf("a foul with two strikes: %d strikes", m.cnt.strikes)
	}
	if ev := m.pitchDone(pitchSwingMiss); ev != evStrikeOut || m.cnt.outs != 1 || m.batter[0] != 1 {
		t.Errorf("strike three: event %d, %d outs, batter %d", ev, m.cnt.outs, m.batter[0])
	}
	m.bases = [3]runner{on(1), on(2), on(3)}
	for i := 0; i < 3; i++ {
		m.pitchDone(pitchBall)
	}
	if ev := m.pitchDone(pitchBall); ev != evWalk || m.score[0] != 1 || !m.bases[0].on || !m.bases[2].on {
		t.Errorf("a walk with the bases loaded: event %d, score %v, bases %v", ev, m.score, m.bases)
	}
	m.cnt.outs = 2
	m.pitchDone(pitchStrike)
	m.pitchDone(pitchStrike)
	m.pitchDone(pitchStrike)
	if !m.top == false || m.inning != 1 || m.top || m.bases[0].on || m.cnt != (count{}) {
		t.Errorf("after the third out: inning %d top %v bases %v count %+v", m.inning, m.top, m.bases, m.cnt)
	}

	// the ninth: the home team leads after the top: the game is over
	m.start(9, false, false)
	m.inning, m.top, m.score = 9, true, [2]int{1, 2}
	m.cnt.outs = 2
	m.pitchDone(pitchStrike)
	m.pitchDone(pitchStrike)
	m.pitchDone(pitchStrike)
	if !m.over || !m.top {
		t.Errorf("home team ahead after the top of the ninth: over %v", m.over)
	}
	// a walk-off walk
	m.start(9, false, false)
	m.inning, m.top, m.score = 9, false, [2]int{3, 3}
	m.bases = [3]runner{on(1), on(2), on(3)}
	m.cnt.balls = 3
	m.pitchDone(pitchBall)
	if !m.over || m.score[1] != 4 {
		t.Errorf("a walk-off walk: over %v, score %v", m.over, m.score)
	}
}

// TestCPUGames plays 200 nine-inning games between the computers: every
// game ends, the line score adds up, and the numbers look like baseball.
func TestCPUGames(t *testing.T) {
	r := mrnd{rand.New(rand.NewSource(1))}
	var m match
	games, runs, pa, h, bb, k := 0, 0, 0, 0, 0, 0
	for g := 0; g < 200; g++ {
		m.start(9, false, false)
		for n := 0; !m.over; n++ {
			if n > 2000 {
				t.Fatal("a game does not end")
			}
			b := m.batting()
			switch m.simPitch(r) {
			case evWalk:
				bb++
				pa++
			case evStrikeOut:
				k++
				pa++
			case evPlay:
				pa++
				if isHit(m.pl.result) {
					h++
				}
			}
			_ = b
		}
		for team := 0; team < 2; team++ {
			sum := 0
			for _, n := range m.line[team] {
				sum += int(n)
			}
			if sum != m.score[team] {
				t.Fatalf("game %d: the line of team %d adds up to %d, the score is %d", g, team, sum, m.score[team])
			}
		}
		if m.inning < 9 || m.inning > maxInnings {
			t.Fatalf("game %d ended in inning %d", g, m.inning)
		}
		games++
		runs += m.score[0] + m.score[1]
	}
	perTeam := float64(runs) / float64(2*games)
	avg := float64(h) / float64(pa-bb)
	bbRate, kRate := float64(bb)/float64(pa), float64(k)/float64(pa)
	t.Logf("runs a team %.2f, average %.3f, walks %.1f%%, strikeouts %.1f%%", perTeam, avg, 100*bbRate, 100*kRate)
	if perTeam < 2 || perTeam > 6 || avg < 0.22 || avg > 0.33 || bbRate < 0.05 || bbRate > 0.14 || kRate < 0.12 || kRate > 0.3 {
		t.Error("the numbers are not like baseball")
	}
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

// TestScreens: the title, the batting view with a pitch on its way, the
// line score.
func TestScreens(t *testing.T) {
	g, d, clock, p := newTest(t, 5)
	g.OneGame = true
	clock.At(200*ms, func() { goldentest.Check(t, "baseball_title", d.Image()) })
	p.Tap(hal.PadA, 300*ms)
	clock.At(700*ms, func() { goldentest.Check(t, "baseball_board", d.Image()) })
	p.Tap(hal.PadA, 900*ms)
	p.Hold(hal.PadDown, 1200*ms, 150*ms)
	clock.At(2500*ms, func() {
		goldentest.Check(t, "baseball_bat", d.Image())
		if !g.m.human[0] || g.m.human[1] || g.m.batting() != 0 {
			t.Errorf("1P: human %v, batting %d", g.m.human, g.m.batting())
		}
	})
	for t := 4 * time.Second; t < 12*time.Second; t += 500 * ms {
		p.Tap(hal.PadY, t) // Y asks, Y again quits
	}
	g.Run()
	if !g.quit {
		t.Error("Y, Y did not quit")
	}
}

// TestPitching: the player pitches: a slider, aimed low and away.
func TestPitching(t *testing.T) {
	g, d, clock, p := newTest(t, 7)
	g.m.start(3, true, false)
	g.m.top = false // the player's team in the field
	g.m.setDefense()
	p.Tap(hal.PadRight, 200*ms)
	clock.At(400*ms, func() { goldentest.Check(t, "baseball_choose", d.Image()) })
	p.Tap(hal.PadA, 500*ms)
	p.Hold(hal.PadDown, 600*ms, 300*ms)
	p.Hold(hal.PadRight, 600*ms, 200*ms)
	clock.At(1000*ms, func() {
		goldentest.Check(t, "baseball_aim", d.Image())
		if g.aim.y > -0.5 || g.aim.x < 0.3 {
			t.Errorf("the aim: %+v", g.aim)
		}
	})
	p.Tap(hal.PadA, 1100*ms)
	g.pitchOnce()
	if g.lastKind != ptSlider || g.lastSpeed < 120 || g.lastSpeed > 140 {
		t.Errorf("the pitch: kind %d, %.0f km/h", g.lastKind, g.lastSpeed)
	}
}

// TestFieldView: a double into the gap with a runner on first, from above.
func TestFieldView(t *testing.T) {
	g, d, clock, _ := newTest(t, 1)
	m := &g.m
	m.start(9, true, false)
	m.bases[0] = runner{true, 8, 27}
	g.sit = m.situation()
	m.pl.resolve(hitBall{98, 22, 18}, g.sit, &m.def)
	clock.At(2500*ms, func() { goldentest.Check(t, "baseball_field", d.Image()) })
	g.fieldPlay()
	if m.pl.result != resDouble {
		t.Errorf("result %s", resultText[m.pl.result])
	}
}

// autoPad taps A every 350 ms: it swings at random moments and throws
// straight balls down the middle, and gets past the line scores.
type autoPad struct{ clock *picotest.Clock }

func (a *autoPad) ReadPad() hal.Buttons {
	if a.clock.T%(350*ms) < 60*ms {
		return hal.PadA
	}
	return 0
}

// TestOneInning plays a whole game of one inning against the computer, and
// Run returns (OneGame).
func TestOneInning(t *testing.T) {
	g, _, clock, _ := newTest(t, 3)
	g.Pad = &autoPad{clock}
	g.innings = 1
	g.OneGame = true
	g.Run()
	if !g.m.over || g.quit || g.m.inning != 1 && g.m.inning < 2 {
		t.Errorf("the game: over %v, inning %d", g.m.over, g.m.inning)
	}
	t.Logf("%d-%d after %v", g.m.score[0], g.m.score[1], clock.T)
}
