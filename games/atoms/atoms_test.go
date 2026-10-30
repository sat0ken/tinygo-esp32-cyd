package atoms

import (
	"testing"
	"time"

	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/internal/goldentest"
	"github.com/sat0ken/tinygo-cyd/internal/picotest"
	"github.com/sat0ken/tinygo-cyd/memlcd"
)

const ms = time.Millisecond

func newTest(t *testing.T) (*Game, *memlcd.Display, *picotest.Clock, *picotest.Pad) {
	d := memlcd.New(320, 240)
	g := New(d, nil, 12345)
	clock := picotest.NewClock(t, 30*time.Minute)
	clock.Step = 500 * time.Microsecond // the first screen draws in a loop without sleeping
	g.Clock = clock
	p := &picotest.Pad{Clock: clock}
	g.Pad = p
	return g, d, clock, p
}

// consistent checks that the players' atoms are the atoms on the board.
func consistent(t *testing.T, g *Game) {
	t.Helper()
	var sum [playerNum]int
	for i, n := range g.atoms {
		if n > 0 {
			sum[g.owner[i]] += int(n)
		}
	}
	for i := range g.players {
		if g.players[i].typ != playerOff && sum[i] != g.players[i].atoms {
			t.Fatalf("player %d: %d atoms on the board, the score says %d", i+1, sum[i], g.players[i].atoms)
		}
	}
}

func TestScreens(t *testing.T) {
	g, d, clock, p := newTest(t)
	clock.At(100*ms, func() { goldentest.Check(t, "atoms_open", d.Image()) })
	p.Tap(hal.PadA, 200*ms) // start: player 1 human, player 2 computer
	clock.At(300*ms, func() {
		goldentest.Check(t, "atoms_game", d.Image())
		if g.playerNum != 2 || g.player != 0 || g.atoms[3*mapW+3] != 1 || g.atoms[3*mapW+6] != 1 {
			t.Errorf("new game: %d players, player %d on turn", g.playerNum, g.player)
		}
	})
	p.Tap(hal.PadY, 400*ms) // back to the first screen
	p.Tap(hal.PadY, 600*ms) // quit
	g.Run()
}

func TestHumanMove(t *testing.T) {
	g, _, clock, p := newTest(t)
	p.Tap(hal.PadA, 100*ms) // start
	// the cursor starts on player 1's atom (3, 3): put a second one there
	// (a place inside holds 3 before it explodes), then the computer plays
	p.Tap(hal.PadA, 300*ms)
	clock.At(250*ms, func() {
		if g.players[0].cursor != 3*mapW+3 {
			t.Errorf("cursor at %d", g.players[0].cursor)
		}
	})
	clock.At(1000*ms, func() {
		if g.atoms[3*mapW+3] != 2 || g.players[0].atoms != 2 {
			t.Errorf("after A: %d atoms there, player 1 has %d", g.atoms[3*mapW+3], g.players[0].atoms)
		}
		if g.player != 0 || g.players[1].atoms != 2 {
			t.Errorf("the computer did not play: player %d on turn, it has %d", g.player, g.players[1].atoms)
		}
		consistent(t, g)
	})
	// the cursor wraps: LEFT from column 3 four times goes to column 9
	for i := 0; i < 4; i++ {
		p.Tap(hal.PadLeft, time.Duration(1100+200*i)*ms)
	}
	clock.At(2000*ms, func() {
		if g.players[0].cursor != 3*mapW+9 {
			t.Errorf("cursor at %d after 4 x LEFT, want %d", g.players[0].cursor, 3*mapW+9)
		}
	})
	p.Tap(hal.PadY, 2100*ms)
	p.Tap(hal.PadY, 2300*ms)
	g.Run()
}

func TestExplosion(t *testing.T) {
	g := New(memlcd.New(320, 240), nil, 1)
	g.Clock = picotest.NewClock(t, time.Minute)
	g.players[1].typ = playerHuman
	g.newGame()
	// player 1 (index 0) puts a second atom in the corner (0, 0): a corner
	// holds 2, so it explodes into (1, 0) and (0, 1), taking player 2's
	// atom there
	g.atoms[1], g.owner[1] = 1, 1
	g.players[1].atoms++
	g.atoms[0], g.owner[0] = 2, 0
	g.players[0].atoms += 2
	g.explo()
	if g.atoms[0] != 0 || g.atoms[1] != 2 || g.owner[1] != 0 || g.atoms[mapW] != 1 || g.owner[mapW] != 0 {
		t.Errorf("after the explosion: %d %d/%d %d/%d", g.atoms[0], g.atoms[1], g.owner[1], g.atoms[mapW], g.owner[mapW])
	}
	consistent(t, g)
}

func TestDemo(t *testing.T) {
	g, _, clock, p := newTest(t)
	p.Tap(hal.PadLeft, 100*ms) // player 1: computer too
	p.Tap(hal.PadA, 300*ms)
	var watch func()
	over := time.Duration(0)
	watch = func() {
		if g.playerHuman != 0 || g.playerNum < 1 {
			t.Fatalf("demo: %d human players, %d players", g.playerHuman, g.playerNum)
		}
		consistent(t, g)
		if g.playerNum == 1 {
			over = clock.T
			return
		}
		clock.At(clock.T+500*ms, watch)
	}
	clock.At(500*ms, watch)
	g.Clock.(*picotest.Clock).At(25*time.Minute, func() {
		if over == 0 {
			t.Fatal("no winner after 25 minutes")
		}
	})
	// a key ends the game at the end, Y quits from the first screen
	var end func()
	end = func() {
		if over == 0 {
			clock.At(clock.T+time.Second, end)
			return
		}
		p.Tap(hal.PadB, clock.T+100*ms)
		p.Tap(hal.PadY, clock.T+500*ms)
	}
	clock.At(time.Second, end)
	g.Run()
	t.Logf("the demo ended after %v", over)
}

// TestOneGame: with OneGame, Run returns when the winner is shown and a
// key pressed, and on Y in a game.
func TestOneGame(t *testing.T) {
	g, _, clock, p := newTest(t)
	g.OneGame = true
	p.Tap(hal.PadA, 100*ms) // player 1 human, player 2 computer
	clock.At(300*ms, func() {
		// player 2's only atom next to the corner, where player 1 has one
		// and puts the second: the explosion takes it
		g.atoms[3*mapW+6] = 0
		g.atoms[1], g.owner[1] = 1, 1
		g.atoms[0], g.owner[0] = 1, 0
		g.players[0].atoms++
		g.players[0].cursor = 0
	})
	p.Tap(hal.PadA, 400*ms)
	clock.At(time.Second, func() {
		if g.playerNum != 1 {
			t.Errorf("%d players left", g.playerNum)
		}
	})
	p.Tap(hal.PadB, 2*time.Second) // the winner blinks until a key
	g.Run()
	if clock.T < 2*time.Second || clock.T > 3*time.Second {
		t.Errorf("Run returned at %v", clock.T)
	}

	g, _, clock, p = newTest(t)
	g.OneGame = true
	p.Tap(hal.PadA, 100*ms)
	p.Tap(hal.PadY, 500*ms)
	g.Run()
	if clock.T > time.Second {
		t.Errorf("Y: Run returned at %v", clock.T)
	}
}
