package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sat0ken/tinygo-cyd/ccproto"
)

// entry is a log line of an answer.
func entry(id, req string, at time.Time, in, out, create, read int) string {
	return `{"parentUuid":"x","type":"assistant","timestamp":"` + at.UTC().Format(time.RFC3339Nano) +
		`","requestId":"` + req + `","message":{"id":"` + id + `","type":"message","role":"assistant","model":"claude-opus-5-5",` +
		`"content":[{"type":"text","text":"the \"usage\" word in a text"}],"usage":{"input_tokens":` + itoa(int64(in)) +
		`,"cache_creation_input_tokens":` + itoa(int64(create)) + `,"cache_read_input_tokens":` + itoa(int64(read)) +
		`,"output_tokens":` + itoa(int64(out)) + `,"service_tier":"standard"}}}` + "\n"
}

func appendFile(t *testing.T, path, s string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(s); err != nil {
		t.Fatal(err)
	}
	f.Close()
}

func TestScanner(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "-home-u-work-a", "s1.jsonl")
	b := filepath.Join(dir, "-home-u-work-b", "s2.jsonl")
	h := func(d time.Duration) time.Time { return now.Add(-d) }

	appendFile(t, a, `{"type":"user","timestamp":"2026-10-30T05:00:00Z","message":{"role":"user","content":"hello"}}`+"\n"+
		// One answer in three lines (one per content block); the usage of
		// the first can be smaller.
		entry("m1", "r1", h(30*time.Minute), 2, 10, 1000, 20000)+
		entry("m1", "r1", h(30*time.Minute), 2, 150, 1000, 20000)+
		entry("m1", "r1", h(30*time.Minute), 2, 150, 1000, 20000)+
		entry("m2", "r2", h(6*time.Hour), 1, 100, 0, 5000)+
		entry("m3", "r3", h(30*time.Hour), 1, 1, 1, 1)+ // too old
		`{"type":"system","content":"not json`+"\n") // a broken line
	// A resumed conversation copies m2 into its own log.
	appendFile(t, b, entry("m2", "r2", h(6*time.Hour), 1, 100, 0, 5000))
	old := filepath.Join(dir, "-home-u-work-c", "old.jsonl")
	appendFile(t, old, entry("m9", "r9", h(2*time.Hour), 1000, 0, 0, 0))
	os.Chtimes(old, h(48*time.Hour), h(48*time.Hour)) // not written for two days: not read

	s := newScanner(dir)
	if err := s.update(now); err != nil {
		t.Fatal(err)
	}
	m1, m2 := int64(2+150+1000+20000), int64(1+100+5000)
	if got := s.sum(now.Add(-5*time.Hour), now); got != m1 {
		t.Errorf("5 hours: %d, want %d", got, m1)
	}
	if got := s.sum(now.Add(-24*time.Hour), now); got != m1+m2 {
		t.Errorf("24 hours: %d, want %d", got, m1+m2)
	}

	// Lines added later are read from where the last update stopped; a line
	// still being written (no newline yet) waits for the next update.
	half := entry("m4", "r4", h(time.Minute), 0, 7, 0, 0)
	appendFile(t, a, half[:40])
	s.update(now)
	if got := s.sum(now.Add(-5*time.Hour), now); got != m1 {
		t.Errorf("with half a line: %d, want %d", got, m1)
	}
	appendFile(t, a, half[40:])
	s.update(now)
	if got := s.sum(now.Add(-5*time.Hour), now); got != m1+7 {
		t.Errorf("after the line: %d, want %d", got, m1+7)
	}

	// 14:10 now: m4 (14:09) is in the current hour, m1 (13:40) in the one
	// before, m2 (08:10) six hours earlier.
	hours := s.hours(now)
	want := [ccproto.Hours]int64{}
	want[ccproto.Hours-1] = 7
	want[ccproto.Hours-2] = m1
	want[ccproto.Hours-7] = m2
	if hours != want {
		t.Errorf("hours\n got %v\nwant %v", hours, want)
	}

	// Answers older than 25 hours are forgotten.
	s.update(now.Add(20 * time.Hour))
	if _, ok := s.msgs["m2/r2"]; ok {
		t.Error("m2 kept")
	}
	if _, ok := s.msgs["m1/r1"]; !ok {
		t.Error("m1 forgotten")
	}
}

func TestCurrent(t *testing.T) {
	dir := t.TempDir()
	logs := filepath.Join(dir, "projects")
	state := filepath.Join(dir, "status.json")
	appendFile(t, filepath.Join(logs, "p", "s.jsonl"),
		entry("m1", "r1", now.Add(-30*time.Minute), 0, 1000, 0, 0)+
			entry("m2", "r2", now.Add(-4*time.Hour), 0, 300, 0, 0)+
			entry("m3", "r3", now.Add(-10*time.Hour), 0, 50, 0, 0))
	// The 5-hour window ends in 2 hours: it started 3 hours ago, after m2.
	// Hours: m1 (13:40) in the one before the current (14:00), m2 (10:10)
	// 4 hours before, m3 (04:10) 10 hours before.
	limits := `"five_hour":{"used_percentage":12,"resets_at":` + itoa(now.Add(2*time.Hour).Unix()) + `}`
	runStatusLine(t, state, report("s", "20", limits), now.Add(-time.Minute))

	u := current(newScanner(logs), state, now)
	if u.Tokens5h != 1000 || u.Tokens24h != 1350 || u.HourMax != 1000 {
		t.Errorf("tokens %d %d %d", u.Tokens5h, u.Tokens24h, u.HourMax)
	}
	if u.FiveHour != (ccproto.Limit{Percent: 12, ResetsIn: 7200}) || u.StatusAge != 60 || u.Context != 20 {
		t.Errorf("status %+v", u)
	}
	lines := string(ccproto.AppendStatus(nil, &u)) + string(ccproto.AppendTokens(nil, &u))
	if want := "U t=1793369400 h=12,7200 c=20 a=60 p=tinygo-cyd m=Opus 5.5\nT f=1000 d=1350 x=1000 b=0000000000000200000b00z0\n"; lines != want {
		t.Errorf("lines\n%s\nwant\n%s", lines, want)
	}

	// Without a status line report and logs: the clock only.
	u = current(newScanner(filepath.Join(dir, "none")), filepath.Join(dir, "none.json"), now)
	if !strings.HasPrefix(string(ccproto.AppendStatus(nil, &u)), "U t=1793369400\n") || u.Tokens5h != -1 {
		t.Errorf("%+v", u)
	}
}
