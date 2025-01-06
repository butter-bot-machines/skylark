package memory

import (
	"sync"
)

// Monitor implements provider.Monitor for testing
type Monitor struct {
	mu sync.RWMutex

	// Request metrics
	requests int
	failures int

	// Token metrics
	promptTokens     int
	completionTokens int
	totalTokens      int

	// Latency metrics
	totalLatency float64
	callCount    int
}

// NewMonitor creates a new memory monitor
func NewMonitor() *Monitor {
	return &Monitor{}
}

// RecordRequest records a request attempt
func (m *Monitor) RecordRequest(success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requests++
	if !success {
		m.failures++
	}
}

// RecordTokens records token usage
func (m *Monitor) RecordTokens(prompt, completion, total int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.promptTokens += prompt
	m.completionTokens += completion
	m.totalTokens += total
}

// RecordLatency records request latency
func (m *Monitor) RecordLatency(duration float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalLatency += duration
	m.callCount++
}

// Metrics returns current metrics
func (m *Monitor) Metrics() Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var avgLatency float64
	if m.callCount > 0 {
		avgLatency = m.totalLatency / float64(m.callCount)
	}

	return Metrics{
		Requests: RequestMetrics{
			Total:    m.requests,
			Failures: m.failures,
		},
		Tokens: TokenMetrics{
			Prompt:     m.promptTokens,
			Completion: m.completionTokens,
			Total:      m.totalTokens,
		},
		Latency: LatencyMetrics{
			Average:    avgLatency,
			TotalCalls: m.callCount,
		},
	}
}

// Metrics holds monitor metrics
type Metrics struct {
	Requests RequestMetrics
	Tokens   TokenMetrics
	Latency  LatencyMetrics
}

// RequestMetrics holds request-related metrics
type RequestMetrics struct {
	Total    int
	Failures int
}

// TokenMetrics holds token usage metrics
type TokenMetrics struct {
	Prompt     int
	Completion int
	Total      int
}

// LatencyMetrics holds latency metrics
type LatencyMetrics struct {
	Average    float64
	TotalCalls int
}
