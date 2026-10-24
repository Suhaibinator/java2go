package stdjava

import "time"

// systemNanoTimeOrigin retains Go's monotonic clock reading for this process.
// Java specifies an arbitrary common origin, not Unix time or a positive value.
var systemNanoTimeOrigin = time.Now()

// SystemNanoTime returns elapsed nanoseconds from the process's arbitrary
// origin. Callers compare differences; the value is unrelated to wall time.
func SystemNanoTime() int64 {
	return time.Since(systemNanoTimeOrigin).Nanoseconds()
}
