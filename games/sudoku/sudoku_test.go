package sudoku

import "testing"

// valid reports whether b is a complete, correct grid.
func valid(b Board) bool {
	for k := 0; k < 9; k++ {
		var row, col, box uint16
		for j := 0; j < 9; j++ {
			row |= 1 << b[k*9+j]
			col |= 1 << b[j*9+k]
			box |= 1 << b[(k/3*3+j/3)*9+k%3*3+j%3]
		}
		if row != allDigits || col != allDigits || box != allDigits {
			return false
		}
	}
	return true
}

func parse(s string) Board {
	var b Board
	i := 0
	for _, c := range s {
		switch {
		case c >= '1' && c <= '9':
			b[i] = uint8(c - '0')
			i++
		case c == '.' || c == '0':
			i++
		}
	}
	return b
}

func TestSolveKnownPuzzle(t *testing.T) {
	// A well-known puzzle with a unique solution.
	p := parse(`
		53..7....
		6..195...
		.98....6.
		8...6...3
		4..8.3..1
		7...2...6
		.6....28.
		...419..5
		....8..79`)
	want := parse(`
		534678912
		672195348
		198342567
		859761423
		426853791
		713924856
		961537284
		287419635
		345286179`)
	got, ok := Solve(p)
	if !ok || got != want {
		t.Fatalf("got %v %v", got, ok)
	}
	if n := CountSolutions(p, 3); n != 1 {
		t.Fatalf("%d solutions", n)
	}
	// Removing clues from a unique puzzle can allow more solutions.
	if n := CountSolutions(Board{}, 2); n != 2 {
		t.Fatalf("empty board: %d", n)
	}
}

func TestGenerate(t *testing.T) {
	for _, df := range difficulties {
		for seed := uint32(1); seed <= 20; seed++ {
			p, s := Generate(seed, df.Clues)
			if !valid(s) {
				t.Fatalf("%s seed %d: solution invalid", df.Name, seed)
			}
			clues := 0
			for i, v := range p {
				if v != 0 {
					clues++
					if v != s[i] {
						t.Fatalf("%s seed %d: clue differs from the solution", df.Name, seed)
					}
				}
			}
			if n := CountSolutions(p, 2); n != 1 {
				t.Fatalf("%s seed %d: %d solutions", df.Name, seed, n)
			}
			// Uniqueness may keep a few more clues than asked for.
			if clues < df.Clues || clues > df.Clues+6 {
				t.Errorf("%s seed %d: %d clues", df.Name, seed, clues)
			}
		}
	}
	a, _ := Generate(7, 30)
	b, _ := Generate(7, 30)
	c, _ := Generate(8, 30)
	if a != b || a == c {
		t.Fatal("generation is not repeatable per seed")
	}
}
