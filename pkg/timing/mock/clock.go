package mock

import (
	"sync"
	"time"

	"github.com/butter-bot-machines/skylark/pkg/timing"
)

// Clock implements timing.Clock with controlled time flow
type Clock struct {
	mu      sync.RWMutex
	now     time.Time
	timers  []*timer
	tickers []*ticker
}

// New creates a new mock clock starting at the given time
func New(now time.Time) *Clock {
	return &Clock{
		now:     now,
		timers:  make([]*timer, 0),
		tickers: make([]*ticker, 0),
	}
}

// Now returns the current mock time
func (c *Clock) Now() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.now
}

// Sleep advances time by the given duration
func (c *Clock) Sleep(d time.Duration) {
	c.Advance(d)
}

// After returns a channel that will send the time after duration d
func (c *Clock) After(d time.Duration) <-chan time.Time {
	return c.NewTimer(d).C()
}

// NewTimer creates a new Timer that will fire after duration d
func (c *Clock) NewTimer(d time.Duration) timing.Timer {
	c.mu.Lock()
	defer c.mu.Unlock()

	t := &timer{
		c:      c,
		when:   c.now.Add(d),
		ch:     make(chan time.Time, 1),
		active: true,
	}
	c.timers = append(c.timers, t)
	return t
}

// AfterFunc waits for duration d then calls f in its own goroutine
func (c *Clock) AfterFunc(d time.Duration, f func()) timing.Timer {
	c.mu.Lock()
	defer c.mu.Unlock()

	t := &timer{
		c:      c,
		when:   c.now.Add(d),
		fn:     f,
		active: true,
	}
	c.timers = append(c.timers, t)
	return t
}

// NewTicker returns a new Ticker that will send the time on its channel
// every duration d, starting at c's next time >= now + d
func (c *Clock) NewTicker(d time.Duration) timing.Ticker {
	if d <= 0 {
		panic("non-positive interval for NewTicker")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	t := &ticker{
		c:        c,
		duration: d,
		next:     c.now.Add(d),
		ch:       make(chan time.Time, 1),
		active:   true,
	}
	c.tickers = append(c.tickers, t)
	return t
}

// Advance moves the clock forward by duration d and fires any timers/tickers
func (c *Clock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	end := c.now.Add(d)
	for c.now.Before(end) {
		next := end

		// Find the next timer/ticker to fire
		for _, t := range c.timers {
			if t.active && t.when.Before(next) {
				next = t.when
			}
		}
		for _, t := range c.tickers {
			if t.active && t.next.Before(next) {
				next = t.next
			}
		}

		c.now = next

		// Fire timers
		var newTimers []*timer
		for _, t := range c.timers {
			if !t.active {
				continue
			}
			if !t.when.After(c.now) {
				t.fire()
			} else {
				newTimers = append(newTimers, t)
			}
		}
		c.timers = newTimers

		// Fire tickers
		for _, t := range c.tickers {
			if !t.active {
				continue
			}
			for !t.next.After(c.now) {
				t.fire()
				t.next = t.next.Add(t.duration)
			}
		}
	}
}

// timer implements timing.Timer
type timer struct {
	c      *Clock
	when   time.Time
	ch     chan time.Time
	fn     func()
	active bool
	mu     sync.RWMutex
}

func (t *timer) C() <-chan time.Time {
	return t.ch
}

func (t *timer) Stop() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	active := t.active
	t.active = false
	return active
}

func (t *timer) Reset(d time.Duration) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.c.mu.Lock()
	defer t.c.mu.Unlock()

	active := t.active
	t.active = true
	t.when = t.c.now.Add(d)
	return active
}

func (t *timer) fire() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.active {
		return
	}

	if t.fn != nil {
		go t.fn()
	} else {
		select {
		case t.ch <- t.when:
		default:
		}
	}
	t.active = false
}

// ticker implements timing.Ticker
type ticker struct {
	c        *Clock
	duration time.Duration
	next     time.Time
	ch       chan time.Time
	active   bool
	mu       sync.RWMutex
}

func (t *ticker) C() <-chan time.Time {
	return t.ch
}

func (t *ticker) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.active = false
}

func (t *ticker) fire() {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.active {
		return
	}

	select {
	case t.ch <- t.next:
	default:
	}
}
