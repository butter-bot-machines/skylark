package watcher

import (
	"sync"
	"time"
)

// Debouncer handles debouncing of file events
type Debouncer struct {
	mu         sync.Mutex
	delay      time.Duration
	maxDelay   time.Duration
	timers     map[string]*time.Timer
	lastEvents map[string]time.Time
}

// newDebouncer creates a new debouncer
func newDebouncer(delay, maxDelay time.Duration) *Debouncer {
	return &Debouncer{
		delay:      delay,
		maxDelay:   maxDelay,
		timers:     make(map[string]*time.Timer),
		lastEvents: make(map[string]time.Time),
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

	now := time.Now()
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
	d.timers[key] = time.AfterFunc(d.delay, func() {
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
	d.timers = make(map[string]*time.Timer)
	d.lastEvents = make(map[string]time.Time)
}
