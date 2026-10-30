package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sat0ken/tinygo-cyd/ccproto"
)

// keepFor is how far back the scanner keeps answers: the 24 hourly bars
// start up to 24 hours ago.
const keepFor = 25 * time.Hour

// scanner reads the tokens of Claude's answers from the conversation logs
// (*.jsonl under the projects directory). Every update reads only what was
// added to the logs since the last one.
//
// An answer is written as several lines (one per content block), each with
// the usage, and a resumed conversation copies the earlier answers into its
// new log: answers are counted once, by message and request ID.
type scanner struct {
	dir   string
	files map[string]int64 // bytes read, by path
	msgs  map[string]answer
}

type answer struct {
	at     time.Time
	tokens int64 // input, output, cache creation and cache read
}

func newScanner(dir string) *scanner {
	return &scanner{dir: dir, files: map[string]int64{}, msgs: map[string]answer{}}
}

// update reads what was added to the logs written in the last 25 hours.
func (s *scanner) update(now time.Time) error {
	since := now.Add(-keepFor)
	seen := map[string]bool{}
	err := filepath.WalkDir(s.dir, func(path string, e fs.DirEntry, err error) error {
		if err != nil {
			if path == s.dir {
				return err
			}
			return nil // skip what cannot be read
		}
		if e.IsDir() || !strings.HasSuffix(path, ".jsonl") {
			return nil
		}
		info, err := e.Info()
		if err != nil || info.ModTime().Before(since) {
			return nil
		}
		seen[path] = true
		return s.read(path, info.Size())
	})
	for path := range s.files {
		if !seen[path] {
			delete(s.files, path)
		}
	}
	for id, a := range s.msgs {
		if a.at.Before(since) {
			delete(s.msgs, id)
		}
	}
	return err
}

// read reads the new complete lines of a log; a line still being written
// is read next time.
func (s *scanner) read(path string, size int64) error {
	off := s.files[path]
	if size < off {
		off = 0 // written again from the start
	}
	if size == off {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil // removed meanwhile
	}
	defer f.Close()
	if _, err := f.Seek(off, io.SeekStart); err != nil {
		return err
	}
	r := bufio.NewReaderSize(f, 1<<20)
	for {
		line, err := r.ReadBytes('\n')
		if err != nil {
			break
		}
		off += int64(len(line))
		s.line(line)
	}
	s.files[path] = off
	return nil
}

var (
	usageKey     = []byte(`"usage"`)
	assistantKey = []byte(`"assistant"`)
)

// line counts the tokens of an answer.
func (s *scanner) line(b []byte) {
	if !bytes.Contains(b, usageKey) || !bytes.Contains(b, assistantKey) {
		return
	}
	var e struct {
		Type      string    `json:"type"`
		Timestamp time.Time `json:"timestamp"`
		RequestID string    `json:"requestId"`
		Message   struct {
			ID    string `json:"id"`
			Usage *struct {
				Input         int64 `json:"input_tokens"`
				Output        int64 `json:"output_tokens"`
				CacheCreation int64 `json:"cache_creation_input_tokens"`
				CacheRead     int64 `json:"cache_read_input_tokens"`
			} `json:"usage"`
		} `json:"message"`
	}
	if json.Unmarshal(b, &e) != nil || e.Type != "assistant" || e.Message.Usage == nil || e.Message.ID == "" {
		return
	}
	u := e.Message.Usage
	a := answer{e.Timestamp, u.Input + u.Output + u.CacheCreation + u.CacheRead}
	id := e.Message.ID + "/" + e.RequestID
	if old, ok := s.msgs[id]; !ok || a.tokens > old.tokens {
		s.msgs[id] = a
	}
}

// sum returns the tokens of the answers from from (included) to to.
func (s *scanner) sum(from, to time.Time) int64 {
	var n int64
	for _, a := range s.msgs {
		if !a.at.Before(from) && a.at.Before(to) {
			n += a.tokens
		}
	}
	return n
}

// hours returns the tokens per hour of the clock, the current hour last.
func (s *scanner) hours(now time.Time) [ccproto.Hours]int64 {
	var h [ccproto.Hours]int64
	_, offset := now.Zone()
	zone := time.Duration(offset) * time.Second
	start := now.Add(zone).Truncate(time.Hour).Add(-zone).Add(-(ccproto.Hours - 1) * time.Hour)
	for _, a := range s.msgs {
		if a.at.Before(start) || !a.at.Before(now) {
			continue
		}
		h[int(a.at.Sub(start)/time.Hour)] += a.tokens
	}
	return h
}
