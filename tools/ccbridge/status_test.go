package main

import (
	"bytes"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sat0ken/tinygo-cyd/ccproto"
)

var now = time.Date(2026, 10, 30, 14, 10, 0, 0, time.FixedZone("JST", 9*3600))

// report is a status line JSON as Claude Code sends it (trimmed).
func report(session string, ctx string, limits string) string {
	s := `{"session_id":"` + session + `","cwd":"/home/u/work/tinygo-cyd",
	"model":{"id":"claude-opus-5-5","display_name":"Opus 5.5"},
	"workspace":{"current_dir":"/home/u/work/tinygo-cyd/cmd","project_dir":"/home/u/work/tinygo-cyd"},
	"version":"2.1.310","context_window":{"used_percentage":` + ctx + `}`
	if limits != "" {
		s += `,"rate_limits":{` + limits + `}`
	}
	return s + "}"
}

func runStatusLine(t *testing.T, path, in string, at time.Time) string {
	t.Helper()
	var out bytes.Buffer
	if err := statusLine(strings.NewReader(in), &out, path, at); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestStatusLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ccmeter", "status.json")
	five := now.Add(3*time.Hour + 20*time.Minute).Unix()
	week := now.Add(53 * time.Hour).Unix()
	limits := `"five_hour":{"used_percentage":41.6,"resets_at":` + itoa(five) +
		`},"seven_day":{"used_percentage":83,"resets_at":` + itoa(week) + `}`
	if got := runStatusLine(t, path, report("s1", "63.4", limits), now); got != "Opus 5.5 | ctx 63% | 5h 42% | 7d 83%\n" {
		t.Errorf("status line %q", got)
	}
	st, err := loadState(path)
	if err != nil {
		t.Fatal(err)
	}
	u := st.usage(now.Add(10 * time.Second))
	want := ccproto.Unknown()
	want.Clock = now.Unix() + 9*3600 + 10
	want.FiveHour = ccproto.Limit{Percent: 42, ResetsIn: 3*3600 + 20*60 - 10}
	want.Week = ccproto.Limit{Percent: 83, ResetsIn: 53*3600 - 10}
	want.Context = 63
	want.StatusAge = 10
	want.Project = "tinygo-cyd"
	want.Model = "Opus 5.5"
	if u != want {
		t.Errorf("usage\n got %+v\nwant %+v", u, want)
	}

	// A new session has no limits before its first answer: the last ones
	// are kept, its context is taken.
	later := now.Add(time.Minute)
	if got := runStatusLine(t, path, report("s2", "null", ""), later); got != "Opus 5.5 | 5h 42% | 7d 83%\n" {
		t.Errorf("status line %q", got)
	}
	st, _ = loadState(path)
	if st.Session != "s2" || st.Context != nil || st.FiveHour == nil || st.SevenDay == nil {
		t.Errorf("state %+v", st)
	}

	// After the 5-hour window resets it is dropped; the meter shows 0%.
	after := time.Unix(five+60, 0)
	runStatusLine(t, path, report("s2", "5", `"seven_day":{"used_percentage":84,"resets_at":`+itoa(week)+`}`), after)
	st, _ = loadState(path)
	if st.FiveHour != nil || st.SevenDay.UsedPercentage != 84 {
		t.Errorf("state %+v", st)
	}
	// A state with a window that has reset since it was written.
	st.FiveHour = &window{UsedPercentage: 90, ResetsAt: five}
	if l := st.usage(after).FiveHour; l != (ccproto.Limit{Percent: 0, ResetsIn: -1}) {
		t.Errorf("reset window %+v", l)
	}
}

func TestStatusLineBadInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	var out bytes.Buffer
	if err := statusLine(strings.NewReader("not json"), &out, path, now); err == nil {
		t.Error("no error")
	}
	// No status line report yet: the clock only.
	st, _ := loadState(path)
	u := st.usage(now)
	if u.StatusAge != -1 || u.FiveHour.Percent != -1 || u.Clock != now.Unix()+9*3600 {
		t.Errorf("%+v", u)
	}
}

func itoa(v int64) string { return strconv.FormatInt(v, 10) }
