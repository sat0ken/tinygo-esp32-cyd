package kart

import "time"

// The camera, in world units: how high it is and how far behind the kart.
const (
	camHeight = 32
	camBack   = 28

	bandRows = 8   // rows drawn and sent at a time
	halfDist = 120 // rows further than this are drawn at half the width
	plainAt  = 300 // rows further than this have no checker pattern
	farthest = 900 // CPU karts further away are not drawn
)

// onScreen is a CPU kart as it is drawn this frame.
type onScreen struct {
	x, y, w, h int16 // the scaled sprite
	s          *sprite
	z          float32
}

// render draws the ground below the horizon, bandRows rows at a time,
// with the karts on it: the CPU's scaled with their distance (the far ones
// first), the player's last.
func (g *Game) render() {
	l := &g.l
	p := g.player()
	sin, cos := sincos(p.angle)
	fx, fy := cos, sin  // forward
	rx, ry := -sin, cos // right
	camX, camY := p.x-fx*camBack, p.y-fy*camBack

	// The player's sprite: its wheels touch the ground on the kart's row.
	pw, ph := p.sprite.w*l.scale, p.sprite.h*l.scale
	kartRow := l.horizon + int16(camHeight*l.focal/camBack)
	px, py := (g.w-pw)/2, kartRow+4-ph
	lean := int16(p.steer * 2 * float32(l.scale))

	// The CPU karts in front of the camera, far to near.
	var shown [3]onScreen
	n := 0
	for i := 1; i < len(g.karts); i++ {
		k := &g.karts[i]
		ox, oy := k.x-camX, k.y-camY
		z := ox*fx + oy*fy
		if z < 6 || z > farthest {
			continue
		}
		side := ox*rx + oy*ry
		w := int16(float32(pw) * camBack / z)
		h := int16(float32(ph) * camBack / z)
		if w < 2 {
			continue
		}
		cx := int16(float32(g.w)/2 + side*l.focal/z)
		bottom := l.horizon + int16(camHeight*l.focal/z+4*camBack/z)
		o := onScreen{x: cx - w/2, y: bottom - h, w: w, h: h, s: k.sprite, z: z}
		j := n
		for j > 0 && shown[j-1].z < z {
			shown[j] = shown[j-1]
			j--
		}
		shown[j] = o
		n++
	}

	half := float32(g.w) / 2
	for y0 := l.horizon; y0 < g.h; y0 += bandRows {
		rows := g.h - y0
		if rows > bandRows {
			rows = bandRows
		}
		buf, bytes := g.bufs[g.next], g.bytes[g.next]
		i := 0
		for y := y0; y < y0+rows; y++ {
			// The row is a line across the map dist units ahead.
			dist := camHeight * l.focal / float32(y-l.horizon+1)
			step := dist / l.focal
			wx := camX + fx*dist - rx*step*half
			wy := camY + fy*dist - ry*step*half
			mask := uint8(0xFF)
			if dist > plainAt {
				mask = 0xFE // plain colours: the pattern would only flicker
			}
			row := buf[i : i+int(g.w)]
			g.groundRow(row, fixed(wx), fixed(wy), fixed(rx*step), fixed(ry*step), mask, dist > halfDist)
			for k := 0; k < n; k++ {
				if o := &shown[k]; y >= o.y && y < o.y+o.h {
					scaledRow(row, o.s, o.x, o.w, (y-o.y)*o.s.h/o.h)
				}
			}
			if y >= py && y < py+ph {
				r := (y - py) / l.scale
				x := px
				if r < 4 {
					x += lean // the driver leans into the turn
				}
				g.kartRow(row, p.sprite, r, x)
			}
			i += int(g.w)
		}
		t := time.Now()
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

// groundRow fills row with the ground: u, v is the map position of the
// first pixel (16.16 fixed point world units), du, dv the step to the
// next; mask clears the shade bit of the cells far away. With half, each
// computed pixel is written twice.
func (g *Game) groundRow(row []uint16, u, v, du, dv int32, mask uint8, half bool) {
	cells := &g.track.cells
	pal := &g.pal
	step := 1
	if half {
		step = 2
		du, dv = du*2, dv*2
	}
	for k := 0; k+step <= len(row); k += step {
		c := g.outside
		cu, cv := uint32(u)>>(16+cellBits), uint32(v)>>(16+cellBits)
		if cu|cv < mapSize {
			c = pal[cells[cv*mapSize+cu]&mask&15]
		}
		row[k] = c
		if half {
			row[k+1] = c
		}
		u += du
		v += dv
	}
}

// kartRow draws row r of sprite s at l.scale over row, from pixel x.
func (g *Game) kartRow(row []uint16, s *sprite, r int16, x int16) {
	for b := int16(0); b < s.w; b++ {
		c, on := s.at(b, r)
		if !on {
			continue
		}
		for k := int16(0); k < g.l.scale; k++ {
			if p := x + b*g.l.scale + k; p >= 0 && int(p) < len(row) {
				row[p] = c
			}
		}
	}
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

func fixed(v float32) int32 { return int32(v * 65536) }
