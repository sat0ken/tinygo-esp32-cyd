package futsal

import "math"

// The pitch, in screen pixels (the playing field is the screen). Boards
// stand on the lines: the ball bounces off them and stays in play, except
// into the goals at the left and right.
const (
	fieldL, fieldR = 16, 304 // the goal lines
	fieldT, fieldB = 38, 232 // the touchlines
	midX, midY     = 160, 135
	goalHalf       = 20 // half the goal mouth
	goalDepth      = 10
	areaR          = 40 // the penalty area: this near the goal mouth
)

type vec struct{ x, y float32 }

func (a vec) add(b vec) vec           { return vec{a.x + b.x, a.y + b.y} }
func (a vec) sub(b vec) vec           { return vec{a.x - b.x, a.y - b.y} }
func (a vec) mul(k float32) vec       { return vec{a.x * k, a.y * k} }
func (a vec) dot(b vec) float32       { return a.x*b.x + a.y*b.y }
func (a vec) len() float32            { return sqrt32(a.x*a.x + a.y*a.y) }
func (a vec) dist(b vec) float32      { return a.sub(b).len() }
func sqrt32(v float32) float32        { return float32(math.Sqrt(float64(v))) }
func clamp(v, lo, hi float32) float32 { return max(lo, min(hi, v)) }

// norm returns a with length 1 (0 stays 0).
func (a vec) norm() vec {
	if l := a.len(); l > 1e-4 {
		return a.mul(1 / l)
	}
	return vec{}
}

// rotate turns a by r radians.
func (a vec) rotate(r float32) vec {
	s, c := math.Sincos(float64(r))
	return vec{a.x*float32(c) - a.y*float32(s), a.x*float32(s) + a.y*float32(c)}
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// rnd is the random numbers (picopad.Console has them).
type rnd interface {
	RandFloat() float32
}

// gauss returns a random number with mean 0 and deviation about sd.
func gauss(r rnd, sd float32) float32 {
	return (r.RandFloat() + r.RandFloat() + r.RandFloat() - 1.5) * 2 * sd
}

// player is a player on the pitch. Players 0..4 are the first team, 5..9
// the second; 0 and 5 are the goalkeepers.
type player struct {
	team int
	gk   bool
	d    *pdata
	pos  vec // the feet
	vel  vec
	face vec // where it looks (length 1)

	stun     float32 // can not move (a missed tackle, a dive)
	tackleCD float32 // until the next tackle
	touchCD  float32 // until it can touch the ball again (after a kick)
	lunge    float32 // a tackle going on
	recv     bool    // a pass is coming to it
	step     float32 // the running animation
	saveTry  int     // the kick it last tried to save (b.kicks)
	think    float32 // until the next decision of the computer
}

func (p *player) runSpeed() float32 {
	if p.gk {
		return 52 + float32(p.d.speed)*2
	}
	return 46 + float32(p.d.speed)*3
}

// ball is the ball: free, or at the feet (or in the hands) of its owner.
type ball struct {
	pos, vel vec
	owner    int     // -1: free
	last     int     // who touched it last
	kicks    int     // the kicks so far (a new kick, a new chance to save)
	shot     bool    // the last kick was a shot
	held     float32 // how long the owner has had it
}

// input is a human team's buttons for a step.
type input struct {
	move   vec     // the arrows (length 0 or 1)
	pass   bool    // A
	tackle bool    // B without the ball
	shoot  bool    // B released with the ball
	power  float32 // of the shot, 0..1
}

// The states of a match.
const (
	stKickoff = iota
	stPlay
	stGoal
	stHalfTime
	stOver
)

// match is the state of a game.
type match struct {
	t       [2]*team
	human   [2]bool
	p       [10]player
	b       ball
	score   [2]int
	shots   [2]int
	dir     [2]float32 // +1: the team attacks to the right
	half    int        // 1, 2
	clock   float32    // the time played in the half (s)
	halfLen float32
	state   int
	stateT  float32 // the time in the state
	kickoff int     // the team kicking off
	first   int     // the team that kicked off the first half
	ctl     [2]int  // the player a human team controls
	in      [2]input
	scorer  int // who scored the last goal
	r       rnd
}

// start begins a match of two halves of halfLen seconds.
func (m *match) start(r rnd, halfLen float32, human0, human1 bool) {
	*m = match{t: [2]*team{&teams[0], &teams[1]}, human: [2]bool{human0, human1},
		dir: [2]float32{1, -1}, half: 1, halfLen: halfLen, r: r}
	for i := range m.p {
		t := i / 5
		m.p[i] = player{team: t, gk: i%5 == 0, d: &m.t[t].p[i%5]}
	}
	m.first = 0
	if r.RandFloat() < 0.5 {
		m.first = 1
	}
	m.setKickoff(m.first)
}

// secondHalf changes ends and kicks off the second half.
func (m *match) secondHalf() {
	m.half = 2
	m.clock = 0
	m.dir[0], m.dir[1] = m.dir[1], m.dir[0]
	m.setKickoff(1 - m.first)
}

// minute is the clock of the match as shown (two halves of 20 minutes),
// in seconds.
func (m *match) minute() int {
	return int(min(m.clock/m.halfLen, 1)*20*60) + (m.half-1)*20*60
}

// goalX is the goal line a team attacks.
func (m *match) goalX(t int) float32 {
	if m.dir[t] > 0 {
		return fieldR
	}
	return fieldL
}

// ownGoal is the middle of the goal a team defends.
func (m *match) ownGoal(t int) vec { return vec{m.goalX(1 - t), midY} }

// oppGoal is the middle of the goal a team attacks.
func (m *match) oppGoal(t int) vec { return vec{m.goalX(t), midY} }

// at returns the point of the pitch at u (0 the own goal line .. 1 the
// other) and v (-1 .. 1 across) for team t.
func (m *match) at(t int, u, v float32) vec {
	x := fieldL + u*(fieldR-fieldL)
	if m.dir[t] < 0 {
		x = fieldR - u*(fieldR-fieldL)
	}
	return vec{x, midY + v*(fieldB-fieldT)/2}
}

// uv returns where p is for team t (the reverse of at).
func (m *match) uv(t int, p vec) (u, v float32) {
	u = (p.x - fieldL) / (fieldR - fieldL)
	if m.dir[t] < 0 {
		u = 1 - u
	}
	return u, (p.y - midY) / ((fieldB - fieldT) / 2)
}

// The formation of the four outfield players (u, v): the back, the wings,
// the pivot.
var formation = [5]vec{{}, {0.22, 0}, {0.42, -0.55}, {0.42, 0.55}, {0.64, 0}}

// setKickoff puts the players in their halves and the ball on the centre
// spot at the feet of team t's pivot.
func (m *match) setKickoff(t int) {
	m.kickoff = t
	m.state, m.stateT = stKickoff, 0
	for i := range m.p {
		p := &m.p[i]
		k := i % 5
		var pos vec
		switch {
		case p.gk:
			pos = m.ownGoal(p.team).add(vec{m.dir[p.team] * 8, 0})
		case p.team == t && k == 4:
			pos = vec{midX - m.dir[t]*3, midY}
		case p.team == t && k == 3:
			pos = m.at(t, 0.46, 0.2)
		default:
			f := formation[k]
			pos = m.at(p.team, min(f.x*0.65, 0.4), f.y)
		}
		p.pos, p.vel = pos, vec{}
		p.face = vec{m.dir[p.team], 0}
		p.stun, p.lunge, p.touchCD, p.recv, p.think = 0, 0, 0, false, 0.6
	}
	m.b = ball{owner: t*5 + 4, last: t*5 + 4, kicks: m.b.kicks + 1}
	m.b.pos = m.p[t*5+4].pos.add(vec{m.dir[t] * 5, 0})
	for t := 0; t < 2; t++ {
		m.ctl[t] = t*5 + 4
	}
}

// step moves the match on by dt seconds.
func (m *match) step(dt float32) {
	m.stateT += dt
	switch m.state {
	case stKickoff:
		m.in = [2]input{}
		if m.stateT >= 1 {
			m.state = stPlay
		}
	case stPlay:
		m.clock += dt
		m.play(dt)
		b := &m.b
		if m.state == stPlay && m.clock >= m.halfLen && !(b.owner < 0 && b.shot && b.vel.len() > 80) {
			if m.half == 1 {
				m.state = stHalfTime
			} else {
				m.state = stOver
			}
			m.stateT = 0
		}
	case stGoal:
		m.in = [2]input{}
		for i := range m.p {
			m.move(i, vec{}, dt)
		}
		m.moveBall(dt)
		if m.stateT >= 2 {
			m.setKickoff(m.kickoff)
		}
	}
}

// play is a step of the game: the decisions, the moves, the ball.
func (m *match) play(dt float32) {
	for i := range m.p {
		p := &m.p[i]
		p.stun = max(p.stun-dt, 0)
		p.tackleCD = max(p.tackleCD-dt, 0)
		p.touchCD = max(p.touchCD-dt, 0)
		if p.lunge > 0 {
			p.lunge -= dt
			if p.lunge <= 0 {
				p.lunge, p.stun = 0, 0.15
			}
		}
	}
	for i := range m.p {
		var want vec
		if t := m.p[i].team; m.human[t] && m.ctl[t] == i {
			want = m.humanPlay(i)
			if want == (vec{}) && m.p[i].recv && m.b.owner < 0 {
				want = m.think(i, dt) // runs to meet the pass by itself
			}
		} else {
			want = m.think(i, dt)
		}
		m.move(i, want, dt)
	}
	m.in = [2]input{}
	m.separate()
	m.tackles(dt)
	m.moveBall(dt)
	if m.state == stPlay {
		m.saves()
		m.touches()
	}
	if m.b.owner >= 0 {
		m.b.held += dt
	}
}

// move runs player i towards want (a direction, length up to 1).
func (m *match) move(i int, want vec, dt float32) {
	p := &m.p[i]
	speed := p.runSpeed()
	if m.b.owner == i {
		speed *= 0.82
	}
	if p.lunge > 0 {
		want, speed = p.face, speed*1.6
	}
	if p.stun > 0 {
		want = vec{}
	}
	if l := want.len(); l > 1 {
		want = want.mul(1 / l)
	}
	p.vel = p.vel.add(want.mul(speed).sub(p.vel).mul(min(1, 9*dt)))
	p.pos = p.pos.add(p.vel.mul(dt))
	p.pos.x = clamp(p.pos.x, fieldL+3, fieldR-3)
	p.pos.y = clamp(p.pos.y, fieldT+3, fieldB)
	if s := p.vel.len(); s > 8 {
		if p.lunge <= 0 {
			p.face = p.vel.mul(1 / s)
		}
		p.step += dt * s / 9
	} else {
		p.step = 0
	}
}

// separate pushes apart players standing on each other.
func (m *match) separate() {
	for i := range m.p {
		for j := i + 1; j < len(m.p); j++ {
			a, b := &m.p[i], &m.p[j]
			d := b.pos.sub(a.pos)
			l := d.len()
			if l >= 7 {
				continue
			}
			if l < 0.01 {
				d, l = vec{1, 0}, 1
			}
			push := d.mul((7 - l) / l / 2)
			a.pos = a.pos.sub(push)
			b.pos = b.pos.add(push)
		}
	}
}

// moveBall moves the ball: with its owner, or rolling, off the boards, into
// a goal.
func (m *match) moveBall(dt float32) {
	b := &m.b
	if b.owner >= 0 {
		o := &m.p[b.owner]
		b.pos = o.pos.add(o.face.mul(5))
		b.vel = o.vel
		return
	}
	b.pos = b.pos.add(b.vel.mul(dt))
	s := b.vel.len()
	if s > 0 {
		ns := max(s-(s*0.9+18)*dt, 0)
		b.vel = b.vel.mul(ns / s)
	}
	if b.pos.y < fieldT {
		b.pos.y, b.vel.y = 2*fieldT-b.pos.y, -b.vel.y*0.6
	} else if b.pos.y > fieldB {
		b.pos.y, b.vel.y = 2*fieldB-b.pos.y, -b.vel.y*0.6
	}
	inMouth := abs32(b.pos.y-midY) < goalHalf-1
	for _, x := range [2]float32{fieldL, fieldR} {
		in := b.pos.x < fieldL && x == fieldL || b.pos.x > fieldR && x == fieldR
		if !in {
			continue
		}
		if !inMouth && m.state != stGoal {
			b.pos.x, b.vel.x = 2*x-b.pos.x, -b.vel.x*0.6
			continue
		}
		// in the goal: the net stops it
		back := x - goalDepth + 2
		if x == fieldR {
			back = x + goalDepth - 2
		}
		if x == fieldL && b.pos.x < back || x == fieldR && b.pos.x > back {
			b.pos.x, b.vel = back, b.vel.mul(0.1)
		}
		b.pos.y = clamp(b.pos.y, midY-goalHalf+2, midY+goalHalf-2)
		if m.state == stPlay {
			t := 0 // the team attacking this goal scores
			if m.goalX(1) == x {
				t = 1
			}
			m.score[t]++
			m.scorer = b.last
			m.kickoff = 1 - t
			m.state, m.stateT = stGoal, 0
		}
	}
}

// kick sends the ball from player i towards to at speed, sd radians off.
func (m *match) kick(i int, to vec, speed, sd float32, shot bool) {
	b := &m.b
	d := to.sub(b.pos).norm()
	if d == (vec{}) {
		d = m.p[i].face
	}
	b.vel = d.rotate(gauss(m.r, sd)).mul(speed)
	b.owner, b.last, b.shot = -1, i, shot
	b.kicks++
	m.p[i].touchCD = 0.25
	for k := range m.p {
		m.p[k].recv = false
	}
	if shot {
		m.shots[m.p[i].team]++
	}
}

// pass plays the ball from i to the teammate to, ahead of it.
func (m *match) pass(i, to int) {
	r := &m.p[to]
	target := r.pos.add(r.vel.mul(0.35))
	d := target.dist(m.b.pos)
	speed := clamp(90+d*1.0, 120, 250)
	m.kick(i, target, speed, 0.02+float32(8-m.p[i].d.tech)*0.012, false)
	r.recv = true
	if t := r.team; m.human[t] && !r.gk {
		m.ctl[t] = to
	}
}

// shoot shoots from i at the goal line, at height y of the mouth, with
// power 0..1.
func (m *match) shoot(i int, y, power float32) {
	p := &m.p[i]
	gx := m.goalX(p.team) + m.dir[p.team]*2
	speed := 170 + 120*power + float32(p.d.kick)*5
	m.kick(i, vec{gx, y}, speed, 0.03+float32(8-p.d.kick)*0.008+power*0.035, true)
}

// passValue rates a pass from i to the teammate j (higher is better; less
// than -50: do not).
func (m *match) passValue(i, j int) float32 {
	from, to := m.p[i].pos, m.p[j].pos
	t := m.p[i].team
	d := from.dist(to)
	if d < 18 || d > 190 {
		return -100
	}
	open := float32(40)
	for k := range m.p {
		o := &m.p[k]
		if o.team == t {
			continue
		}
		open = min(open, o.pos.dist(to))
		// the lane: how near the line from i to j it stands
		ab := to.sub(from)
		s := clamp(o.pos.sub(from).dot(ab)/ab.dot(ab), 0, 1)
		if lane := o.pos.dist(from.add(ab.mul(s))); lane < 9 && s > 0.1 {
			return -100
		}
	}
	ui, _ := m.uv(t, from)
	uj, _ := m.uv(t, to)
	v := open + (uj-ui)*80 - d*0.05
	if m.p[j].gk {
		v -= 40
	}
	return v
}

// bestPass returns the teammate i passes to best, and how good it is
// (-1: none). With dir set, only those that way.
func (m *match) bestPass(i int, dir vec) (int, float32) {
	best, bv := -1, float32(-50)
	t := m.p[i].team
	for j := t * 5; j < t*5+5; j++ {
		if j == i {
			continue
		}
		v := m.passValue(i, j)
		if dir != (vec{}) {
			c := m.p[j].pos.sub(m.p[i].pos).norm().dot(dir)
			if c < 0.5 {
				continue
			}
			v = max(v, -49) + c*40 // the player chose the direction
		}
		if v > bv {
			best, bv = j, v
		}
	}
	return best, bv
}

// humanPlay applies a human team's buttons to its player i and returns
// where it runs.
func (m *match) humanPlay(i int) vec {
	p := &m.p[i]
	in := m.in[p.team]
	m.in[p.team].pass, m.in[p.team].tackle, m.in[p.team].shoot = false, false, false // once
	b := &m.b
	if b.owner == i {
		switch {
		case in.shoot:
			y := midY + in.move.y*(goalHalf-5)
			if in.move.y == 0 {
				y = midY + gauss(m.r, 6)
			}
			m.shoot(i, y, in.power)
		case in.pass:
			dir := in.move
			if dir == (vec{}) {
				dir = p.face
			}
			if j, _ := m.bestPass(i, dir); j >= 0 {
				m.pass(i, j)
			} else {
				m.kick(i, p.pos.add(dir), 150, 0.05, false)
			}
		}
	} else {
		mates := b.owner >= 0 && m.p[b.owner].team == p.team
		if in.pass && !mates {
			m.ctl[p.team] = m.nearest(p.team, i)
		}
		if in.tackle && p.tackleCD <= 0 && p.stun <= 0 {
			if in.move != (vec{}) {
				p.face = in.move
			}
			p.tackleCD, p.lunge = 0.9, 0.22
		}
	}
	return in.move
}

// nearest returns team t's outfield player nearest the ball but not
// (-1: anyone).
func (m *match) nearest(t, not int) int {
	best, bd := not, float32(1e9)
	for j := t*5 + 1; j < t*5+5; j++ {
		if j == not {
			continue
		}
		if d := m.p[j].pos.dist(m.b.pos); d < bd {
			best, bd = j, d
		}
	}
	return best
}

// tackles resolves the tackles going on, and the ball won by a defender
// at the carrier's feet.
func (m *match) tackles(dt float32) {
	b := &m.b
	if b.owner < 0 {
		return
	}
	o := &m.p[b.owner]
	if o.gk && m.inArea(o.team, o.pos) {
		return // the keeper has it in the hands
	}
	for i := range m.p {
		p := &m.p[i]
		if p.team == o.team || p.stun > 0 {
			continue
		}
		d := p.pos.dist(b.pos)
		win := 0.5 + 0.06*float32(p.d.def-o.d.tech)
		if p.lunge <= 0 {
			if d < 6 && p.touchCD <= 0 && m.r.RandFloat() < contact*win*dt {
				m.steal(i)
				return
			}
			continue
		}
		if d > 9 {
			continue
		}
		p.lunge = 0
		if m.r.RandFloat() < win {
			o.stun, o.touchCD = 0.5, 0.5
			b.owner, b.last = -1, i
			b.kicks++
			b.vel = p.face.mul(60).add(vec{-p.face.y, p.face.x}.mul(gauss(m.r, 30)))
			return
		}
		p.stun = 0.5
	}
}

// contact is how often (a second) a defender at the carrier's feet wins
// the ball, times its chance of winning a tackle.
const contact = 1.5

// steal gives the ball to the defender i, the carrier knocked off it.
func (m *match) steal(i int) {
	o := &m.p[m.b.owner]
	o.stun, o.touchCD = 0.3, 0.5
	m.b.vel = vec{}
	m.take(i)
}

// inArea reports whether pos is in the penalty area team t defends.
func (m *match) inArea(t int, pos vec) bool {
	g := m.ownGoal(t)
	dy := max(abs32(pos.y-midY)-goalHalf, 0)
	dx := abs32(pos.x - g.x)
	return dx*dx+dy*dy < areaR*areaR
}

// saves lets the keepers try to stop shots coming at their goals.
func (m *match) saves() {
	b := &m.b
	s := b.vel.len()
	if b.owner >= 0 || s < 60 {
		return
	}
	for t := 0; t < 2; t++ {
		k := &m.p[t*5]
		if k.saveTry == b.kicks || k.stun > 0 || b.vel.x*m.dir[t] > 0 || m.p[b.last].team == t && !b.shot {
			continue
		}
		reach := 6 + float32(k.d.def)*0.6
		if k.pos.dist(b.pos) > reach || !m.inArea(t, k.pos) {
			continue
		}
		k.saveTry = b.kicks
		chance := clamp(0.64+float32(k.d.def)*0.05-(s-200)/320, 0.15, 0.95)
		if m.r.RandFloat() >= chance {
			k.stun = 0.4 // dived the wrong way
			continue
		}
		if s < 230 {
			m.take(t * 5)
		} else { // parried out
			b.vel = vec{m.dir[t] * abs32(b.vel.x) * 0.35, b.vel.y*0.3 + gauss(m.r, 70)}
			b.last, b.shot = t*5, false
			b.kicks++
			k.touchCD = 0.4
		}
		return
	}
}

// touches gives the free ball to a player who reaches it.
func (m *match) touches() {
	b := &m.b
	if b.owner >= 0 {
		return
	}
	best, bd := -1, float32(1e9)
	for i := range m.p {
		p := &m.p[i]
		r := float32(6)
		if p.gk && m.inArea(p.team, p.pos) {
			r = 8
		}
		d := p.pos.dist(b.pos)
		if d < r && p.touchCD <= 0 && p.stun <= 0 && d < bd {
			best, bd = i, d
		}
	}
	if best < 0 {
		return
	}
	p := &m.p[best]
	s := b.vel.len()
	if !p.gk && (b.shot && s > 170 || s > 260) && m.r.RandFloat() < 0.5 {
		// too fast to control: it bounces off
		b.vel = b.vel.mul(0.4).rotate(gauss(m.r, 0.8))
		b.last = best
		b.kicks++
		p.touchCD = 0.3
		return
	}
	m.take(best)
}

// take gives the ball to player i.
func (m *match) take(i int) {
	b := &m.b
	p := &m.p[i]
	prev := -1
	if b.last >= 0 {
		prev = m.p[b.last].team
	}
	b.owner, b.last, b.held, b.shot = i, i, 0, false
	p.think = 0.15 + m.r.RandFloat()*0.2
	for k := range m.p {
		m.p[k].recv = false
	}
	if b.vel.len() > 5 {
		p.face = b.vel.norm().mul(-1)
		if p.gk {
			p.face = vec{m.dir[p.team], 0}
		}
	}
	t := p.team
	if m.human[t] && !p.gk {
		m.ctl[t] = i
	}
	if prev >= 0 && prev != t && m.human[1-t] {
		// the human's side lost it: control the one nearest the ball
		m.ctl[1-t] = m.nearest(1-t, -1)
	}
}
