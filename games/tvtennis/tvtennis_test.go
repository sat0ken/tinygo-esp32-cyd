package tvtennis

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

func newTest(t *testing.T) (*Game, *memlcd.Display, *picotest.Clock, *picotest.Pad) {
	d := memlcd.New(320, 240)
	g := New(d, nil, 12345)
	clock := picotest.NewClock(t, 10*time.Minute)
	clock.Step = 100 * time.Microsecond // the game loop does not sleep
	g.Clock = clock
	p := &picotest.Pad{Clock: clock}
	g.Pad = p
	return g, d, clock, p
}

func pixel(d *memlcd.Display, x, y int) picopad.Color { return picopad.Color(d.Pix[y*int(d.W)+x]) }

func TestTitle(t *testing.T) {
	g, d, clock, p := newTest(t)
	clock.At(100*ms, func() { goldentest.Check(t, "tvtennis_title", d.Image()) })
	p.Tap(hal.PadY, 200*ms)
	g.Run()
}

func TestTwoPlayers(t *testing.T) {
	g, d, clock, p := newTest(t)
	var title *image.RGBA
	clock.At(100*ms, func() { title = d.Image() })
	p.Tap(hal.PadDown, 200*ms) // 2 players
	clock.At(500*ms, func() {
		goldentest.Check(t, "tvtennis_game", d.Image())
	})
	p.Hold(hal.PadUp, 600*ms, time.Second) // left paddle up to the top
	p.Hold(hal.PadA, 600*ms, time.Second)  // right paddle down to the bottom
	clock.At(1700*ms, func() {
		if g.paddleY[0] != paddleH/2 || g.paddleY[1] != height-paddleH/2 {
			t.Errorf("paddles at %v, want %d and %d", g.paddleY, paddleH/2, height-paddleH/2)
		}
		// drawn at their new places, and nothing left where they were
		if pixel(d, paddleLX, 1) != picopad.ColYellow || pixel(d, paddleLX, height/2+paddleH/2) != picopad.ColBlack ||
			pixel(d, paddleRX, height-1) != picopad.ColYellow || pixel(d, paddleRX, height/2-paddleH/2) != picopad.ColBlack {
			t.Error("the paddles are not drawn where they are")
		}
	})
	p.Tap(hal.PadLeft, 1800*ms) // the left player serves
	clock.At(2100*ms, func() {
		if g.ballState != ballIn || g.ballX <= paddleLX+paddleW+ballSize/2 {
			t.Errorf("after the serve: state %d, x %v", g.ballState, g.ballX)
		}
		if pixel(d, int(g.ballX), int(g.ballY)) != picopad.ColRed {
			t.Error("the ball is not drawn where it is")
		}
	})
	p.Tap(hal.PadY, 2200*ms) // back to the first screen
	clock.At(2400*ms, func() {
		if diff := goldentest.Compare(title, d.Image()); diff != "" {
			t.Errorf("Y: not the first screen: %s", diff)
		}
	})
	p.Tap(hal.PadY, 2500*ms)
	g.Run()
}

func TestBall(t *testing.T) {
	g := New(memlcd.New(320, 240), nil, 1)
	g.paddleY = [2]float32{120, 120}
	g.timeDelta = 10000 // 3 pixels a step

	// to the left, at the left paddle: it bounces to the right
	g.ballState, g.ballX, g.ballY, g.ballDir = ballIn, paddleLX+paddleW+ballSize/2+1, 110, pi
	g.gameBall()
	if c := cosf(g.ballDir); c <= 0 || g.ballX != paddleLX+paddleW+ballSize/2 {
		t.Errorf("at the left paddle: direction %v (cos %v), x %v", g.ballDir, c, g.ballX)
	}

	// the left paddle elsewhere: the ball goes out, the right player scores
	g.paddleY[0] = 30
	g.ballDir = pi
	for i := 0; i < 20 && g.ballState == ballIn; i++ {
		g.gameBall()
	}
	if g.ballState != ballL || g.score != [2]uint8{0, 1} {
		t.Errorf("out on the left: state %d, score %v", g.ballState, g.score)
	}

	// the top edge
	g.ballState, g.ballX, g.ballY, g.ballDir = ballIn, 160, ballSize/2+1, pi/4
	g.gameBall()
	if g.ballDir != -pi/4 || g.ballY != ballSize/2 {
		t.Errorf("at the top: direction %v, y %v", g.ballDir, g.ballY)
	}
}

func TestDemo(t *testing.T) {
	g, _, clock, p := newTest(t)
	p.Tap(hal.PadUp, 100*ms) // the computer plays both paddles
	served := false
	var watch func()
	watch = func() {
		served = served || g.ballState == ballIn
		if clock.T < 120*time.Second {
			clock.At(clock.T+100*ms, watch)
		}
	}
	clock.At(200*ms, watch)
	clock.At(120*time.Second, func() {
		if !served || g.score[0]+g.score[1] == 0 {
			t.Errorf("demo: served %v, score %v", served, g.score)
		}
	})
	p.Tap(hal.PadY, 121*time.Second)
	p.Tap(hal.PadY, 121500*ms)
	g.Run()
}
