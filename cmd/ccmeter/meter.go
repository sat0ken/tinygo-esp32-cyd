package main

import (
	"image/color"
	"time"
	"unsafe"

	"github.com/sat0ken/tinygo-cyd/ccproto"
	"github.com/sat0ken/tinygo-cyd/hal"
	"github.com/sat0ken/tinygo-cyd/romfont/freesans"
	"github.com/sat0ken/tinygo-cyd/romfont/proggy"
	"tinygo.org/x/tinyfont"
)

var (
	bgColor    = color.RGBA{20, 20, 24, 255}
	titleColor = color.RGBA{196, 98, 66, 255}
	white      = color.RGBA{255, 255, 255, 255}
	labelColor = color.RGBA{150, 150, 165, 255}
	textColor  = color.RGBA{220, 220, 228, 255}
	trackColor = color.RGBA{48, 48, 58, 255}
	okColor    = color.RGBA{80, 200, 120, 255}
	warnColor  = color.RGBA{240, 180, 60, 255}
	fullColor  = color.RGBA{235, 80, 70, 255}
	hourColor  = color.RGBA{110, 140, 190, 255}
	nowColor   = color.RGBA{236, 140, 100, 255}
)

// lostAfter is how long without a line from the PC before the meter says
// so (ccbridge sends every few seconds).
const lostAfter = 20 * time.Second

// Rows of limits.
const (
	rowFiveHour = iota
	rowWeek
	rowContext
	rows
)

// layout is where everything is on a w x h screen.
type layout struct {
	small   bool
	margin  int16
	titleH  int16
	rowY    [rows]int16 // top of each row
	label   int16       // label baseline, from the row top
	barY    int16       // bar top, from the row top
	barH    int16
	barW    int16
	tokY    int16 // top of the tokens section
	histY   int16 // the hourly bars
	histH   int16
	footY   int16 // footer baseline
	titleF  tinyfont.Fonter
	labelF  tinyfont.Fonter
	infoF   tinyfont.Fonter
	pctF    tinyfont.Fonter
	footF   tinyfont.Fonter
	titleBL int16 // title baseline
	labels  [rows + 1]string
	infoX   int16 // left edge of the texts right of the labels
}

// layoutFor returns the layout of a w x h screen: 480x320 (ESP32-3248S035C),
// 480x272 (ESP32-4827S043, the browser) or 320x240 (ESP32-2432S028R).
func layoutFor(w, h int16) layout {
	if w >= 480 {
		l := layout{
			margin: 12, titleH: 34, titleBL: 24, label: 15, barY: 22, barH: 24,
			titleF: &freesans.Bold12pt7b, labelF: &freesans.Bold9pt7b, infoF: &freesans.Regular9pt7b,
			pctF: &freesans.Bold18pt7b, footF: &proggy.TinySZ8pt7b,
			labels: [...]string{"5-HOUR LIMIT", "WEEKLY LIMIT", "CONTEXT", "TOKENS / HOUR"},
			infoX:  152,
		}
		rowH := int16(62)
		if h < 300 {
			l.titleH, l.titleBL = 30, 22
			rowH, l.barY, l.barH = 52, 21, 18
		}
		l.barW = w - 2*l.margin - 100
		for i := range l.rowY {
			l.rowY[i] = l.titleH + 8 + int16(i)*rowH
		}
		l.tokY = l.rowY[rows-1] + rowH
		l.footY = h - 5
		l.histY = l.tokY + 22
		l.histH = l.footY - 14 - l.histY
		return l
	}
	l := layout{
		small: true, margin: 8, titleH: 26, titleBL: 19, label: 13, barY: 17, barH: 14,
		titleF: &freesans.Bold9pt7b, labelF: &freesans.Bold9pt7b, infoF: &proggy.TinySZ8pt7b,
		pctF: &freesans.Bold12pt7b, footF: &proggy.TinySZ8pt7b,
		labels: [...]string{"5 HOURS", "WEEK", "CONTEXT", "TOKENS/HOUR"},
		infoX:  104,
	}
	rowH := int16(44)
	l.barW = w - 2*l.margin - 72
	for i := range l.rowY {
		l.rowY[i] = l.titleH + 4 + int16(i)*rowH
	}
	l.tokY = l.rowY[rows-1] + rowH
	l.footY = h - 4
	l.histY = l.tokY + 18
	l.histH = l.footY - 12 - l.histY
	return l
}

// text is a line of text on the screen, redrawn only when it changes.
type text struct {
	x, y, w, h int16 // cleared before drawing
	base       int16 // baseline
	right      bool  // aligned to x+w
	font       tinyfont.Fonter
	bg         color.RGBA

	shown [48]byte
	n     int
	fg    color.RGBA
	drawn bool
}

// newText makes a text area w wide with its baseline at base, as high as
// the font's line.
func newText(font tinyfont.Fonter, x, base, w int16, right bool, bg color.RGBA) text {
	ya := int16(font.GetYAdvance())
	return text{x: x, y: base - ya*3/4, w: w, h: ya, base: base, right: right, font: font, bg: bg}
}

func (t *text) set(d hal.Display, s []byte, fg color.RGBA) {
	if len(s) > len(t.shown) {
		s = s[:len(t.shown)]
	}
	if t.drawn && fg == t.fg && equal(s, t.shown[:t.n]) {
		return
	}
	t.n = copy(t.shown[:], s)
	t.fg, t.drawn = fg, true
	d.FillRectangle(t.x, t.y, t.w, t.h, t.bg)
	// Cut what does not fit (a long model or project name).
	for n := t.n; n > 0; n-- {
		str := unsafe.String(&t.shown[0], n)
		_, w := tinyfont.LineWidth(t.font, str)
		if int16(w) > t.w {
			continue
		}
		x := t.x
		if t.right {
			x = t.x + t.w - int16(w)
		}
		writeLine(d, t.font, x, t.base, str, fg)
		return
	}
}

// writeLine is tinyfont.WriteLine (without line breaks) that does not
// allocate: WriteLine makes a slice for the colour and puts a
// RotatedDisplay in an interface, on every call.
func writeLine(d hal.Display, font tinyfont.Fonter, x, y int16, s string, c color.RGBA) {
	for _, r := range s {
		g := font.GetGlyph(r)
		g.Draw(d, x, y, c)
		x += int16(g.Info().XAdvance)
	}
}

func equal(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// gauge is a bar filled to a percentage.
type gauge struct {
	x, y, w, h int16
	pct        int // shown; -2: nothing yet
}

func (g *gauge) set(d hal.Display, pct int) {
	if pct == g.pct {
		return
	}
	g.pct = pct
	fill := int16(0)
	c := okColor
	if pct >= 0 {
		fill = int16(min(pct, 100)) * g.w / 100
		if pct > 0 && fill == 0 {
			fill = 1
		}
		switch {
		case pct >= 80:
			c = fullColor
		case pct >= 50:
			c = warnColor
		}
	}
	d.FillRectangle(g.x, g.y, fill, g.h, c)
	d.FillRectangle(g.x+fill, g.y, g.w-fill, g.h, trackColor)
}

// Meter shows the Claude Code usage that tools/ccbridge sends.
type Meter struct {
	d hal.Display
	l layout
	w int16
	h int16

	u        ccproto.Usage
	statusAt time.Time // when the last U line came
	lineAt   time.Time // when the last line came
	got      bool      // a line came

	in   [160]byte // the line being received
	inN  int
	long bool // longer than in: skipped

	buf [48]byte // to format a text

	clock, model text
	info, pct    [rows]text
	bar          [rows]gauge
	tokens, foot text
	hours        [ccproto.Hours]uint8
	hoursKnown   bool
	hoursDrawn   bool

	now time.Time // of the last Update
}

// NewMeter makes the meter for the size of d. Call Draw once, then Feed
// with the bytes from the PC and Update now and then.
func NewMeter(d hal.Display) *Meter {
	w, h := d.Size()
	m := &Meter{d: d, w: w, h: h, l: layoutFor(w, h), u: ccproto.Unknown()}
	l := &m.l
	mg := l.margin
	if l.small {
		m.clock = newText(l.titleF, w-mg-50, l.titleBL, 50, true, titleColor)
		m.model = newText(&proggy.TinySZ8pt7b, 120, l.titleBL-1, w-mg-50-8-120, true, titleColor)
	} else {
		m.clock = newText(l.titleF, w-mg-70, l.titleBL, 70, true, titleColor)
		m.model = newText(l.infoF, 170, l.titleBL-1, w-mg-70-12-170, true, titleColor)
	}
	for i := range m.info {
		y := l.rowY[i]
		m.info[i] = newText(l.infoF, l.infoX, y+l.label, mg+l.barW-l.infoX, true, bgColor)
		px := mg + l.barW + 8
		m.pct[i] = newText(l.pctF, px, y+l.barY+l.barH, w-mg-px, true, bgColor)
		m.bar[i] = gauge{x: mg, y: y + l.barY, w: l.barW, h: l.barH, pct: -2}
	}
	m.tokens = newText(l.infoF, mg+150, l.tokY+l.label, w-2*mg-150, true, bgColor)
	m.foot = newText(l.footF, mg, l.footY, w-2*mg, false, bgColor)
	return m
}

// Draw draws the whole screen.
func (m *Meter) Draw() {
	d, l := m.d, &m.l
	d.FillRectangle(0, 0, m.w, m.h, bgColor)
	d.FillRectangle(0, 0, m.w, l.titleH, titleColor)
	writeLine(d, l.titleF, l.margin, l.titleBL, "Claude Code", white)
	for i, s := range l.labels {
		y := l.tokY
		if i < rows {
			y = l.rowY[i]
		}
		writeLine(d, l.labelF, l.margin, y+l.label, s, labelColor)
	}
	m.clock.drawn, m.model.drawn, m.tokens.drawn, m.foot.drawn = false, false, false, false
	for i := range m.info {
		m.info[i].drawn, m.pct[i].drawn, m.bar[i].pct = false, false, -2
	}
	m.hoursDrawn = false
	m.Update(m.now)
}

// Feed takes a byte from the PC, received at now.
func (m *Meter) Feed(b byte, now time.Time) {
	switch {
	case b == '\n':
		if !m.long {
			switch m.u.Parse(m.in[:m.inN]) {
			case ccproto.Status:
				m.statusAt, m.lineAt, m.got = now, now, true
			case ccproto.Tokens:
				m.lineAt, m.got = now, true
			}
		}
		m.inN, m.long = 0, false
	case b == '\r':
	case m.inN == len(m.in):
		m.long = true
	default:
		m.in[m.inN] = b
		m.inN++
	}
}

// Update redraws what changed by now (the zero time: before anything came).
func (m *Meter) Update(now time.Time) {
	m.now = now
	d, u := m.d, &m.u
	el := 0 // seconds since the last U line
	if m.got && !now.IsZero() {
		el = int(now.Sub(m.statusAt) / time.Second)
	}
	lost := !m.got || now.IsZero() || now.Sub(m.lineAt) > lostAfter

	b := m.buf[:0]
	if u.Clock > 0 {
		b = appendClock(b, u.Clock+int64(el))
	}
	m.clock.set(d, b, white)
	m.model.set(d, append(m.buf[:0], u.Model...), white)

	limits := [2]ccproto.Limit{u.FiveHour, u.Week}
	for i, lim := range limits {
		b = m.buf[:0]
		if lim.Percent >= 0 && lim.ResetsIn >= 0 && u.Clock > 0 {
			b = m.appendReset(b, u.Clock+int64(lim.ResetsIn), lim.ResetsIn-el, i == rowWeek)
		}
		m.info[i].set(d, b, textColor)
		m.setPercent(i, lim.Percent)
	}
	m.info[rowContext].set(d, append(m.buf[:0], u.Project...), textColor)
	m.setPercent(rowContext, u.Context)

	b = m.buf[:0]
	if u.Tokens5h >= 0 {
		b = append(b, "5h "...)
		b = appendTokens(b, u.Tokens5h)
	}
	if u.Tokens24h >= 0 {
		if len(b) > 0 {
			b = append(b, "   "...)
		}
		b = append(b, "24h "...)
		b = appendTokens(b, u.Tokens24h)
	}
	m.tokens.set(d, b, textColor)
	m.setHours(u.HourMax >= 0, &u.HourLevel)

	b = m.buf[:0]
	c := labelColor
	switch {
	case !m.got:
		b = append(b, "Waiting for ccbridge on the PC (USB serial)"...)
	case lost:
		c = warnColor
		b = append(b, "No data from ccbridge for "...)
		b = appendAgo(b, int(now.Sub(m.lineAt)/time.Second))
	case u.StatusAge < 0:
		b = append(b, "PC link OK, no status line report yet"...)
	default:
		b = append(b, "PC link OK, status line "...)
		b = appendAgo(b, u.StatusAge+el)
		b = append(b, " ago"...)
	}
	m.foot.set(d, b, c)
}

func (m *Meter) setPercent(row, pct int) {
	b := m.buf[:0]
	if pct >= 0 {
		b = appendInt(b, pct)
		b = append(b, '%')
	} else {
		b = append(b, "--"...)
	}
	m.pct[row].set(m.d, b, white)
	m.bar[row].set(m.d, pct)
}

// appendReset appends when a limit resets: at local time at, in left
// seconds ("resets 15:00 (1h 23m)", with the day for the weekly limit).
func (m *Meter) appendReset(b []byte, at int64, left int, week bool) []byte {
	if !m.l.small {
		b = append(b, "resets "...)
	}
	if week {
		b = append(b, weekdays[(at/86400+4)%7]...)
		b = append(b, ' ')
	}
	b = appendClock(b, at)
	if left <= 0 {
		return b
	}
	b = append(b, " ("...)
	b = appendDuration(b, left)
	return append(b, ')')
}

var weekdays = [7]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"} // 1970-01-01 was a Thursday

func (m *Meter) setHours(known bool, levels *[ccproto.Hours]uint8) {
	if m.hoursDrawn && known == m.hoursKnown && *levels == m.hours {
		return
	}
	m.hours, m.hoursKnown, m.hoursDrawn = *levels, known, true
	d, l := m.d, &m.l
	w := m.w - 2*l.margin
	slot := w / ccproto.Hours
	bw := slot * 3 / 4
	x0 := l.margin + (w-slot*ccproto.Hours)/2 + (slot-bw)/2
	d.FillRectangle(l.margin, l.histY, w, l.histH, bgColor)
	for i, lv := range m.hours {
		x := x0 + int16(i)*slot
		bh := int16(lv) * (l.histH - 1) / (ccproto.Levels - 1)
		c := hourColor
		if i == ccproto.Hours-1 {
			c = nowColor
		}
		d.FillRectangle(x, l.histY+l.histH-1, bw, 1, trackColor)
		if known && bh > 0 {
			d.FillRectangle(x, l.histY+l.histH-1-bh, bw, bh, c)
		}
	}
}

// appendClock appends "HH:MM" of a local time in seconds.
func appendClock(b []byte, t int64) []byte {
	s := t % 86400
	if s < 0 {
		s += 86400
	}
	b = append2(b, int(s/3600))
	b = append(b, ':')
	return append2(b, int(s/60%60))
}

func append2(b []byte, v int) []byte {
	return append(b, byte('0'+v/10), byte('0'+v%10))
}

// appendDuration appends seconds as "45m", "1h 23m" or "3d 4h" ("<1m"
// under a minute).
func appendDuration(b []byte, s int) []byte {
	switch {
	case s < 60:
		return append(b, "<1m"...)
	case s < 3600:
		b = appendInt(b, s/60)
		return append(b, 'm')
	case s < 86400:
		b = appendInt(b, s/3600)
		b = append(b, "h "...)
		b = appendInt(b, s/60%60)
		return append(b, 'm')
	}
	b = appendInt(b, s/86400)
	b = append(b, "d "...)
	b = appendInt(b, s/3600%24)
	return append(b, 'h')
}

// appendAgo appends seconds as "12s", "5m", "3h" or "2d".
func appendAgo(b []byte, s int) []byte {
	switch {
	case s < 60:
		b = appendInt(b, s)
		return append(b, 's')
	case s < 3600:
		b = appendInt(b, s/60)
		return append(b, 'm')
	case s < 86400:
		b = appendInt(b, s/3600)
		return append(b, 'h')
	}
	b = appendInt(b, s/86400)
	return append(b, 'd')
}

// appendTokens appends a token count with 2 or 3 digits: "950", "1.2k",
// "34k", "123M", "1.5G".
func appendTokens(b []byte, n int64) []byte {
	const units = "kMGT"
	if n < 1000 {
		return appendInt(b, int(n))
	}
	u := -1
	for n >= 999500 && u < len(units)-2 { // n/1000 would round to 1000 or more
		n /= 1000
		u++
	}
	u++ // show n/1000 in units[u], rounded
	if n < 9950 {
		t := (n + 50) / 100 // tenths
		b = appendInt(b, int(t/10))
		b = append(b, '.', byte('0'+t%10))
	} else {
		b = appendInt(b, int((n+500)/1000))
	}
	return append(b, units[u])
}

func appendInt(b []byte, v int) []byte {
	if v < 0 {
		b = append(b, '-')
		v = -v
	}
	var tmp [20]byte
	i := len(tmp)
	for {
		i--
		tmp[i] = byte('0' + v%10)
		v /= 10
		if v == 0 {
			break
		}
	}
	return append(b, tmp[i:]...)
}
