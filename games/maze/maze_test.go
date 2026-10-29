package maze

import (
	"image"
	"testing"
	"time"

	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/internal/goldentest"
	"github.com/sat0ken/tinygo-cyd/internal/picotest"
	"github.com/sat0ken/tinygo-cyd/memlcd"
	"github.com/sat0ken/tinygo-cyd/picopad"
)

const ms = time.Millisecond

func newTest(t *testing.T, w, h int16) (*Game, *memlcd.Display, *picotest.Clock, *picotest.Pad) {
	d := memlcd.New(w, h)
	g := New(d, nil, 12345)
	clock := picotest.NewClock(t, 5*time.Minute)
	clock.Step = 100 * time.Microsecond // the main loop draws every 100 turns
	g.Clock = clock
	p := &picotest.Pad{Clock: clock}
	g.Pad = p
	return g, d, clock, p
}

// screen returns the PicoPad's screen in the middle of d.
func screen(d *memlcd.Display) *image.RGBA {
	x, y := (int(d.W)-picopad.Width)/2, (int(d.H)-picopad.Height)/2
	img := d.Image()
	out := image.NewRGBA(image.Rect(0, 0, picopad.Width, picopad.Height))
	for yy := 0; yy < picopad.Height; yy++ {
		copy(out.Pix[yy*out.Stride:], img.Pix[(yy+y)*img.Stride+x*4:(yy+y)*img.Stride+(x+picopad.Width)*4])
	}
	return out
}

// hidden counts the places of the maze not shown yet.
func hidden(g *Game) int {
	n := 0
	for _, b := range g.board {
		if b == sEmptyHide || b == sWallHide || b == sDoorHide {
			n++
		}
	}
	return n
}

// path returns the shortest way from the face to the door as buttons.
func path(g *Game) []hal.Buttons {
	type pos struct{ x, y int }
	start, stop := pos{g.posX, g.posY}, pos{g.stopX, g.stopY}
	from := map[pos]pos{start: start}
	queue := []pos{start}
	dirs := []struct {
		dx, dy int
		b      hal.Buttons
	}{{-1, 0, hal.PadLeft}, {1, 0, hal.PadRight}, {0, -1, hal.PadUp}, {0, 1, hal.PadDown}}
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		for _, d := range dirs {
			n := pos{p.x + d.dx, p.y + d.dy}
			if _, seen := from[n]; seen || !g.isEmpty(n.x, n.y) {
				continue
			}
			from[n] = p
			queue = append(queue, n)
		}
	}
	if _, ok := from[stop]; !ok {
		return nil
	}
	var bs []hal.Buttons
	for p := stop; p != start; p = from[p] {
		q := from[p]
		for _, d := range dirs {
			if q.x+d.dx == p.x && q.y+d.dy == p.y {
				bs = append([]hal.Buttons{d.b}, bs...)
			}
		}
	}
	return bs
}

func TestStart(t *testing.T) {
	for _, size := range []struct{ w, h int16 }{{320, 240}, {480, 272}} {
		g, d, clock, p := newTest(t, size.w, size.h)
		clock.At(50*ms, func() {
			goldentest.Check(t, "maze_start", screen(d))
			// only the face and the places around it are shown
			if n := hidden(g); n < mazeW*mazeH-5 {
				t.Errorf("%dx%d: %d places hidden, want all but 5 at most", size.w, size.h, n)
			}
		})
		p.Tap(hal.PadY, 100*ms)
		g.Run()
		if clock.T > 300*ms {
			t.Errorf("%dx%d: Run returned at %v", size.w, size.h, clock.T)
		}
	}
}

func TestGenerator(t *testing.T) {
	for seed := uint32(0); seed < 20; seed++ {
		g := New(memlcd.New(320, 240), nil, seed)
		g.gener()
		// every stone became a wall, and the paths are one piece
		start := -1
		empty := 0
		for y := 0; y < mazeH; y++ {
			for x := 0; x < mazeW; x++ {
				switch b := *g.at(x, y); {
				case x%2 == 1 && y%2 == 1 && b != sWallHide:
					t.Fatalf("seed %d: stone at %d, %d is %d", seed, x, y, b)
				case b == sEmptyHide:
					empty++
					if start < 0 {
						start = x + y*mapW
					}
				}
			}
		}
		g.posX, g.posY = start%mapW, start/mapW
		g.stopX, g.stopY = -1, -1
		reached := 0
		seen := map[int]bool{start: true}
		queue := []int{start}
		for len(queue) > 0 {
			p := queue[0]
			queue = queue[1:]
			reached++
			x, y := p%mapW, p/mapW
			for _, n := range [][2]int{{x - 1, y}, {x + 1, y}, {x, y - 1}, {x, y + 1}} {
				k := n[0] + n[1]*mapW
				if n[0] >= 0 && n[0] < mazeW && n[1] >= 0 && n[1] < mazeH && !seen[k] && *g.at(n[0], n[1]) == sEmptyHide {
					seen[k] = true
					queue = append(queue, k)
				}
			}
		}
		if reached != empty {
			t.Errorf("seed %d: %d of %d places reachable", seed, reached, empty)
		}
	}
}

func TestSolve(t *testing.T) {
	g, d, clock, p := newTest(t, 320, 240)
	var done time.Duration // when the last step is pressed
	var startX, startY int
	clock.At(50*ms, func() {
		startX, startY = g.posX, g.posY
		steps := path(g)
		if len(steps) == 0 {
			t.Fatal("no way to the door")
		}
		// 200ms apart: a button counts as released 50ms after it goes up,
		// so taps closer together in the same direction would be one
		for i, b := range steps {
			done = clock.T + time.Duration(i+1)*200*ms
			p.Tap(b, done)
		}
		// at the door: the whole maze is shown, then a button starts a
		// new one
		clock.At(done+4*time.Second, func() {
			if g.posX != g.stopX || g.posY != g.stopY {
				t.Fatalf("at %d, %d, the door is at %d, %d", g.posX, g.posY, g.stopX, g.stopY)
			}
			if n := hidden(g); n != 0 {
				t.Errorf("solved: %d places still hidden", n)
			}
			goldentest.Check(t, "maze_solved", screen(d))
		})
		p.Tap(hal.PadB, done+4500*ms)
		clock.At(done+5*time.Second, func() {
			if g.posX == startX && g.posY == startY || hidden(g) < mazeW*mazeH-5 {
				t.Errorf("no new maze: at %d, %d, %d hidden", g.posX, g.posY, hidden(g))
			}
		})
		p.Tap(hal.PadY, done+5500*ms)
	})
	g.Run()
	if clock.T < done+5500*ms {
		t.Errorf("Run returned at %v, before Y", clock.T)
	}
}

func TestShowDoor(t *testing.T) {
	g, d, clock, p := newTest(t, 320, 240)
	p.Tap(hal.PadA, 100*ms)
	var before [mapSize]uint8
	// the door's tile on the screen, at the place of the door
	doorShown := func() bool {
		x, y := g.stopX*tileSize+tileSize/2, g.stopY*tileSize+tileSize/2
		for ty := 0; ty < tileSize; ty++ {
			for tx := 0; tx < tileSize; tx++ {
				b := tilesImg[(sDoor*tileSize+ty)*tileSize/2+tx/2]
				if tx%2 == 0 {
					b >>= 4
				}
				e := int(b & 0x0F)
				want := uint16(tilesImgPal[2*e]) | uint16(tilesImgPal[2*e+1])<<8
				if d.Pix[(y+ty)*int(d.W)+x+tx] != want {
					return false
				}
			}
		}
		return true
	}
	clock.At(50*ms, func() {
		before = g.board
		if doorShown() {
			t.Error("the door is shown at the start")
		}
	})
	clock.At(110*ms, func() {
		if !doorShown() {
			t.Error("A: the door is not shown")
		}
	})
	clock.At(700*ms, func() {
		if g.board != before || doorShown() {
			t.Error("after A: the board changed or the door is still shown")
		}
	})
	p.Tap(hal.PadY, 800*ms)
	g.Run()
}
