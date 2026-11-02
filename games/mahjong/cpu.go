package mahjong

// visibleTiles counts the tiles seat can see.
func (t *Table) visibleTiles(seat int) Counts {
	v := t.P[seat].Hand
	for _, ind := range t.Indicators() {
		v[ind]++
	}
	for s := range t.P {
		p := &t.P[s]
		for i := 0; i < p.NRiv; i++ {
			if !p.River[i].Called {
				v[p.River[i].Tile]++
			}
		}
		for i := 0; i < p.NMeld; i++ {
			m := &p.Melds[i]
			for k := Tile(0); k < numKinds; k++ {
				v[k] += uint8(m.count(k))
			}
		}
	}
	return v
}

// ukeire counts the tiles still unseen that bring the hand closer.
func ukeire(h *Counts, melds, sh int, seen *Counts) int {
	n := 0
	for k := Tile(0); k < numKinds; k++ {
		if seen[k] >= 4 {
			continue
		}
		h[k]++
		if shanten(h, melds) < sh {
			n += 4 - int(seen[k])
		}
		h[k]--
	}
	return n
}

// yakuhai reports a tile whose set is a yaku for the seat.
func (t *Table) yakuhai(seat int, k Tile) bool {
	return k >= white || k == t.SeatWind(seat) || k == t.RoundWind
}

// The ways an open hand can get its yaku.
const (
	planAny     = iota // it has one already: a set of honors
	planTanyao         // only simples
	planFlush          // one suit and honors
	planNothing        // nothing in sight
)

// openPlan is how an open hand of the seat goes on.
func (t *Table) openPlan(seat int) (plan, suit int) {
	p := &t.P[seat]
	simple, oneSuit := true, -1
	flush := true
	for i := 0; i < p.NMeld; i++ {
		m := &p.Melds[i]
		if m.Kind != chiMeld && m.Tile.isHonor() && t.yakuhai(seat, m.Tile) {
			return planAny, 0
		}
		if m.Tile.isEdge() || m.Kind == chiMeld && m.Tile.number() == 6 {
			simple = false
		}
		if !m.Tile.isHonor() {
			if oneSuit >= 0 && oneSuit != m.Tile.suit() {
				flush = false
			}
			oneSuit = m.Tile.suit()
		}
	}
	switch {
	case simple:
		return planTanyao, 0
	case flush:
		if oneSuit < 0 {
			oneSuit = t.bestSuit(&p.Hand)
		}
		return planFlush, oneSuit
	}
	return planNothing, 0
}

func (t *Table) bestSuit(h *Counts) int {
	best, bestN := 0, -1
	for s := 0; s < 3; s++ {
		n := 0
		for _, v := range h[s*9 : s*9+9] {
			n += int(v)
		}
		if n > bestN {
			best, bestN = s, n
		}
	}
	return best
}

// safety says how safe k is against a player in riichi: higher is safer.
func (t *Table) safety(seat int, k Tile, seen *Counts) int {
	score := 0
	for s := range t.P {
		q := &t.P[s]
		if s == seat || !q.Riichi {
			continue
		}
		score += t.safeAgainst(q, k, seen)
	}
	return score
}

func (t *Table) safeAgainst(q *Player, k Tile, seen *Counts) int {
	for i := 0; i < q.NRiv; i++ {
		if q.River[i].Tile == k {
			return 100 // the player cannot win with it
		}
	}
	switch {
	case k.isHonor():
		return 50 + 10*int(seen[k])
	case t.isSuji(q, k):
		return 40
	case k.isTerminal():
		return 25
	}
	return 0
}

// isSuji reports that k is three away from something the player discarded.
func (t *Table) isSuji(q *Player, k Tile) bool {
	for i := 0; i < q.NRiv; i++ {
		r := q.River[i].Tile
		if r.isHonor() || r.suit() != k.suit() {
			continue
		}
		d := r.number() - k.number()
		if d == 3 || d == -3 {
			return true
		}
	}
	return false
}

// keepValue is how much the CPU wants to keep k, besides the shape.
func (t *Table) keepValue(seat int, k Tile, dora *Counts) int {
	p := &t.P[seat]
	v := int(dora[k]) * 30
	switch {
	case k.isHonor():
		if t.yakuhai(seat, k) {
			v += 12
			if p.Hand[k] >= 2 {
				v += 30
			}
		} else if p.Hand[k] >= 2 {
			v += 8
		}
	case k.isTerminal():
		v += 2
	default:
		v += 6
	}
	return v
}

// discardScore compares the discards: higher is better.
func (t *Table) discardScore(seat int, k Tile, seen, dora *Counts, plan, suit int) (score, sh int) {
	p := &t.P[seat]
	p.Hand[k]--
	sh = shanten(&p.Hand, p.NMeld)
	score = -2000 * sh
	if sh >= 0 {
		score += 10 * ukeire(&p.Hand, p.NMeld, sh, seen)
	}
	p.Hand[k]++
	score -= t.keepValue(seat, k, dora)
	if p.isOpen() {
		switch plan {
		case planTanyao:
			if k.isEdge() {
				score += 500
			}
		case planFlush:
			if !k.isHonor() && k.suit() != suit {
				score += 800
			}
		}
	}
	return score, sh
}

// CPUTurn does what the CPU does on its turn: win, kan, discard.
func (t *Table) CPUTurn() {
	seat := t.Turn
	p := &t.P[seat]
	if t.CanTsumo() {
		t.Tsumo()
		return
	}
	if t.CanAbort() {
		if shanten(&p.Hand, p.NMeld) >= 3 {
			t.Abort()
			return
		}
	}
	if p.Riichi {
		if kans := t.KanTiles(); kans != 0 {
			t.Kan(p.Drawn)
			return
		}
		t.DiscardTile(p.Drawn, false)
		return
	}

	seen := t.visibleTiles(seat)
	var dora Counts
	for _, ind := range t.Indicators() {
		dora[doraOf(ind)]++
	}
	danger := false
	for s := range t.P {
		danger = danger || s != seat && t.P[s].Riichi
	}

	// A kan that does not spoil the hand.
	if kans := t.KanTiles(); kans != 0 && !danger {
		before := shanten(&p.Hand, p.NMeld)
		for k := Tile(0); k < numKinds; k++ {
			if !kans.has(k) {
				continue
			}
			h := p.Hand
			added := h[k] == 1
			h[k] = 0
			melds := p.NMeld + 1
			if added {
				melds = p.NMeld
			}
			if shanten(&h, melds) <= before {
				t.Kan(k)
				return
			}
		}
	}

	plan, suit := planAny, 0
	if p.isOpen() {
		plan, suit = t.openPlan(seat)
	}
	best, bestScore, bestSh := noTile, 0, 0
	for k := Tile(0); k < numKinds; k++ {
		if !t.CanDiscard(k) {
			continue
		}
		score, sh := t.discardScore(seat, k, &seen, &dora, plan, suit)
		if best == noTile || score > bestScore {
			best, bestScore, bestSh = k, score, sh
		}
	}

	// Not close and somebody is in riichi: the safest tile.
	if danger && bestSh >= 1 {
		safest, safestScore := best, -1
		for k := Tile(0); k < numKinds; k++ {
			if !t.CanDiscard(k) {
				continue
			}
			if v := t.safety(seat, k, &seen); v > safestScore {
				safest, safestScore = k, v
			}
		}
		t.DiscardTile(safest, false)
		return
	}

	if t.RiichiTiles().has(best) {
		t.DiscardTile(best, true)
		return
	}
	t.DiscardTile(best, false)
}

// CPUAnswer answers to the discard for a seat.
func (t *Table) CPUAnswer(seat int) {
	o := t.OptsOf(seat)
	if o.Ron {
		t.AnswerClaim(seat, Claim{Kind: claimRon})
		return
	}
	t.AnswerClaim(seat, t.cpuCall(seat))
}

func (t *Table) cpuCall(seat int) Claim {
	o, p, k := t.OptsOf(seat), &t.P[seat], t.Discard0
	for s := range t.P {
		if s != seat && t.P[s].Riichi {
			return Claim{} // not with somebody in riichi
		}
	}
	before := shanten(&p.Hand, p.NMeld)
	if !p.isOpen() && before == 0 {
		return Claim{} // ready and closed: wait, the riichi is worth more
	}
	if o.Pon {
		h := p.Hand
		h[k] -= 2
		if shanten(&h, p.NMeld+1) < before && t.callHasYaku(seat, &h, k, k, true) {
			return Claim{Kind: claimPon}
		}
	}
	for i := 0; i < o.NChi; i++ {
		h := p.Hand
		lo := o.Chi[i]
		for x := lo; x <= lo+2; x++ {
			if x != k {
				h[x]--
			}
		}
		if shanten(&h, p.NMeld+1) < before && t.callHasYaku(seat, &h, lo, k, false) {
			return Claim{Kind: claimChi, Low: lo}
		}
	}
	return Claim{}
}

// callHasYaku reports that a hand can still get a yaku after a call: it keeps
// the CPU from opening hands that could never win.
func (t *Table) callHasYaku(seat int, h *Counts, low, k Tile, isPon bool) bool {
	p := &t.P[seat]
	if isPon && t.yakuhai(seat, k) {
		return true
	}
	for i := 0; i < p.NMeld; i++ {
		m := &p.Melds[i]
		if m.Kind != chiMeld && t.yakuhai(seat, m.Tile) {
			return true
		}
	}
	for x := Tile(0); x < numKinds; x++ {
		if x.isHonor() && h[x] >= 2 && t.yakuhai(seat, x) {
			return true // a pair that can become a set
		}
	}

	// Only simples: the tiles left and the groups called.
	edge := 0
	for x := Tile(0); x < numKinds; x++ {
		if x.isEdge() {
			edge += int(h[x])
		}
	}
	simple := edge <= 2
	if isPon && k.isEdge() || !isPon && (low.number() == 0 || low.number() == 6) {
		simple = false
	}
	for i := 0; i < p.NMeld; i++ {
		m := &p.Melds[i]
		if m.Tile.isEdge() || m.Kind == chiMeld && m.Tile.number() == 6 {
			simple = false
		}
	}
	if simple {
		return true
	}

	// One suit and honors.
	s := t.bestSuit(h)
	if !k.isHonor() && k.suit() != s {
		return false
	}
	off := 0
	for x := Tile(0); x < 27; x++ {
		if x.suit() != s {
			off += int(h[x])
		}
	}
	for i := 0; i < p.NMeld; i++ {
		if m := &p.Melds[i]; !m.Tile.isHonor() && m.Tile.suit() != s {
			return false
		}
	}
	return off <= 2
}
