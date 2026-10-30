// Command ccbridge sends the Claude Code usage to cmd/ccmeter on the board,
// over the USB serial port (package ccproto).
//
//	go install ./tools/ccbridge
//	ccbridge statusline     # the status line command of Claude Code
//	ccbridge serve [-port /dev/ttyUSB0] [-every 5s] [-v]
//	ccbridge print          # print the lines once, to check
//
// The plan limits (5-hour and weekly) and the context window come from the
// status line: Claude Code runs its status line command after its answers
// with a JSON on stdin, and "ccbridge statusline" keeps what it needs in
// ~/.cache/ccmeter/status.json. Set it in ~/.claude/settings.json:
//
//	"statusLine": {"type": "command", "command": "/home/you/go/bin/ccbridge statusline"}
//
// The tokens come from the conversation logs, ~/.claude/projects/*/*.jsonl
// ($CLAUDE_CONFIG_DIR/projects if set).
//
// "ccbridge serve" keeps the serial port open (stop it before flashing the
// board) and sends the usage every few seconds. Only Linux for now.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/sat0ken/tinygo-cyd/ccproto"
)

func usage() {
	fmt.Fprintf(os.Stderr, `usage:
  ccbridge statusline   status line command for Claude Code (JSON on stdin)
  ccbridge serve [-port /dev/ttyUSB0] [-every 5s] [-v]
  ccbridge print        print the lines once
`)
	os.Exit(2)
}

func main() {
	log.SetFlags(log.Ltime)
	if len(os.Args) < 2 {
		usage()
	}
	cmd, args := os.Args[1], os.Args[2:]
	switch cmd {
	case "statusline":
		if err := statusLine(os.Stdin, os.Stdout, stateFile(), time.Now()); err != nil {
			// Claude Code shows the first line of stdout: say what is wrong there.
			fmt.Println("ccbridge:", err)
		}
	case "serve":
		fs := flag.NewFlagSet("serve", flag.ExitOnError)
		port := fs.String("port", "/dev/ttyUSB0", "serial port of the board")
		every := fs.Duration("every", 5*time.Second, "how often to send")
		verbose := fs.Bool("v", false, "print what is sent and what the board prints")
		fs.Parse(args)
		serve(*port, *every, *verbose)
	case "print":
		sc := newScanner(projectsDir())
		u := current(sc, stateFile(), time.Now())
		os.Stdout.Write(ccproto.AppendTokens(ccproto.AppendStatus(nil, &u), &u))
	default:
		usage()
	}
}

// stateFile is where "ccbridge statusline" keeps the last status line
// report: $XDG_CACHE_HOME/ccmeter/status.json (~/.cache/ccmeter/status.json).
func stateFile() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "ccmeter", "status.json")
}

// projectsDir is where Claude Code keeps the conversation logs.
func projectsDir() string {
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, "projects")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "projects")
}

// serve sends the usage to the board every so often, opening the port
// again when it goes away (the board unplugged or reset by a flash).
func serve(port string, every time.Duration, verbose bool) {
	sc := newScanner(projectsDir())
	state := stateFile()
	for {
		f, err := openSerial(port)
		if err != nil {
			log.Printf("%v (trying again)", err)
			time.Sleep(3 * time.Second)
			continue
		}
		log.Printf("sending to %s every %v", port, every)
		go func() {
			// Read what the board prints, so it does not pile up in the
			// port's buffer; show it with -v.
			s := bufio.NewScanner(f)
			for s.Scan() {
				if verbose {
					log.Printf("board: %s", s.Bytes())
				}
			}
		}()
		for {
			u := current(sc, state, time.Now())
			status := ccproto.AppendStatus(nil, &u)
			tokens := ccproto.AppendTokens(nil, &u)
			if verbose {
				log.Printf("sent: %s", status[:len(status)-1])
				log.Printf("sent: %s", tokens[:len(tokens)-1])
			}
			if _, err = f.Write(status); err != nil {
				break
			}
			// The board keeps only 128 bytes while it draws: let it
			// read the first line before the second comes.
			time.Sleep(200 * time.Millisecond)
			if _, err = f.Write(tokens); err != nil {
				break
			}
			time.Sleep(every)
		}
		log.Printf("%s: %v", port, err)
		f.Close()
		time.Sleep(time.Second)
	}
}

// current is the usage now: the last status line report from state and
// the tokens from the logs.
func current(sc *scanner, state string, now time.Time) ccproto.Usage {
	st, err := loadState(state)
	if err != nil && !os.IsNotExist(err) {
		log.Print(err)
	}
	u := st.usage(now)
	if err := sc.update(now); err != nil {
		log.Print(err)
		return u
	}
	// Tokens in the current 5-hour window, when the status line said when
	// it ends; the last 5 hours otherwise.
	from := now.Add(-5 * time.Hour)
	if w := st.FiveHour; w != nil && w.ResetsAt > now.Unix() {
		from = time.Unix(w.ResetsAt, 0).Add(-5 * time.Hour)
	}
	u.Tokens5h = sc.sum(from, now)
	u.Tokens24h = sc.sum(now.Add(-24*time.Hour), now)
	hours := sc.hours(now)
	u.HourMax = 0
	for _, v := range hours {
		u.HourMax = max(u.HourMax, v)
	}
	for i, v := range hours {
		u.HourLevel[i] = ccproto.Level(v, u.HourMax)
	}
	return u
}
