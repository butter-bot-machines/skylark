package concrete

import (
	"sync"
	"time"

	"github.com/butter-bot-machines/skylark/pkg/timing"
	"github.com/butter-bot-machines/skylark/pkg/watcher"
)

// debouncerImpl implements watcher.Debouncer
type debouncerImpl struct {
	delay     time.Duration
	maxDelay  time.Duration
	timers    map[string]*timerCtx
	mu        sync.Mutex
	done      chan struct{}
	clock     timing.Clock
}

type timerCtx struct {
	timer     timing.Timer
	lastEvent time.Time
}

// newDebouncer creates a new debouncer
func newDebouncer(delay, maxDelay time.Duration, clock timing.Clock) watcher.Debouncer {
	if clock == nil {
		clock = timing.New()
	}
	return &debouncerImpl{
		delay:    delay,
		maxDelay: maxDelay,
		timers:   make(map[string]*timerCtx),
		done:     make(chan struct{}),
		clock:    clock,
	}
}

// Debounce delays execution of fn until events settle
func (d *debouncerImpl) Debounce(key string, fn func()) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if already stopped
	select {
	case <-d.done:
		return
	default:
	}

	// Get or create timer context
	ctx, ok := d.timers[key]
	if !ok {
		ctx = &timerCtx{}
		d.timers[key] = ctx
	}

	// Stop existing timer
	if ctx.timer != nil {
		ctx.timer.Stop()
	}

	now := d.clock.Now()
	ctx.lastEvent = now

	// Create new timer
	ctx.timer = d.clock.AfterFunc(d.delay, func() {
		d.mu.Lock()
		defer d.mu.Unlock()

		// Check if already stopped
		select {
		case <-d.done:
			return
		default:
		}

		// Check if max delay exceeded
		if d.clock.Now().Sub(ctx.lastEvent) >= d.maxDelay {
			delete(d.timers, key)
			go fn()
			return
		}

		// If not exceeded, check if events have settled
		if d.clock.Now().Sub(ctx.lastEvent) >= d.delay {
			delete(d.timers, key)
			go fn()
		}
	})
}

// Stop stops the debouncer
func (d *debouncerImpl) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	select {
	case <-d.done:
		return
	default:
		close(d.done)
	}

	// Stop all timers
	for _, ctx := range d.timers {
		if ctx.timer != nil {
			ctx.timer.Stop()
		}
	}
	d.timers = nil
}
