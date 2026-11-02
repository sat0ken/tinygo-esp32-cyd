package mahjong

// yakuID is a scoring element of a winning hand.
type yakuID uint8

const (
	yRiichi yakuID = iota
	yDoubleRiichi
	yIppatsu
	yTsumo
	yTanyao
	yPinfu
	yIipeikou
	yWhite
	yGreen
	yRed
	ySeatWind
	yRoundWind
	yHaitei
	yHoutei
	yRinshan
	yChankan
	yChanta
	yIttsu
	ySanshoku
	ySanshokuKou
	yToitoi
	ySanankou
	ySankantsu
	yShousangen
	yHonroutou
	yJunchan
	yHonitsu
	yRyanpeikou
	yChinitsu
	yChiitoi
	yDora
	yUra
	// Yakuman.
	yKokushi
	ySuuankou
	yDaisangen
	yShousuushii
	yDaisuushii
	yTsuuiisou
	yRyuuiisou
	yChinroutou
	yChuuren
	ySuukantsu
	yTenhou
	yChiihou
)

var yakuNames = [...]string{
	yRiichi: "リーチ", yDoubleRiichi: "ダブルリーチ", yIppatsu: "一発", yTsumo: "門前ツモ",
	yTanyao: "タンヤオ", yPinfu: "ピンフ", yIipeikou: "一盃口", yWhite: "白", yGreen: "發", yRed: "中",
	ySeatWind: "自風", yRoundWind: "場風", yHaitei: "海底", yHoutei: "河底",
	yRinshan: "嶺上開花", yChankan: "槍槓", yChanta: "チャンタ", yIttsu: "一気通貫", ySanshoku: "三色同順",
	ySanshokuKou: "三色同刻", yToitoi: "トイトイ", ySanankou: "三暗刻", ySankantsu: "三槓子",
	yShousangen: "小三元", yHonroutou: "混老頭", yJunchan: "純チャン", yHonitsu: "混一色",
	yRyanpeikou: "二盃口", yChinitsu: "清一色", yChiitoi: "七対子", yDora: "ドラ", yUra: "裏ドラ",
	yKokushi: "国士無双", ySuuankou: "四暗刻", yDaisangen: "大三元", yShousuushii: "小四喜",
	yDaisuushii: "大四喜", yTsuuiisou: "字一色", yRyuuiisou: "緑一色", yChinroutou: "清老頭",
	yChuuren: "九蓮宝燈", ySuukantsu: "四槓子", yTenhou: "天和", yChiihou: "地和",
}

// yakuHan is one line of the scoring of a hand.
type yakuHan struct {
	id  yakuID
	han uint8
}

// Score is the value of a winning hand.
type Score struct {
	yaku    [18]yakuHan
	nYaku   int
	Han     int
	Fu      int
	Yakuman int
	Base    int // the base points; see pay
}

func (s *Score) add(id yakuID, han int) {
	s.yaku[s.nYaku] = yakuHan{id, uint8(han)}
	s.nYaku++
	s.Han += han
}

// Limit is the name of the limit the hand reached, or "".
func (s *Score) Limit() string {
	switch {
	case s.Yakuman > 1:
		return "倍役満" // after the number
	case s.Yakuman == 1:
		return "役満"
	case s.Base >= 8000:
		return "数え役満"
	case s.Base >= 6000:
		return "三倍満"
	case s.Base >= 4000:
		return "倍満"
	case s.Base >= 3000:
		return "跳満"
	case s.Base >= 2000:
		return "満貫"
	}
	return ""
}

// winCtx is everything about a win besides the tiles' shape.
type winCtx struct {
	hand      Counts // the closed tiles, with the winning tile
	melds     []Meld
	win       Tile
	tsumo     bool
	seatWind  Tile
	roundWind Tile

	riichi, doubleRiichi, ippatsu    bool
	haitei, houtei, rinshan, chankan bool
	tenhou, chiihou                  bool

	dora, ura []Tile // the indicators
}

const (
	groupRun = iota
	groupSet
)

// group is a group of a decomposed hand.
type group struct {
	tile Tile
	kind uint8
	open bool // called, or a set completed by a discard
	kan  bool
}

// evaluate returns the best score of a complete hand. ok is false if the
// shape is not complete or the hand has no yaku.
func evaluate(c *winCtx) (best Score, ok bool) {
	e := evaluator{c: c}
	if len(c.melds) == 0 {
		e.kokushi()
		e.chiitoi()
	}
	e.regular()
	if !e.found {
		return Score{}, false
	}
	return e.best, true
}

type evaluator struct {
	c      *winCtx
	best   Score
	found  bool
	groups [4]group
	closed bool
}

func (e *evaluator) consider(s *Score) {
	if s.nYaku == 0 {
		return
	}
	c := e.c
	if s.Yakuman == 0 {
		dora := countDora(c, c.dora)
		if dora > 0 {
			s.add(yDora, dora)
		}
		if c.riichi {
			if ura := countDora(c, c.ura); ura > 0 {
				s.add(yUra, ura)
			}
		}
	}
	s.Base = baseOf(s)
	if !e.found || s.Base > e.best.Base || s.Base == e.best.Base &&
		(s.Han > e.best.Han || s.Han == e.best.Han && s.Fu > e.best.Fu) {
		e.best, e.found = *s, true
	}
}

func baseOf(s *Score) int {
	switch {
	case s.Yakuman > 0:
		return 8000 * s.Yakuman
	case s.Han >= 13:
		return 8000
	case s.Han >= 11:
		return 6000
	case s.Han >= 8:
		return 4000
	case s.Han >= 6:
		return 3000
	case s.Han == 5:
		return 2000
	}
	return min(s.Fu<<(2+s.Han), 2000)
}

func countDora(c *winCtx, inds []Tile) int {
	n := 0
	for _, ind := range inds {
		d := doraOf(ind)
		n += int(c.hand[d])
		for i := range c.melds {
			n += c.melds[i].count(d)
		}
	}
	return n
}

// luck adds what depends on how the hand was won and not on its shape.
func (e *evaluator) luck(s *Score, closed bool) {
	c := e.c
	if c.doubleRiichi {
		s.add(yDoubleRiichi, 2)
	} else if c.riichi {
		s.add(yRiichi, 1)
	}
	if c.ippatsu {
		s.add(yIppatsu, 1)
	}
	if c.tsumo && closed {
		s.add(yTsumo, 1)
	}
	if c.haitei {
		s.add(yHaitei, 1)
	}
	if c.houtei {
		s.add(yHoutei, 1)
	}
	if c.rinshan {
		s.add(yRinshan, 1)
	}
	if c.chankan {
		s.add(yChankan, 1)
	}
}

// heavenly adds the yakuman of a win with the first tiles.
func (e *evaluator) heavenly(s *Score) {
	if e.c.tenhou {
		s.add(yTenhou, 0)
		s.Yakuman++
	}
	if e.c.chiihou {
		s.add(yChiihou, 0)
		s.Yakuman++
	}
}

func (e *evaluator) kokushi() {
	c := e.c
	if shantenOrphans(&c.hand) != -1 {
		return
	}
	var s Score
	e.heavenly(&s)
	s.add(yKokushi, 0)
	s.Yakuman++
	s.Fu = 25
	e.consider(&s)
}

func (e *evaluator) chiitoi() {
	c := e.c
	suits := [4]int{}
	allEdge, noEdge, honorsOnly := true, true, true
	for t := Tile(0); t < numKinds; t++ {
		switch c.hand[t] {
		case 0:
			continue
		case 2:
		default:
			return // seven different pairs only
		}
		suits[t.suit()]++
		allEdge = allEdge && t.isEdge()
		noEdge = noEdge && !t.isEdge()
		honorsOnly = honorsOnly && t.isHonor()
	}
	var s Score
	s.Fu = 25
	if honorsOnly {
		e.heavenly(&s)
		s.add(yTsuuiisou, 0)
		s.Yakuman++
		e.consider(&s)
		return
	}
	e.heavenly(&s)
	if s.Yakuman > 0 {
		e.consider(&s)
		return
	}
	e.luck(&s, true)
	s.add(yChiitoi, 2)
	if noEdge {
		s.add(yTanyao, 1)
	}
	if allEdge {
		s.add(yHonroutou, 2)
	}
	addFlush(&s, suits, true)
	e.consider(&s)
}

// addFlush adds half and full flush. suits counts the tiles of each suit.
func addFlush(s *Score, suits [4]int, closed bool) {
	n := 0
	for _, v := range suits[:3] {
		if v > 0 {
			n++
		}
	}
	if n != 1 {
		return
	}
	if suits[3] > 0 {
		s.add(yHonitsu, ifClosed(closed, 3, 2))
	} else {
		s.add(yChinitsu, ifClosed(closed, 6, 5))
	}
}

func ifClosed(closed bool, yes, no int) int {
	if closed {
		return yes
	}
	return no
}

func (e *evaluator) regular() {
	c := e.c
	e.closed = true
	n := len(c.melds)
	for i := range c.melds {
		m := &c.melds[i]
		g := group{tile: m.Tile, kind: groupSet, open: m.Kind != closedKan, kan: m.isKan()}
		if m.Kind == chiMeld {
			g.kind = groupRun
		}
		if g.open {
			e.closed = false
		}
		e.groups[i] = g
	}
	hand := c.hand
	for p := Tile(0); p < numKinds; p++ {
		if hand[p] < 2 {
			continue
		}
		hand[p] -= 2
		e.split(&hand, 0, n, n, p)
		hand[p] += 2
	}
}

// split splits the closed tiles into groups from tile i on: groups n0 up to n
// are found.
func (e *evaluator) split(c *Counts, i Tile, n0, n int, pair Tile) {
	for i < numKinds && c[i] == 0 {
		i++
	}
	if i == numKinds {
		if n == 4 {
			e.waits(n0, pair)
		}
		return
	}
	if n == 4 {
		return
	}
	if c[i] >= 3 {
		c[i] -= 3
		e.groups[n] = group{tile: i, kind: groupSet}
		e.split(c, i, n0, n+1, pair)
		c[i] += 3
	}
	if !i.isHonor() && i.number() <= 6 && c[i+1] > 0 && c[i+2] > 0 {
		c[i]--
		c[i+1]--
		c[i+2]--
		e.groups[n] = group{tile: i, kind: groupRun}
		e.split(c, i, n0, n+1, pair)
		c[i]++
		c[i+1]++
		c[i+2]++
	}
}

const (
	waitTwoSided = iota
	waitOther    // one tile only: 2 fu
	waitSet      // the set of a double pair wait
)

// waits scores the decomposition once for each way the winning tile fits.
func (e *evaluator) waits(n0 int, pair Tile) {
	c := e.c
	if pair == c.win {
		e.shape(n0, pair, -1, waitOther)
	}
	for i := n0; i < 4; i++ {
		g := e.groups[i]
		switch {
		case g.kind == groupSet && g.tile == c.win:
			e.shape(n0, pair, i, waitSet)
		case g.kind == groupRun && c.win >= g.tile && c.win <= g.tile+2:
			switch {
			case c.win == g.tile+1, c.win == g.tile && g.tile.number() == 6, c.win == g.tile+2 && g.tile.number() == 0:
				e.shape(n0, pair, i, waitOther) // middle or edge
			default:
				e.shape(n0, pair, i, waitTwoSided)
			}
		}
	}
}

// shape scores a decomposition; group winGroup has the winning tile (-1: the
// pair).
func (e *evaluator) shape(n0 int, pair Tile, winGroup, wait int) {
	c := e.c
	gs := e.groups
	if wait == waitSet && !c.tsumo {
		gs[winGroup].open = true
	}
	closed := e.closed

	var s Score
	e.heavenly(&s)
	addYakuman(c, &s, gs[:], pair, closed)
	if s.Yakuman > 0 {
		s.Fu = 20
		e.consider(&s)
		return
	}

	e.luck(&s, closed)

	// The kinds of tiles.
	suits := [4]int{}
	suits[pair.suit()]++
	allSimple, allEdgeNoRun := !pair.isEdge(), pair.isEdge()
	allHaveEdge, noHonor := pair.isEdge(), !pair.isHonor()
	runs, sets, kans, closedSets := 0, 0, 0, 0
	for i := range gs {
		g := &gs[i]
		suits[g.tile.suit()]++
		if g.kind == groupRun {
			runs++
			edge := g.tile.number() == 0 || g.tile.number() == 6
			allSimple = allSimple && !edge
			allEdgeNoRun = false
			allHaveEdge = allHaveEdge && edge
			continue
		}
		sets++
		if g.kan {
			kans++
		}
		if !g.open {
			closedSets++
		}
		allSimple = allSimple && !g.tile.isEdge()
		allEdgeNoRun = allEdgeNoRun && g.tile.isEdge()
		allHaveEdge = allHaveEdge && g.tile.isEdge()
		noHonor = noHonor && !g.tile.isHonor()
	}
	if allSimple {
		s.add(yTanyao, 1)
	}

	// Sets of honors.
	yakuhaiPair := pair >= white || pair == c.seatWind || pair == c.roundWind
	for i := range gs {
		g := &gs[i]
		if g.kind != groupSet || !g.tile.isHonor() {
			continue
		}
		switch g.tile {
		case white:
			s.add(yWhite, 1)
		case green:
			s.add(yGreen, 1)
		case red:
			s.add(yRed, 1)
		}
		if g.tile == c.seatWind {
			s.add(ySeatWind, 1)
		}
		if g.tile == c.roundWind {
			s.add(yRoundWind, 1)
		}
	}

	pinfu := closed && runs == 4 && !yakuhaiPair && wait == waitTwoSided
	if pinfu {
		s.add(yPinfu, 1)
	}

	// Same runs.
	if closed && runs >= 2 {
		same := 0
		var used [4]bool
		for i := 0; i < 4; i++ {
			if used[i] || gs[i].kind != groupRun {
				continue
			}
			for j := i + 1; j < 4; j++ {
				if !used[j] && gs[j].kind == groupRun && gs[j].tile == gs[i].tile {
					used[j] = true
					same++
					break
				}
			}
		}
		if same == 2 {
			s.add(yRyanpeikou, 3)
		} else if same == 1 {
			s.add(yIipeikou, 1)
		}
	}

	// Pure straight and three colours.
	var runAt [3][9]bool
	var setAt [3][9]bool
	for i := range gs {
		g := &gs[i]
		if g.tile.isHonor() {
			continue
		}
		if g.kind == groupRun {
			runAt[g.tile.suit()][g.tile.number()] = true
		} else {
			setAt[g.tile.suit()][g.tile.number()] = true
		}
	}
	for su := 0; su < 3; su++ {
		if runAt[su][0] && runAt[su][3] && runAt[su][6] {
			s.add(yIttsu, ifClosed(closed, 2, 1))
		}
	}
	for n := 0; n < 9; n++ {
		if runAt[0][n] && runAt[1][n] && runAt[2][n] {
			s.add(ySanshoku, ifClosed(closed, 2, 1))
		}
		if setAt[0][n] && setAt[1][n] && setAt[2][n] {
			s.add(ySanshokuKou, 2)
		}
	}

	if sets == 4 {
		s.add(yToitoi, 2)
	}
	if closedSets == 3 {
		s.add(ySanankou, 2)
	}
	if kans == 3 {
		s.add(ySankantsu, 2)
	}

	// Three dragons with two sets and the pair.
	dragons := 0
	for i := range gs {
		if gs[i].kind == groupSet && gs[i].tile >= white {
			dragons++
		}
	}
	if dragons == 2 && pair >= white {
		s.add(yShousangen, 2)
	}

	switch {
	case allEdgeNoRun:
		s.add(yHonroutou, 2)
	case allHaveEdge && noHonor:
		s.add(yJunchan, ifClosed(closed, 3, 2))
	case allHaveEdge:
		s.add(yChanta, ifClosed(closed, 2, 1))
	}
	addFlush(&s, suits, closed)

	s.Fu = fuOf(c, gs[:], pair, wait, pinfu, closed)
	e.consider(&s)
}

// fuOf is the rounded fu of a decomposition.
func fuOf(c *winCtx, gs []group, pair Tile, wait int, pinfu, closed bool) int {
	if pinfu {
		if c.tsumo {
			return 20
		}
		return 30
	}
	fu := 20
	if closed && !c.tsumo {
		fu += 10
	}
	if c.tsumo {
		fu += 2
	}
	if wait == waitOther {
		fu += 2
	}
	if pair >= white {
		fu += 2
	}
	if pair == c.seatWind {
		fu += 2
	}
	if pair == c.roundWind {
		fu += 2
	}
	for i := range gs {
		g := &gs[i]
		if g.kind != groupSet {
			continue
		}
		v := 2
		if !g.open {
			v *= 2
		}
		if g.tile.isEdge() {
			v *= 2
		}
		if g.kan {
			v *= 4
		}
		fu += v
	}
	if fu == 20 {
		return 30 // an open hand with nothing in it
	}
	return (fu + 9) / 10 * 10
}

// addYakuman adds the yakuman of a shape.
func addYakuman(c *winCtx, s *Score, gs []group, pair Tile, closed bool) {
	dragons, winds, closedSets, kans := 0, 0, 0, 0
	allHonor, allTerminal, allGreen := pair.isHonor(), pair.isTerminal(), isGreen(pair)
	for i := range gs {
		g := &gs[i]
		if g.kind == groupRun {
			allHonor, allTerminal = false, false
			n := g.tile.number()
			allGreen = allGreen && g.tile.suit() == 2 && n == 1 // 234 only
			continue
		}
		if g.tile >= white {
			dragons++
		} else if g.tile.isHonor() {
			winds++
		}
		if !g.open {
			closedSets++
		}
		if g.kan {
			kans++
		}
		allHonor = allHonor && g.tile.isHonor()
		allTerminal = allTerminal && g.tile.isTerminal()
		allGreen = allGreen && isGreen(g.tile)
	}
	add := func(id yakuID) {
		s.add(id, 0)
		s.Yakuman++
	}
	if closedSets == 4 {
		add(ySuuankou)
	}
	if dragons == 3 {
		add(yDaisangen)
	}
	if winds == 4 {
		add(yDaisuushii)
	} else if winds == 3 && pair >= east && pair <= north {
		add(yShousuushii)
	}
	if allHonor {
		add(yTsuuiisou)
	}
	if allGreen {
		add(yRyuuiisou)
	}
	if allTerminal {
		add(yChinroutou)
	}
	if kans == 4 {
		add(ySuukantsu)
	}
	if closed && isNineGates(&c.hand) {
		add(yChuuren)
	}
}

func isGreen(t Tile) bool {
	if t == green {
		return true
	}
	if t.suit() != 2 || t.isHonor() {
		return false
	}
	switch t.number() {
	case 1, 2, 3, 5, 7:
		return true
	}
	return false
}

// isNineGates reports 1112345678999 and one more, of one suit.
func isNineGates(c *Counts) bool {
	if c.total() != 14 {
		return false
	}
	for su := 0; su < 3; su++ {
		s := c[su*9 : su*9+9]
		n := 0
		ok := s[0] >= 3 && s[8] >= 3
		for i, v := range s {
			n += int(v)
			if i > 0 && i < 8 && v == 0 {
				ok = false
			}
		}
		if n == 14 {
			return ok
		}
	}
	return false
}

// ceil100 rounds up to the next 100.
func ceil100(v int) int { return (v + 99) / 100 * 100 }

// pay returns what a win costs: for a win by discard the amount the discarder
// pays, for a self-draw what a non-dealer and what the dealer pay (nothing
// for the dealer when the dealer wins). honba is not included.
func pay(base int, dealerWins, tsumo bool) (fromOther, fromDealer int) {
	switch {
	case !tsumo && dealerWins:
		return ceil100(base * 6), 0
	case !tsumo:
		return ceil100(base * 4), 0
	case dealerWins:
		return ceil100(base * 2), 0
	}
	return ceil100(base), ceil100(base * 2)
}
