package kart

// The track is a map of mapSize x mapSize cells, cellSize world units
// square. A world unit is one texel of the ground.
const (
	mapSize   = 128
	cellBits  = 3
	cellSize  = 1 << cellBits
	worldSize = mapSize * cellSize

	roadRadius = 3.0 // cells from the middle line to the edge of the road
	curbWidth  = 1.0 // cells
)

// tile is what a cell of the map is.
type tile uint8

const (
	grass tile = iota
	road
	curb
	startLine
	numTiles
)

// coursePoints is the middle line of the course in cells, in driving
// order; it closes back on the first point.
var coursePoints = [...][2]float32{
	{24, 100}, {22, 64}, {26, 36}, {42, 20}, {66, 22}, {78, 44},
	{100, 36}, {112, 54}, {110, 84}, {94, 100}, {74, 90}, {58, 108}, {36, 112},
}

// track is the map and the sampled middle line.
type track struct {
	// cells are tile<<1 | shade: shade picks one of the tile's two
	// colours, in a checker pattern (of 2 cells on grass, 4 on the road, 1
	// on curbs and the start line) that shows the speed.
	cells [mapSize * mapSize]uint8
	path  [][2]float32 // the middle line in cells, about one cell apart
}

// newTrack draws the course into the map: curbs, then the road over them,
// then the start line across the road at the first point.
func newTrack() *track {
	t := &track{}
	n := len(coursePoints)
	for i := 0; i < n; i++ {
		p0, p1 := coursePoints[(i+n-1)%n], coursePoints[i]
		p2, p3 := coursePoints[(i+1)%n], coursePoints[(i+2)%n]
		steps := int(dist(p1, p2)) + 1
		for s := 0; s < steps; s++ {
			t.path = append(t.path, catmullRom(p0, p1, p2, p3, float32(s)/float32(steps)))
		}
	}
	t.stamp(roadRadius+curbWidth, curb)
	t.stamp(roadRadius, road)

	// The start line: the road cells less than a cell from the first
	// point along the driving direction.
	p, q := t.path[0], t.path[1]
	dx, dy := q[0]-p[0], q[1]-p[1]
	l := sqrt(dx*dx + dy*dy)
	dx, dy = dx/l, dy/l
	for cy := 0; cy < mapSize; cy++ {
		for cx := 0; cx < mapSize; cx++ {
			ox, oy := float32(cx)+0.5-p[0], float32(cy)+0.5-p[1]
			along, across := ox*dx+oy*dy, -ox*dy+oy*dx
			if along > -1 && along < 1 && across > -roadRadius && across < roadRadius {
				t.cells[cy*mapSize+cx] = uint8(startLine)
			}
		}
	}
	for cy := 0; cy < mapSize; cy++ {
		for cx := 0; cx < mapSize; cx++ {
			k := tile(t.cells[cy*mapSize+cx])
			b := checkerBits[k]
			shade := (cx>>b ^ cy>>b) & 1
			t.cells[cy*mapSize+cx] = uint8(k)<<1 | uint8(shade)
		}
	}
	return t
}

// checkerBits: the checker squares of each tile are 1<<checkerBits cells.
var checkerBits = [numTiles]uint8{grass: 1, road: 2, curb: 0, startLine: 0}

// stamp sets every cell within r cells of the middle line to k.
func (t *track) stamp(r float32, k tile) {
	for _, p := range t.path {
		for cy := int(p[1] - r - 1); cy <= int(p[1]+r+1); cy++ {
			for cx := int(p[0] - r - 1); cx <= int(p[0]+r+1); cx++ {
				if cx < 0 || cy < 0 || cx >= mapSize || cy >= mapSize {
					continue
				}
				ox, oy := float32(cx)+0.5-p[0], float32(cy)+0.5-p[1]
				if ox*ox+oy*oy <= r*r {
					t.cells[cy*mapSize+cx] = uint8(k)
				}
			}
		}
	}
}

// at returns the tile under world position (x, y); off the map is grass.
func (t *track) at(x, y float32) tile {
	cx, cy := int(x)>>cellBits, int(y)>>cellBits
	if x < 0 || y < 0 || cx >= mapSize || cy >= mapSize {
		return grass
	}
	return tile(t.cells[cy*mapSize+cx] >> 1)
}

// catmullRom is the point at s (0..1) between p1 and p2 of the spline
// through p0..p3.
func catmullRom(p0, p1, p2, p3 [2]float32, s float32) [2]float32 {
	s2, s3 := s*s, s*s*s
	var r [2]float32
	for k := 0; k < 2; k++ {
		r[k] = 0.5 * (2*p1[k] + (p2[k]-p0[k])*s + (2*p0[k]-5*p1[k]+4*p2[k]-p3[k])*s2 + (3*p1[k]-p0[k]-3*p2[k]+p3[k])*s3)
	}
	return r
}

func dist(a, b [2]float32) float32 {
	dx, dy := b[0]-a[0], b[1]-a[1]
	return sqrt(dx*dx + dy*dy)
}

// sqrt is Newton's method in float32 (math.Sqrt is float64, which the
// ESP32 does in software).
func sqrt(v float32) float32 {
	if v <= 0 {
		return 0
	}
	x := v
	if x < 1 {
		x = 1
	}
	for i := 0; i < 12; i++ {
		x = 0.5 * (x + v/x)
	}
	return x
}
