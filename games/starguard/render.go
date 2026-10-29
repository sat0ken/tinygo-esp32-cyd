package starguard

import "time"

const bandRows = 8 // rows drawn and sent at a time

// shown is an enemy (or explosion) as it is drawn this frame.
type shown struct {
	x, y, w, h int16 // the scaled sprite
	s          *sprite
	z          float32
}

// dot is a star as it is drawn this frame.
type dot struct {
	x, y, size int16
	c          uint16
}

// render draws the view (rows 0 to viewH): the stars, the enemies and the
// mothership far to near, the lasers, the crosshair, the mothership's
// gauge, a red frame after a hit. A band of rows is drawn into one buffer
// while the other one is sent.
func (g *Game) render() {
	l := &g.l
	// The stars: nearer is brighter and bigger.
	nd := 0
	for i := range g.stars {
		s := &g.stars[i]
		sx, sy, _ := g.project(s.x, s.y, s.z, 0)
		if sx < 0 || sy < 0 || sx >= float32(g.w) || sy >= float32(l.viewH) {
			continue
		}
		d := dot{x: int16(sx), y: int16(sy), size: 1}
		switch {
		case s.z < starFar/5:
			d.c, d.size = g.starColors[2], 2
		case s.z < starFar/2:
			d.c = g.starColors[1]
		default:
			d.c = g.starColors[0]
		}
		g.dots[nd] = d
		nd++
	}

	// The enemies, explosions and the mothership in front of the ship,
	// far to near.
	g.nShown = 0
	if b := &g.boss; b.active && b.z > 1 {
		sx, sy, sr := g.project(b.x, b.y, b.z, bossRadius)
		g.addShown(g.mother, sx, sy, 2*1.4*sr, b.z)
	}
	for i := range g.enemies {
		e := &g.enemies[i]
		if (!e.alive && e.boom <= 0) || e.z < 1 {
			continue
		}
		sx, sy, sr := g.project(e.x, e.y, e.z, e.radius())
		s := g.kinds[e.kind]
		if e.boom > 0 {
			f := int((boomTime - e.boom) / boomTime * float32(len(g.boom)))
			if f >= len(g.boom) {
				f = len(g.boom) - 1
			}
			s, sr = g.boom[f], sr*e.big
		}
		g.addShown(s, sx, sy, 2*sr, e.z)
	}
	n := g.nShown

	ax, ay := int16(g.aimX), int16(g.aimY)
	cross := g.crossColor
	if g.target() >= 0 {
		cross = g.lockColor
	}
	for y0 := int16(0); y0 < l.viewH; y0 += bandRows {
		rows := l.viewH - y0
		if rows > bandRows {
			rows = bandRows
		}
		buf, bytes := g.bufs[g.next], g.bytes[g.next]
		clear(buf[:int(rows)*int(g.w)])
		for k := 0; k < nd; k++ {
			d := &g.dots[k]
			for yy := d.y; yy < d.y+d.size; yy++ {
				if yy >= y0 && yy < y0+rows {
					row := buf[int(yy-y0)*int(g.w):][:g.w]
					for xx := d.x; xx < d.x+d.size && xx < g.w; xx++ {
						row[xx] = d.c
					}
				}
			}
		}
		for y := y0; y < y0+rows; y++ {
			row := buf[int(y-y0)*int(g.w):][:g.w]
			for k := 0; k < n; k++ {
				if o := &g.shown[k]; y >= o.y && y < o.y+o.h {
					scaledRow(row, o.s, o.x, o.w, (y-o.y)*o.s.h/o.h)
				}
			}
			if g.laser > 0 && y > ay {
				g.laserRow(row, y, ax, ay)
			}
			if g.screen != menuScreen {
				g.crossRow(row, y, ax, ay, cross)
			}
			if g.boss.active && g.boss.dying <= 0 && y >= 6 && y < 12 {
				g.gaugeRow(row, y-6)
			}
			if g.flash > 0 && (y < 3 || y >= l.viewH-3) {
				for x := range row {
					row[x] = g.flashColor
				}
			} else if g.flash > 0 {
				for x := 0; x < 3; x++ {
					row[x], row[len(row)-1-x] = g.flashColor, g.flashColor
				}
			}
		}
		t := time.Now()
		i := int(rows) * int(g.w)
		switch {
		case g.ad != nil:
			g.ad.DrawRGBBitmap8Async(0, y0, bytes[:i*2], g.w, rows) // waits for the other buffer
		case g.bd != nil:
			g.bd.DrawRGBBitmap8(0, y0, bytes[:i*2], g.w, rows)
		}
		g.sendTime += time.Since(t)
		g.next ^= 1
	}
}

// gaugeRow draws row r (0 to 5) of the mothership's gauge at the top of
// the view: a frame and the hit points left.
func (g *Game) gaugeRow(row []uint16, r int16) {
	x0, x1 := g.w/4, g.w*3/4
	if r == 0 || r == 5 {
		for x := x0; x <= x1; x++ {
			row[x] = g.gaugeEdge
		}
		return
	}
	row[x0], row[x1] = g.gaugeEdge, g.gaugeEdge
	full := x0 + 1 + (x1-x0-1)*int16(g.boss.hp)/int16(g.boss.max)
	for x := x0 + 1; x < full; x++ {
		row[x] = g.gaugeFill
	}
}

// laserRow draws row y of the two beams from the bottom corners of the
// view to the crosshair at (ax, ay).
func (g *Game) laserRow(row []uint16, y, ax, ay int16) {
	bottom := g.l.viewH - 1
	for _, x0 := range [2]int16{0, g.w - 1} {
		// x on the line from (x0, bottom) to (ax, ay), 3 pixels wide.
		x := x0 + int16(int32(ax-x0)*int32(bottom-y)/int32(bottom-ay))
		for k := x - 1; k <= x+1; k++ {
			if k >= 0 && k < g.w {
				row[k] = g.laserColor
			}
		}
	}
}

// crossRow draws row y of the crosshair at (ax, ay): four arms with a gap
// in the middle and a dot.
func (g *Game) crossRow(row []uint16, y, ax, ay int16, c uint16) {
	const arm, gap = 7, 3
	switch d := y - ay; {
	case d == 0:
		for x := ax - gap - arm; x < ax-gap; x++ {
			set(row, x, c)
		}
		for x := ax + gap + 1; x <= ax+gap+arm; x++ {
			set(row, x, c)
		}
		set(row, ax, c)
	case (d >= -gap-arm && d < -gap) || (d > gap && d <= gap+arm):
		set(row, ax, c)
	}
}

// set sets pixel x of row if it is inside.
func set(row []uint16, x int16, c uint16) {
	if x >= 0 && int(x) < len(row) {
		row[x] = c
	}
}

// addShown adds a sprite width pixels wide at (sx, sy) to the shown list,
// which is kept far to near. (A method, not a closure: a closure would be
// allocated every frame.)
func (g *Game) addShown(s *sprite, sx, sy, width, z float32) {
	size := int16(width)
	if size < 2 {
		size = 2
	}
	h := size * s.h / s.w
	o := shown{x: int16(sx) - size/2, y: int16(sy) - h/2, w: size, h: h, s: s, z: z}
	j := g.nShown
	for j > 0 && g.shown[j-1].z < o.z {
		g.shown[j] = g.shown[j-1]
		j--
	}
	g.shown[j] = o
	g.nShown++
}

// scaledRow draws row r of sprite s stretched to w pixels over row, from
// pixel x (nearest neighbour).
func scaledRow(row []uint16, s *sprite, x, w int16, r int16) {
	x0, x1 := x, x+w
	if x0 < 0 {
		x0 = 0
	}
	if int(x1) > len(row) {
		x1 = int16(len(row))
	}
	for p := x0; p < x1; p++ {
		if c, on := s.at((p-x)*s.w/w, r); on {
			row[p] = c
		}
	}
}
