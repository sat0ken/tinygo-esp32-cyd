package baseball

// maxInnings is how long a game can go (then it is a draw).
const maxInnings = 12

// match is the state of a game.
type match struct {
	t       [2]*team // the visitors (bat first) and the home team
	human   [2]bool
	innings int // regulation
	inning  int // 1..
	top     bool
	cnt     count
	bases   [3]runner
	score   [2]int
	line    [2][maxInnings]int8
	hits    [2]int
	batter  [2]int // the next batter of each team
	over    bool
	pl      play // the last ball in play
	def     [numPos]fielder
}

// The events between pitches, for the screen.
const (
	evNone = iota
	evBall
	evStrike
	evFoul
	evStrikeOut
	evWalk
	evPlay
)

// newMatch starts a game of innings innings.
func (m *match) start(innings int, human0, human1 bool) {
	*m = match{t: [2]*team{&teams[0], &teams[1]}, human: [2]bool{human0, human1}, innings: innings, inning: 1, top: true}
	m.setDefense()
}

func (m *match) batting() int {
	if m.top {
		return 0
	}
	return 1
}

func (m *match) fielding() int { return 1 - m.batting() }

// curBatter is the batter at the plate.
func (m *match) curBatter() *player {
	b := m.batting()
	return &m.t[b].order[m.batter[b]]
}

// curPitcher is the pitcher on the mound.
func (m *match) curPitcher() *pitcher { return &m.t[m.fielding()].pitcher }

// fielderAt returns the player of the fielding team at position pos.
func (m *match) fielderAt(pos int) *player {
	t := m.t[m.fielding()]
	for i := range t.order {
		if t.order[i].pos == pos {
			return &t.order[i]
		}
	}
	return &t.order[0]
}

// setDefense fills m.def from the fielding team.
func (m *match) setDefense() {
	for pos := range m.def {
		p := m.fielderAt(pos)
		m.def[pos] = fielder{p.rangeSpeed(), p.throwSpeed()}
	}
}

// situation is the play's starting point.
func (m *match) situation() situation {
	b := m.batting()
	p := m.curBatter()
	return situation{outs: m.cnt.outs, bases: m.bases, batter: runner{true, int8(m.batter[b]), p.runSpeed()}}
}

// addRuns scores n runs for the batting team.
func (m *match) addRuns(n int) {
	b := m.batting()
	m.score[b] += n
	i := min(m.inning, maxInnings) - 1
	m.line[b][i] += int8(n)
}

// nextBatter brings up the next batter, with a new count.
func (m *match) nextBatter() {
	b := m.batting()
	m.batter[b] = (m.batter[b] + 1) % 9
	m.cnt.balls, m.cnt.strikes = 0, 0
}

// pitchDone applies a pitch that was not put in play, and returns what
// happened.
func (m *match) pitchDone(res int) int {
	switch res {
	case pitchBall:
		m.cnt.balls++
		if m.cnt.balls < 4 {
			return evBall
		}
		m.walk()
		return evWalk
	case pitchFoul:
		if m.cnt.strikes < 2 {
			m.cnt.strikes++
		}
		return evFoul
	default: // a strike, called or swinging
		m.cnt.strikes++
		if m.cnt.strikes < 3 {
			return evStrike
		}
		m.cnt.outs++
		m.nextBatter()
		m.checkHalf()
		return evStrikeOut
	}
}

// walk puts the batter on first and pushes the runners who must move.
func (m *match) walk() {
	s := m.situation()
	if m.bases[0].on {
		if m.bases[1].on {
			if m.bases[2].on {
				m.addRuns(1)
			}
			m.bases[2] = m.bases[1]
		}
		m.bases[1] = m.bases[0]
	}
	m.bases[0] = s.batter
	m.nextBatter()
	m.checkWalkOff()
}

// playDone applies the ball in play m.pl.
func (m *match) playDone() {
	pl := &m.pl
	b := m.batting()
	if isHit(pl.result) {
		m.hits[b]++
	}
	m.cnt.outs = pl.after.outs
	m.bases = pl.after.bases
	m.addRuns(pl.scored)
	m.nextBatter()
	m.checkWalkOff()
	m.checkHalf()
}

// checkWalkOff ends the game when the home team takes the lead in the last
// inning.
func (m *match) checkWalkOff() {
	if !m.top && m.inning >= m.innings && m.score[1] > m.score[0] {
		m.over = true
	}
}

// checkHalf ends the half inning after three outs (and the game when it is
// decided).
func (m *match) checkHalf() {
	if m.over || m.cnt.outs < 3 {
		return
	}
	m.cnt = count{}
	m.bases = [3]runner{}
	if m.top {
		if m.inning >= m.innings && m.score[1] > m.score[0] {
			m.over = true // the home team need not bat
			return
		}
		m.top = false
	} else {
		if m.inning >= m.innings && m.score[0] != m.score[1] || m.inning >= maxInnings {
			m.over = true
			return
		}
		m.inning++
		m.top = true
	}
	m.setDefense()
}

// halfOver reports whether a new half inning has just begun (no outs, no
// count, nobody on, and nothing yet in its line).
func (m *match) halfStart() bool {
	return m.cnt == count{} && m.bases == [3]runner{}
}

// simPitch plays a pitch with the computer on both sides (for the demo's
// fast forward and the tests): it returns the event.
func (m *match) simPitch(r rnd) int {
	pt := m.curPitcher()
	kind, level, aim := cpuPitch(r, pt, m.cnt)
	p := throwPitch(r, pt, kind, level, aim)
	b := m.curBatter()
	sw := cpuSwing(r, &p, b, m.cnt)
	res, hb := result(r, &p, sw, b)
	if res != pitchInPlay {
		return m.pitchDone(res)
	}
	m.pl.resolve(hb, m.situation(), &m.def)
	m.playDone()
	return evPlay
}
