package timing

import "time"

// Clock defines the interface for time operations
type Clock interface {
	// Time operations
	Now() time.Time
	Sleep(d time.Duration)
	After(d time.Duration) <-chan time.Time

	// Timer operations
	NewTimer(d time.Duration) Timer
	AfterFunc(d time.Duration, f func()) Timer

	// Ticker operations
	NewTicker(d time.Duration) Ticker
}

// Timer defines the interface for timer operations
type Timer interface {
	// C returns the timer's channel
	C() <-chan time.Time
	// Stop prevents the timer from firing
	Stop() bool
	// Reset changes the timer's duration
	Reset(d time.Duration) bool
}

// Ticker defines the interface for ticker operations
type Ticker interface {
	// C returns the ticker's channel
	C() <-chan time.Time
	// Stop prevents the ticker from firing
	Stop()
}

// Error types for timing operations
var (
	ErrInvalidDuration = Error{"invalid duration"}
	ErrTimerStopped   = Error{"timer already stopped"}
	ErrTickerStopped  = Error{"ticker already stopped"}
)

// Error represents a timing error
type Error struct {
	Message string
}

func (e Error) Error() string {
	return e.Message
}
