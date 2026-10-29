package picopad

import (
	"time"

	"github.com/sat0ken/tinygo-cyd/hal"
)

// The buttons of the PicoPad (KEY_*), as KeyGet returns them.
const (
	NoKey    = 0
	KeyUp    = 1
	KeyLeft  = 2
	KeyRight = 3
	KeyDown  = 4
	KeyX     = 5
	KeyY     = 6
	KeyA     = 7
	KeyB     = 8

	KeyRelease = 0x80 // KeyGetRel: the button was released
)

// The timing of PicoLibSDK (picopad_key.c): the buttons are scanned every
// SysTick (5ms); a held button repeats after 400ms, then every 100ms; it
// counts as released after 50ms up.
const (
	scanEvery    = 5 * time.Millisecond
	repeatFirst  = 400 * time.Millisecond
	repeatNext   = 100 * time.Millisecond
	releaseAfter = 50 * time.Millisecond
	keyBufSize   = 16
)

// busyFor: KeyGet, KeyPressed and Time sleep 1ms when the game has not
// slept for this long (it is waiting in a loop), so that the browser can
// run its events.
const busyFor = 20 * time.Millisecond

// keyButtons are the controller buttons of KeyUp ... KeyB. Pressing the
// stick is A.
var keyButtons = [8]hal.Buttons{
	hal.PadUp, hal.PadLeft, hal.PadRight, hal.PadDown,
	hal.PadX, hal.PadY, hal.PadA | hal.PadOK, hal.PadB,
}

// touchButtons are the buttons of the parts of the display, in 3 x 3:
// the middle of each edge is a direction, the corners are Y, X, B and A,
// and the middle is A.
var touchButtons = [3][3]hal.Buttons{
	{hal.PadY, hal.PadUp, hal.PadX},
	{hal.PadLeft, hal.PadA, hal.PadRight},
	{hal.PadB, hal.PadDown, hal.PadA},
}

// Clock is the time of a console: the real one (NewClock), or a fake one
// in tests.
type Clock interface {
	Now() time.Duration // since any fixed moment
	Sleep(d time.Duration)
}

// NewClock returns the real time (time.Now and time.Sleep).
func NewClock() Clock { return &sysClock{start: time.Now()} }

type sysClock struct{ start time.Time }

func (s *sysClock) Now() time.Duration    { return time.Since(s.start) }
func (s *sysClock) Sleep(d time.Duration) { time.Sleep(d) }

// keys is the button state of a console.
type keys struct {
	// Pad, if set, is the game controller (the stick is the d-pad).
	Pad hal.Pad
	// Clock is the time; New sets the real one.
	Clock Clock

	t        hal.Touch
	tw, th   int // the display, for touchButtons
	scanned  bool
	lastScan time.Duration
	slept    time.Duration // when the game last slept

	down    [8]bool          // KeyPressMap
	repeat  [8]time.Duration // KeyLastPress: when it last repeated
	seen    [8]time.Duration // KeyLastRelease: when it was last seen down
	initial [8]bool          // held since the first scan: not a press

	buf    [keyBufSize]uint8
	rd, wr int
}

func (k *keys) init(t hal.Touch, w, h int16) {
	k.t, k.tw, k.th = t, int(w), int(h)
	k.Clock = NewClock()
}

// buttons returns the buttons held on the controller and the touch panel.
func (k *keys) buttons() hal.Buttons {
	var b hal.Buttons
	if k.Pad != nil {
		b = k.Pad.ReadPad()
	}
	if k.t != nil {
		if x, y, pressed := k.t.ReadTouch(); pressed {
			col := min(max(int(x)*3/k.tw, 0), 2)
			row := min(max(int(y)*3/k.th, 0), 2)
			b |= touchButtons[row][col]
		}
	}
	return b
}

// scan is KeyScan: it reads the buttons (at most every scanEvery) and
// writes the presses, repeats and releases to the buffer.
func (k *keys) scan() {
	now := k.Clock.Now()
	if k.scanned && now-k.lastScan < scanEvery {
		return
	}
	first := !k.scanned
	k.scanned, k.lastScan = true, now
	b := k.buttons()
	for i, kb := range keyButtons {
		switch {
		case b&kb != 0 && first:
			// Still held from before the game (A on the launcher).
			k.down[i], k.initial[i], k.seen[i] = true, true, now
		case b&kb != 0:
			if !k.down[i] {
				k.down[i] = true
				k.repeat[i] = now + repeatFirst - repeatNext
				k.write(uint8(i + 1))
			} else if !k.initial[i] && now-k.repeat[i] >= repeatNext {
				k.repeat[i] = now
				k.write(uint8(i + 1))
			}
			k.seen[i] = now
		case k.down[i] && now-k.seen[i] >= releaseAfter:
			k.down[i] = false
			if !k.initial[i] {
				k.write(uint8(i+1) | KeyRelease)
			}
			k.initial[i] = false
		}
	}
}

func (k *keys) write(key uint8) {
	w := (k.wr + 1) % keyBufSize
	if w != k.rd { // full: the key is lost, as on the PicoPad
		k.buf[k.wr] = key
		k.wr = w
	}
}

// sleep sleeps d and notes it.
func (k *keys) sleep(d time.Duration) {
	k.Clock.Sleep(d)
	k.slept = k.Clock.Now()
}

// poll scans the buttons for the functions a game may call in a loop
// while it waits, sleeping a moment if it has been busy for busyFor.
func (k *keys) poll() {
	if k.Clock.Now()-k.slept >= busyFor {
		k.sleep(time.Millisecond)
	}
	k.scan()
}

// KeyGetRel returns the next button from the buffer, with KeyRelease set
// for a release, or NoKey.
func (k *keys) KeyGetRel() uint8 {
	k.poll()
	if k.rd == k.wr {
		return NoKey
	}
	key := k.buf[k.rd]
	k.rd = (k.rd + 1) % keyBufSize
	return key
}

// KeyGet returns the next button pressed (or repeated), or NoKey.
func (k *keys) KeyGet() uint8 {
	for {
		if key := k.KeyGetRel(); key&KeyRelease == 0 {
			return key
		}
	}
}

// KeyPressed reports whether a button (KeyUp ... KeyB) is held.
func (k *keys) KeyPressed(key uint8) bool {
	k.poll()
	key &^= KeyRelease
	if key < KeyUp || key > KeyB {
		return false
	}
	return k.down[key-1]
}

// KeyNoPressed reports whether no button is held.
func (k *keys) KeyNoPressed() bool {
	k.poll()
	for _, d := range k.down {
		if d {
			return false
		}
	}
	return true
}

// KeyWaitNoPressed waits until no button is held.
func (k *keys) KeyWaitNoPressed() {
	for !k.KeyNoPressed() {
	}
}

// KeyFlush empties the buffer.
func (k *keys) KeyFlush() { k.rd = k.wr }

// Time returns the time in microseconds (it wraps around after 71
// minutes, like the PicoPad's timer).
func (k *keys) Time() uint32 {
	k.poll()
	return uint32(k.Clock.Now() / time.Microsecond)
}

// WaitMs waits ms milliseconds, scanning the buttons meanwhile.
func (k *keys) WaitMs(ms int) { k.wait(time.Duration(ms) * time.Millisecond) }

// WaitUs waits us microseconds, scanning the buttons meanwhile.
func (k *keys) WaitUs(us int) { k.wait(time.Duration(us) * time.Microsecond) }

func (k *keys) wait(d time.Duration) {
	end := k.Clock.Now() + d
	for {
		k.scan()
		rest := end - k.Clock.Now()
		if rest <= 0 {
			return
		}
		k.sleep(min(rest, scanEvery))
	}
}
