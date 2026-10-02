package othello

import "math/bits"

// Rules and the CPU player. Pure logic, no drawing.
//
// The board is two bitboards: bit r*8+c is the square in row r, column c
// (a1 = bit 0 is the top-left square, h8 = bit 63 the bottom-right).

// Position is the board and who is to move.
type Position struct {
	Me, Opp uint64 // stones of the side to move and of the other side
}

const (
	notColA = 0xfefefefefefefefe // all squares except column 0
	notColH = 0x7f7f7f7f7f7f7f7f // all squares except column 7
)

// shift moves every stone of x one square in direction d (0..7) without
// wrapping around the board edges.
func shift(x uint64, d int) uint64 {
	switch d {
	case 0: // east
		return (x << 1) & notColA
	case 1: // west
		return (x >> 1) & notColH
	case 2: // south
		return x << 8
	case 3: // north
		return x >> 8
	case 4: // south-east
		return (x << 9) & notColA
	case 5: // south-west
		return (x << 7) & notColH
	case 6: // north-east
		return (x >> 7) & notColA
	default: // north-west
		return (x >> 9) & notColH
	}
}

// Start is the standard opening position, black (the first player) to move.
func Start() Position {
	return Position{
		Me:  1<<(3*8+4) | 1<<(4*8+3), // e4, d5: black
		Opp: 1<<(3*8+3) | 1<<(4*8+4), // d4, e5: white
	}
}

// Moves returns the legal moves of the side to move.
func (p Position) Moves() uint64 {
	empty := ^(p.Me | p.Opp)
	moves := uint64(0)
	for d := 0; d < 8; d++ {
		// Runs of opponent stones next to my stones in this direction...
		t := shift(p.Me, d) & p.Opp
		for i := 0; i < 5; i++ {
			t |= shift(t, d) & p.Opp
		}
		// ...followed by an empty square.
		moves |= shift(t, d) & empty
	}
	return moves
}

// Flips returns the stones turned by playing square sq (0..63).
func (p Position) Flips(sq int) uint64 {
	m := uint64(1) << sq
	flips := uint64(0)
	for d := 0; d < 8; d++ {
		f := uint64(0)
		x := shift(m, d)
		for x&p.Opp != 0 {
			f |= x
			x = shift(x, d)
		}
		if x&p.Me != 0 {
			flips |= f
		}
	}
	return flips
}

// Play returns the position after the side to move plays sq; the other
// side is then to move. sq must be legal.
func (p Position) Play(sq int) Position {
	f := p.Flips(sq)
	me := p.Me | f | 1<<sq
	return Position{Me: p.Opp &^ f, Opp: me}
}

// Pass returns the position with the other side to move.
func (p Position) Pass() Position { return Position{Me: p.Opp, Opp: p.Me} }

// Over reports whether neither side can move.
func (p Position) Over() bool { return p.Moves() == 0 && p.Pass().Moves() == 0 }

// Count returns the stones of the side to move and of the other side.
func (p Position) Count() (me, opp int) { return bits.OnesCount64(p.Me), bits.OnesCount64(p.Opp) }

func (p Position) empties() int { return 64 - bits.OnesCount64(p.Me|p.Opp) }

// ---- evaluation ----

// Square values: corners are best, the squares next to them are bad
// because they let the opponent take the corner.
var weights = [64]int{
	120, -20, 20, 5, 5, 20, -20, 120,
	-20, -40, -5, -5, -5, -5, -40, -20,
	20, -5, 15, 3, 3, 15, -5, 20,
	5, -5, 3, 3, 3, 3, -5, 5,
	5, -5, 3, 3, 3, 3, -5, 5,
	20, -5, 15, 3, 3, 15, -5, 20,
	-20, -40, -5, -5, -5, -5, -40, -20,
	120, -20, 20, 5, 5, 20, -20, 120,
}

// evaluate scores p for the side to move (positive is good for it).
func evaluate(p Position) int {
	s := 0
	for b := p.Me; b != 0; b &= b - 1 {
		s += weights[bits.TrailingZeros64(b)]
	}
	for b := p.Opp; b != 0; b &= b - 1 {
		s -= weights[bits.TrailingZeros64(b)]
	}
	// Mobility: having more moves than the opponent is good.
	s += 8 * (bits.OnesCount64(p.Moves()) - bits.OnesCount64(p.Pass().Moves()))
	return s
}

const (
	inf      = 1 << 20
	winScore = 1 << 16 // final positions beat any evaluation
)

// final scores a finished game by the stone difference.
func final(p Position) int {
	me, opp := p.Count()
	switch {
	case me > opp:
		return winScore + me - opp
	case me < opp:
		return -winScore + me - opp
	}
	return 0
}

// ---- search ----

// Level is how strong the CPU plays.
type Level struct {
	Name     string
	Depth    int // look-ahead in moves
	Exact    int // search to the end when this many squares or fewer are empty
	MaxNodes int // node budget per move (keeps thinking time bounded)
	Noise    int // random noise added to move scores (makes it weaker)
}

var levels = []Level{
	{Name: "EASY", Depth: 1, Noise: 60, MaxNodes: 1000},
	{Name: "NORMAL", Depth: 3, Exact: 8, MaxNodes: 40_000},
	{Name: "HARD", Depth: 6, Exact: 12, MaxNodes: 250_000},
}

type searcher struct {
	nodes, max int
	aborted    bool
}

// negamax returns the score of p for the side to move, searching depth
// moves ahead (or to the end of the game when exact).
func (s *searcher) negamax(p Position, depth int, alpha, beta int, exact bool, passed bool) int {
	s.nodes++
	if s.nodes > s.max {
		s.aborted = true
		return 0
	}
	moves := p.Moves()
	if moves == 0 {
		if passed { // neither side can move
			return final(p)
		}
		return -s.negamax(p.Pass(), depth, -beta, -alpha, exact, true)
	}
	if !exact && depth == 0 {
		return evaluate(p)
	}
	var list [maxMoves]int
	n := order(moves, &list)
	best := -inf
	for _, sq := range list[:n] {
		v := -s.negamax(p.Play(sq), depth-1, -beta, -alpha, exact, false)
		if s.aborted {
			return 0
		}
		if v > best {
			best = v
		}
		if v > alpha {
			alpha = v
		}
		if alpha >= beta {
			break
		}
	}
	return best
}

// maxMoves is more than the most legal moves an othello position can have (33).
const maxMoves = 34

// order writes the moves into list best-looking first (by square value),
// which makes alpha-beta cut off more, and returns how many there are.
// The list lives in the caller's stack frame, so the search does not
// allocate.
func order(moves uint64, list *[maxMoves]int) int {
	n := 0
	for b := moves; b != 0; b &= b - 1 {
		sq := bits.TrailingZeros64(b)
		i := n
		for i > 0 && weights[list[i-1]] < weights[sq] {
			list[i] = list[i-1]
			i--
		}
		list[i] = sq
		n++
	}
	return n
}

// BestMove picks a move for the side to move (which must have one).
// It searches 1, 2, ... moves ahead (iterative deepening) up to the depth
// of the level, then, near the end of the game, once to the very end. If
// the node budget runs out, the move of the last complete search is used.
// noise(n) supplies randomness for weak levels. It also returns the depth
// reached (-1 for a search to the end) and the nodes searched.
func BestMove(p Position, lv Level, noise func(n int) int) (sq, depth, nodes int) {
	var list [maxMoves]int
	n := order(p.Moves(), &list)
	moves := list[:n]
	sq = moves[0]

	maxDepth := lv.Depth
	if e := p.empties(); maxDepth > e {
		maxDepth = e
	}
	exact := p.empties() <= lv.Exact
	plan := make([]int, 0, maxDepth+1)
	for d := 1; d <= maxDepth; d++ {
		plan = append(plan, d)
	}
	if exact {
		plan = append(plan, -1) // to the end
	}

	total := 0
	for _, d := range plan {
		s := &searcher{max: lv.MaxNodes - total}
		best, bestSq := -inf, moves[0]
		alpha := -inf
		for _, m := range moves {
			v := -s.negamax(p.Play(m), d-1, -inf, -alpha, d < 0, false)
			if s.aborted {
				break
			}
			if lv.Noise > 0 && noise != nil {
				v += noise(lv.Noise)
			}
			if v > best {
				best, bestSq = v, m
			}
			if v > alpha {
				alpha = v
			}
		}
		total += s.nodes
		if s.aborted {
			break
		}
		sq, depth = bestSq, d
	}
	return sq, depth, total
}
