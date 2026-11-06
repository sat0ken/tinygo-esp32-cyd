package futsal

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

func newMatch(seed int64) *match {
	m := &match{}
	m.start(mrnd{rand.New(rand.NewSource(seed))}, 120, true, false)
	m.state = stPlay
	return m
}

// free puts the ball loose at pos with velocity vel, nobody near it.
func free(m *match, pos, vel vec) {
	for i := range m.p {
		m.p[i].pos = vec{midX, fieldB - 4} // out of the way
		m.p[i].touchCD = 10
	}
	m.b = ball{pos: pos, vel: vel, owner: -1, last: 4}
}

// TestBall: the ball rolls to a stop, comes off the boards and goes into
// the goals.
func TestBall(t *testing.T) {
	m := newMatch(1)
	free(m, vec{100, 100}, vec{120, 0})
	for i := 0; i < 300; i++ {
		m.moveBall(dt)
	}
	if m.b.vel != (vec{}) || m.b.pos.x < 150 || m.b.pos.x > 230 {
		t.Errorf("rolled to %+v, %+v", m.b.pos, m.b.vel)
	}
	free(m, vec{100, fieldT + 5}, vec{0, -200}) // off the top board
	for i := 0; i < 5; i++ {
		m.moveBall(dt)
	}
	if m.b.vel.y <= 0 || m.b.pos.y < fieldT {
		t.Errorf("the board: %+v, %+v", m.b.pos, m.b.vel)
	}
	free(m, vec{fieldR - 20, midY + goalHalf + 10}, vec{250, 0}) // beside the goal
	for i := 0; i < 10; i++ {
		m.moveBall(dt)
	}
	if m.state != stPlay || m.b.vel.x >= 0 {
		t.Errorf("beside the goal: state %d, %+v", m.state, m.b.vel)
	}
	free(m, vec{fieldR - 20, midY + 5}, vec{250, 0}) // into it
	for i := 0; i < 10; i++ {
		m.moveBall(dt)
	}
	if m.state != stGoal || m.score != [2]int{1, 0} || m.kickoff != 1 || m.b.pos.x > fieldR+goalDepth {
		t.Errorf("a goal: state %d, score %v, kickoff %d, ball %+v", m.state, m.score, m.kickoff, m.b.pos)
	}
	for m.state == stGoal {
		m.step(dt)
	}
	if m.state != stKickoff || m.b.owner != 9 {
		t.Errorf("after the goal: state %d, the ball at %d", m.state, m.b.owner)
	}
}

// TestKeeper: the keeper catches a soft shot at him and keeps most hard
// ones out from far.
func TestKeeper(t *testing.T) {
	m := newMatch(2)
	k := &m.p[5] // the second team's keeper, at the right
	free(m, vec{fieldR - 60, midY}, vec{150, 0})
	k.pos, k.touchCD, k.stun = vec{fieldR - 8, midY}, 0, 0
	m.b.last, m.b.shot = 4, true
	for i := 0; i < 30 && m.b.owner < 0 && m.state == stPlay; i++ {
		m.play(dt)
	}
	if m.b.owner != 5 {
		t.Errorf("a soft shot: the ball at %d, state %d", m.b.owner, m.state)
	}
	goals := 0
	for n := 0; n < 200; n++ {
		m := newMatch(int64(n))
		free(m, vec{midX + 40, midY + 30}, vec{})
		k := &m.p[5]
		k.pos, k.touchCD = vec{fieldR - 8, midY}, 0
		m.p[4].pos, m.p[4].touchCD = m.b.pos, 0
		m.take(4)
		m.shoot(4, midY+10, 0.8)
		for i := 0; i < 60 && m.state == stPlay; i++ {
			m.play(dt)
		}
		if m.state == stGoal {
			goals++
		}
	}
	if goals < 15 || goals > 110 {
		t.Errorf("%d goals of 200 hard shots from far", goals)
	}
}

// TestHuman: the buttons pass, shoot and change the player.
func TestHuman(t *testing.T) {
	m := newMatch(3)
	m.setKickoff(0)
	m.state = stPlay
	if m.ctl[0] != 4 || m.b.owner != 4 {
		t.Fatalf("the kickoff: control %d, ball %d", m.ctl[0], m.b.owner)
	}
	m.b.held = 1
	m.in[0] = input{pass: true, move: vec{0, -1}} // up: the left wing
	m.play(dt)
	if m.b.owner >= 0 || m.ctl[0] != 2 || !m.p[2].recv {
		t.Fatalf("the pass: ball %d, control %d", m.b.owner, m.ctl[0])
	}
	for i := 0; i < 60 && m.b.owner < 0; i++ {
		m.in[0] = input{}
		m.play(dt)
	}
	if m.b.owner != 2 {
		t.Fatalf("the pass came to %d", m.b.owner)
	}
	m.in[0] = input{shoot: true, power: 1, move: vec{0, 1}}
	m.play(dt)
	if m.shots[0] != 1 || m.b.vel.x < 200 {
		t.Errorf("the shot: %d shots, %+v", m.shots[0], m.b.vel)
	}
	// the computer's ball: A changes to the player nearest to it
	m = newMatch(4)
	m.setKickoff(1)
	m.state = stPlay
	m.ctl[0] = 1
	m.in[0] = input{pass: true}
	want := m.nearest(0, 1)
	m.play(dt)
	if m.ctl[0] != want || want == 1 {
		t.Errorf("A without the ball: control %d, want %d", m.ctl[0], want)
	}
}

// TestTackle: B lunges at the carrier and wins the ball (the best
// defender against the worst dribbler, many times).
func TestTackle(t *testing.T) {
	won := 0
	for n := 0; n < 100; n++ {
		m := newMatch(int64(n))
		m.setKickoff(1)
		m.state = stPlay
		c := &m.p[9]
		d := &m.p[1] // ISHII, def 7
		d.pos = c.pos.add(vec{-8, 0})
		d.face = vec{1, 0}
		m.ctl[0] = 1
		m.in[0] = input{tackle: true}
		for i := 0; i < 10; i++ {
			m.play(dt)
			if m.b.owner < 0 || m.p[m.b.owner].team == 0 {
				won++
				break
			}
		}
	}
	if won < 40 || won > 90 {
		t.Errorf("%d tackles of 100 won", won)
	}
}

// TestCPUGames plays 100 games of two 2-minute halves between the
// computers: every game ends, the ends change at half time, and the
// numbers look like a game.
func TestCPUGames(t *testing.T) {
	r := mrnd{rand.New(rand.NewSource(1))}
	var m match
	goals, shots, wins := 0, 0, [2]int{}
	for g := 0; g < 100; g++ {
		m.start(r, 120, false, false)
		first := m.first
		for n := 0; m.state != stOver; n++ {
			if n > 20000 {
				t.Fatal("a game does not end")
			}
			m.step(dt)
			if m.state == stHalfTime {
				if m.minute() != 20*60 {
					t.Errorf("half time at %d s", m.minute())
				}
				m.secondHalf()
				if m.dir != [2]float32{-1, 1} || m.kickoff != 1-first || m.b.owner != m.kickoff*5+4 {
					t.Fatalf("the second half: dir %v, kickoff %d (first %d)", m.dir, m.kickoff, first)
				}
			}
		}
		goals += m.score[0] + m.score[1]
		shots += m.shots[0] + m.shots[1]
		if m.score[0] > m.score[1] {
			wins[0]++
		} else if m.score[1] > m.score[0] {
			wins[1]++
		}
	}
	t.Logf("goals a game %.2f, shots %.1f, wins %v", float64(goals)/100, float64(shots)/100, wins)
	if goals < 300 || goals > 1100 || wins[0] < 20 || wins[1] < 20 {
		t.Error("the numbers are not like a game")
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

// TestScreens: the title, the kickoff, the play; Y twice quits.
func TestScreens(t *testing.T) {
	g, d, clock, p := newTest(t, 5)
	g.OneGame = true
	g.halfLen = 60
	clock.At(200*ms, func() { goldentest.Check(t, "futsal_title", d.Image()) })
	p.Tap(hal.PadRight, 250*ms) // 3 minutes
	p.Tap(hal.PadLeft, 350*ms)  // 1 again
	p.Tap(hal.PadA, 450*ms)
	clock.At(700*ms, func() { goldentest.Check(t, "futsal_kickoff", d.Image()) })
	p.Hold(hal.PadRight, 1500*ms, 1500*ms)
	clock.At(5*time.Second, func() {
		goldentest.Check(t, "futsal_play", d.Image())
		if g.halfLen != 60 || !g.m.human[0] {
			t.Errorf("half %d s, human %v", g.halfLen, g.m.human)
		}
	})
	for at := 6 * time.Second; at < 12*time.Second; at += 500 * ms {
		p.Tap(hal.PadY, at)
	}
	g.Run()
	if !g.quit || g.m.state == stOver {
		t.Errorf("Y, Y: quit %v, state %d", g.quit, g.m.state)
	}
}

// TestBoard: the score at half time, until A.
func TestBoard(t *testing.T) {
	g, d, clock, p := newTest(t, 5)
	g.m.start(g, 60, true, false)
	g.m.score, g.m.shots = [2]int{2, 1}, [2]int{9, 6}
	g.showPitch()
	clock.At(300*ms, func() { goldentest.Check(t, "futsal_halftime", d.Image()) })
	p.Tap(hal.PadA, 500*ms)
	g.board("HALF TIME")
	if clock.T > time.Second {
		t.Errorf("A came back after %v", clock.T)
	}
}

// TestGoalScreen: GOAL! and the scorer.
func TestGoalScreen(t *testing.T) {
	g, d, _, _ := newTest(t, 1)
	m := &g.m
	m.start(g, 120, true, false)
	m.state = stPlay
	g.showPitch()
	for i := range m.p {
		m.p[i].pos = vec{midX, fieldB - 4}
	}
	m.p[5].pos = vec{fieldR - 8, midY - 12}
	m.p[4].pos = vec{fieldR - 40, midY}
	m.b.pos = vec{fieldR - 35, midY}
	m.take(4)
	m.shoot(4, midY+8, 1)
	m.p[5].stun = 5 // the keeper does not move
	for i := 0; i < 30 && m.state == stPlay; i++ {
		m.step(dt)
	}
	for i := 0; i < 10; i++ {
		m.step(dt)
	}
	g.scene()
	g.DispUpdate()
	goldentest.Check(t, "futsal_goal", d.Image())
	if m.state != stGoal || m.score[0] != 1 || m.scorer != 4 {
		t.Errorf("state %d, score %v, scorer %d", m.state, m.score, m.scorer)
	}
}

// botPad plays the human side simply: to the ball, to the goal, B near it.
type botPad struct {
	clock *picotest.Clock
	g     *Game
}

func (b *botPad) ReadPad() hal.Buttons {
	m := &b.g.m
	if m.state != stPlay {
		if b.clock.T%(400*ms) < 60*ms {
			return hal.PadA
		}
		return 0
	}
	p := &m.p[m.ctl[0]]
	to := m.b.pos
	var btn hal.Buttons
	if m.b.owner == m.ctl[0] {
		to = m.oppGoal(0)
		if p.pos.dist(to) < 90 && b.clock.T%(300*ms) < 150*ms {
			btn |= hal.PadB
		}
	}
	d := to.sub(p.pos)
	if d.x > 3 {
		btn |= hal.PadRight
	} else if d.x < -3 {
		btn |= hal.PadLeft
	}
	if d.y > 3 {
		btn |= hal.PadDown
	} else if d.y < -3 {
		btn |= hal.PadUp
	}
	return btn
}

// TestOneGame plays a whole game of 1-minute halves with the buttons, and
// Run returns (OneGame).
func TestOneGame(t *testing.T) {
	g, _, clock, _ := newTest(t, 3)
	g.Pad = &botPad{clock, g}
	g.halfLen = 60
	g.OneGame = true
	g.Run()
	if g.m.state != stOver || g.quit || g.m.shots[0] == 0 {
		t.Errorf("the game: state %d, quit %v, shots %v", g.m.state, g.quit, g.m.shots)
	}
	t.Logf("%v after %v, shots %v", g.m.score, clock.T, g.m.shots)
}
