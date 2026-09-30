package platform

import "time"

// Halt blocks forever, e.g. after showing an error.
//
// Use it instead of `select {}`: on the board TinyGo reports an empty
// select with no timer pending as "fatal error: deadlocked: no event
// source" and aborts, which would also stop the display from showing the
// error. A sleeping loop keeps the scheduler alive on every target.
func Halt() {
	for {
		time.Sleep(time.Hour)
	}
}
