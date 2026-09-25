package clock

import (
	"sync/atomic"
	"time"
)

var offset atomic.Int64 // nanoseconds

// Now returns the current time as the process sees it: wall time plus the
// offset.
func Now() time.Time {
	return time.Now().Add(time.Duration(offset.Load()))
}

// Offset returns the current offset.
func Offset() time.Duration {
	return time.Duration(offset.Load())
}

// SetOffset replaces the offset. It must not be smaller than the current one:
// the clock only runs ahead, so nothing that already expired comes back to
// life and nothing issued at the later time becomes "not yet valid" (JWT
// nbf). Reset is the only way down.
func SetOffset(d time.Duration) bool {
	if d < 0 {
		return false
	}
	next := int64(d)
	for {
		cur := offset.Load()
		if next < cur {
			return false // never behind the current offset
		}
		if offset.CompareAndSwap(cur, next) {
			return true
		}
		// A concurrent SetOffset won the race; re-read and re-check.
	}
}

// Reset returns the clock to wall time.
func Reset() {
	offset.Store(0)
}

// System reads Now; it is the clock handed to code that takes a Now() source
// (entity.TimeMachine) rather than calling this package directly.
type System struct{}

// Now returns Now.
func (System) Now() time.Time {
	return Now()
}
