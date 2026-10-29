package chess

import (
	"fmt"
	"strings"
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
	clock.Step = 100 * time.Microsecond // the cursor blinks in a loop without sleeping
	g.Clock = clock
	p := &picotest.Pad{Clock: clock}
	g.Pad = p
	return g, d, clock, p
}

// perft counts the positions after depth plies, walking the moves as
// moveCompLevel does.
func (g *Game) perft(depth int) int {
	level := g.moveDeep
	g.searchMoves(0)
	n := g.moveStackNum[level]
	if depth <= 1 {
		return n
	}
	total := 0
	g.moveDeep = level + 1
	start := g.moveStackStart[level]
	for k := 0; k < n; k++ {
		m := &g.moveStack[start+k]
		g.movePassantPiece[level+1] = 0
		g.movePassantPos[level+1] = 0
		if m.srcPiece&(pieceMask|moveMask) == pawn|noMoving {
			if dif := int(m.dst) - int(m.src); dif == 20 || dif == -20 {
				g.movePassantPiece[level+1] = m.dst
				g.movePassantPos[level+1] = uint8(int(m.src) + dif/2)
			}
		}
		g.doMove(m)
		g.player ^= 1
		top := g.moveStackTop
		total += g.perft(depth - 1)
		g.moveStackTop = top
		g.player ^= 1
		g.undoMove(m)
	}
	g.moveDeep = level
	return total
}

func TestPerft(t *testing.T) {
	g := New(memlcd.New(320, 240), nil, 1)
	g.newGame()
	// the known numbers of chess, which the C source also gives
	// (testdata/cperft/run.sh)
	for d, want := range []int{20, 400, 8902, 197281} {
		g.moveDeep, g.moveStackTop = 0, 0
		if got := g.perft(d + 1); got != want {
			t.Errorf("perft %d: %d, want %d", d+1, got, want)
		}
	}
}

// TestComputerMatchesC plays the computer against itself and compares the
// moves with the C source compiled on the host with the same random
// numbers (PicoLibSDK's lib_rand.c seeded like picopad.New with 7):
// testdata/cperft/run.sh prints them.
func TestComputerMatchesC(t *testing.T) {
	want := map[int]string{
		2: "35-55 84-74 22-41 85-65 34-44 82-72 41-22 97-85 37-47 87-77 31-51 83-73 25-35 93-84 33-53 73-63",
		3: "35-55 84-74 24-46 87-77 27-35 96-87 46-44 83-73 35-56 87-65 56-35 95-96 44-53 81-71 53-44 94-83",
	}
	for level, moves := range want {
		g, _, _, _ := newTest(t, 7)
		g.newGame()
		g.deepMaxN = level
		var got []string
		for k := 0; k < 16; k++ {
			g.moveComp()
			if !g.lastMoveOK {
				got = append(got, "END")
				break
			}
			got = append(got, fmt.Sprintf("%d-%d", g.lastMove.src, g.lastMove.dst))
			g.player ^= 1
			if g.player == whitePlayer {
				g.moveNum++
			}
		}
		if s := strings.Join(got, " "); s != moves {
			t.Errorf("level %d:\n got %s\nwant %s", level, s, moves)
		}
	}
}

func TestScreens(t *testing.T) {
	g, d, clock, p := newTest(t, 3)
	clock.At(100*ms, func() { goldentest.Check(t, "chess_open", d.Image()) })
	p.Tap(hal.PadLeft, 200*ms) // play white
	clock.At(300*ms, func() {
		// the board, the cursor on the king (e1)
		goldentest.Check(t, "chess_board", d.Image())
	})
	p.Tap(hal.PadY, 400*ms)
	p.Tap(hal.PadY, 600*ms)
	g.Run()
}

func TestHumanMove(t *testing.T) {
	g, _, clock, p := newTest(t, 3)
	p.Tap(hal.PadA, 100*ms)    // level: SUPERPRO
	p.Tap(hal.PadA, 300*ms)    // level: BEGINNER
	p.Tap(hal.PadLeft, 500*ms) // play white
	// e2-e4: up to the pawn, A, up twice, A
	at := 700 * ms
	for _, b := range []hal.Buttons{hal.PadUp, hal.PadA, hal.PadUp, hal.PadUp, hal.PadA} {
		p.Tap(b, at)
		at += 200 * ms
	}
	// the move is shown (1.6 seconds), then the computer answers
	clock.At(at+5*time.Second, func() {
		if g.board[35] != empty || g.board[55] != pawn+white {
			t.Errorf("e2-e4: e2 %#x, e4 %#x", g.board[35], g.board[55])
		}
		if row := string(g.textBuf[:textW]); row != "  1:e2-e4 " { // "%3d:" as the C source
			t.Errorf("text console: %q", row)
		}
		if row := string(g.textBuf[textW : 2*textW]); !strings.HasPrefix(row, "    ") || row[4] < 'a' || row[4] > 'h' {
			t.Errorf("black's answer: %q", row)
		}
		if g.player != whitePlayer || g.moveNum != 2 {
			t.Errorf("player %d on turn, move %d", g.player, g.moveNum)
		}
	})
	p.Tap(hal.PadY, at+5500*ms)
	p.Tap(hal.PadY, at+6*time.Second)
	g.Run()
}

func TestCancelPick(t *testing.T) {
	g, _, clock, p := newTest(t, 3)
	p.Tap(hal.PadDown, 100*ms) // 2 players
	p.Tap(hal.PadUp, 300*ms)   // to the pawn e2
	p.Tap(hal.PadA, 500*ms)    // pick it up
	p.Tap(hal.PadUp, 700*ms)
	p.Tap(hal.PadB, 900*ms) // put it back
	clock.At(1200*ms, func() {
		if g.board[35] != pawn+white+noMoving || g.board[45] != empty || g.players[whitePlayer].curPos != 35 {
			t.Errorf("after B: e2 %#x, e3 %#x, cursor %d", g.board[35], g.board[45], g.players[whitePlayer].curPos)
		}
	})
	p.Tap(hal.PadY, 1400*ms)
	p.Tap(hal.PadY, 1600*ms)
	g.Run()
}
