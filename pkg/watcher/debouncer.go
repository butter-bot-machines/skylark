package watcher

import (
	"sync"
	"time"

	"github.com/butter-bot-machines/skylark/pkg/timing"
)

// Debouncer handles debouncing of file events
type Debouncer struct {
	mu         sync.Mutex
	delay      time.Duration
	maxDelay   time.Duration
	timers     map[string]timing.Timer
	lastEvents map[string]time.Time
	clock      timing.Clock
}

// newDebouncer creates a new debouncer
func newDebouncer(delay, maxDelay time.Duration, clock timing.Clock) *Debouncer {
	if clock == nil {
		clock = timing.New()
	}
	return &Debouncer{
		delay:      delay,
		maxDelay:   maxDelay,
		timers:     make(map[string]timing.Timer),
		lastEvents: make(map[string]time.Time),
		clock:      clock,
	}
}

// Debounce executes the callback after the debounce delay
func (d *Debouncer) Debounce(key string, callback func()) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Cancel existing timer if any
	if timer, exists := d.timers[key]; exists {
		timer.Stop()
	}

	now := d.clock.Now()
	lastEvent, exists := d.lastEvents[key]

	// If max delay exceeded, execute immediately
	if exists && now.Sub(lastEvent) >= d.maxDelay {
		delete(d.timers, key)
		delete(d.lastEvents, key)
		go callback()
		return
	}

	// Update last event time
	d.lastEvents[key] = now

	// Create new timer
	d.timers[key] = d.clock.AfterFunc(d.delay, func() {
		d.mu.Lock()
		delete(d.timers, key)
		delete(d.lastEvents, key)
		d.mu.Unlock()
		callback()
	})
}

// Stop stops all timers
func (d *Debouncer) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, timer := range d.timers {
		timer.Stop()
	}
	d.timers = make(map[string]timing.Timer)
	d.lastEvents = make(map[string]time.Time)
}
