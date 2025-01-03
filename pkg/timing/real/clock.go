package real

import (
	"time"

	"github.com/butter-bot-machines/skylark/pkg/timing"
)

// Clock implements timing.Clock using the standard time package
type Clock struct{}

// New creates a new real clock
func New() *Clock {
	return &Clock{}
}

// Now returns the current time
func (c *Clock) Now() time.Time {
	return time.Now()
}

// Sleep pauses the current goroutine for the specified duration
func (c *Clock) Sleep(d time.Duration) {
	time.Sleep(d)
}

// After waits for the duration to elapse and then sends the current time on the returned channel
func (c *Clock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

// NewTimer creates a new Timer that will send the current time on its channel after at least duration d
func (c *Clock) NewTimer(d time.Duration) timing.Timer {
	return &timer{time.NewTimer(d)}
}

// AfterFunc waits for the duration to elapse and then calls f in its own goroutine
func (c *Clock) AfterFunc(d time.Duration, f func()) timing.Timer {
	return &timer{time.AfterFunc(d, f)}
}

// NewTicker returns a new Ticker containing a channel that will send the current time on the channel after each tick
func (c *Clock) NewTicker(d time.Duration) timing.Ticker {
	return &ticker{time.NewTicker(d)}
}

// timer implements timing.Timer
type timer struct {
	*time.Timer
}

func (t *timer) C() <-chan time.Time {
	return t.Timer.C
}

// ticker implements timing.Ticker
type ticker struct {
	*time.Ticker
}

func (t *ticker) C() <-chan time.Time {
	return t.Ticker.C
}
