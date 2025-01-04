package memory

import (
	"testing"
)

func TestMonitor(t *testing.T) {
	t.Run("Request Metrics", func(t *testing.T) {
		m := NewMonitor()

		// Record some requests
		m.RecordRequest(true)  // Success
		m.RecordRequest(false) // Failure
		m.RecordRequest(true)  // Success

		metrics := m.Metrics()
		if metrics.Requests.Total != 3 {
			t.Errorf("expected 3 total requests, got %d", metrics.Requests.Total)
		}
		if metrics.Requests.Failures != 1 {
			t.Errorf("expected 1 failure, got %d", metrics.Requests.Failures)
		}
	})

	t.Run("Token Metrics", func(t *testing.T) {
		m := NewMonitor()

		// Record token usage
		m.RecordTokens(10, 20, 30) // First request
		m.RecordTokens(5, 15, 20)  // Second request

		metrics := m.Metrics()
		if metrics.Tokens.Prompt != 15 {
			t.Errorf("expected 15 prompt tokens, got %d", metrics.Tokens.Prompt)
		}
		if metrics.Tokens.Completion != 35 {
			t.Errorf("expected 35 completion tokens, got %d", metrics.Tokens.Completion)
		}
		if metrics.Tokens.Total != 50 {
			t.Errorf("expected 50 total tokens, got %d", metrics.Tokens.Total)
		}
	})

	t.Run("Latency Metrics", func(t *testing.T) {
		m := NewMonitor()

		// Record latencies
		m.RecordLatency(100.0) // 100ms
		m.RecordLatency(200.0) // 200ms
		m.RecordLatency(300.0) // 300ms

		metrics := m.Metrics()
		if metrics.Latency.TotalCalls != 3 {
			t.Errorf("expected 3 calls, got %d", metrics.Latency.TotalCalls)
		}

		expectedAvg := 200.0 // (100 + 200 + 300) / 3
		if metrics.Latency.Average != expectedAvg {
			t.Errorf("expected average latency %.2f, got %.2f", expectedAvg, metrics.Latency.Average)
		}
	})

	t.Run("Concurrent Access", func(t *testing.T) {
		m := NewMonitor()

		// Run concurrent operations
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func() {
				m.RecordRequest(true)
				m.RecordTokens(10, 20, 30)
				m.RecordLatency(100.0)
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		metrics := m.Metrics()
		if metrics.Requests.Total != 10 {
			t.Errorf("expected 10 requests, got %d", metrics.Requests.Total)
		}
		if metrics.Tokens.Total != 300 {
			t.Errorf("expected 300 total tokens, got %d", metrics.Tokens.Total)
		}
		if metrics.Latency.TotalCalls != 10 {
			t.Errorf("expected 10 latency records, got %d", metrics.Latency.TotalCalls)
		}
	})

	t.Run("Zero Values", func(t *testing.T) {
		m := NewMonitor()
		metrics := m.Metrics()

		// Check zero values
		if metrics.Requests.Total != 0 {
			t.Errorf("expected 0 requests, got %d", metrics.Requests.Total)
		}
		if metrics.Tokens.Total != 0 {
			t.Errorf("expected 0 tokens, got %d", metrics.Tokens.Total)
		}
		if metrics.Latency.Average != 0 {
			t.Errorf("expected 0 average latency, got %.2f", metrics.Latency.Average)
		}
	})
}
