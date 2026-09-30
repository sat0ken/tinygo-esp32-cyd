package rgblcd

import "testing"

var boardTiming = Timing{
	PclkHz:          9_000_000,
	HSyncPulseWidth: 4, HSyncBackPorch: 43, HSyncFrontPorch: 8,
	VSyncPulseWidth: 4, VSyncBackPorch: 12, VSyncFrontPorch: 8,
	HSyncIdleLow: true, VSyncIdleLow: true, PclkActiveNeg: true,
}

func TestCalcClock9MHz(t *testing.T) {
	// 160MHz / 9MHz / 256 + 1 = 1 -> MO forced to 2 because pclk_active_neg.
	// 160MHz / 18MHz = 8 remainder 16MHz; gcd(18M,16M) = 2M -> A=9, B=8.
	// lcd_clk = 160M*9 / (8*9+8) = 18MHz, pclk = 9MHz.
	d, ok := CalcClock(160_000_000, 9_000_000, false)
	if !ok {
		t.Fatal("not ok")
	}
	want := ClockDiv{N: 8, A: 9, B: 8, MO: 2, PclkHz: 9_000_000}
	if d != want {
		t.Fatalf("got %+v, want %+v", d, want)
	}
}

func TestCalcClockOther(t *testing.T) {
	tests := []struct {
		src, want uint32
		allowEq   bool
		exp       ClockDiv
	}{
		// 160/16 = 10 exactly: denominator stays 2, numerator 0.
		{160_000_000, 8_000_000, false, ClockDiv{N: 10, A: 2, B: 0, MO: 2, PclkHz: 8_000_000}},
		// 160/14 = 11 remainder 6M; gcd(14M,6M)=2M -> 7, 3.
		{160_000_000, 7_000_000, false, ClockDiv{N: 11, A: 7, B: 3, MO: 2, PclkHz: 7_000_000}},
		// Equal sysclk allowed: MO stays 1.
		{160_000_000, 10_000_000, true, ClockDiv{N: 16, A: 2, B: 0, MO: 1, PclkHz: 10_000_000}},
		// Low clock needs MO > 1 regardless: 160M/100k/256+1 = 7, 160M/700k = 228 + 4/7.
		{160_000_000, 100_000, true, ClockDiv{N: 228, A: 7, B: 4, MO: 7, PclkHz: 100_000}},
	}
	for _, tc := range tests {
		d, ok := CalcClock(tc.src, tc.want, tc.allowEq)
		if !ok {
			t.Errorf("%d: not ok", tc.want)
			continue
		}
		if d.N != tc.exp.N || d.MO != tc.exp.MO || d.A*tc.exp.B != d.B*tc.exp.A {
			t.Errorf("%d: got %+v, want %+v", tc.want, d, tc.exp)
		}
		if diff := int(d.PclkHz) - int(tc.want); diff < -1000 || diff > 1000 {
			t.Errorf("%d: pclk %d", tc.want, d.PclkHz)
		}
	}
}

func TestCalcTimingBoard(t *testing.T) {
	r, err := CalcTiming(480, 272, boardTiming)
	if err != nil {
		t.Fatal(err)
	}
	want := TimingRegs{
		HSyncWidth: 3,   // 4-1
		HBFront:    46,  // 43+4-1
		HAWidth:    479, // 480-1
		HTWidth:    534, // 4+43+480+8-1
		VSyncWidth: 3,   // 4-1
		VBFront:    15,  // 12+4-1
		VAHeight:   271, // 272-1
		VTHeight:   295, // 4+12+272+8-1
	}
	if r != want {
		t.Fatalf("got %+v, want %+v", r, want)
	}
	hz := RefreshHz(9_000_000, 480, 272, boardTiming)
	if hz < 56.8 || hz > 56.9 { // 9e6 / (535*296) = 56.83
		t.Fatalf("refresh %f", hz)
	}
}

func TestCalcTimingOverflow(t *testing.T) {
	if _, err := CalcTiming(480, 1100, boardTiming); err != ErrBadTiming {
		t.Fatalf("want ErrBadTiming, got %v", err)
	}
}

func TestDescLayout(t *testing.T) {
	lines, n, err := DescLayout(480*2, 272)
	if err != nil || lines != 4 || n != 68 {
		t.Fatalf("got %d lines x %d, %v", lines, n, err)
	}
	if lines*480*2 != 3840 || n*lines*480*2 != 480*272*2 {
		t.Fatal("total bytes mismatch")
	}
	if _, _, err := DescLayout(4096, 10); err != ErrLineTooLong {
		t.Fatal(err)
	}
}

func TestDescWord0(t *testing.T) {
	// size=3840 (0xF00), length=3840<<12, owner bit 31.
	if w := DescWord0(3840, false); w != 0x80F00F00 {
		t.Fatalf("got %#x", w)
	}
	if w := DescWord0(3840, true); w != 0xC0F00F00 {
		t.Fatalf("got %#x", w)
	}
}
