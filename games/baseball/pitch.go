package baseball

// The duel at the plate. The strike zone is the square |u| <= 1, |v| <= 1
// (u to the right as the catcher sees it, v up); a pitch crosses the plate
// at a point of that plane.

// The pitches.
const (
	ptFast = iota
	ptSlider
	ptCurve
	ptShoot
	ptFork
	numPitches
)

// pitchInfo is how each pitch moves (for a right-handed pitcher; a
// left-hander's breaks the other way).
var pitchInfo = [numPitches]struct {
	name   string
	speed  float32 // of the fastball
	bu, bv float32 // the break at level 4, in zone units
}{
	ptFast:   {"STRAIGHT", 1, 0, 0.12},
	ptSlider: {"SLIDER", 0.9, 0.95, -0.15},
	ptCurve:  {"CURVE", 0.8, 0.55, -1.15},
	ptShoot:  {"SHOOT", 0.94, -0.75, -0.2},
	ptFork:   {"FORK", 0.88, 0, -1.25},
}

// ballR is the radius of the ball in zone units, for the strike zone and
// the cursor.
const ballR = 0.12

// slowdown stretches the time of a pitch (a real one takes 0.4 s).
const slowdown = 1.7

// pitch is a pitch on its way.
type pitch struct {
	kind   int
	speed  float32 // km/h
	aim    vec     // where the pitcher aimed (zone units)
	end    vec     // where it crosses the plate
	brk    vec     // its break
	flight float32 // from the release to the plate (s)
}

// rnd is the random numbers (picopad.Console has them).
type rnd interface {
	RandFloat() float32
}

// gauss returns a random number with mean 0 and deviation about sd.
func gauss(r rnd, sd float32) float32 {
	return (r.RandFloat() + r.RandFloat() + r.RandFloat() - 1.5) * 2 * sd
}

// throwPitch makes a pitch of kind (level 1..7) aimed at aim.
func throwPitch(r rnd, p *pitcher, kind, level int, aim vec) pitch {
	info := &pitchInfo[kind]
	k := float32(level) / 4
	pt := pitch{kind: kind, speed: p.velocity * info.speed, aim: aim}
	pt.speed += gauss(r, 1.5)
	pt.brk = vec{info.bu * k, info.bv * k}
	if kind == ptFast {
		pt.brk = vec{info.bu, info.bv}
	}
	miss := 0.32 - float32(p.control)*0.03 // the control
	pt.end = aim.add(vec{gauss(r, miss), gauss(r, miss)})
	pt.flight = 0.4 * 145 / pt.speed * slowdown
	return pt
}

// at returns where the pitch seems to be heading at t (0..1 of its flight):
// the point of the plate it would reach, which moves as it breaks.
func (p *pitch) at(t float32) vec {
	straight := p.end.sub(p.brk)
	return straight.add(p.brk.mul(t * t))
}

// strike reports whether the pitch crosses the strike zone.
func (p *pitch) strike() bool {
	return abs32(p.end.x) <= 1+ballR && abs32(p.end.y) <= 1+ballR
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// swing is what the batter did.
type swing struct {
	swung bool
	t     float32 // when (s after the release)
	at    vec     // the center of the cursor
	power bool    // a power swing: a smaller cursor, harder hit
}

// cursorSize is the half width and half height of the batter's cursor.
func cursorSize(b *player, power bool) vec {
	k := 0.8 + float32(b.contact)*0.06 // 0.86 .. 1.28
	if power {
		return vec{0.3 * k, 0.24 * k}
	}
	return vec{0.48 * k, 0.38 * k}
}

// window is how far from the right moment a swing can still hit (s).
func window(b *player, power bool) float32 {
	w := 0.07 + float32(b.contact)*0.006
	if power {
		w *= 0.75
	}
	return w
}

// The results of a pitch.
const (
	pitchBall = iota
	pitchStrike
	pitchSwingMiss
	pitchFoul
	pitchInPlay
)

// result returns what the pitch and the swing come to, and the ball off
// the bat for pitchInPlay.
func result(r rnd, p *pitch, sw swing, b *player) (int, hitBall) {
	if !sw.swung {
		if p.strike() {
			return pitchStrike, hitBall{}
		}
		return pitchBall, hitBall{}
	}
	cs := cursorSize(b, sw.power)
	d := p.end.sub(sw.at)
	if abs32(d.x) > cs.x+ballR || abs32(d.y) > cs.y+ballR {
		return pitchSwingMiss, hitBall{}
	}
	w := window(b, sw.power)
	te := (sw.t - p.flight) / w
	if te < -1 || te > 1 {
		return pitchSwingMiss, hitBall{}
	}
	dx, dy := d.x/(cs.x+ballR), d.y/(cs.y+ballR)
	q := 1 - 0.55*sqrt32(dx*dx+dy*dy) - 0.4*abs32(te)
	if q < 0.45 {
		return pitchFoul, hitBall{}
	}
	speed := 84 + float32(b.power)*3.5
	if sw.power {
		speed += 10
	}
	hb := hitBall{
		speed:  speed*(0.6+0.4*q) + gauss(r, 3),
		launch: 12 + 34*dy + gauss(r, 6),
		dir:    55*te + 12*p.end.x + gauss(r, 7),
	}
	if b.left { // the other way round: early is to the right
		hb.dir = -55*te + 12*p.end.x + gauss(r, 7)
	}
	if hb.foul() {
		return pitchFoul, hb
	}
	return pitchInPlay, hb
}

// count is the balls, strikes and outs.
type count struct {
	balls, strikes, outs int
}

// cpuPitch chooses the computer's pitch: the kind, its level and where to.
func cpuPitch(r rnd, p *pitcher, c count) (kind, level int, aim vec) {
	kind, level = ptFast, 4
	if r.RandFloat() < 0.5 {
		n := 0
		for _, pl := range p.pitches {
			if pl.level > 0 {
				n++
			}
		}
		if n > 0 {
			pl := p.pitches[int(r.RandFloat()*float32(n))%n]
			kind, level = int(pl.kind), int(pl.level)
		}
	}
	// the corners, and off the plate when ahead in the count
	sx, sy := float32(1), float32(1)
	if r.RandFloat() < 0.5 {
		sx = -1
	}
	if r.RandFloat() < 0.5 {
		sy = -1
	}
	edge := float32(0.95)
	if c.strikes == 2 && c.balls < 3 {
		edge = 1.2
	} else if c.balls == 3 {
		edge = 0.8
	}
	aim = vec{sx * (edge + gauss(r, 0.3)), sy * (edge + gauss(r, 0.3))}
	if kind == ptFork || kind == ptCurve {
		aim.y = 0.4 + gauss(r, 0.25) // it falls into the zone
	}
	return kind, level, aim
}

// cpuSwing decides the computer batter's swing at p.
func cpuSwing(r rnd, p *pitch, b *player, c count) swing {
	// patient early in the count, protecting the plate with two strikes
	want := [3]float32{0.14, 0.2, 0.3}[c.strikes] // chasing a ball
	if p.strike() {
		want = [3]float32{0.45, 0.62, 0.82}[c.strikes]
	}
	if c.balls == 3 && c.strikes < 2 {
		want -= 0.25
	}
	if p.kind != ptFast && !p.strike() {
		want += 0.12 // fooled by the break
	}
	if r.RandFloat() >= want {
		return swing{}
	}
	skill := 1.6 - float32(b.contact)*0.12 // 1.5 .. 0.64
	te := gauss(r, 0.055*skill)
	if p.kind != ptFast {
		te -= 0.03 // early on the slower pitches
	}
	guess := p.end
	if p.kind != ptFast {
		guess = p.end.sub(p.brk.mul(0.3 * skill)) // misreads part of the break
	}
	sw := swing{
		swung: true,
		t:     p.flight + te,
		at:    guess.add(vec{gauss(r, 0.22*skill), gauss(r, 0.22*skill)}),
		power: b.power >= 6 && c.strikes < 2 && r.RandFloat() < 0.5,
	}
	return sw
}
