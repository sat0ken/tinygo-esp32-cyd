package main

// Puzzle generation and solving. Pure logic, no drawing.
//
// The search is iterative (an explicit stack instead of recursion) so its
// stack use on the board is small and fixed.

// Board is a 9x9 grid in row-major order; 0 means empty.
type Board [81]uint8

const allDigits = 0x3FE // bits 1..9

// rng is a small xorshift32 generator, so games are repeatable in tests.
type rng struct{ s uint32 }

func newRNG(seed uint32) *rng {
	if seed == 0 {
		seed = 0x9E3779B9
	}
	return &rng{s: seed}
}

func (r *rng) next() uint32 {
	r.s ^= r.s << 13
	r.s ^= r.s >> 17
	r.s ^= r.s << 5
	return r.s
}

// intn returns 0..n-1.
func (r *rng) intn(n int) int { return int(r.next() % uint32(n)) }

// candidates returns the digits (as bits 1..9) that can go into cell i.
func (b *Board) candidates(i int) uint16 {
	row, col := i/9, i%9
	used := uint16(0)
	for k := 0; k < 9; k++ {
		used |= 1 << b[row*9+k]
		used |= 1 << b[k*9+col]
	}
	br, bc := row/3*3, col/3*3
	for r := br; r < br+3; r++ {
		for c := bc; c < bc+3; c++ {
			used |= 1 << b[r*9+c]
		}
	}
	return allDigits &^ used
}

func popcount(m uint16) int {
	n := 0
	for ; m != 0; m &= m - 1 {
		n++
	}
	return n
}

// mostConstrained returns the empty cell with the fewest candidates, or -1
// if the board is full.
func (b *Board) mostConstrained() (cell int, cand uint16) {
	cell, best := -1, 10
	for i := 0; i < 81; i++ {
		if b[i] != 0 {
			continue
		}
		m := b.candidates(i)
		if n := popcount(m); n < best {
			cell, cand, best = i, m, n
			if n <= 1 {
				break
			}
		}
	}
	return cell, cand
}

// takeDigit removes one digit from *m and returns it: the lowest one, or a
// random one if r is not nil.
func takeDigit(m *uint16, r *rng) uint8 {
	k := 0
	if r != nil {
		k = r.intn(popcount(*m))
	}
	for d := uint8(1); d <= 9; d++ {
		if *m&(1<<d) != 0 {
			if k == 0 {
				*m &^= 1 << d
				return d
			}
			k--
		}
	}
	return 0
}

// search runs a depth-first search on a copy of b. It stops after `limit`
// solutions and returns how many it found (at most limit) and the first
// one. With r != nil, digits are tried in random order.
func search(b Board, limit int, r *rng) (count int, first Board) {
	var cells [81]int
	var rest [81]uint16 // digits not tried yet for cells[d]
	depth := 0
	for {
		cell, cand := b.mostConstrained()
		descend := false
		switch {
		case cell < 0: // full board: a solution
			if count == 0 {
				first = b
			}
			count++
			if count >= limit {
				return count, first
			}
		case cand != 0:
			cells[depth] = cell
			b[cell] = takeDigit(&cand, r)
			rest[depth] = cand
			depth++
			descend = true
		}
		if descend {
			continue
		}
		// Backtrack to the deepest cell that still has digits to try.
		for {
			if depth == 0 {
				return count, first
			}
			top := depth - 1
			c := cells[top]
			if rest[top] != 0 {
				b[c] = takeDigit(&rest[top], r)
				break
			}
			b[c] = 0
			depth--
		}
	}
}

// Solve returns the first solution of b, and whether there is one.
func Solve(b Board) (Board, bool) {
	n, s := search(b, 1, nil)
	return s, n == 1
}

// CountSolutions returns the number of solutions of b, stopping at limit.
func CountSolutions(b Board, limit int) int {
	n, _ := search(b, limit, nil)
	return n
}

// Generate makes a puzzle with a unique solution. It starts from a random
// full grid and empties cells in random order, keeping a cell only if
// emptying it would allow a second solution, until `clues` cells are left
// (or no more cell can be emptied).
func Generate(seed uint32, clues int) (puzzle, solution Board) {
	r := newRNG(seed)
	_, solution = search(Board{}, 1, r)
	puzzle = solution

	var order [81]int
	for i := range order {
		order[i] = i
	}
	for i := 80; i > 0; i-- {
		j := r.intn(i + 1)
		order[i], order[j] = order[j], order[i]
	}
	filled := 81
	for _, i := range order {
		if filled <= clues {
			break
		}
		v := puzzle[i]
		puzzle[i] = 0
		if CountSolutions(puzzle, 2) != 1 {
			puzzle[i] = v // needed for uniqueness
			continue
		}
		filled--
	}
	return puzzle, solution
}
