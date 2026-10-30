package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sat0ken/tinygo-cyd/ccproto"
)

// statusInput is the part of the status line JSON that ccbridge uses (see
// "statusLine" in the Claude Code settings documentation).
type statusInput struct {
	SessionID string `json:"session_id"`
	Cwd       string `json:"cwd"`
	Model     struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
	} `json:"model"`
	Workspace struct {
		CurrentDir string `json:"current_dir"`
		ProjectDir string `json:"project_dir"`
	} `json:"workspace"`
	ContextWindow struct {
		UsedPercentage *float64 `json:"used_percentage"`
	} `json:"context_window"`
	// Only for claude.ai subscribers, after the first answer; a window is
	// left out once it has reset.
	RateLimits *struct {
		FiveHour *window `json:"five_hour"`
		SevenDay *window `json:"seven_day"`
	} `json:"rate_limits"`
}

type window struct {
	UsedPercentage float64 `json:"used_percentage"`
	ResetsAt       int64   `json:"resets_at"` // unix seconds
}

// state is the last status line report, from any session: the limits are
// the same for every session, the context and the model are the latest
// session's.
type state struct {
	Updated  int64    `json:"updated"` // unix seconds
	Session  string   `json:"session,omitempty"`
	Project  string   `json:"project,omitempty"`
	Model    string   `json:"model,omitempty"`
	Context  *float64 `json:"context,omitempty"`
	FiveHour *window  `json:"five_hour,omitempty"`
	SevenDay *window  `json:"seven_day,omitempty"`
}

// statusLine reads the status line JSON from r, merges it into the state
// file and writes the status line to w.
func statusLine(r io.Reader, w io.Writer, path string, now time.Time) error {
	data, err := io.ReadAll(io.LimitReader(r, 4<<20))
	if err != nil {
		return err
	}
	var in statusInput
	if err := json.Unmarshal(data, &in); err != nil {
		return fmt.Errorf("status line input: %v", err)
	}
	st, err := loadState(path)
	if err != nil && !os.IsNotExist(err) {
		st = state{} // a broken file: start again
	}
	st.merge(&in, now)
	if err := saveState(path, &st); err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, st.text(now))
	return err
}

// merge takes a status line report, keeping the limits the report does not
// have until they reset (a new session has none before its first answer).
func (st *state) merge(in *statusInput, now time.Time) {
	st.Updated = now.Unix()
	st.Session = in.SessionID
	st.Model = in.Model.DisplayName
	if st.Model == "" {
		st.Model = in.Model.ID
	}
	dir := in.Workspace.ProjectDir
	if dir == "" {
		dir = in.Workspace.CurrentDir
	}
	if dir == "" {
		dir = in.Cwd
	}
	st.Project = ""
	if dir != "" {
		st.Project = filepath.Base(dir)
	}
	st.Context = in.ContextWindow.UsedPercentage
	var five, week *window
	if in.RateLimits != nil {
		five, week = in.RateLimits.FiveHour, in.RateLimits.SevenDay
	}
	st.FiveHour = keep(five, st.FiveHour, now)
	st.SevenDay = keep(week, st.SevenDay, now)
}

func keep(w, old *window, now time.Time) *window {
	if w != nil {
		return w
	}
	if old != nil && old.ResetsAt > now.Unix() {
		return old
	}
	return nil
}

// text is the status line Claude Code shows.
func (st *state) text(now time.Time) string {
	var parts []string
	if st.Model != "" {
		parts = append(parts, st.Model)
	}
	if st.Context != nil {
		parts = append(parts, fmt.Sprintf("ctx %.0f%%", *st.Context))
	}
	for _, l := range []struct {
		name string
		w    *window
	}{{"5h", st.FiveHour}, {"7d", st.SevenDay}} {
		if l.w != nil && l.w.ResetsAt > now.Unix() {
			parts = append(parts, fmt.Sprintf("%s %.0f%%", l.name, l.w.UsedPercentage))
		}
	}
	return strings.Join(parts, " | ")
}

// usage is the status part of what the meter shows at now.
func (st *state) usage(now time.Time) ccproto.Usage {
	u := ccproto.Unknown()
	_, offset := now.Zone()
	u.Clock = now.Unix() + int64(offset)
	if st.Updated == 0 {
		return u
	}
	u.StatusAge = int(max(now.Unix()-st.Updated, 0))
	u.Project, u.Model = st.Project, st.Model
	if st.Context != nil {
		u.Context = percent(*st.Context)
	}
	u.FiveHour = limit(st.FiveHour, now)
	u.Week = limit(st.SevenDay, now)
	return u
}

func limit(w *window, now time.Time) ccproto.Limit {
	switch {
	case w == nil:
		return ccproto.Limit{Percent: -1, ResetsIn: -1}
	case w.ResetsAt <= now.Unix():
		return ccproto.Limit{Percent: 0, ResetsIn: -1} // it has reset since
	}
	return ccproto.Limit{Percent: percent(w.UsedPercentage), ResetsIn: int(w.ResetsAt - now.Unix())}
}

func percent(v float64) int {
	if math.IsNaN(v) || v < 0 {
		return -1
	}
	return int(math.Round(min(v, 999)))
}

func loadState(path string) (state, error) {
	var st state
	data, err := os.ReadFile(path)
	if err != nil {
		return st, err
	}
	if err := json.Unmarshal(data, &st); err != nil {
		return state{}, fmt.Errorf("%s: %v", path, err)
	}
	return st, nil
}

// saveState writes the state file through a temporary file, so ccbridge
// serve never reads half of it.
func saveState(path string, st *state) error {
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".status-*.json")
	if err != nil {
		return err
	}
	if _, err := f.Write(append(data, '\n')); err != nil {
		f.Close()
		os.Remove(f.Name())
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return err
	}
	if err := os.Rename(f.Name(), path); err != nil {
		os.Remove(f.Name())
		return err
	}
	return nil
}
