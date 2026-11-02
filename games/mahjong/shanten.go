package mahjong

// suitTable is the best a suit (or the honors) can do: the number of
// incomplete groups for each number of complete groups (0-4) and pair (0-1),
// -1 if that is not possible.
type suitTable [5][2]int8

// suitCache remembers the tables: the CPU asks for the same suits again and
// again while it compares discards.
var suitCache [256]struct {
	key uint32 // 0: empty
	tab suitTable
}

// tableOf returns the table of counts c (9 kinds, or 7 honors without runs).
func tableOf(c []uint8, runs bool) *suitTable {
	key := uint32(1)
	for _, v := range c {
		key = key<<3 | uint32(v)
	}
	if runs {
		key |= 1 << 31 // 9 x 3 bits and the leading 1 end below this one
	}
	e := &suitCache[(key^key>>9^key>>18)&255]
	if e.key == key {
		return &e.tab
	}
	e.key = key
	for i := range e.tab {
		e.tab[i] = [2]int8{-1, -1}
	}
	var w [9]uint8
	copy(w[:], c)
	scanSuit(&w, len(c), runs, 0, 0, 0, 0, &e.tab)
	return &e.tab
}

// scanSuit tries every way to split the tiles from position i on.
func scanSuit(c *[9]uint8, n int, runs bool, i, groups, partial, pair int, tab *suitTable) {
	for i < n && c[i] == 0 {
		i++
	}
	if i == n {
		if int8(partial) > tab[groups][pair] {
			tab[groups][pair] = int8(partial)
		}
		return
	}
	if groups < 4 && c[i] >= 3 {
		c[i] -= 3
		scanSuit(c, n, runs, i, groups+1, partial, pair, tab)
		c[i] += 3
	}
	if groups < 4 && runs && i+2 < n && c[i+1] > 0 && c[i+2] > 0 {
		c[i]--
		c[i+1]--
		c[i+2]--
		scanSuit(c, n, runs, i, groups+1, partial, pair, tab)
		c[i]++
		c[i+1]++
		c[i+2]++
	}
	if c[i] >= 2 {
		c[i] -= 2
		if pair == 0 {
			scanSuit(c, n, runs, i, groups, partial, 1, tab)
		}
		if partial < 4 {
			scanSuit(c, n, runs, i, groups, partial+1, pair, tab)
		}
		c[i] += 2
	}
	if runs && partial < 4 {
		for d := 1; d <= 2; d++ {
			if i+d < n && c[i+d] > 0 {
				c[i]--
				c[i+d]--
				scanSuit(c, n, runs, i, groups, partial+1, pair, tab)
				c[i]++
				c[i+d]++
			}
		}
	}
	// This copy of the tile stays alone.
	saved := c[i]
	c[i] = 0
	scanSuit(c, n, runs, i+1, groups, partial, pair, tab)
	c[i] = saved
}

// shantenGroups is how many tiles the hand is from being ready as groups and
// a pair (-1: it is complete). melds is the number of groups already called.
func shantenGroups(c *Counts, melds int) int {
	cur := [5][2]int8{}
	for i := range cur {
		cur[i] = [2]int8{-1, -1}
	}
	cur[melds][0] = 0
	for s := 0; s < 4; s++ {
		var t *suitTable
		if s < 3 {
			t = tableOf(c[s*9:s*9+9], true)
		} else {
			t = tableOf(c[27:34], false)
		}
		var next [5][2]int8
		for i := range next {
			next[i] = [2]int8{-1, -1}
		}
		for g := 0; g <= 4; g++ {
			for p := 0; p < 2; p++ {
				if cur[g][p] < 0 {
					continue
				}
				for g2 := 0; g+g2 <= 4; g2++ {
					for p2 := 0; p+p2 < 2; p2++ {
						if t[g2][p2] >= 0 {
							if v := cur[g][p] + t[g2][p2]; v > next[g+g2][p+p2] {
								next[g+g2][p+p2] = v
							}
						}
					}
				}
			}
		}
		cur = next
	}
	best := 8
	for g := 0; g <= 4; g++ {
		for p := 0; p < 2; p++ {
			if cur[g][p] < 0 {
				continue
			}
			partial := int(cur[g][p])
			if g+partial > 4 {
				partial = 4 - g // five blocks are not better than four
			}
			if v := 8 - 2*g - partial - p; v < best {
				best = v
			}
		}
	}
	return best
}

// shantenPairs is the same for seven pairs.
func shantenPairs(c *Counts) int {
	pairs, kinds := 0, 0
	for _, v := range c {
		if v > 0 {
			kinds++
		}
		if v >= 2 {
			pairs++
		}
	}
	v := 6 - pairs
	if kinds < 7 {
		v += 7 - kinds
	}
	return v
}

// shantenOrphans is the same for thirteen orphans.
func shantenOrphans(c *Counts) int {
	kinds, pair := 0, 0
	for t := Tile(0); t < numKinds; t++ {
		if t.isEdge() && c[t] > 0 {
			kinds++
			if c[t] >= 2 {
				pair = 1
			}
		}
	}
	return 13 - kinds - pair
}

// shanten is how many tiles the hand is from being ready, with any shape.
func shanten(c *Counts, melds int) int {
	best := shantenGroups(c, melds)
	if melds == 0 {
		best = min(best, shantenPairs(c), shantenOrphans(c))
	}
	return best
}

// waitsOf is the tiles that complete a 13-tile hand (less for each meld). A
// tile the hand holds all four of does not count.
func waitsOf(c *Counts, melds int) Mask {
	if shanten(c, melds) != 0 {
		return 0
	}
	var m Mask
	for t := Tile(0); t < numKinds; t++ {
		if c[t] >= 4 {
			continue
		}
		c[t]++
		if shanten(c, melds) == -1 {
			m |= 1 << t
		}
		c[t]--
	}
	return m
}
