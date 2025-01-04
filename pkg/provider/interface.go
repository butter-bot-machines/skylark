// Package provider defines interfaces for AI model providers
package provider

import (
	"net/http"
)

// HTTPClient abstracts HTTP operations for testing
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
	CloseIdleConnections()
}

// Monitor tracks provider metrics
type Monitor interface {
	// RecordRequest records a request attempt
	RecordRequest(success bool)
	// RecordTokens records token usage
	RecordTokens(prompt, completion, total int)
	// RecordLatency records request latency
	RecordLatency(duration float64)
}
