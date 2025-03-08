package roundutil

import "time"

// Clock is an interface for time-related operations.
type Clock interface {
	Now() time.Time
	NowUTC() time.Time
	After(d time.Duration) <-chan time.Time
	Sleep(d time.Duration)
	Parse(layout, value string) (time.Time, error)
	LoadLocation(name string) (*time.Location, error)
}

// RealClock is a real implementation of the Clock interface.
type RealClock struct{}

// Now returns the current local time.
func (RealClock) Now() time.Time {
	return time.Now()
}

// NowUTC returns the current UTC time.
func (RealClock) NowUTC() time.Time {
	return time.Now().UTC()
}

// After returns a channel that receives the current time after the specified duration.
func (RealClock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

// Sleep pauses the current goroutine for the specified duration.
func (RealClock) Sleep(d time.Duration) {
	time.Sleep(d)
}

// Parse parses a formatted string and returns the time value.
func (RealClock) Parse(layout, value string) (time.Time, error) {
	return time.Parse(layout, value)
}

func (RealClock) LoadLocation(name string) (*time.Location, error) {
	return time.LoadLocation(name)
}
