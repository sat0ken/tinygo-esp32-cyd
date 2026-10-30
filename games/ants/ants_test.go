package ants

import (
	"fmt"
	"hash/crc32"
	"testing"
	"time"

	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/internal/goldentest"
	"github.com/sat0ken/tinygo-cyd/internal/picotest"
	"github.com/sat0ken/tinygo-cyd/memlcd"
)

const ms = time.Millisecond

func newTest(t *testing.T, seed uint32) (*Game, *memlcd.Display, *picotest.Clock, *picotest.Pad) {
	d := memlcd.New(320, 240)
	g := New(d, nil, seed)
	clock := picotest.NewClock(t, 30*time.Minute)
	clock.Step = 100 * time.Microsecond
	g.Clock = clock
	p := &picotest.Pad{Clock: clock}
	g.Pad = p
	return g, d, clock, p
}

// TestMatchesC plays 2,000 turns of the computer against itself as
// testdata/csim/sim.c does with the C source compiled on the host
// (testdata/csim/run.sh), with the same random numbers, and compares the
// parameters and the cards of both players after each turn by their
// checksums (21 games won on the way).
func TestMatchesC(t *testing.T) {
	want := map[int]string{
		99:   "550a304a",
		499:  "842f0ffc",
		999:  "63be43b8",
		1499: "7fe6fd4c",
		1999: "cb082f0a",
	}
	g, _, clock, _ := newTest(t, 6)
	clock.Limit = 100 * time.Hour
	clock.Step = time.Millisecond // faster through the animations
	g.newGame(playerComp, playerComp)
	crc := crc32.NewIEEE()
	var wins [2]int
	for turn := 0; turn < 2000; turn++ {
		g.endGame = -1
		g.autoPlay()
		if g.endGame >= 0 {
			wins[g.endGame]++
		}
		d := ""
		if g.lastDiscard {
			d = "d"
		}
		line := fmt.Sprintf("%d: p%d last %d%s", turn, g.player, g.last, d)
		for i := range g.players {
			line += " |"
			for _, v := range g.players[i].par {
				line += fmt.Sprintf(" %d", v)
			}
			line += " :"
			for _, c := range g.players[i].cards {
				line += fmt.Sprintf(" %d", c)
			}
		}
		crc.Write([]byte(line + "\n"))
		if w, ok := want[turn]; ok {
			if got := fmt.Sprintf("%08x", crc.Sum32()); got != w {
				t.Fatalf("turn %d: checksum %s, want %s; %s", turn, got, w, line)
			}
		}
	}
	if wins != [2]int{10, 11} {
		t.Errorf("wins %v", wins)
	}
}

func TestScreens(t *testing.T) {
	g, d, clock, p := newTest(t, 1)
	clock.At(200*ms, func() { goldentest.Check(t, "ants_open", d.Image()) })
	p.Tap(hal.PadRight, 400*ms) // the black ants against the computer
	p.Tap(hal.PadA, 600*ms)
	clock.At(1200*ms, func() {
		goldentest.Check(t, "ants_game", d.Image())
		if g.player != 0 || g.selCard != 3 || g.players[1].player != playerComp {
			t.Errorf("start: player %d, card %d", g.player, g.selCard)
		}
	})
	// X shows the best card, A plays it; then the computer plays
	p.Tap(hal.PadX, 1400*ms)
	var best, card int
	clock.At(1700*ms, func() {
		best = g.selCard
		card = g.players[0].cards[best]
	})
	p.Tap(hal.PadA, 1800*ms)
	clock.At(1950*ms, func() { // in the delays of the turn
		if g.last != card || g.player != 0 {
			t.Errorf("A: the last card %d (played %d), player %d", g.last, card, g.player)
		}
	})
	clock.At(15*time.Second, func() {
		if g.player != 0 || g.players[0].cards[best] < 0 {
			t.Errorf("after the computer: player %d, card %d", g.player, g.players[0].cards[best])
		}
		goldentest.Check(t, "ants_turn", d.Image())
	})
	p.Tap(hal.PadY, 16*time.Second) // back to the open screen
	p.Tap(hal.PadY, 17*time.Second)
	g.Run()
}

// TestCards plays the cards that do more than one thing on a known state.
func TestCards(t *testing.T) {
	g, _, _, _ := newTest(t, 1)
	for _, tc := range []struct {
		card       int
		own, enemy [parNum]int // the changes
	}{
		{3, [parNum]int{parCastle: 8, parFence: -4}, [parNum]int{}},                                                                 // reserve
		{6, [parNum]int{parCastle: 8}, [parNum]int{parCastle: -4}},                                                                  // wain
		{16, [parNum]int{}, [parNum]int{parBricks: -4, parWeapons: -4, parCrystals: -3}},                                            // saboteur (3 crystals left)
		{17, [parNum]int{parBricks: 5, parWeapons: 5, parCrystals: 3}, [parNum]int{parBricks: -5, parWeapons: -5, parCrystals: -3}}, // thief
		{29, [parNum]int{1, 1, 1, 1, 1, 1, 1, 1}, [parNum]int{-1, -1, -1, -1, -1, -1, -1, -1}},                                      // curse
		{19, [parNum]int{}, [parNum]int{parCastle: -22, parFence: -10}},                                                             // banshee: the fence, then the castle
	} {
		g.newGame(playerHuman, playerHuman)
		g.players[1].par[parCrystals] = 3
		before := g.players
		g.player = 0
		g.players[0].par[cardTab[tc.card].typ] += cardTab[tc.card].costs // to pay it
		g.players[0].cards[0] = tc.card
		g.permission()
		g.putCard(0, false)
		for i := 0; i < parNum; i++ {
			own := g.players[0].par[i] - before[0].par[i]
			enemy := g.players[1].par[i] - before[1].par[i]
			// the enemy (now on turn) got its stocks
			switch i {
			case parBricks:
				enemy -= g.players[1].par[parBuilders]
			case parWeapons:
				enemy -= g.players[1].par[parSoldiers]
			case parCrystals:
				enemy -= g.players[1].par[parWizards]
			}
			if own != tc.own[i] || enemy != tc.enemy[i] {
				t.Errorf("card %d, parameter %d: %+d and %+d, want %+d and %+d", tc.card, i, own, enemy, tc.own[i], tc.enemy[i])
			}
		}
	}
}

// TestOneGame: with OneGame, Run returns when the winner has danced (a
// key), and on Y in a game.
func TestOneGame(t *testing.T) {
	g, _, clock, p := newTest(t, 1)
	g.OneGame = true
	p.Tap(hal.PadRight, 400*ms) // the black ants against the computer
	p.Tap(hal.PadA, 600*ms)
	clock.At(1200*ms, func() { g.players[1].par[parCastle] = 0 }) // the next card wins
	p.Tap(hal.PadA, 1400*ms)
	clock.At(5*time.Second, func() {
		if g.endGame != 0 {
			t.Errorf("the winner: %d", g.endGame)
		}
	})
	p.Tap(hal.PadB, 6*time.Second) // ends the dance
	g.Run()
	if clock.T < 6*time.Second || clock.T > 6500*ms {
		t.Errorf("Run returned at %v", clock.T)
	}

	g, _, clock, p = newTest(t, 1)
	g.OneGame = true
	p.Tap(hal.PadRight, 400*ms)
	p.Tap(hal.PadA, 600*ms)
	p.Tap(hal.PadY, 1400*ms)
	g.Run()
	if clock.T > 1800*ms {
		t.Errorf("Y: Run returned at %v", clock.T)
	}
}
