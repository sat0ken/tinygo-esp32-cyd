// Package ccproto is the line format in which tools/ccbridge (on the PC)
// sends the Claude Code usage to cmd/ccmeter (on the board) over the serial
// port, 115200 baud.
//
// Two kinds of lines, both ASCII, ending in '\n', with fields "key=value"
// separated by one space. Unknown keys are skipped and a missing key means
// "unknown", so either side can be extended.
//
//	U t=1793287800 h=42,5400 w=17,280000 c=63 a=12 p=tinygo-cyd m=Opus 5.5
//	T f=34567890 d=123456789 x=9876543 b=000000000000000012359azk
//
// U (status, from the Claude Code status line):
//
//	t  the PC's local time: seconds since 1970-01-01 00:00 in its time zone
//	h  the 5-hour limit: percent used, then seconds until it resets
//	w  the weekly limit, the same way
//	c  percent of the context window used by the latest session
//	a  seconds since the status line last reported
//	p  the project (directory name) of the latest session, no spaces
//	m  the model name; it takes the rest of the line, spaces included
//
// T (tokens, from the conversation logs; input, output and cache tokens
// together):
//
//	f  tokens in the current 5-hour window (the last 5 hours if unknown)
//	d  tokens in the last 24 hours
//	x  tokens in the busiest of the last 24 hours
//	b  24 levels, oldest hour first, the current hour last: a digit or
//	   lower-case letter, 0..35 of x
//
// A line is at most MaxLine bytes: the board keeps 128 received bytes
// (TinyGo's UART ring buffer) while it is busy drawing.
package ccproto

import "strconv"

// MaxLine is the longest line, '\n' included.
const MaxLine = 127

// Name lengths in a U line; longer names are cut.
const (
	MaxProject = 20
	MaxModel   = 20
)

// Hours is the number of hourly levels; Levels is the highest level + 1.
const (
	Hours  = 24
	Levels = 36
)

// Limit is a plan limit.
type Limit struct {
	Percent  int // -1: unknown
	ResetsIn int // seconds; -1: unknown (also when the window has not started)
}

// Usage is what the meter shows. The zero value is not "unknown": start
// from Unknown().
type Usage struct {
	// U line
	Clock     int64 // local time, seconds since 1970 in the PC's time zone; 0: unknown
	FiveHour  Limit
	Week      Limit
	Context   int // percent; -1: unknown
	StatusAge int // seconds; -1: no status line report yet
	Project   string
	Model     string

	// T line
	Tokens5h  int64 // -1: unknown
	Tokens24h int64 // -1: unknown
	HourMax   int64 // -1: unknown
	HourLevel [Hours]uint8
}

// Unknown returns a Usage with every value unknown.
func Unknown() Usage {
	var u Usage
	u.clearStatus()
	u.clearTokens()
	return u
}

// clearStatus sets the status values to unknown, but the names: Parse
// keeps them while they do not change, so as not to allocate.
func (u *Usage) clearStatus() {
	u.Clock = 0
	u.FiveHour = Limit{-1, -1}
	u.Week = Limit{-1, -1}
	u.Context = -1
	u.StatusAge = -1
}

func (u *Usage) clearTokens() {
	u.Tokens5h, u.Tokens24h, u.HourMax = -1, -1, -1
	u.HourLevel = [Hours]uint8{}
}

// Line kinds returned by Parse.
const (
	None   = 0
	Status = 'U'
	Tokens = 'T'
)

// Parse reads one line (without '\n') into u: a U line sets every status
// value (to unknown when its key is missing), a T line every token value.
// It returns the kind of the line, or None for anything else (u is then
// unchanged). It does not allocate, except for a project or model name
// that changed.
func (u *Usage) Parse(line []byte) int {
	if len(line) < 1 || (len(line) > 1 && line[1] != ' ') {
		return None
	}
	kind := int(line[0])
	switch kind {
	case Status:
		u.clearStatus()
	case Tokens:
		u.clearTokens()
	default:
		return None
	}
	project, model := false, false
	rest := line[1:]
	for len(rest) > 0 {
		if rest[0] == ' ' {
			rest = rest[1:]
			continue
		}
		eq := index(rest, '=')
		if eq < 0 {
			break
		}
		key := rest[:eq]
		rest = rest[eq+1:]
		if kind == Status && len(key) == 1 && key[0] == 'm' {
			setName(&u.Model, rest, MaxModel)
			model = true
			break
		}
		end := index(rest, ' ')
		if end < 0 {
			end = len(rest)
		}
		val := rest[:end]
		rest = rest[end:]
		if len(key) != 1 {
			continue
		}
		switch {
		case kind == Tokens:
			u.tokensField(key[0], val)
		case key[0] == 'p':
			setName(&u.Project, val, MaxProject)
			project = true
		default:
			u.statusField(key[0], val)
		}
	}
	if kind == Status {
		if !project {
			u.Project = ""
		}
		if !model {
			u.Model = ""
		}
	}
	return kind
}

func (u *Usage) statusField(k byte, v []byte) {
	switch k {
	case 't':
		u.Clock = num(v, 0)
	case 'h':
		u.FiveHour = limit(v)
	case 'w':
		u.Week = limit(v)
	case 'c':
		u.Context = int(num(v, -1))
	case 'a':
		u.StatusAge = int(num(v, -1))
	}
}

func (u *Usage) tokensField(k byte, v []byte) {
	switch k {
	case 'f':
		u.Tokens5h = num(v, -1)
	case 'd':
		u.Tokens24h = num(v, -1)
	case 'x':
		u.HourMax = num(v, -1)
	case 'b':
		if len(v) != Hours {
			return
		}
		for i, c := range v {
			u.HourLevel[i] = uint8(digit(c))
		}
	}
}

// limit parses "percent" or "percent,seconds".
func limit(v []byte) Limit {
	l := Limit{-1, -1}
	if c := index(v, ','); c >= 0 {
		l.ResetsIn = int(num(v[c+1:], -1))
		v = v[:c]
	}
	if l.Percent = int(num(v, -1)); l.Percent < 0 {
		return Limit{-1, -1}
	}
	return l
}

// num parses a non-negative decimal number, or returns bad.
func num(v []byte, bad int64) int64 {
	if len(v) == 0 || len(v) > 18 {
		return bad
	}
	var n int64
	for _, c := range v {
		if c < '0' || c > '9' {
			return bad
		}
		n = n*10 + int64(c-'0')
	}
	return n
}

// digit returns the level of a base-36 digit (0 for anything else).
func digit(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'z':
		return int(c-'a') + 10
	}
	return 0
}

func index(b []byte, c byte) int {
	for i, x := range b {
		if x == c {
			return i
		}
	}
	return -1
}

// setName sets *s to v (cut to n bytes), allocating only if it changed.
func setName(s *string, v []byte, n int) {
	if len(v) > n {
		v = v[:n]
	}
	if len(*s) == len(v) {
		same := true
		for i := range v {
			if (*s)[i] != v[i] {
				same = false
				break
			}
		}
		if same {
			return
		}
	}
	*s = string(v)
}

// AppendStatus appends the U line of u, '\n' included.
func AppendStatus(b []byte, u *Usage) []byte {
	b = append(b, 'U')
	if u.Clock > 0 {
		b = append(b, " t="...)
		b = strconv.AppendInt(b, u.Clock, 10)
	}
	b = appendLimit(b, 'h', u.FiveHour)
	b = appendLimit(b, 'w', u.Week)
	if u.Context >= 0 {
		b = append(b, " c="...)
		b = strconv.AppendInt(b, int64(u.Context), 10)
	}
	if u.StatusAge >= 0 {
		b = append(b, " a="...)
		b = strconv.AppendInt(b, int64(u.StatusAge), 10)
	}
	if p := name(u.Project, MaxProject); p != "" {
		b = append(b, " p="...)
		for i := 0; i < len(p); i++ {
			if c := p[i]; c == ' ' {
				b = append(b, '_')
			} else {
				b = append(b, c)
			}
		}
	}
	if m := name(u.Model, MaxModel); m != "" {
		b = append(b, " m="...)
		b = append(b, m...)
	}
	return append(b, '\n')
}

func appendLimit(b []byte, key byte, l Limit) []byte {
	if l.Percent < 0 {
		return b
	}
	b = append(b, ' ', key, '=')
	b = strconv.AppendInt(b, int64(l.Percent), 10)
	if l.ResetsIn >= 0 {
		b = append(b, ',')
		b = strconv.AppendInt(b, int64(l.ResetsIn), 10)
	}
	return b
}

// name cuts s to n bytes and drops what cannot be sent (control
// characters and non-ASCII, which the board's fonts do not have anyway).
func name(s string, n int) string {
	ok := true
	for i := 0; i < len(s); i++ {
		if s[i] < ' ' || s[i] > '~' {
			ok = false
			break
		}
	}
	if !ok {
		b := make([]byte, 0, len(s))
		for i := 0; i < len(s); i++ {
			if s[i] >= ' ' && s[i] <= '~' {
				b = append(b, s[i])
			}
		}
		s = string(b)
	}
	if len(s) > n {
		s = s[:n]
	}
	return s
}

// AppendTokens appends the T line of u, '\n' included.
func AppendTokens(b []byte, u *Usage) []byte {
	b = append(b, 'T')
	b = appendNum(b, 'f', u.Tokens5h)
	b = appendNum(b, 'd', u.Tokens24h)
	b = appendNum(b, 'x', u.HourMax)
	if u.HourMax >= 0 {
		b = append(b, " b="...)
		for _, l := range u.HourLevel {
			b = append(b, "0123456789abcdefghijklmnopqrstuvwxyz"[min(int(l), Levels-1)])
		}
	}
	return append(b, '\n')
}

func appendNum(b []byte, key byte, v int64) []byte {
	if v < 0 {
		return b
	}
	b = append(b, ' ', key, '=')
	return strconv.AppendInt(b, v, 10)
}

// Level returns the level (0..35) of v tokens in an hour when the busiest
// hour had top: any use gives at least 1.
func Level(v, top int64) uint8 {
	if v <= 0 || top <= 0 {
		return 0
	}
	l := (v*(Levels-1) + top - 1) / top
	return uint8(min(max(l, 1), Levels-1))
}
