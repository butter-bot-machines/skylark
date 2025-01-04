package timing

import (
	"testing"
	"time"
)

func TestClock_BasicOperations(t *testing.T) {
	t.Run("Now", func(t *testing.T) {
		mock := NewMock()
		now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		mock.Set(now)
		
		if got := mock.Now(); !got.Equal(now) {
			t.Errorf("Now() = %v, want %v", got, now)
		}
	})

	t.Run("After", func(t *testing.T) {
		mock := NewMock()
		now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		mock.Set(now)

		ch := mock.After(time.Second)
		select {
		case <-ch:
			t.Error("After() should not trigger before time advance")
		default:
			// Expected
		}

		mock.Add(time.Second)
		select {
		case got := <-ch:
			if !got.Equal(now.Add(time.Second)) {
				t.Errorf("After() = %v, want %v", got, now.Add(time.Second))
			}
		default:
			t.Error("After() should trigger after time advance")
		}
	})
}

func TestClock_TimerOperations(t *testing.T) {
	t.Run("NewTimer", func(t *testing.T) {
		mock := NewMock()
		now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		mock.Set(now)

		timer := mock.NewTimer(time.Second)
		select {
		case <-timer.C():
			t.Error("Timer should not trigger before time advance")
		default:
			// Expected
		}

		mock.Add(time.Second)
		select {
		case got := <-timer.C():
			if !got.Equal(now.Add(time.Second)) {
				t.Errorf("Timer = %v, want %v", got, now.Add(time.Second))
			}
		default:
			t.Error("Timer should trigger after time advance")
		}
	})

	t.Run("Stop and Reset", func(t *testing.T) {
		mock := NewMock()
		timer := mock.NewTimer(time.Second)

		// Stop before firing
		if !timer.Stop() {
			t.Error("Stop() should return true for active timer")
		}

		mock.Add(time.Second)
		select {
		case <-timer.C():
			t.Error("Timer should not trigger after being stopped")
		default:
			// Expected
		}

		// Reset and verify it works
		timer.Reset(time.Second)
		mock.Add(time.Second)
		select {
		case <-timer.C():
			// Expected
		default:
			t.Error("Timer should trigger after reset")
		}
	})
}

func TestClock_TickerOperations(t *testing.T) {
	t.Run("NewTicker", func(t *testing.T) {
		mock := NewMock()
		now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		mock.Set(now)

		ticker := mock.NewTicker(time.Second)
		defer ticker.Stop()

		// Should not tick immediately
		select {
		case <-ticker.C():
			t.Error("Ticker should not trigger before time advance")
		default:
			// Expected
		}

		// Should tick after duration
		mock.Add(time.Second)
		select {
		case got := <-ticker.C():
			if !got.Equal(now.Add(time.Second)) {
				t.Errorf("Ticker = %v, want %v", got, now.Add(time.Second))
			}
		default:
			t.Error("Ticker should trigger after time advance")
		}

		// Should tick again
		mock.Add(time.Second)
		select {
		case got := <-ticker.C():
			if !got.Equal(now.Add(2 * time.Second)) {
				t.Errorf("Ticker = %v, want %v", got, now.Add(2*time.Second))
			}
		default:
			t.Error("Ticker should trigger after second advance")
		}
	})

	t.Run("Stop", func(t *testing.T) {
		mock := NewMock()
		ticker := mock.NewTicker(time.Second)
		ticker.Stop()

		mock.Add(time.Second)
		select {
		case <-ticker.C():
			t.Error("Ticker should not trigger after being stopped")
		default:
			// Expected
		}
	})
}

func TestClock_AfterFunc(t *testing.T) {
	mock := NewMock()
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	mock.Set(now)

	called := false
	timer := mock.AfterFunc(time.Second, func() {
		called = true
	})

	if called {
		t.Error("AfterFunc should not trigger before time advance")
	}

	mock.Add(time.Second)
	// Give the goroutine a chance to run
	time.Sleep(10 * time.Millisecond)
	if !called {
		t.Error("AfterFunc should trigger after time advance")
	}

	// Stop should prevent future calls
	called = false
	timer.Reset(time.Second)
	timer.Stop()
	mock.Add(time.Second)
	time.Sleep(10 * time.Millisecond)
	if called {
		t.Error("AfterFunc should not trigger after being stopped")
	}
}
