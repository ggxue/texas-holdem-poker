package poker

import "time"

// Clock is the deterministic time seam, never selectable through HTTP.
type Clock interface {
	Now() time.Time
	AfterFunc(time.Duration, func()) Timer
}
type Timer interface{ Stop() bool }
type systemClock struct{}

func (systemClock) Now() time.Time                            { return time.Now() }
func (systemClock) AfterFunc(d time.Duration, f func()) Timer { return time.AfterFunc(d, f) }
