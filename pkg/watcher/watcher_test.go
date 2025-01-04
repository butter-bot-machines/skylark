package watcher

import (
	"testing"
	"time"

	"github.com/butter-bot-machines/skylark/pkg/timing"
)

func TestDebouncer(t *testing.T) {
	tests := []struct {
		name     string
		delay    time.Duration
		maxDelay time.Duration
		events   []time.Duration // Delays between events
		want     int            // Expected number of callbacks
	}{
		{
			name:     "single event",
			delay:    100 * time.Millisecond,
			maxDelay: time.Second,
			events:   []time.Duration{0},
			want:     1,
		},
		{
			name:     "rapid events",
			delay:    100 * time.Millisecond,
			maxDelay: time.Second,
			events:   []time.Duration{0, 50 * time.Millisecond, 50 * time.Millisecond},
			want:     1,
		},
		{
			name:     "spaced events",
			delay:    100 * time.Millisecond,
			maxDelay: time.Second,
			events:   []time.Duration{0, 200 * time.Millisecond, 200 * time.Millisecond},
			want:     3,
		},
		{
			name:     "max delay exceeded",
			delay:    100 * time.Millisecond,
			maxDelay: 500 * time.Millisecond,
			events:   []time.Duration{0, 600 * time.Millisecond},
			want:     2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var count int
			mock := timing.NewMock()
			d := newDebouncer(tt.delay, tt.maxDelay, mock)
			defer d.Stop()

			// Send events with specified delays
			for _, delay := range tt.events {
				mock.Add(delay)
				d.Debounce("test", func() {
					count++
				})
			}

			// Wait for all events to process
			mock.Add(tt.delay * 2)

			if count != tt.want {
				t.Errorf("got %d callbacks, want %d", count, tt.want)
			}
		})
	}
}

func TestDebouncerStop(t *testing.T) {
	mock := timing.NewMock()
	d := newDebouncer(100*time.Millisecond, time.Second, mock)

	// Queue some events
	for i := 0; i < 5; i++ {
		d.Debounce("test", func() {
			t.Error("callback should not be called after stop")
		})
	}

	// Stop immediately
	d.Stop()

	// Advance time to ensure no callbacks are executed
	mock.Add(200 * time.Millisecond)
}

func TestDebouncerConcurrent(t *testing.T) {
	mock := timing.NewMock()
	d := newDebouncer(50*time.Millisecond, time.Second, mock)
	defer d.Stop()

	// Send events concurrently
	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			d.Debounce("test", func() {})
			mock.Add(time.Millisecond)
		}
		close(done)
	}()

	select {
	case <-done:
		// Test completed successfully
	case <-mock.After(time.Second):
		t.Fatal("test timed out")
	}
}
