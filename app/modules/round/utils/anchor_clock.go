package roundutil

import "time"

// AnchorClock is a Clock implementation whose Now/NowUTC always return the
// provided anchor time. Useful for parsing relative user input deterministically
// even if the message is processed later (e.g. queue delay / retries).
type AnchorClock struct {
	anchor time.Time
}

// NewAnchorClock creates a new AnchorClock. If t is the zero value, the current
// real UTC time is used.
func NewAnchorClock(t time.Time) AnchorClock {
	if t.IsZero() {
		return AnchorClock{anchor: time.Now().UTC()}
	}
	return AnchorClock{anchor: t.UTC()}
}

func (c AnchorClock) Now() time.Time    { return c.anchor }
func (c AnchorClock) NowUTC() time.Time { return c.anchor.UTC() }

// The remaining methods delegate to the real clock since anchoring is only
// relevant for "Now" semantics during parsing & validation; timers/sleeping
// should use real passage of time.
func (c AnchorClock) After(d time.Duration) <-chan time.Time        { return time.After(d) }
func (c AnchorClock) Sleep(d time.Duration)                         { time.Sleep(d) }
func (c AnchorClock) Parse(layout, value string) (time.Time, error) { return time.Parse(layout, value) }
func (c AnchorClock) LoadLocation(name string) (*time.Location, error) {
	return time.LoadLocation(name)
}
