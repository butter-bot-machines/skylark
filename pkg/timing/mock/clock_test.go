package mock

import (
	"sync"
	"testing"
	"time"
)

func TestClock_BasicOperations(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := New(start)

	// Test Now
	t.Run("Now", func(t *testing.T) {
		if now := clock.Now(); !now.Equal(start) {
			t.Errorf("Got time %v, want %v", now, start)
		}
	})

	// Test Sleep
	t.Run("Sleep", func(t *testing.T) {
		before := clock.Now()
		clock.Sleep(time.Hour)
		after := clock.Now()

		if expected := before.Add(time.Hour); !after.Equal(expected) {
			t.Errorf("Got time %v, want %v", after, expected)
		}
	})

	// Test After
	t.Run("After", func(t *testing.T) {
		done := make(chan bool)
		go func() {
			<-clock.After(time.Hour)
			done <- true
		}()

		clock.Advance(30 * time.Minute)
		select {
		case <-done:
			t.Error("Timer fired too early")
		default:
		}

		clock.Advance(31 * time.Minute)
		select {
		case <-done:
		case <-time.After(10 * time.Millisecond):
			t.Error("Timer didn't fire")
		}
	})
}

func TestClock_TimerOperations(t *testing.T) {
	clock := New(time.Now())

	// Test NewTimer
	t.Run("NewTimer", func(t *testing.T) {
		timer := clock.NewTimer(time.Hour)
		fired := false

		go func() {
			<-timer.C()
			fired = true
		}()

		clock.Advance(30 * time.Minute)
		if fired {
			t.Error("Timer fired too early")
		}

		clock.Advance(31 * time.Minute)
		if !fired {
			t.Error("Timer didn't fire")
		}
	})

	// Test AfterFunc
	t.Run("AfterFunc", func(t *testing.T) {
		done := make(chan bool)
		timer := clock.AfterFunc(time.Hour, func() {
			done <- true
		})

		clock.Advance(30 * time.Minute)
		select {
		case <-done:
			t.Error("Timer fired too early")
		default:
		}

		clock.Advance(31 * time.Minute)
		select {
		case <-done:
		case <-time.After(10 * time.Millisecond):
			t.Error("Timer didn't fire")
		}

		if timer.Stop() {
			t.Error("Timer still active after firing")
		}
	})

	// Test Stop and Reset
	t.Run("Stop and Reset", func(t *testing.T) {
		timer := clock.NewTimer(time.Hour)
		fired := false

		go func() {
			<-timer.C()
			fired = true
		}()

		if !timer.Stop() {
			t.Error("Stop returned false for active timer")
		}

		clock.Advance(2 * time.Hour)
		if fired {
			t.Error("Timer fired after being stopped")
		}

		if !timer.Reset(time.Hour) {
			t.Error("Reset returned false")
		}

		clock.Advance(2 * time.Hour)
		if !fired {
			t.Error("Timer didn't fire after reset")
		}
	})
}

func TestClock_TickerOperations(t *testing.T) {
	clock := New(time.Now())

	// Test NewTicker
	t.Run("NewTicker", func(t *testing.T) {
		ticker := clock.NewTicker(time.Hour)
		ticks := 0

		go func() {
			for range ticker.C() {
				ticks++
			}
		}()

		// Advance by 3.5 hours, should get 3 ticks
		clock.Advance(3*time.Hour + 30*time.Minute)
		if ticks != 3 {
			t.Errorf("Got %d ticks, want 3", ticks)
		}

		ticker.Stop()
		clock.Advance(2 * time.Hour)
		if ticks != 3 {
			t.Errorf("Got %d ticks after stop, want 3", ticks)
		}
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

func TestClock_Concurrency(t *testing.T) {
	clock := New(time.Now())
	var wg sync.WaitGroup
	workers := 10
	iterations := 100

	// Concurrent timers
	t.Run("Concurrent Timers", func(t *testing.T) {
		wg.Add(workers)
		for i := 0; i < workers; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					timer := clock.NewTimer(time.Duration(id+1) * time.Second)
					<-timer.C()
				}
			}(i)
		}

		// Advance clock while timers are running
		for i := 0; i < iterations; i++ {
			clock.Advance(time.Duration(workers+1) * time.Second)
		}
		wg.Wait()
	})

	// Concurrent tickers
	t.Run("Concurrent Tickers", func(t *testing.T) {
		wg.Add(workers)
		tickers := make([]time.Duration, workers)
		for i := 0; i < workers; i++ {
			tickers[i] = time.Duration(i+1) * time.Second
		}

		for i := 0; i < workers; i++ {
			go func(id int) {
				defer wg.Done()
				ticker := clock.NewTicker(tickers[id])
				for j := 0; j < iterations; j++ {
					<-ticker.C()
				}
				ticker.Stop()
			}(i)
		}

		// Advance clock while tickers are running
		maxDuration := time.Duration(workers+1) * time.Second
		for i := 0; i < iterations; i++ {
			clock.Advance(maxDuration)
		}
		wg.Wait()
	})
}

func TestClock_EdgeCases(t *testing.T) {
	clock := New(time.Now())

	// Test multiple stops
	t.Run("Multiple Stops", func(t *testing.T) {
		timer := clock.NewTimer(time.Hour)
		if !timer.Stop() {
			t.Error("First stop returned false")
		}
		if timer.Stop() {
			t.Error("Second stop returned true")
		}
	})

	// Test multiple resets
	t.Run("Multiple Resets", func(t *testing.T) {
		timer := clock.NewTimer(time.Hour)
		if !timer.Reset(time.Minute) {
			t.Error("First reset returned false")
		}
		if !timer.Reset(time.Second) {
			t.Error("Second reset returned false")
		}
	})

	// Test zero advance
	t.Run("Zero Advance", func(t *testing.T) {
		before := clock.Now()
		clock.Advance(0)
		after := clock.Now()
		if !before.Equal(after) {
			t.Error("Time changed with zero advance")
		}
	})

	// Test negative advance
	t.Run("Negative Advance", func(t *testing.T) {
		before := clock.Now()
		clock.Advance(-time.Hour)
		after := clock.Now()
		if !before.Equal(after) {
			t.Error("Time changed with negative advance")
		}
	})
}
