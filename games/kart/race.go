package kart

import (
	"image/color"
	"math"
)

const laps = 3

// Driving, in world units and seconds.
const (
	maxSpeed   = 140
	boostSpeed = 190 // for a moment after a boost pad
	boostTime  = 1.0
	grassSpeed = 45 // on the grass the kart slows down to this
	accel      = 70
	brake      = 140
	drag       = 20
	grassDrag  = 120
	turnRate   = 1.9 // radians per second at full lock

	bumpDist = 9 // karts closer than this push each other apart
)

// Level is how fast the CPU karts go.
type Level struct {
	Name string
	Top  float32 // top speed of the CPU karts on a straight
}

var levels = [...]Level{{"EASY", 100}, {"NORMAL", 118}, {"HARD", 132}}

// kartColors are the body colours of the player's kart and the CPU's.
var kartColors = [4][2]color.RGBA{
	{{40, 90, 220, 255}, {20, 50, 140, 255}},
	{{220, 50, 40, 255}, {140, 25, 20, 255}},
	{{40, 170, 60, 255}, {20, 100, 35, 255}},
	{{240, 190, 30, 255}, {170, 120, 10, 255}},
}

// kart is the player's kart or a CPU's.
type kart struct {
	x, y, angle float32 // in world units; angle 0 is east (+x), y grows south
	speed       float32
	steer       float32 // -1..1, for the lean of the sprite
	boostT      float32

	// The player: the nearest point of the course and the laps started
	// (-1 before the start line).
	idx        int
	lap        int
	checkpoint bool // half way round since the last lap

	// A CPU: where it is along the course (in path points, from the start
	// line, growing) and its lane (cells to the right of the middle).
	cpu      bool
	s, lane  float32
	top      float32
	progress float32 // path points from the start line: the order of the race

	sprite     *sprite
	dot        color.RGBA // on the minimap
	mapX, mapY int16      // where the dot is drawn, -1 none
}

// The grid behind the start line, in path points and lanes: the player's
// place first (the last), then the CPU's. (Two arrays, not an array of
// structs: on the ESP32, TinyGo 0.42 read a local [4]struct{s, lane
// float32} indexed by a variable one field off.)
var (
	gridS    = [4]float32{-8, -2, -4, -6}
	gridLane = [4]float32{1.2, -1.2, 1.2, -1.2}
)

// newRace puts the karts on the grid behind the start line: the CPU's
// in front, the player last.
func (g *Game) newRace() {
	n := float32(len(g.track.path))
	for i := range g.karts {
		k := &g.karts[i]
		*k = kart{sprite: k.sprite, dot: kartColors[i][0], mapX: -1, mapY: -1}
		s := gridS[i]
		k.x, k.y = g.track.point(s, gridLane[i])
		dx, dy := g.track.dir(s)
		k.angle = atan2(dy, dx)
		if i == 0 {
			k.idx = int(n + s)
			k.lap, k.checkpoint = -1, true
			k.progress = s
			continue
		}
		k.cpu, k.s, k.lane = true, s, gridLane[i]
		k.top = levels[g.level].Top * (0.96 + 0.03*float32(i))
		k.progress = s
	}
	g.raceTime, g.lapStart, g.bestLap, g.lastLap = 0, 0, 0, 0
}

// player is the player's kart.
func (g *Game) player() *kart { return &g.karts[0] }

// drive moves the player's kart: speed, steering (more grip at speed),
// the grass and the boost pads.
func (g *Game) drive(k *kart, dt float32, gas, brakes bool, steer float32) {
	under := g.track.at(k.x, k.y)
	if under == boost {
		k.boostT = boostTime
	}
	if k.boostT > 0 {
		k.boostT -= dt
	}
	top := float32(maxSpeed)
	switch {
	case k.boostT > 0:
		top = boostSpeed
		gas = true
	case under == grass:
		top = grassSpeed
	}
	switch {
	case brakes:
		k.speed -= brake * dt
	case gas && k.boostT > 0:
		k.speed += 3 * accel * dt
	case gas:
		k.speed += accel * dt
	default:
		k.speed -= drag * dt
	}
	if k.speed > top {
		k.speed -= grassDrag * dt
		if k.speed < top {
			k.speed = top
		}
	}
	if k.speed < 0 {
		k.speed = 0
	}
	grip := k.speed / 40
	if grip > 1 {
		grip = 1
	}
	k.steer = steer
	k.angle += steer * turnRate * grip * dt
	sin, cos := sincos(k.angle)
	k.x += cos * k.speed * dt
	k.y += sin * k.speed * dt
	const margin = 2 * cellSize
	k.x = clamp(k.x, margin, worldSize-margin)
	k.y = clamp(k.y, margin, worldSize-margin)
}

// driveCPU moves a CPU kart along its lane: slower before curves, a
// little faster when far behind the player.
func (g *Game) driveCPU(k *kart, dt float32) {
	n := float32(len(g.track.path))
	target := k.top * (1 - 0.3*g.track.curve(k.s, 8))
	if target < 0.55*k.top {
		target = 0.55 * k.top
	}
	if g.player().progress-k.s > n/4 {
		target *= 1.12
	}
	if k.speed < target {
		k.speed += accel * dt
		if k.speed > target {
			k.speed = target
		}
	} else {
		k.speed -= brake * dt / 2
	}
	// The path points are about a cell apart.
	a := g.track.path[mod(int(floor(k.s)), int(n))]
	b := g.track.path[mod(int(floor(k.s))+1, int(n))]
	step := dist(a, b)
	if step < 0.2 {
		step = 0.2
	}
	k.s += k.speed * dt / (step * cellSize)
	k.x, k.y = g.track.point(k.s, k.lane)
	dx, dy := g.track.dir(k.s)
	k.angle = atan2(dy, dx)
	k.progress = k.s
}

// follow updates the player's place on the course: the nearest point, the
// laps (the line must be crossed forwards, after going half way round),
// the lap times. It returns true when the last lap is done.
func (g *Game) follow(k *kart) bool {
	n := len(g.track.path)
	old := k.idx
	k.idx = g.track.nearest(k.x, k.y, k.idx, 12)
	switch {
	case old > 3*n/4 && k.idx < n/4: // over the line forwards
		if k.checkpoint {
			k.lap++
			k.checkpoint = false
			if k.lap >= 1 {
				g.lastLap = g.raceTime - g.lapStart
				if g.bestLap == 0 || g.lastLap < g.bestLap {
					g.bestLap = g.lastLap
				}
			}
			g.lapStart = g.raceTime
		}
	case old < n/4 && k.idx > 3*n/4: // backwards
		k.lap--
		k.checkpoint = true
	case k.idx > 2*n/5 && k.idx < 3*n/5:
		k.checkpoint = true
	}
	k.progress = float32(k.lap*n + k.idx)
	return k.lap >= laps
}

// placeNow is the player's position in the race, 1 to 4.
func (g *Game) placeNow() int {
	p := 1
	for i := 1; i < len(g.karts); i++ {
		if g.karts[i].progress > g.player().progress {
			p++
		}
	}
	return p
}

// bump pushes the player out of a CPU kart it runs into; the one behind
// slows down.
func (g *Game) bump() {
	p := g.player()
	for i := 1; i < len(g.karts); i++ {
		c := &g.karts[i]
		dx, dy := p.x-c.x, p.y-c.y
		d := sqrt(dx*dx + dy*dy)
		if d >= bumpDist || d < 0.01 {
			continue
		}
		push := (bumpDist - d) / d
		p.x += dx * push
		p.y += dy * push
		if p.progress < c.progress {
			p.speed *= 0.75
		} else {
			c.speed *= 0.75
		}
	}
}

func atan2(y, x float32) float32 { return float32(math.Atan2(float64(y), float64(x))) }
