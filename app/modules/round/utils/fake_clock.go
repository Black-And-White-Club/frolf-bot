package roundutil

import "time"

// FakeClock is a fake implementation of the Clock interface.
type FakeClock struct {
	NowFn          func() time.Time
	NowUTCFn       func() time.Time
	AfterFn        func(d time.Duration) <-chan time.Time
	SleepFn        func(d time.Duration)
	ParseFn        func(layout, value string) (time.Time, error)
	LoadLocationFn func(name string) (*time.Location, error)
}

func (f *FakeClock) Now() time.Time {
	if f.NowFn != nil {
		return f.NowFn()
	}
	return time.Now()
}

func (f *FakeClock) NowUTC() time.Time {
	if f.NowUTCFn != nil {
		return f.NowUTCFn()
	}
	return time.Now().UTC()
}

func (f *FakeClock) After(d time.Duration) <-chan time.Time {
	if f.AfterFn != nil {
		return f.AfterFn(d)
	}
	return time.After(d)
}

func (f *FakeClock) Sleep(d time.Duration) {
	if f.SleepFn != nil {
		f.SleepFn(d)
	}
}

func (f *FakeClock) Parse(layout, value string) (time.Time, error) {
	if f.ParseFn != nil {
		return f.ParseFn(layout, value)
	}
	return time.Parse(layout, value)
}

func (f *FakeClock) LoadLocation(name string) (*time.Location, error) {
	if f.LoadLocationFn != nil {
		return f.LoadLocationFn(name)
	}
	return time.LoadLocation(name)
}
