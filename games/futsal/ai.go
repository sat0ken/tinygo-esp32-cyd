package futsal

// The computer's players.

// think decides where computer player i runs, and kicks or tackles.
func (m *match) think(i int, dt float32) vec {
	p := &m.p[i]
	b := &m.b
	if p.gk {
		return m.keeper(i, dt)
	}
	if b.owner == i {
		return m.carry(i, dt)
	}
	t := p.team
	ours := b.owner >= 0 && m.p[b.owner].team == t
	if p.recv && b.owner < 0 {
		// meet the pass on its line
		d := b.vel.norm()
		s := max(p.pos.sub(b.pos).dot(d), 0)
		return m.toward(p, b.pos.add(d.mul(s)), 2)
	}
	if !ours {
		first, second := m.chasers(t)
		switch i {
		case first:
			to := b.pos.add(b.vel.mul(0.25))
			if b.owner >= 0 && p.pos.dist(b.pos) < 10 && p.tackleCD <= 0 && p.stun <= 0 && m.r.RandFloat() < 4*dt {
				p.face = b.pos.sub(p.pos).norm()
				p.tackleCD, p.lunge = 1.1, 0.22
			}
			return m.toward(p, to, 1)
		case second:
			if b.owner >= 0 { // cover: between the ball and the goal
				g := m.ownGoal(t)
				if p.pos.dist(b.pos) < 24 { // close in when the carrier comes
					return m.toward(p, b.pos, 1)
				}
				return m.toward(p, b.pos.add(g.sub(b.pos).norm().mul(26)), 4)
			}
		}
	}
	return m.toward(p, m.place(i, ours), 6)
}

// toward runs p to the point to, slowing down within slow pixels.
func (m *match) toward(p *player, to vec, slow float32) vec {
	d := to.sub(p.pos)
	l := d.len()
	if l < 1 {
		return vec{}
	}
	return d.mul(min(1, l/slow) / l)
}

// chasers returns the outfield players of team t nearest to the ball
// (the first goes for it, the second covers).
func (m *match) chasers(t int) (int, int) {
	first, second := -1, -1
	var d1, d2 float32 = 1e9, 1e9
	to := m.b.pos.add(m.b.vel.mul(0.3))
	for j := t*5 + 1; j < t*5+5; j++ {
		p := &m.p[j]
		d := p.pos.dist(to) / p.runSpeed()
		if p.stun > 0 {
			d += 0.5
		}
		switch {
		case d < d1:
			second, d2 = first, d1
			first, d1 = j, d
		case d < d2:
			second, d2 = j, d
		}
	}
	return first, second
}

// place is where outfield player i stands in the formation: moved with
// the ball, higher up when its team has the ball, away from markers.
func (m *match) place(i int, ours bool) vec {
	p := &m.p[i]
	t := p.team
	f := formation[i%5]
	bu, bv := m.uv(t, m.b.pos)
	u := f.x + (bu-0.5)*0.55
	if ours {
		u += 0.1
	} else {
		u -= 0.06
	}
	v := f.y*0.9 + bv*0.25
	pos := m.at(t, clamp(u, 0.08, 0.9), clamp(v, -0.9, 0.9))
	if ours { // get free of the nearest opponent
		for k := (1 - t) * 5; k < (1-t)*5+5; k++ {
			if d := pos.sub(m.p[k].pos); d.len() < 18 {
				pos = pos.add(d.norm().mul(18 - d.len()))
			}
		}
	}
	return pos
}

// carry is the computer with the ball: dribble to the goal, pass, shoot.
func (m *match) carry(i int, dt float32) vec {
	p := &m.p[i]
	t := p.team
	b := &m.b
	goal := m.oppGoal(t)
	p.think -= dt
	if p.think <= 0 && m.state == stPlay {
		p.think = 0.2
		d := p.pos.dist(goal)
		press, ahead := m.pressure(i)
		switch {
		case b.held > 0.3 && (d < 78 || d < 125 && m.r.RandFloat() < 0.1):
			m.shoot(i, m.aim(t), clamp((d-40)/90, 0.3, 1))
			return vec{}
		case b.held > 0.2 && (press < 14 && m.r.RandFloat() < 0.6 || ahead && m.r.RandFloat() < 0.15 || m.r.RandFloat() < 0.08):
			if j, v := m.bestPass(i, vec{}); j >= 0 && v > 0 {
				m.pass(i, j)
				return vec{}
			}
		}
	}
	// dribble: to the goal, around the opponents ahead
	want := goal.sub(p.pos).norm()
	for k := (1 - t) * 5; k < (1-t)*5+5; k++ {
		d := m.p[k].pos.sub(p.pos)
		l := d.len()
		if l > 35 || d.dot(want) < 0 {
			continue
		}
		side := vec{-want.y, want.x}
		if d.dot(side) > 0 {
			side = side.mul(-1)
		}
		want = want.add(side.mul((35 - l) / 25))
	}
	// keep off the boards
	if p.pos.y < fieldT+14 {
		want.y += 0.6
	} else if p.pos.y > fieldB-14 {
		want.y -= 0.6
	}
	return want.norm()
}

// pressure returns how near the nearest opponent of i is, and whether one
// stands in front of it.
func (m *match) pressure(i int) (float32, bool) {
	p := &m.p[i]
	t := p.team
	near, ahead := float32(1e9), false
	fwd := vec{m.dir[t], 0}
	for k := (1 - t) * 5; k < (1-t)*5+5; k++ {
		d := m.p[k].pos.sub(p.pos)
		l := d.len()
		near = min(near, l)
		if l < 28 && d.dot(fwd) > l*0.6 {
			ahead = true
		}
	}
	return near, ahead
}

// aim chooses where in the goal mouth team t shoots: the side the keeper
// leaves open.
func (m *match) aim(t int) float32 {
	k := &m.p[(1-t)*5]
	off := float32(goalHalf - 5)
	if m.r.RandFloat() < 0.25 {
		off = m.r.RandFloat() * off
	}
	if k.pos.y > midY {
		return midY - off
	}
	return midY + off
}

// keeper is the goalkeeper: on the line between the ball and the goal,
// to the shots, out for loose balls in the area, and throwing the ball out.
func (m *match) keeper(i int, dt float32) vec {
	p := &m.p[i]
	t := p.team
	b := &m.b
	g := m.ownGoal(t)
	if b.owner == i {
		if b.held > 0.9 && m.state == stPlay {
			if j, _ := m.bestPass(i, vec{}); j >= 0 {
				m.pass(i, j)
			} else {
				m.kick(i, m.at(t, 0.5, gauss(m.r, 0.5)), 200, 0.1, false)
			}
		}
		return m.toward(p, g.add(vec{m.dir[t] * 10, 0}), 4)
	}
	if b.owner < 0 && m.inArea(t, b.pos) && b.vel.len() < 120 {
		return m.toward(p, b.pos, 1)
	}
	y := midY + (b.pos.y-midY)*0.35
	y = clamp(y, midY-goalHalf+3, midY+goalHalf-3)
	out := clamp(28-abs32(b.pos.x-g.x)/6, 4, 16)
	if b.owner < 0 && b.vel.x*m.dir[t] < 0 && abs32(b.vel.x) > 50 {
		// a shot or a pass coming: where it crosses the line
		tt := (g.x - b.pos.x) / b.vel.x
		yc := b.pos.y + b.vel.y*tt
		if abs32(yc-midY) < goalHalf+8 {
			y = clamp(yc, midY-goalHalf-2, midY+goalHalf+2)
			out = 6
		}
	}
	return m.toward(p, vec{g.x + m.dir[t]*out, y}, 3)
}
