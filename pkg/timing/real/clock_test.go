package real

import (
	"testing"
	"time"
)

func TestClock_BasicOperations(t *testing.T) {
	clock := New()

	// Test Now
	t.Run("Now", func(t *testing.T) {
		t1 := time.Now()
		c := clock.Now()
		t2 := time.Now()

		if c.Before(t1) || c.After(t2) {
			t.Errorf("Clock time %v not between %v and %v", c, t1, t2)
		}
	})

	// Test Sleep
	t.Run("Sleep", func(t *testing.T) {
		d := 10 * time.Millisecond
		start := time.Now()
		clock.Sleep(d)
		elapsed := time.Since(start)

		if elapsed < d {
			t.Errorf("Sleep returned too early: got %v, want >= %v", elapsed, d)
		}
	})

	// Test After
	t.Run("After", func(t *testing.T) {
		d := 10 * time.Millisecond
		start := time.Now()
		<-clock.After(d)
		elapsed := time.Since(start)

		if elapsed < d {
			t.Errorf("After returned too early: got %v, want >= %v", elapsed, d)
		}
	})
}

func TestClock_TimerOperations(t *testing.T) {
	clock := New()

	// Test NewTimer
	t.Run("NewTimer", func(t *testing.T) {
		d := 10 * time.Millisecond
		timer := clock.NewTimer(d)
		start := time.Now()
		<-timer.C()
		elapsed := time.Since(start)

		if elapsed < d {
			t.Errorf("Timer fired too early: got %v, want >= %v", elapsed, d)
		}
	})

	// Test AfterFunc
	t.Run("AfterFunc", func(t *testing.T) {
		d := 10 * time.Millisecond
		done := make(chan bool)
		start := time.Now()

		timer := clock.AfterFunc(d, func() {
			done <- true
		})

		<-done
		elapsed := time.Since(start)

		if elapsed < d {
			t.Errorf("AfterFunc fired too early: got %v, want >= %v", elapsed, d)
		}

		if timer.Stop() {
			t.Error("Stop returned true after timer fired")
		}
	})

	// Test Stop and Reset
	t.Run("Stop and Reset", func(t *testing.T) {
		timer := clock.NewTimer(time.Hour)
		if !timer.Stop() {
			t.Error("Stop returned false for active timer")
		}

		if timer.Stop() {
			t.Error("Stop returned true for already stopped timer")
		}

		d := 10 * time.Millisecond
		start := time.Now()
		if !timer.Reset(d) {
			t.Error("Reset returned false")
		}

		<-timer.C()
		elapsed := time.Since(start)

		if elapsed < d {
			t.Errorf("Timer fired too early after reset: got %v, want >= %v", elapsed, d)
		}
	})
}

func TestClock_TickerOperations(t *testing.T) {
	clock := New()

	// Test NewTicker
	t.Run("NewTicker", func(t *testing.T) {
		d := 10 * time.Millisecond
		ticker := clock.NewTicker(d)
		start := time.Now()

		// Wait for 3 ticks
		for i := 0; i < 3; i++ {
			<-ticker.C()
		}

		elapsed := time.Since(start)
		expected := d * 3

		if elapsed < expected {
			t.Errorf("Ticker ticked too fast: got %v, want >= %v", elapsed, expected)
		}

		ticker.Stop()
	})

	// Test invalid duration
	t.Run("Invalid Duration", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("NewTicker with zero duration didn't panic")
			}
		}()
		clock.NewTicker(0)
	})
}

func TestClock_ErrorCases(t *testing.T) {
	clock := New()

	// Test negative duration
	t.Run("Negative Duration Timer", func(t *testing.T) {
		timer := clock.NewTimer(-time.Hour)
		select {
		case <-timer.C():
		case <-time.After(10 * time.Millisecond):
			t.Error("Timer with negative duration didn't fire immediately")
		}
	})

	// Test zero duration
	t.Run("Zero Duration Timer", func(t *testing.T) {
		timer := clock.NewTimer(0)
		select {
		case <-timer.C():
		case <-time.After(10 * time.Millisecond):
			t.Error("Timer with zero duration didn't fire immediately")
		}
	})

	// Test negative duration AfterFunc
	t.Run("Negative Duration AfterFunc", func(t *testing.T) {
		done := make(chan bool)
		clock.AfterFunc(-time.Hour, func() {
			done <- true
		})
		select {
		case <-done:
		case <-time.After(10 * time.Millisecond):
			t.Error("AfterFunc with negative duration didn't fire immediately")
		}
	})
}
