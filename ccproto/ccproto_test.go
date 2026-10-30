package ccproto

import (
	"strings"
	"testing"
)

func sample() Usage {
	u := Unknown()
	u.Clock = 1793287800
	u.FiveHour = Limit{42, 5400}
	u.Week = Limit{17, 280000}
	u.Context = 63
	u.StatusAge = 12
	u.Project = "tinygo-cyd"
	u.Model = "Opus 5.5"
	u.Tokens5h = 34567890
	u.Tokens24h = 123456789
	u.HourMax = 9876543
	for i := range u.HourLevel {
		u.HourLevel[i] = uint8(i * 3 / 2)
	}
	return u
}

func TestLines(t *testing.T) {
	u := sample()
	if got, want := string(AppendStatus(nil, &u)), "U t=1793287800 h=42,5400 w=17,280000 c=63 a=12 p=tinygo-cyd m=Opus 5.5\n"; got != want {
		t.Errorf("status line\n got %q\nwant %q", got, want)
	}
	if got, want := string(AppendTokens(nil, &u)), "T f=34567890 d=123456789 x=9876543 b=0134679acdfgijlmoprsuvxy\n"; got != want {
		t.Errorf("tokens line\n got %q\nwant %q", got, want)
	}
}

func TestRoundTrip(t *testing.T) {
	for _, u := range []Usage{sample(), Unknown(), func() Usage {
		u := sample()
		u.FiveHour = Limit{0, -1} // the window has reset
		u.Week = Limit{-1, -1}
		u.Model = ""
		u.Project = ""
		return u
	}()} {
		got := Unknown()
		got.Model, got.Project = "old", "old" // replaced, or cleared when missing
		for _, line := range [][]byte{AppendStatus(nil, &u), AppendTokens(nil, &u)} {
			if len(line) > MaxLine {
				t.Errorf("%q: %d bytes", line, len(line))
			}
			if kind := got.Parse(line[:len(line)-1]); kind != int(line[0]) {
				t.Errorf("%q: kind %c", line, kind)
			}
		}
		if got != u {
			t.Errorf("\n got %+v\nwant %+v", got, u)
		}
	}
}

func TestLongestLines(t *testing.T) {
	u := sample()
	u.Clock = 99999999999
	u.FiveHour = Limit{100, 18000}
	u.Week = Limit{100, 604800}
	u.Context = 100
	u.StatusAge = 9999999
	u.Project = strings.Repeat("p", 40)
	u.Model = strings.Repeat("m", 40)
	u.Tokens5h, u.Tokens24h, u.HourMax = 999999999999, 999999999999, 999999999999
	for _, line := range [][]byte{AppendStatus(nil, &u), AppendTokens(nil, &u)} {
		if len(line) > MaxLine {
			t.Errorf("%q: %d bytes, more than %d", line, len(line), MaxLine)
		}
	}
}

func TestNames(t *testing.T) {
	u := sample()
	u.Project = "my project"
	u.Model = "Opus\t5.5 é"
	line := AppendStatus(nil, &u)
	if !strings.Contains(string(line), " p=my_project m=Opus5.5 \n") {
		t.Errorf("%q", line)
	}
}

func TestParseBadInput(t *testing.T) {
	u := sample()
	for _, s := range []string{"", "X t=1", "Ut=1", "hello", "boot: ets Jun  8 2016"} {
		if kind := u.Parse([]byte(s)); kind != None {
			t.Errorf("%q: kind %c", s, kind)
		}
	}
	if u != sample() {
		t.Error("changed by lines that are not ours")
	}
	// Bad values are unknown, the others still count; unknown keys are skipped.
	u.Parse([]byte("U t=12x h=,5 w=17 c=-3 zz=1 a=5 q"))
	want := Unknown()
	want.Week = Limit{17, -1}
	want.StatusAge = 5
	if u.Clock != 0 || u.FiveHour != want.FiveHour || u.Week != want.Week || u.Context != -1 || u.StatusAge != 5 {
		t.Errorf("%+v", u)
	}
	u.Parse([]byte("T f=1 b=0123")) // b has the wrong length
	if u.Tokens5h != 1 || u.HourMax != -1 || u.HourLevel != [Hours]uint8{} {
		t.Errorf("%+v", u)
	}
}

func TestLevel(t *testing.T) {
	for _, c := range []struct {
		v, top int64
		want   uint8
	}{
		{0, 100, 0}, {1, 1000000, 1}, {50, 100, 18}, {100, 100, 35}, {200, 100, 35}, {5, 0, 0},
	} {
		if got := Level(c.v, c.top); got != c.want {
			t.Errorf("Level(%d, %d) = %d, want %d", c.v, c.top, got, c.want)
		}
	}
}

func TestParseNoAllocs(t *testing.T) {
	u := Unknown()
	status := AppendStatus(nil, func() *Usage { s := sample(); return &s }())
	tokens := AppendTokens(nil, func() *Usage { s := sample(); return &s }())
	u.Parse(status[:len(status)-1])
	if n := testing.AllocsPerRun(20, func() {
		u.Parse(status[:len(status)-1])
		u.Parse(tokens[:len(tokens)-1])
	}); n != 0 {
		t.Errorf("%v allocations per parse", n)
	}
}
