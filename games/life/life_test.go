package life

import (
	"image"
	"math/rand"
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
	g := New(d, nil)
	clock := picotest.NewClock(t, 5*time.Minute)
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

// checkShown checks that d shows the middle of the world, tile by tile.
func checkShown(t *testing.T, g *Game, d *memlcd.Display) {
	t.Helper()
	ox, oy := (int(d.W)-picopad.Width)/2, (int(d.H)-picopad.Height)/2
	for i := 0; i < mapH; i++ {
		for j := 0; j < mapW; j++ {
			b := int(g.board[(i+mapY)*mapW0+j+mapX])
			for ty := 0; ty < tileSize; ty++ {
				for tx := 0; tx < tileSize; tx++ {
					v := tiles16Img[(b*tileSize+ty)*tileSize/2+tx/2]
					if tx%2 == 0 {
						v >>= 4
					}
					e := int(v & 0x0F)
					want := uint16(tiles16ImgPal[2*e]) | uint16(tiles16ImgPal[2*e+1])<<8
					if got := d.Pix[(oy+i*tileSize+ty)*int(d.W)+ox+j*tileSize+tx]; got != want {
						t.Fatalf("cell %d, %d (tile %d) pixel %d, %d: %#04x, want %#04x", j, i, b, tx, ty, got, want)
					}
				}
			}
		}
	}
}

// alive returns the world as booleans (the cursor's cell as its cell).
func alive(g *Game) [mapSize]bool {
	var a [mapSize]bool
	for i, b := range g.board {
		a[i] = b == tileFull || b == tileSelFull
	}
	return a
}

func TestStart(t *testing.T) {
	for _, size := range []struct{ w, h int16 }{{320, 240}, {480, 272}} {
		g, d, clock, p := newTest(t, size.w, size.h)
		clock.At(50*ms, func() {
			// slot 1 (penta-decathlon), the cursor in the middle
			goldentest.Check(t, "life_start", screen(d))
			checkShown(t, g, d)
		})
		p.Tap(hal.PadY, 100*ms)
		g.Run()
	}
}

// next is the rule written plainly, to check generation against.
func next(a [mapSize]bool) [mapSize]bool {
	var b [mapSize]bool
	for y := 0; y < mapH0; y++ {
		for x := 0; x < mapW0; x++ {
			n := 0
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if (dx != 0 || dy != 0) && a[(x+dx+mapW0)%mapW0+(y+dy+mapH0)%mapH0*mapW0] {
						n++
					}
				}
			}
			i := x + y*mapW0
			b[i] = n == 3 || n == 2 && a[i]
		}
	}
	return b
}

func TestGeneration(t *testing.T) {
	g := New(memlcd.New(320, 240), nil)
	r := rand.New(rand.NewSource(1))
	for i := range g.board {
		if r.Intn(3) == 0 {
			g.board[i] = tileFull
		}
	}
	for gen := 0; gen < 30; gen++ {
		want := next(alive(g))
		g.generation()
		if alive(g) != want {
			t.Fatalf("generation %d differs from the rule", gen+1)
		}
	}

	// a glider goes over the corner of the world and comes back after
	// 4 x 50 generations
	g.board = [mapSize]uint8{}
	for _, c := range [][2]int{{2, 0}, {0, 1}, {2, 1}, {1, 2}, {2, 2}} {
		g.board[48+c[0]-3+(48+c[1]-3)*mapW0] = tileFull
	}
	start := alive(g)
	for gen := 0; gen < 4*mapW0; gen++ {
		g.generation()
	}
	if alive(g) != start {
		t.Error("the glider did not come back around the world")
	}
}

func TestSlots(t *testing.T) {
	g, _, clock, p := newTest(t, 320, 240)
	// sample k in the middle of the world
	sample := func(k int) [mapSize]bool {
		var a [mapSize]bool
		s := 0
		for ; k > 0; k-- {
			s += 2 + int(samples[s])*int(samples[s+1])
		}
		w, h := int(samples[s]), int(samples[s+1])
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				a[(mapW0-w)/2+x+((mapH0-h)/2+y)*mapW0] = samples[s+2+y*w+x] == 1
			}
		}
		return a
	}
	var flipped bool
	clock.At(50*ms, func() {
		if alive(g) != sample(1) {
			t.Error("start: not the penta-decathlon")
		}
		flipped = alive(g)[g.curX+g.curY*mapW0]
	})
	p.Tap(hal.PadA, 100*ms) // flip the cell under the cursor in slot 1
	at := 300 * ms
	for k := 2; k < 2+slotNum; k++ {
		p.Tap(hal.PadX, at)
		k := k % slotNum
		clock.At(at+150*ms, func() {
			want := sample(k)
			if k == 1 {
				want[g.curX+g.curY*mapW0] = !flipped // kept in the slot
			}
			if alive(g) != want {
				t.Errorf("slot %d: not its sample", k)
			}
		})
		at += 200 * ms
	}
	p.Tap(hal.PadY, at)
	g.Run()
}

func TestRun(t *testing.T) {
	g, d, clock, p := newTest(t, 320, 240)
	var before [mapSize]uint8
	clock.At(50*ms, func() { before = g.board })
	p.Tap(hal.PadB, 100*ms) // run
	clock.At(1100*ms, func() {
		// about 5 generations of the penta-decathlon (period 15)
		if g.board == before {
			t.Error("running: nothing changed")
		}
		checkShown(t, g, d)
	})
	p.Tap(hal.PadA, 1200*ms) // stop: the world as before the run
	clock.At(1500*ms, func() {
		if g.board != before {
			t.Error("after the run: not the world before it")
		}
		checkShown(t, g, d)
	})
	p.Tap(hal.PadY, 1600*ms)
	g.Run()
}

func TestCursorWraps(t *testing.T) {
	g, _, clock, p := newTest(t, 320, 240)
	// from the middle (25) to the left edge (15) and over it to the right
	for i := 0; i < 11; i++ {
		p.Tap(hal.PadLeft, time.Duration(100+200*i)*ms)
	}
	clock.At(2400*ms, func() {
		if g.curX != mapX+mapW-1 || g.board[g.curX+g.curY*mapW0] < tileSelEmpty {
			t.Errorf("cursor at %d (cell %d), want %d", g.curX, g.board[g.curX+g.curY*mapW0], mapX+mapW-1)
		}
	})
	p.Tap(hal.PadY, 2500*ms)
	g.Run()
}
