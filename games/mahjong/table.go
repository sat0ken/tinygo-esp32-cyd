package mahjong

const (
	wallSize = 136
	liveSize = wallSize - 14 // the last 14 are the dead wall
	startPts = 25000
	topPts   = 30000 // the score the dealer needs to stop in the last hand
)

// The kinds of melds.
const (
	chiMeld = iota
	ponMeld
	openKan
	closedKan
	addedKan
)

// Meld is a called group or a declared kan.
type Meld struct {
	Kind   uint8
	Tile   Tile // the lowest tile
	Called Tile // the tile taken from the discarder
	From   int8 // the discarder's seat, -1 for a closed kan
}

func (m *Meld) isKan() bool { return m.Kind >= openKan }

// count is how many tiles of kind t the meld has.
func (m *Meld) count(t Tile) int {
	switch {
	case m.Kind == chiMeld:
		if t >= m.Tile && t <= m.Tile+2 {
			return 1
		}
	case t != m.Tile:
	case m.isKan():
		return 4
	default:
		return 3
	}
	return 0
}

// River is one discard.
type River struct {
	Tile   Tile
	Riichi bool // the riichi was declared with this tile
	Called bool // somebody took it
}

// Player is one seat at the table.
type Player struct {
	Hand  Counts // the closed tiles, with the drawn one
	Drawn Tile   // the tile just drawn, noTile if none
	Melds [4]Meld
	NMeld int
	River [36]River
	NRiv  int
	Score int

	Riichi, DoubleRiichi bool
	ippatsu              bool
	riichiStick          bool // declared with the last discard, not paid yet
	Waits                Mask // what completes the hand when it is 13 tiles
	furitenPass          bool // passed on a winning tile since the last discard
	furitenRiichi        bool // passed on a winning tile after the riichi
	forbid               Mask // the discards a call does not allow
	Last                 string
}

func (p *Player) isOpen() bool {
	for i := 0; i < p.NMeld; i++ {
		if p.Melds[i].Kind != closedKan {
			return true
		}
	}
	return false
}

func (p *Player) melds() []Meld { return p.Melds[:p.NMeld] }

// Furiten reports that the player cannot win with a discard.
func (p *Player) Furiten() bool {
	if p.furitenPass || p.furitenRiichi {
		return true
	}
	for i := 0; i < p.NRiv; i++ {
		if p.Waits.has(p.River[i].Tile) {
			return true
		}
	}
	return false
}

// The phases of a hand.
const (
	phaseDraw   = iota // the player whose turn it is draws next
	phaseTurn          // the player whose turn it is acts
	phaseClaims        // the others answer to a discard or an added kan
	phaseOver          // the hand has ended: see Result
)

// The kinds of answers to a discard.
const (
	claimPass = iota
	claimRon
	claimChi
	claimPon
	claimKan
)

// Claim is an answer to a discard.
type Claim struct {
	Kind uint8
	Low  Tile // chi: the lowest tile of the run
}

// ClaimOpts is what a player may answer to a discard.
type ClaimOpts struct {
	Ron, Pon, Kan bool
	Chi           [3]Tile // the lowest tiles of the runs possible
	NChi          int
}

func (o *ClaimOpts) any() bool { return o.Ron || o.Pon || o.Kan || o.NChi > 0 }

// The kinds of ends of a hand.
const (
	endWin = iota
	endDraw
	endAbort
)

// WinInfo is one winner of a hand.
type WinInfo struct {
	Seat, From int // From == Seat: self-draw
	Tile       Tile
	Hand       Counts
	Melds      [4]Meld
	NMeld      int
	Score      Score
	Points     int // what the hand is worth to the winner, without sticks
	Ura        [5]Tile
	NUra       int
}

// Result is how a hand ended.
type Result struct {
	Kind   int
	Reason string // why a hand was aborted
	Wins   [3]WinInfo
	NWins  int
	Delta  [4]int
	Tenpai [4]bool
	Hands  [4]Counts // the closed hands at a draw

	dealerKeeps bool
}

// Table is a game of mahjong: the rules, with no screen. The UI and the CPU
// act through Discard, Tsumo, Kan, AnswerClaim and so on.
type Table struct {
	rng uint32

	Long      bool // east and south rounds, not only east
	RoundWind Tile
	HandNo    int // 0-3 in the round: also the dealer's offset
	Dealer    int
	first     int // the first dealer's seat
	Honba     int
	Sticks    int

	P     [4]Player
	Turn  int
	Phase int

	wall   [wallSize]Tile
	next   int
	kans   int
	NDora  int // indicators shown
	doraTo int // indicators to show after the next discard

	Discarder int
	Discard0  Tile // the tile being answered to
	chankan   bool
	opts      [4]ClaimOpts
	claims    [4]Claim
	answered  [4]bool

	rinshan   bool
	pure      bool // no call yet and nobody has discarded twice
	discards  int
	kanSeats  [4]int
	riichis   int
	windFirst Tile

	Result   Result
	GameOver bool
}

// NewTable starts a game. long: east and south rounds.
func NewTable(seed uint32, long bool) *Table {
	if seed == 0 {
		seed = 1
	}
	t := &Table{rng: seed, Long: long, RoundWind: east}
	for i := range t.P {
		t.P[i].Score = startPts
	}
	t.first = t.random(4)
	t.Dealer = t.first
	t.deal()
	return t
}

func (t *Table) random(n int) int {
	t.rng ^= t.rng << 13
	t.rng ^= t.rng >> 17
	t.rng ^= t.rng << 5
	return int(t.rng % uint32(n))
}

// SeatWind is the wind of a seat in this hand.
func (t *Table) SeatWind(seat int) Tile { return east + Tile((seat-t.Dealer+4)%4) }

// Live is how many tiles can still be drawn.
func (t *Table) Live() int { return liveSize - t.next - t.kans }

// Indicators returns the dora indicators shown.
func (t *Table) Indicators() []Tile {
	var b [5]Tile
	for i := 0; i < t.NDora; i++ {
		b[i] = t.wall[liveSize+4+2*i]
	}
	return b[:t.NDora]
}

func (t *Table) uraIndicators() []Tile {
	var b [5]Tile
	for i := 0; i < t.NDora; i++ {
		b[i] = t.wall[liveSize+5+2*i]
	}
	return b[:t.NDora]
}

func (t *Table) deal() {
	for k := 0; k < numKinds; k++ {
		for c := 0; c < 4; c++ {
			t.wall[k*4+c] = Tile(k)
		}
	}
	for i := wallSize - 1; i > 0; i-- {
		j := t.random(i + 1)
		t.wall[i], t.wall[j] = t.wall[j], t.wall[i]
	}
	for i := range t.P {
		p := &t.P[i]
		score := p.Score
		*p = Player{Drawn: noTile, Score: score}
		for k := 0; k < 13; k++ {
			p.Hand[t.wall[i*13+k]]++
		}
		p.Waits = waitsOf(&p.Hand, 0)
	}
	t.next = 52
	t.kans, t.NDora, t.doraTo = 0, 1, 0
	t.chankan, t.rinshan = false, false
	t.pure = true
	t.discards = 0
	t.kanSeats = [4]int{}
	t.riichis = 0
	t.windFirst = noTile
	t.Turn = t.Dealer
	t.Phase = phaseDraw
	t.Result = Result{}
}

// Draw gives the next tile to the player whose turn it is, or ends the hand
// if there is none.
func (t *Table) Draw() {
	if t.Live() == 0 {
		t.exhausted()
		return
	}
	p := &t.P[t.Turn]
	p.Drawn = t.wall[t.next]
	t.next++
	p.Hand[p.Drawn]++
	t.Phase = phaseTurn
}

// drawKan gives the replacement tile after a kan.
func (t *Table) drawKan() {
	p := &t.P[t.Turn]
	p.Drawn = t.wall[liveSize+t.kans]
	t.kans++
	p.Hand[p.Drawn]++
	t.rinshan = true
	t.Phase = phaseTurn
}

func (t *Table) ctx(seat int, win Tile, tsumo bool) winCtx {
	p := &t.P[seat]
	c := winCtx{
		hand: p.Hand, melds: p.melds(), win: win, tsumo: tsumo,
		seatWind: t.SeatWind(seat), roundWind: t.RoundWind,
		riichi: p.Riichi, doubleRiichi: p.DoubleRiichi, ippatsu: p.ippatsu,
		dora: t.Indicators(),
	}
	if p.Riichi {
		c.ura = t.uraIndicators()
	}
	if tsumo {
		c.haitei = t.Live() == 0 && !t.rinshan
		c.rinshan = t.rinshan
		if t.pure && p.NRiv == 0 && p.NMeld == 0 {
			c.tenhou = seat == t.Dealer
			c.chiihou = seat != t.Dealer
		}
	} else {
		c.hand[win]++
		c.houtei = t.Live() == 0 && !t.chankan
		c.chankan = t.chankan
	}
	return c
}

// CanTsumo reports that the player whose turn it is can win with the drawn
// tile.
func (t *Table) CanTsumo() bool {
	p := &t.P[t.Turn]
	if p.Drawn == noTile || !p.Waits.has(p.Drawn) {
		return false
	}
	c := t.ctx(t.Turn, p.Drawn, true)
	_, ok := evaluate(&c)
	return ok
}

// CanAbort reports nine different terminals and honors in the first tiles.
func (t *Table) CanAbort() bool {
	p := &t.P[t.Turn]
	if !t.pure || p.NRiv > 0 || p.Drawn == noTile {
		return false
	}
	n := 0
	for k := Tile(0); k < numKinds; k++ {
		if k.isEdge() && p.Hand[k] > 0 {
			n++
		}
	}
	return n >= 9
}

// Abort ends the hand with nine terminals and honors.
func (t *Table) Abort() { t.abort("九種九牌") }

// CanDiscard reports that the player whose turn it is may discard k.
func (t *Table) CanDiscard(k Tile) bool {
	p := &t.P[t.Turn]
	if p.Hand[k] == 0 || p.forbid.has(k) {
		return false
	}
	return !p.Riichi || k == p.Drawn
}

// RiichiTiles is the discards with which the player can declare a riichi.
func (t *Table) RiichiTiles() Mask {
	p := &t.P[t.Turn]
	if p.Riichi || p.isOpen() || p.Score < 1000 || t.Live() < 4 || p.Drawn == noTile {
		return 0
	}
	var m Mask
	for k := Tile(0); k < numKinds; k++ {
		if p.Hand[k] == 0 {
			continue
		}
		p.Hand[k]--
		if shanten(&p.Hand, p.NMeld) == 0 && waitsOf(&p.Hand, p.NMeld) != 0 {
			m |= 1 << k
		}
		p.Hand[k]++
	}
	return m
}

// KanTiles is the kans the player can declare now.
func (t *Table) KanTiles() Mask {
	p := &t.P[t.Turn]
	if p.Drawn == noTile || t.Live() == 0 || t.totalKans() == 4 {
		return 0
	}
	var m Mask
	if p.Riichi {
		// Only with the drawn tile, and only if the waits stay.
		k := p.Drawn
		if p.Hand[k] == 4 {
			h := p.Hand
			h[k] = 0
			if waitsOf(&h, p.NMeld+1) == p.Waits {
				m |= 1 << k
			}
		}
		return m
	}
	for k := Tile(0); k < numKinds; k++ {
		if p.Hand[k] == 4 {
			m |= 1 << k
		}
	}
	for i := 0; i < p.NMeld; i++ {
		if q := &p.Melds[i]; q.Kind == ponMeld && p.Hand[q.Tile] > 0 {
			m |= 1 << q.Tile
		}
	}
	return m
}

func (t *Table) totalKans() int { return t.kans }

// Tsumo wins with the drawn tile.
func (t *Table) Tsumo() {
	seat := t.Turn
	t.commitStick()
	t.Result = Result{Kind: endWin}
	t.addWin(seat, seat, t.P[seat].Drawn)
	t.finishWins(seat)
}

// Kan declares a kan of k (closed, or added to a pon).
func (t *Table) Kan(k Tile) {
	p := &t.P[t.Turn]
	t.clearIppatsu()
	t.pure = false
	for i := 0; i < p.NMeld; i++ {
		if q := &p.Melds[i]; q.Kind == ponMeld && q.Tile == k {
			// The others may win with the added tile first.
			t.Discarder, t.Discard0, t.chankan = t.Turn, k, true
			if t.startClaims() {
				return
			}
			t.markPassed(-1)
			t.addKan()
			return
		}
	}
	p.Hand[k] = 0
	p.Melds[p.NMeld] = Meld{Kind: closedKan, Tile: k, Called: k, From: -1}
	p.NMeld++
	p.Drawn = noTile
	p.Last = "カン"
	t.NDora++ // shown at once
	t.kanSeats[t.Turn]++
	t.drawKan()
	t.updateWaits(p)
}

// addKan completes an added kan: nobody won with the tile.
func (t *Table) addKan() {
	p := &t.P[t.Discarder]
	t.chankan = false
	k := t.Discard0
	p.Hand[k]--
	for i := 0; i < p.NMeld; i++ {
		if q := &p.Melds[i]; q.Kind == ponMeld && q.Tile == k {
			q.Kind = addedKan
		}
	}
	p.Drawn = noTile
	p.Last = "カン"
	t.doraTo++
	t.kanSeats[t.Discarder]++
	t.Turn = t.Discarder
	t.drawKan()
	t.updateWaits(p)
}

// updateWaits computes the waits of a hand with no extra tile. With the
// replacement tile in the hand it does so without that tile.
func (t *Table) updateWaits(p *Player) {
	h := p.Hand
	if p.Drawn != noTile {
		h[p.Drawn]--
	}
	p.Waits = waitsOf(&h, p.NMeld)
}

func (t *Table) clearIppatsu() {
	for i := range t.P {
		t.P[i].ippatsu = false
	}
}

// DiscardTile discards k; riichi declares a riichi with it.
func (t *Table) DiscardTile(k Tile, riichi bool) {
	seat := t.Turn
	p := &t.P[seat]
	p.Hand[k]--
	p.Drawn = noTile
	p.forbid = 0
	p.furitenPass = false
	p.Last = ""
	if p.Riichi {
		p.ippatsu = false
	}
	if riichi {
		p.Riichi, p.ippatsu, p.riichiStick = true, true, true
		p.DoubleRiichi = t.pure && p.NRiv == 0
		p.Last = "リーチ"
		t.riichis++
	}
	p.River[p.NRiv] = River{Tile: k, Riichi: riichi}
	p.NRiv++
	t.updateWaits(p)
	t.rinshan = false
	t.NDora += t.doraTo
	t.doraTo = 0
	t.discards++
	if t.discards == 1 {
		t.windFirst = k
	}
	if t.windFirst != k || k < east || k > north {
		t.windFirst = noTile
	}
	t.Discarder, t.Discard0 = seat, k
	if !t.startClaims() {
		t.markPassed(-1)
		t.noClaim()
	}
}

// startClaims works out who can answer to the tile; false if nobody can.
func (t *Table) startClaims() bool {
	k := t.Discard0
	found := false
	for s := range t.P {
		t.opts[s] = ClaimOpts{}
		t.answered[s] = true
		t.claims[s] = Claim{}
		if s == t.Discarder {
			continue
		}
		t.opts[s] = t.optsFor(s, k)
		if t.opts[s].any() {
			t.answered[s] = false
			found = true
		}
	}
	if found {
		t.Phase = phaseClaims
	}
	return found
}

func (t *Table) optsFor(s int, k Tile) ClaimOpts {
	p := &t.P[s]
	var o ClaimOpts
	if p.Waits.has(k) && !p.Furiten() {
		c := t.ctx(s, k, false)
		_, o.Ron = evaluate(&c)
	}
	if t.chankan || p.Riichi || t.Live() == 0 {
		return o
	}
	if p.Hand[k] >= 2 && t.canDiscardAfter(p, k, k, k) {
		o.Pon = true
	}
	o.Kan = p.Hand[k] == 3 && t.totalKans() < 4
	if s == (t.Discarder+1)%4 && !k.isHonor() {
		n := k.number()
		for low := max(n-2, 0); low <= n && low <= 6; low++ {
			lo := k - Tile(n-low)
			a, b := lo, lo+1
			switch k {
			case lo:
				a, b = lo+1, lo+2
			case lo + 1:
				a, b = lo, lo+2
			}
			if p.Hand[a] > 0 && p.Hand[b] > 0 && t.canDiscardAfter(p, a, b, k) {
				o.Chi[o.NChi] = lo
				o.NChi++
			}
		}
	}
	return o
}

// forbidOf is what may not be discarded after a call with a and b of tile k:
// the same tile and, for a run called at one end, the tile at the other end.
func forbidOf(a, b, k Tile) Mask {
	m := Mask(1) << k
	lo, hi := min(a, b), max(a, b)
	if k.isHonor() || hi != lo+1 {
		return m
	}
	if k+1 == lo && hi.number() < 8 {
		m |= 1 << (hi + 1)
	}
	if k == hi+1 && lo.number() > 0 {
		m |= 1 << (lo - 1)
	}
	return m
}

// canDiscardAfter reports that a call leaves a tile that may be discarded.
func (t *Table) canDiscardAfter(p *Player, a, b, k Tile) bool {
	h := p.Hand
	h[a]--
	h[b]--
	forbid := forbidOf(a, b, k)
	for i := Tile(0); i < numKinds; i++ {
		if h[i] > 0 && !forbid.has(i) {
			return true
		}
	}
	return false
}

// OptsOf is what a seat may answer to the discard now (nothing if it has
// answered).
func (t *Table) OptsOf(seat int) *ClaimOpts { return &t.opts[seat] }

// Waiting reports that the seat has yet to answer to the discard.
func (t *Table) Waiting(seat int) bool { return t.Phase == phaseClaims && !t.answered[seat] }

// AnswerClaim records the answer of a seat. After the last answer it
// resolves the claims.
func (t *Table) AnswerClaim(seat int, c Claim) {
	t.claims[seat] = c
	t.answered[seat] = true
	for _, a := range t.answered {
		if !a {
			return
		}
	}
	t.resolve()
}

func (t *Table) resolve() {
	k := t.Discard0
	rons := 0
	for s := range t.P {
		if t.claims[s].Kind == claimRon {
			rons++
		}
	}
	t.markPassed(claimRon)
	switch rons {
	case 0:
	case 3:
		t.abort("三家和")
		return
	default:
		t.Result = Result{Kind: endWin}
		// The nearest winner after the discarder gets the sticks.
		firstSeat := -1
		for d := 1; d <= 3; d++ {
			s := (t.Discarder + d) % 4
			if t.claims[s].Kind == claimRon {
				if firstSeat < 0 {
					firstSeat = s
				}
				t.addWin(s, t.Discarder, k)
			}
		}
		// A riichi declared with this tile does not count.
		if p := &t.P[t.Discarder]; p.riichiStick {
			p.riichiStick = false
			p.Riichi = false
			t.riichis--
		}
		t.finishWins(firstSeat)
		return
	}
	if t.chankan {
		t.addKan()
		return
	}
	for s := range t.P {
		if c := t.claims[s]; c.Kind == claimKan || c.Kind == claimPon {
			t.call(s, c)
			return
		}
	}
	for s := range t.P {
		if c := t.claims[s]; c.Kind == claimChi {
			t.call(s, c)
			return
		}
	}
	t.noClaim()
}

// markPassed puts in furiten everyone who could have won with the tile and did
// not. ron is the kind of claim that won, -1 if nobody answered.
func (t *Table) markPassed(ron int) {
	for s := range t.P {
		p := &t.P[s]
		if s == t.Discarder || !p.Waits.has(t.Discard0) || ron >= 0 && int(t.claims[s].Kind) == ron {
			continue
		}
		p.furitenPass = true
		if p.Riichi {
			p.furitenRiichi = true
		}
	}
}

// commitStick pays the stick of a riichi declared with the last discard.
func (t *Table) commitStick() {
	for i := range t.P {
		if p := &t.P[i]; p.riichiStick {
			p.riichiStick = false
			p.Score -= 1000
			t.Sticks++
		}
	}
}

// noClaim goes on after a discard nobody took.
func (t *Table) noClaim() {
	t.commitStick()
	switch {
	case t.discards == 4 && t.windFirst != noTile && t.pure:
		t.abort("四風連打")
	case t.riichis == 4:
		t.abort("四家立直")
	case t.totalKans() == 4 && t.kanSeats[0] != 4 && t.kanSeats[1] != 4 && t.kanSeats[2] != 4 && t.kanSeats[3] != 4:
		t.abort("四開槓")
	default:
		if t.discards >= 4 {
			t.pure = false
		}
		t.Turn = (t.Discarder + 1) % 4
		t.Phase = phaseDraw
	}
}

// call gives the discard to seat s.
func (t *Table) call(s int, c Claim) {
	t.commitStick()
	t.clearIppatsu()
	t.pure = false
	k := t.Discard0
	d := &t.P[t.Discarder]
	d.River[d.NRiv-1].Called = true
	p := &t.P[s]
	m := Meld{Tile: k, Called: k, From: int8(t.Discarder)}
	switch c.Kind {
	case claimChi:
		m.Kind, m.Tile = chiMeld, c.Low
		a, b := c.Low, c.Low+1
		switch k {
		case c.Low:
			a, b = c.Low+1, c.Low+2
		case c.Low + 1:
			a, b = c.Low, c.Low+2
		}
		p.Hand[a]--
		p.Hand[b]--
		p.forbid = forbidOf(a, b, k)
		p.Last = "チー"
	case claimPon:
		m.Kind = ponMeld
		p.Hand[k] -= 2
		p.forbid = forbidOf(k, k, k)
		p.Last = "ポン"
	case claimKan:
		m.Kind = openKan
		p.Hand[k] -= 3
		p.Last = "カン"
	}
	p.Melds[p.NMeld] = m
	p.NMeld++
	t.Turn = s
	if c.Kind == claimKan {
		t.doraTo++
		t.kanSeats[s]++
		t.drawKan()
		t.updateWaits(p)
		return
	}
	t.Phase = phaseTurn
}

func (t *Table) addWin(seat, from int, k Tile) {
	p := &t.P[seat]
	c := t.ctx(seat, k, seat == from)
	sc, _ := evaluate(&c)
	r := &t.Result
	w := &r.Wins[r.NWins]
	r.NWins++
	*w = WinInfo{Seat: seat, From: from, Tile: k, Hand: c.hand, Melds: p.Melds, NMeld: p.NMeld, Score: sc}
	if p.Riichi {
		w.NUra = copy(w.Ura[:], t.uraIndicators())
	}

	dealerWins := seat == t.Dealer
	if seat == from {
		fromOther, fromDealer := pay(sc.Base, dealerWins, true)
		for s := range t.P {
			if s == seat {
				continue
			}
			v := fromOther
			if s == t.Dealer {
				v = fromDealer
			}
			v += t.Honba * 100
			r.Delta[s] -= v
			r.Delta[seat] += v
			w.Points += v
		}
		return
	}
	v, _ := pay(sc.Base, dealerWins, false)
	v += t.Honba * 300
	r.Delta[from] -= v
	r.Delta[seat] += v
	w.Points = v
}

// finishWins pays the wins of the hand; the sticks go to seat.
func (t *Table) finishWins(seat int) {
	r := &t.Result
	r.Delta[seat] += t.Sticks * 1000
	t.Sticks = 0
	r.dealerKeeps = false
	for i := 0; i < r.NWins; i++ {
		if r.Wins[i].Seat == t.Dealer {
			r.dealerKeeps = true
		}
	}
	t.applyDelta()
	t.Phase = phaseOver
}

func (t *Table) applyDelta() {
	for s := range t.P {
		t.P[s].Score += t.Result.Delta[s]
	}
}

// exhausted ends a hand with no tiles left: those not ready pay.
func (t *Table) exhausted() {
	r := &t.Result
	*r = Result{Kind: endDraw}
	ready := 0
	for s := range t.P {
		p := &t.P[s]
		r.Tenpai[s] = waitsOf(&p.Hand, p.NMeld) != 0
		r.Hands[s] = p.Hand
		if r.Tenpai[s] {
			ready++
		}
	}
	if ready > 0 && ready < 4 {
		for s := range t.P {
			if r.Tenpai[s] {
				r.Delta[s] = 3000 / ready
			} else {
				r.Delta[s] = -3000 / (4 - ready)
			}
		}
	}
	r.dealerKeeps = r.Tenpai[t.Dealer]
	t.applyDelta()
	t.Phase = phaseOver
}

func (t *Table) abort(why string) {
	t.commitStick()
	t.Result = Result{Kind: endAbort, Reason: why, dealerKeeps: true}
	t.Phase = phaseOver
}

// NextHand starts the next hand, or sets GameOver if the game has ended.
func (t *Table) NextHand() {
	r := &t.Result
	lastHand := t.HandNo == 3 && (t.RoundWind == south || !t.Long)
	top := t.topSeat()
	switch {
	case t.anyBroke():
		t.endGame()
		return
	case lastHand && !r.dealerKeeps:
		t.endGame()
		return
	case lastHand && top == t.Dealer && t.P[top].Score >= topPts && r.Kind != endAbort:
		t.endGame()
		return
	}
	switch {
	case r.dealerKeeps || r.Kind == endDraw:
		t.Honba++
	default:
		t.Honba = 0
	}
	if !r.dealerKeeps {
		t.HandNo++
		t.Dealer = (t.Dealer + 1) % 4
		if t.HandNo == 4 {
			t.HandNo = 0
			t.RoundWind++
		}
	}
	t.deal()
}

func (t *Table) anyBroke() bool {
	for s := range t.P {
		if t.P[s].Score < 0 {
			return true
		}
	}
	return false
}

// topSeat is the seat with the most points; on a tie the seat nearest after
// the first dealer.
func (t *Table) topSeat() int {
	best := t.first
	for d := 1; d < 4; d++ {
		if s := (t.first + d) % 4; t.P[s].Score > t.P[best].Score {
			best = s
		}
	}
	return best
}

func (t *Table) endGame() {
	t.P[t.topSeat()].Score += t.Sticks * 1000
	t.Sticks = 0
	t.GameOver = true
}

// Ranking returns the seats from first to last.
func (t *Table) Ranking() [4]int {
	var r [4]int
	for d := range r {
		r[d] = (t.first + d) % 4
	}
	for i := 1; i < 4; i++ {
		for j := i; j > 0 && t.P[r[j]].Score > t.P[r[j-1]].Score; j-- {
			r[j], r[j-1] = r[j-1], r[j]
		}
	}
	return r
}
