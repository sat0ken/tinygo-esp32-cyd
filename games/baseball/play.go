package baseball

import "math"

// The field is in feet: home plate at (0, 0), x to the right as seen from
// behind home plate (the first-base side), y toward center field.

type vec struct{ x, y float32 }

func (a vec) add(b vec) vec           { return vec{a.x + b.x, a.y + b.y} }
func (a vec) sub(b vec) vec           { return vec{a.x - b.x, a.y - b.y} }
func (a vec) mul(k float32) vec       { return vec{a.x * k, a.y * k} }
func (a vec) len() float32            { return sqrt32(a.x*a.x + a.y*a.y) }
func dist(a, b vec) float32           { return a.sub(b).len() }
func lerp(a, b vec, t float32) vec    { return a.add(b.sub(a).mul(t)) }
func sqrt32(v float32) float32        { return float32(math.Sqrt(float64(v))) }
func clamp(v, lo, hi float32) float32 { return min(max(v, lo), hi) }

// basePos are home plate, first, second and third base.
var basePos = [4]vec{{0, 0}, {63.64, 63.64}, {0, 127.28}, {-63.64, 63.64}}

// The positions of the fielders.
const (
	posP = iota
	posC
	pos1B
	pos2B
	pos3B
	posSS
	posLF
	posCF
	posRF
	numPos
)

// posName are the positions as a scorer writes them.
var posName = [numPos]string{"P", "C", "1B", "2B", "3B", "SS", "LF", "CF", "RF"}

// spot is where each fielder stands before the pitch.
var spot = [numPos]vec{
	posP: {0, 60.5}, posC: {0, -4},
	pos1B: {70, 88}, pos2B: {36, 142}, pos3B: {-68, 88}, posSS: {-38, 142},
	posLF: {-150, 262}, posCF: {0, 305}, posRF: {150, 262},
}

// fenceDist is the distance to the fence in direction dir (degrees from
// center field, negative to the left): 330 ft at the lines, 400 in center.
func fenceDist(dir float32) float32 {
	return 330 + 70*float32(math.Cos(2*float64(dir)*math.Pi/180))
}

const fenceHeight = 10

// direction returns the direction of p from home plate in degrees from
// center field.
func direction(p vec) float32 {
	return float32(math.Atan2(float64(p.x), float64(p.y)) * 180 / math.Pi)
}

// fielder is a fielder's ability: running speed and throwing speed (ft/s).
type fielder struct {
	speed, arm float32
}

// hitBall is a ball off the bat.
type hitBall struct {
	speed  float32 // mph
	launch float32 // degrees above the ground
	dir    float32 // degrees from center field, negative to the left
}

// foul reports whether the ball goes foul (outside the lines).
func (b hitBall) foul() bool { return b.dir < -45 || b.dir > 45 }

// sample is the ball at a moment of its flight.
type sample struct {
	p vec
	z float32 // height (ft)
}

const (
	sampleT    = float32(1) / 30 // between samples (s)
	maxSamples = 360             // 12 s
	gravity    = 32.2
	dragK      = 0.0011 // air resistance per (ft/s) of speed
	rollFric   = 16     // deceleration of a rolling ball (ft/s²)
)

// flight is the path of a batted ball until a fielder takes it.
type flight struct {
	s       [maxSamples]sample
	n       int
	landed  int // sample of the first bounce (-1: none)
	hr      bool
	hrAt    int // the sample where it passes the fence
	stopped int // the sample where it stops rolling (-1: still moving)
}

// fly computes the path of b (it fills f.s with up to maxSamples samples).
func (f *flight) fly(b hitBall) {
	v := b.speed * 1.4667 // ft/s
	la := float64(b.launch) * math.Pi / 180
	dr := float64(b.dir) * math.Pi / 180
	vh := v * float32(math.Cos(la))
	vx, vy := vh*float32(math.Sin(dr)), vh*float32(math.Cos(dr))
	vz := v * float32(math.Sin(la))
	x, y, z := float32(0), float32(0), float32(3)
	f.n, f.landed, f.hr, f.hrAt, f.stopped = 0, -1, false, -1, -1
	bounces := 0
	wall := false
	const sub = 4
	dt := sampleT / sub
	for f.n < maxSamples {
		f.s[f.n] = sample{vec{x, y}, z}
		f.n++
		for k := 0; k < sub; k++ {
			if z > 0 || vz > 0 { // in the air
				sp := sqrt32(vx*vx + vy*vy + vz*vz)
				vx -= dragK * sp * vx * dt
				vy -= dragK * sp * vy * dt
				vz -= (dragK*sp*vz + gravity) * dt
				x, y, z = x+vx*dt, y+vy*dt, z+vz*dt
				if z <= 0 && vz < 0 { // a bounce
					z = 0
					if f.landed < 0 {
						f.landed = f.n
					}
					bounces++
					keep := float32(0.85)
					if bounces == 1 {
						keep = 0.8
					}
					vx, vy = vx*keep, vy*keep
					vz = -vz * 0.35
					if vz < 5 {
						vz = 0
					}
				}
			} else { // rolling
				sp := sqrt32(vx*vx + vy*vy)
				if sp <= rollFric*dt {
					vx, vy = 0, 0
					if f.stopped < 0 {
						f.stopped = f.n
					}
				} else {
					k := (sp - rollFric*dt) / sp
					vx, vy = vx*k, vy*k
				}
				x, y = x+vx*dt, y+vy*dt
			}
			// the fence
			if d := sqrt32(x*x + y*y); !wall && d >= fenceDist(direction(vec{x, y})) {
				if z > fenceHeight && f.landed < 0 {
					f.hr = true
					f.hrAt = f.n
				} else if !f.hr {
					wall = true // off the wall: back toward the field
					vx, vy = -vx*0.3, -vy*0.3
				}
			}
		}
		if f.hr && f.n > f.hrAt+20 {
			return
		}
	}
}

// carry is how far b flies before it lands (or 0 if it does not), for
// tuning and tests.
func carry(b hitBall) float32 {
	var f flight
	f.fly(b)
	if f.landed < 0 {
		return f.s[f.n-1].p.len()
	}
	return f.s[f.landed].p.len()
}

// runner is a runner on a base (or the batter).
type runner struct {
	on    bool
	id    int8    // the batter in the batting order
	speed float32 // ft/s
}

// situation is what the play starts from.
type situation struct {
	outs   int
	bases  [3]runner // first, second, third
	batter runner
}

// The results of a ball in play.
const (
	resGroundOut = iota
	resFlyOut
	resLineOut
	resPopOut
	resDoublePlay
	resFieldersChoice
	resSacFly
	resInfieldHit
	resSingle
	resDouble
	resTriple
	resHomeRun
	resInsideHR
)

// resultText is what the screen says.
var resultText = [...]string{
	resGroundOut: "GROUND OUT", resFlyOut: "FLY OUT", resLineOut: "LINE OUT", resPopOut: "POP OUT",
	resDoublePlay: "DOUBLE PLAY", resFieldersChoice: "FIELDER'S CHOICE", resSacFly: "SACRIFICE FLY",
	resInfieldHit: "INFIELD HIT", resSingle: "SINGLE", resDouble: "DOUBLE", resTriple: "TRIPLE",
	resHomeRun: "HOME RUN!", resInsideHR: "HOME RUN!",
}

// isHit reports whether a result is a hit.
func isHit(r int) bool { return r >= resInfieldHit }

// run is a runner going from base to base during the play (0 home ... 4
// home again).
type run struct {
	id       int8
	from, to int8
	t0, t1   float32
	out      bool // put out on arriving
}

// throw is the ball thrown from one place to another.
type throw struct {
	from, to vec
	t0, t1   float32
}

// play is a ball in play and what comes of it.
type play struct {
	ball   hitBall
	fl     flight
	who    int     // the fielder who gets the ball (-1: home run)
	fieldT float32 // when
	fieldP vec     // where
	caught bool    // on the fly
	cover  [numPos]vec

	throws  [3]throw
	nThrows int
	runs    [4]run
	nRuns   int

	result int
	after  situation // outs and bases after the play
	scored int       // runs
	end    float32   // the time the play is over
}

// reaction is how long a fielder takes to start (s).
func reaction(pos int) float32 {
	switch pos {
	case posP:
		return 0.55
	case posC:
		return 0.5
	case posLF, posCF, posRF:
		return 0.4
	}
	return 0.3
}

// reachTime is how long a fielder takes to get to a ball d feet away: he
// starts, speeds up, and reaches out with the glove.
func reachTime(pos int, d, speed float32) float32 {
	d = max(d-4, 0)
	if d == 0 {
		return reaction(pos)
	}
	return reaction(pos) + 0.3 + d/speed
}

// resolve plays the ball b against the fielders def from the situation
// sit.
func (pl *play) resolve(b hitBall, sit situation, def *[numPos]fielder) {
	*pl = play{ball: b, who: -1}
	pl.fl.fly(b)
	pl.cover = spot
	fl := &pl.fl

	if fl.hr {
		pl.result = resHomeRun
		pl.homeRun(sit, float32(fl.hrAt)*sampleT)
		return
	}

	// the first moment a fielder can reach the ball, low enough to take it
	k := 0
	for ; k < maxSamples; k++ {
		i := min(k, fl.n-1)
		s := fl.s[i]
		t := float32(k) * sampleT
		best := -1
		for pos := range def {
			if pos == posC && s.p.len() > 40 {
				continue
			}
			reach := float32(8)
			if pos == posP {
				reach = 6
			}
			if s.z > reach {
				continue
			}
			if reachTime(pos, dist(spot[pos], s.p), def[pos].speed) <= t {
				if best < 0 || dist(spot[pos], s.p) < dist(spot[best], s.p) {
					best = pos
				}
			}
		}
		if best >= 0 {
			pl.who, pl.fieldT, pl.fieldP = best, t, s.p
			pl.caught = s.z > 0 && (fl.landed < 0 || k < fl.landed)
			break
		}
	}
	if pl.who < 0 { // nobody: the ball rolls dead
		pl.who, pl.fieldT, pl.fieldP = posCF, float32(maxSamples)*sampleT, fl.s[fl.n-1].p
	}
	pl.cover[pl.who] = pl.fieldP

	after := sit
	after.batter = runner{}
	switch {
	case pl.caught:
		pl.flyOut(sit, &after)
	case pl.who <= posSS: // an infielder, the pitcher or the catcher has it
		pl.infield(sit, &after, def)
	default:
		pl.outfield(sit, &after, def)
	}
	pl.after = after
	if pl.after.outs >= 3 {
		pl.scored = 0 // the third out: no runs count
	}
}

// addRun adds a runner's move.
func (pl *play) addRun(id int8, from, to int8, t0, speed float32, out bool) float32 {
	t1 := t0 + float32(to-from)*90/speed + 0.25*float32(to-from-1)
	if pl.nRuns < len(pl.runs) {
		pl.runs[pl.nRuns] = run{id, from, to, t0, t1, out}
		pl.nRuns++
	}
	pl.end = max(pl.end, t1)
	return t1
}

// addThrow adds a throw and returns when it arrives.
func (pl *play) addThrow(from, to vec, t0, arm float32) float32 {
	t1 := t0 + dist(from, to)/arm
	if pl.nThrows < len(pl.throws) {
		pl.throws[pl.nThrows] = throw{from, to, t0, t1}
		pl.nThrows++
	}
	pl.end = max(pl.end, t1)
	return t1
}

// homeRun: everybody scores.
func (pl *play) homeRun(sit situation, t float32) {
	for b := 2; b >= 0; b-- {
		if r := sit.bases[b]; r.on {
			pl.addRun(r.id, int8(b+1), 4, 0.3, r.speed*0.8, false)
			pl.scored++
		}
	}
	pl.addRun(sit.batter.id, 0, 4, 0.8, sit.batter.speed*0.8, false)
	pl.scored++
	pl.after = situation{outs: sit.outs}
	pl.end = max(pl.end, t+1)
}

// flyOut: the ball is caught; a runner on third tags up and scores on a deep
// fly with fewer than two outs.
func (pl *play) flyOut(sit situation, after *situation) {
	after.outs++
	switch {
	case pl.ball.launch > 50 && pl.fieldP.len() < 160:
		pl.result = resPopOut
	case pl.ball.launch < 16:
		pl.result = resLineOut
	default:
		pl.result = resFlyOut
	}
	d := pl.fieldP.len()
	pl.end = pl.fieldT + 0.8
	if after.outs < 3 {
		if r := sit.bases[2]; r.on && d > 210 && pl.who >= posLF {
			pl.addRun(r.id, 3, 4, pl.fieldT, r.speed, false)
			after.bases[2] = runner{}
			pl.scored++
			pl.result = resSacFly
		}
		if r := sit.bases[1]; r.on && d > 260 && pl.who >= posLF && !after.bases[2].on {
			pl.addRun(r.id, 2, 3, pl.fieldT, r.speed, false)
			after.bases[1], after.bases[2] = runner{}, r
		}
	}
}

const transfer = 0.6 // from fielding to throwing (s)

// infield: a ground ball (or a ball that fell in) taken by an infielder: a
// throw to first, or to second for a double play.
func (pl *play) infield(sit situation, after *situation, def *[numPos]fielder) {
	bt := sit.batter
	tBatter := 0.8 + 90/bt.speed // to first base
	arm := def[pl.who].arm
	t0 := pl.fieldT + transfer + dist(spot[pl.who], pl.fieldP)*0.012 // set his feet after running
	if pl.who == posP || pl.who == posC {
		t0 -= 0.1
	}

	// the double play: to second, then to first
	if r1 := sit.bases[0]; r1.on && sit.outs < 2 {
		tRunner := 0.3 + 90/r1.speed
		cover := posSS
		if pl.who == posSS || pl.fieldP.x < -10 {
			cover = pos2B
		}
		pl.cover[cover] = basePos[2]
		if t2 := t0 + dist(pl.fieldP, basePos[2])/arm; t2 < tRunner {
			pl.addThrow(pl.fieldP, basePos[2], t0, arm)
			pl.addRun(r1.id, 1, 2, 0.3, r1.speed, true)
			after.outs++
			pl.cover[pos1B] = basePos[1]
			t1 := t2 + 0.5 + 90*1.414/def[cover].arm
			out := t1 < tBatter && after.outs < 3
			pl.addThrow(basePos[2], basePos[1], t2+0.5, def[cover].arm)
			pl.addRun(bt.id, 0, 1, 0.8, bt.speed, out)
			after.bases[0] = runner{}
			if out {
				after.outs++
				pl.result = resDoublePlay
			} else {
				after.bases[0] = bt
				pl.result = resFieldersChoice
			}
			pl.advanceOthers(sit, after, false)
			return
		}
	}

	// to first
	pl.cover[pos1B] = basePos[1]
	if pl.who == pos1B {
		pl.cover[posP] = basePos[1]
		arm = 60 // an underhand toss
	}
	t1 := t0 + dist(pl.fieldP, basePos[1])/arm
	pl.addThrow(pl.fieldP, basePos[1], t0, arm)
	if t1 < tBatter {
		pl.addRun(bt.id, 0, 1, 0.8, bt.speed, true)
		after.outs++
		pl.result = resGroundOut
		if after.outs < 3 { // the runners move up while the throw goes to first
			pl.advanceOthers(sit, after, true)
		}
		return
	}
	pl.addRun(bt.id, 0, 1, 0.8, bt.speed, false)
	pl.result = resInfieldHit
	pl.advanceOthers(sit, after, false)
	after.bases[0] = bt
}

// advanceOthers moves the runners already on base: one base each if all
// (on a ground out to first), else only those forced.
func (pl *play) advanceOthers(sit situation, after *situation, all bool) {
	forced := [3]bool{true, sit.bases[0].on, sit.bases[0].on && sit.bases[1].on}
	for b := 2; b >= 0; b-- {
		r := sit.bases[b]
		if !r.on || b == 0 && after.bases[0] != r { // the runner from first was put out
			continue
		}
		if !all && !forced[b] {
			continue
		}
		pl.addRun(r.id, int8(b+1), int8(b+2), 0.3, r.speed, false)
		after.bases[b] = runner{}
		if b == 2 {
			pl.scored++
		} else {
			after.bases[b+1] = r
		}
	}
}

// outfield: a hit; the batter runs as far as he can before the ball gets
// back in, and the runners ahead of him too.
func (pl *play) outfield(sit situation, after *situation, def *[numPos]fielder) {
	arm := def[pl.who].arm
	t0 := pl.fieldT + transfer + 0.5 // an outfielder stops, picks up the ball and turns
	ballAt := func(base int) float32 {
		d := dist(pl.fieldP, basePos[base%4])
		t := t0 + d/arm
		if d > 200 { // a relay
			t += 0.5
		}
		return t
	}
	bt := sit.batter
	runT := func(r runner, from, to int, start float32) float32 {
		return start + float32(to-from)*90/r.speed + 0.25*float32(to-from-1)
	}
	bases := 1
	for k := 2; k <= 4; k++ {
		if runT(bt, 0, k, 0.8)+0.3 < ballAt(k) {
			bases = k
		}
	}

	// the runners, the lead one first
	taken := [5]bool{} // bases 1..4 taken by a runner after the play
	for b := 2; b >= 0; b-- {
		r := sit.bases[b]
		if !r.on {
			continue
		}
		from := b + 1
		to := min(from+bases, 4)
		if to < 4 && runT(r, from, to+1, 0.3)+0.2 < ballAt(to+1) && !taken[to+1] {
			to++
		}
		for to < 4 && taken[to] {
			to++
		}
		pl.addRun(r.id, int8(from), int8(to), 0.3, r.speed, false)
		after.bases[b] = runner{}
		if to >= 4 {
			pl.scored++
		} else {
			after.bases[to-1] = r
			taken[to] = true
		}
	}
	for bases < 4 && taken[bases] {
		bases-- // cannot pass a runner: stay behind
	}
	bases = max(bases, 1)
	pl.addRun(bt.id, 0, int8(bases), 0.8, bt.speed, false)
	if bases == 4 {
		pl.scored++
		pl.result = resInsideHR
	} else {
		after.bases[bases-1] = bt
		pl.result = resSingle + bases - 1
	}
	target := min(bases+1, 4)
	pl.addThrow(pl.fieldP, basePos[target%4], t0, arm)
}
