package timing

import (
	"time"

	"github.com/benbjohnson/clock"
)

// Clock represents an interface to time-related functionality.
// It wraps benbjohnson/clock.Clock to provide mockable time operations.
type Clock interface {
	// Now returns the current time
	Now() time.Time

	// After waits for the duration to elapse and then sends the current time on the returned channel
	After(d time.Duration) <-chan time.Time

	// Sleep pauses the current goroutine for at least the duration d
	Sleep(d time.Duration)

	// NewTimer creates a new Timer that will send the current time on its channel after at least duration d
	NewTimer(d time.Duration) Timer

	// AfterFunc waits for the duration to elapse and then calls f in its own goroutine
	AfterFunc(d time.Duration, f func()) Timer

	// NewTicker returns a new Ticker containing a channel that will send the current time on the channel after each tick
	NewTicker(d time.Duration) Ticker
}

// MockClock extends Clock with methods to control time in tests
type MockClock interface {
	Clock

	// Add moves the mock clock forward by the specified duration
	Add(d time.Duration)

	// Set sets the mock clock to a specific time
	Set(t time.Time)
}

// Timer represents a single event
type Timer interface {
	// C returns the channel on which the time will be sent
	C() <-chan time.Time

	// Stop prevents the Timer from firing
	Stop() bool

	// Reset changes the timer to expire after duration d
	Reset(d time.Duration) bool
}

// Ticker represents a periodic event
type Ticker interface {
	// C returns the channel on which ticks are delivered
	C() <-chan time.Time

	// Stop turns off a ticker
	Stop()
}

// clockAdapter adapts benbjohnson/clock.Clock to our Clock interface
type clockAdapter struct {
	c clock.Clock
}

func (c *clockAdapter) Now() time.Time                           { return c.c.Now() }
func (c *clockAdapter) After(d time.Duration) <-chan time.Time   { return c.c.After(d) }
func (c *clockAdapter) Sleep(d time.Duration)                    { c.c.Sleep(d) }
func (c *clockAdapter) NewTimer(d time.Duration) Timer {
	t := c.c.Timer(d)
	return &timerAdapter{t}
}
func (c *clockAdapter) AfterFunc(d time.Duration, f func()) Timer {
	t := c.c.AfterFunc(d, f)
	return &timerAdapter{t}
}
func (c *clockAdapter) NewTicker(d time.Duration) Ticker {
	t := c.c.Ticker(d)
	return &tickerAdapter{t}
}

// mockClockAdapter adapts *clock.Mock to our MockClock interface
type mockClockAdapter struct {
	clockAdapter
	mock *clock.Mock
}

func (m *mockClockAdapter) Add(d time.Duration) { m.mock.Add(d) }
func (m *mockClockAdapter) Set(t time.Time)     { m.mock.Set(t) }

// timerAdapter adapts *clock.Timer to our Timer interface
type timerAdapter struct {
	t *clock.Timer
}

func (t *timerAdapter) C() <-chan time.Time { return t.t.C }
func (t *timerAdapter) Stop() bool          { return t.t.Stop() }
func (t *timerAdapter) Reset(d time.Duration) bool { return t.t.Reset(d) }

// tickerAdapter adapts *clock.Ticker to our Ticker interface
type tickerAdapter struct {
	t *clock.Ticker
}

func (t *tickerAdapter) C() <-chan time.Time { return t.t.C }
func (t *tickerAdapter) Stop()               { t.t.Stop() }

// New returns a new Clock that uses the system clock
func New() Clock {
	return &clockAdapter{clock.New()}
}

// NewMock returns a new MockClock that can be moved forward in time programmatically
func NewMock() MockClock {
	mock := clock.NewMock()
	return &mockClockAdapter{
		clockAdapter: clockAdapter{mock},
		mock:        mock,
	}
}
