package mahjong

import "testing"

func TestShanten(t *testing.T) {
	for _, c := range []struct {
		hand  string
		melds int
		want  int
	}{
		{"123m456p789s11122z", 0, -1},
		{"123m456p789s1112z", 0, 0},
		{"123m456p789s1122z", 0, 0}, // two pairs: ready for either
		{"1m", 4, 0},
		{"11m", 4, -1},
		{"19m19p19s1234567z", 0, 0},
		{"19m19p19s12345677z", 0, -1},
		{"1122m3344p5566s7z", 0, 0},
		{"13579m13579p135s", 0, 4}, // four two-tile groups and no pair
		{"1112345678999m", 0, 0},
	} {
		h := parseHand(c.hand)
		if got := shanten(&h, c.melds); got != c.want {
			t.Errorf("%s: shanten %d, want %d", c.hand, got, c.want)
		}
	}
	h := parseHand("1112345678999m")
	if w := waitsOf(&h, 0); w != 0x1FF {
		t.Errorf("nine gates waits %x", uint64(w))
	}
}

// score evaluates a hand; a win east round, south seat unless ctx says otherwise.
func score(t *testing.T, c winCtx, hand string) Score {
	t.Helper()
	c.hand = parseHand(hand)
	if c.seatWind == 0 {
		c.seatWind = south
	}
	if c.roundWind == 0 {
		c.roundWind = east
	}
	s, ok := evaluate(&c)
	if !ok {
		t.Fatalf("%s: no win", hand)
	}
	return s
}

func hasYaku(s *Score, id yakuID) bool {
	for i := 0; i < s.nYaku; i++ {
		if s.yaku[i].id == id {
			return true
		}
	}
	return false
}

func TestScores(t *testing.T) {
	for _, c := range []struct {
		name    string
		ctx     winCtx
		hand    string
		han, fu int
		base    int
		yaku    []yakuID
	}{
		{"pinfu tsumo", winCtx{win: man + 1, tsumo: true}, "234m567m234p56788s", 0, 0, 0, nil},
		{"pinfu ron", winCtx{win: pin, riichi: true}, "234m678m123p567s22m", 2, 30, 480, []yakuID{yRiichi, yPinfu}},
		{"edge wait is no pinfu", winCtx{win: pin + 2, riichi: true}, "234m678m123p567s22m", 1, 40, 320, []yakuID{yRiichi}},
		{"tanyao open sets", winCtx{win: pin + 3, melds: []Meld{{Kind: ponMeld, Tile: man + 1, From: 1}}},
			"456m234p678s44p", 1, 30, 240, []yakuID{yTanyao}},
		{"chiitoi", winCtx{win: man, tsumo: true}, "1133m55p77s22z66z88s", 3, 25, 800, []yakuID{yTsumo, yChiitoi}},
		{"chinitsu", winCtx{win: man, tsumo: true}, "123456789m12399m", 11, 0, 6000, []yakuID{yTsumo, yChinitsu, yIttsu, yPinfu, yIipeikou}},
		{"kokushi", winCtx{win: east}, "19m19p19s12345677z", 0, 0, 8000, []yakuID{yKokushi}},
		{"daisangen", winCtx{win: man + 2}, "555666777z123m11p", 0, 0, 8000, []yakuID{yDaisangen}},
		{"suuankou tsumo", winCtx{win: man, tsumo: true}, "111m222p333s444m55z", 0, 0, 8000, []yakuID{ySuuankou}},
		{"chuuren", winCtx{win: man + 4}, "11123455678999m", 0, 0, 8000, []yakuID{yChuuren}},
		{"ryuuiisou", winCtx{win: sou + 1, tsumo: true}, "234s234s666s888s66z", 0, 0, 8000, []yakuID{yRyuuiisou}},
	} {
		s := score(t, c.ctx, c.hand)
		if c.name == "pinfu tsumo" {
			if s.Fu != 20 || !hasYaku(&s, yPinfu) || !hasYaku(&s, yTsumo) {
				t.Errorf("%s: %+v", c.name, s)
			}
			continue
		}
		if c.han != 0 && s.Han != c.han {
			t.Errorf("%s: %d han, want %d (%+v)", c.name, s.Han, c.han, s.yaku[:s.nYaku])
		}
		if c.fu != 0 && s.Fu != c.fu {
			t.Errorf("%s: %d fu, want %d", c.name, s.Fu, c.fu)
		}
		if c.base != 0 && s.Base != c.base {
			t.Errorf("%s: base %d, want %d", c.name, s.Base, c.base)
		}
		for _, y := range c.yaku {
			if !hasYaku(&s, y) {
				t.Errorf("%s: no %s in %+v", c.name, yakuNames[y], s.yaku[:s.nYaku])
			}
		}
	}
}

func TestNoYaku(t *testing.T) {
	c := winCtx{win: pin + 3, seatWind: south, roundWind: east, hand: parseHand("123m456p789s11z"),
		melds: []Meld{{Kind: chiMeld, Tile: man + 3, From: 1}}}
	if s, ok := evaluate(&c); ok {
		t.Errorf("open hand without a yaku won: %+v", s)
	}
}

// TestWaitFu checks that a ron on a double pair wait makes the set open.
func TestWaitFu(t *testing.T) {
	s := score(t, winCtx{win: man, riichi: true}, "111m222p333s444m77p")
	if hasYaku(&s, ySuuankou) {
		t.Error("suuankou by ron on a double pair wait")
	}
	if !hasYaku(&s, ySanankou) || !hasYaku(&s, yToitoi) {
		t.Errorf("no sanankou and toitoi: %+v", s.yaku[:s.nYaku])
	}
	s = score(t, winCtx{win: pin + 6, riichi: true}, "111m222p333s444m77p") // the pair: all four stay closed
	if !hasYaku(&s, ySuuankou) {
		t.Errorf("no suuankou on a pair wait: %+v", s.yaku[:s.nYaku])
	}
}

// TestFourWinds plays the same wind four times.
func TestFourWinds(t *testing.T) {
	tb := NewTable(1, false)
	for i := 0; i < 4; i++ {
		tb.Draw()
		p := &tb.P[tb.Turn]
		p.Hand[p.Drawn]--
		p.Hand[east]++ // tiles do not matter here
		p.Drawn = east
		tb.DiscardTile(east, false)
		for s := range tb.P {
			if tb.Waiting(s) {
				tb.AnswerClaim(s, Claim{})
			}
		}
	}
	if tb.Phase != phaseOver || tb.Result.Reason != "四風連打" {
		t.Fatalf("phase %d %q", tb.Phase, tb.Result.Reason)
	}
}

func TestPay(t *testing.T) {
	for _, c := range []struct {
		han, fu    int
		dealer     bool
		tsumo      bool
		other, dlr int
	}{
		{1, 30, false, false, 1000, 0},
		{2, 30, false, false, 2000, 0},
		{3, 30, false, false, 3900, 0},
		{3, 30, true, false, 5800, 0},
		{4, 30, false, false, 7700, 0},
		{5, 30, false, false, 8000, 0},
		{1, 30, false, true, 300, 500},
		{2, 20, false, true, 400, 700},
		{3, 30, true, true, 2000, 0},
		{4, 40, true, true, 4000, 0},
	} {
		s := Score{Han: c.han, Fu: c.fu}
		other, dlr := pay(baseOf(&s), c.dealer, c.tsumo)
		if other != c.other || dlr != c.dlr {
			t.Errorf("%d han %d fu dealer=%v tsumo=%v: %d/%d, want %d/%d", c.han, c.fu, c.dealer, c.tsumo, other, dlr, c.other, c.dlr)
		}
	}
}

func TestForbid(t *testing.T) {
	for _, c := range []struct {
		a, b, k Tile
		want    []Tile
	}{
		{man + 1, man + 2, man, []Tile{man, man + 3}},         // 1 with 23: 4 is out too
		{man + 5, man + 6, man + 7, []Tile{man + 7, man + 4}}, // 8 with 67: 5 is out too
		{man, man + 2, man + 1, []Tile{man + 1}},              // 2 with 13
		{man + 6, man + 7, man + 8, []Tile{man + 8, man + 5}},
		{man, man + 1, man + 2, []Tile{man + 2}}, // 3 with 12: no tile below
		{east, east, east, []Tile{east}},
	} {
		var want Mask
		for _, k := range c.want {
			want |= 1 << k
		}
		if got := forbidOf(c.a, c.b, c.k); got != want {
			t.Errorf("%d %d %d: %x, want %x", c.a, c.b, c.k, uint64(got), uint64(want))
		}
	}
}

// checkTiles checks that all the tiles of the game are somewhere.
func checkTiles(t *testing.T, tb *Table) {
	t.Helper()
	var c Counts
	for i := range tb.P {
		p := &tb.P[i]
		for k, v := range p.Hand {
			c[k] += v
		}
		for j := 0; j < p.NRiv; j++ {
			if !p.River[j].Called {
				c[p.River[j].Tile]++
			}
		}
		for j := 0; j < p.NMeld; j++ {
			for k := Tile(0); k < numKinds; k++ {
				c[k] += uint8(p.Melds[j].count(k))
			}
		}
	}
	for i := tb.next; i < liveSize; i++ {
		c[tb.wall[i]]++
	}
	for i := liveSize + tb.kans; i < wallSize; i++ {
		c[tb.wall[i]]++
	}
	for k, v := range c {
		if v != 4 {
			t.Fatalf("%d of tile %d at phase %d", v, k, tb.Phase)
		}
	}
}

// TestCPUGames plays whole games with four CPUs and checks what must always
// hold.
func TestCPUGames(t *testing.T) {
	var wins, draws, aborts, hands, tsumos, rons, riichis, calls, kans int
	yaku := map[yakuID]int{}
	for seed := uint32(1); seed <= 60; seed++ {
		tb := NewTable(seed*2654435761, seed%2 == 0)
		for !tb.GameOver {
			for steps := 0; tb.Phase != phaseOver; steps++ {
				if steps > 2000 {
					t.Fatalf("seed %d: the hand does not end", seed)
				}
				switch tb.Phase {
				case phaseDraw:
					tb.Draw()
				case phaseTurn:
					tb.CPUTurn()
				case phaseClaims:
					for s := range tb.P {
						if tb.Waiting(s) {
							tb.CPUAnswer(s)
							break
						}
					}
				}
				checkTiles(t, tb)
				total := tb.Sticks * 1000
				for s := range tb.P {
					total += tb.P[s].Score
				}
				if total != 4*startPts {
					t.Fatalf("seed %d: %d points in the game", seed, total)
				}
			}
			hands++
			r := &tb.Result
			switch r.Kind {
			case endWin:
				wins++
				for i := 0; i < r.NWins; i++ {
					w := &r.Wins[i]
					if w.Seat == w.From {
						tsumos++
					} else {
						rons++
					}
					if w.Score.nYaku == 0 || w.Points <= 0 {
						t.Fatalf("seed %d: a win without a yaku: %+v", seed, w)
					}
					for j := 0; j < w.Score.nYaku; j++ {
						yaku[w.Score.yaku[j].id]++
					}
				}
			case endDraw:
				draws++
			default:
				aborts++
			}
			for s := range tb.P {
				if tb.P[s].Riichi {
					riichis++
				}
				calls += tb.P[s].NMeld
				for j := 0; j < tb.P[s].NMeld; j++ {
					if tb.P[s].Melds[j].isKan() {
						kans++
					}
				}
			}
			tb.NextHand()
		}
		rk := tb.Ranking()
		for i := 1; i < 4; i++ {
			if tb.P[rk[i-1]].Score < tb.P[rk[i]].Score {
				t.Fatalf("ranking %v", rk)
			}
		}
	}
	t.Logf("%d hands: %d wins (%d tsumo, %d ron), %d draws, %d aborted; %d riichi, %d melds, %d kans", hands, wins, tsumos, rons, draws, aborts, riichis, calls, kans)
	for id, n := range yaku {
		t.Logf("%-16s %d", yakuNames[id], n)
	}
	if wins == 0 || draws == 0 || riichis == 0 || calls == 0 {
		t.Error("something never happens")
	}
	// A plausible share of wins: real games end in a win about 4 times out of 5.
	if wins*100/hands < 50 {
		t.Errorf("only %d of %d hands were won", wins, hands)
	}
}

// screenTexts is what the screen writes besides the names of the yaku.
var screenTexts = []string{
	"麻雀", "リーチ麻雀  自分とコンピュータ3人", "東風戦", "半荘戦", "東場のみ", "東場と南場",
	"ツモ", "リーチ", "カン", "九種", "ロン", "ポン", "チー", "パス", "戻る", "終了", "終了?",
	"テンパイ", "フリテン", "東", "南", "局", "本場", "供託", "残り", "から", "ドラ", "裏", "翻", "符", "点",
	"流局", "ノーテン", "途中流局", "九種九牌", "三家和", "四風連打", "四家立直", "四開槓", "タップで次へ", "終局", "位",
	"役満", "倍役満", "数え役満", "三倍満", "倍満", "跳満", "満貫",
	seatNames[0], seatNames[1], seatNames[2], seatNames[3],
}

// TestFont checks that the font has every character the game writes.
func TestFont(t *testing.T) {
	texts := append([]string{}, yakuNames[:]...)
	texts = append(texts, screenTexts...)
	for _, s := range texts {
		for _, r := range s {
			if r >= 0x80 && runeIndex(r) < 0 {
				t.Errorf("%q: no %c", s, r)
			}
		}
	}
	if len(fontSmall) != len(fontRunes)*8 || len(fontBig) != len(fontRunes)*32 || len(fontBigASCII) != len(bigASCII)*16 {
		t.Errorf("%d and %d rows for %d characters", len(fontSmall), len(fontBig), len(fontRunes))
	}
}
