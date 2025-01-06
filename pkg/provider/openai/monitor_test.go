package openai

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"github.com/butter-bot-machines/skylark/pkg/config"
	"github.com/butter-bot-machines/skylark/pkg/provider"
	"github.com/butter-bot-machines/skylark/pkg/tool"
)

// mockMonitor implements provider.Monitor for testing
type mockMonitor struct {
	mu sync.Mutex

	requests  int
	successes int
	failures  int

	promptTokens     int
	completionTokens int
	totalTokens      int

	latencies []float64
}

func (m *mockMonitor) RecordRequest(success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requests++
	if success {
		m.successes++
	} else {
		m.failures++
	}
}

func (m *mockMonitor) RecordTokens(prompt, completion, total int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.promptTokens += prompt
	m.completionTokens += completion
	m.totalTokens += total
}

func (m *mockMonitor) RecordLatency(duration float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.latencies = append(m.latencies, duration)
}

func TestProviderMonitoring(t *testing.T) {
	t.Run("Success Metrics", func(t *testing.T) {
		// Setup mocks
		monitor := &mockMonitor{}
		client := &http.Client{
			Transport: &mockHTTPClient{
				responses: []mockResponse{
					{
						body: `{
							"choices": [{"message": {"content": "test"}}],
							"usage": {
								"prompt_tokens": 10,
								"completion_tokens": 20,
								"total_tokens": 30
							}
						}`,
						statusCode: http.StatusOK,
					},
				},
			},
		}

		// Create provider
		p, err := New("gpt-4", config.ModelConfig{
			APIKey: "test-key",
		}, Options{
			HTTPClient: client,
			Monitor:    monitor,
		})
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		// Send request with default options
		_, err = p.Send(context.Background(), "test", provider.DefaultRequestOptions)
		if err != nil {
			t.Fatalf("Send failed: %v", err)
		}

		// Verify metrics
		if monitor.requests != 1 {
			t.Errorf("Expected 1 request, got %d", monitor.requests)
		}
		if monitor.successes != 1 {
			t.Errorf("Expected 1 success, got %d", monitor.successes)
		}
		if monitor.failures != 0 {
			t.Errorf("Expected 0 failures, got %d", monitor.failures)
		}

		if monitor.promptTokens != 10 {
			t.Errorf("Expected 10 prompt tokens, got %d", monitor.promptTokens)
		}
		if monitor.completionTokens != 20 {
			t.Errorf("Expected 20 completion tokens, got %d", monitor.completionTokens)
		}
		if monitor.totalTokens != 30 {
			t.Errorf("Expected 30 total tokens, got %d", monitor.totalTokens)
		}

		if len(monitor.latencies) != 1 {
			t.Errorf("Expected 1 latency record, got %d", len(monitor.latencies))
		}
	})

	t.Run("Error Metrics", func(t *testing.T) {
		// Setup mocks
		monitor := &mockMonitor{}
		client := &http.Client{
			Transport: &mockHTTPClient{
				responses: []mockResponse{
					{
						body: `{
							"error": {
								"message": "test error",
								"type": "test",
								"code": "rate_limit_exceeded"
							}
						}`,
						statusCode: http.StatusTooManyRequests,
					},
				},
			},
		}

		// Create provider
		p, err := New("gpt-4", config.ModelConfig{
			APIKey: "test-key",
		}, Options{
			HTTPClient: client,
			Monitor:    monitor,
		})
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		// Send request (expect error) with default options
		_, err = p.Send(context.Background(), "test", provider.DefaultRequestOptions)
		if err == nil {
			t.Fatal("Expected error but got none")
		}

		// Verify metrics
		if monitor.requests != 1 {
			t.Errorf("Expected 1 request, got %d", monitor.requests)
		}
		if monitor.successes != 0 {
			t.Errorf("Expected 0 successes, got %d", monitor.successes)
		}
		if monitor.failures != 1 {
			t.Errorf("Expected 1 failure, got %d", monitor.failures)
		}

		if len(monitor.latencies) != 1 {
			t.Errorf("Expected 1 latency record, got %d", len(monitor.latencies))
		}
	})

	t.Run("Tool Call Metrics", func(t *testing.T) {
		// Setup mocks
		monitor := &mockMonitor{}
		client := &http.Client{
			Transport: &mockHTTPClient{
				responses: []mockResponse{
					{
						// Initial response with tool call
						body: `{
							"choices": [{
								"message": {
									"content": "test",
									"tool_calls": [{
										"id": "call_123",
										"function": {
											"name": "test_tool",
											"arguments": "{\"input\":\"test\"}"
										}
									}]
								}
							}],
							"usage": {
								"prompt_tokens": 10,
								"completion_tokens": 20,
								"total_tokens": 30
							}
						}`,
						statusCode: http.StatusOK,
					},
					{
						// Final response after tool call
						body: `{
							"choices": [{"message": {"content": "test result"}}],
							"usage": {
								"prompt_tokens": 15,
								"completion_tokens": 25,
								"total_tokens": 40
							}
						}`,
						statusCode: http.StatusOK,
					},
				},
			},
		}

		// Create provider
		p, err := New("gpt-4", config.ModelConfig{
			APIKey: "test-key",
		}, Options{
			HTTPClient: client,
			Monitor:    monitor,
		})
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		// Register test tool
		schema := tool.Schema{}
		schema.Schema.Description = "A test tool"
		schema.Schema.Parameters = map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"input": map[string]interface{}{
					"type": "string",
				},
			},
		}
		p.RegisterTool("test_tool", &testTool{schema: schema})

		// Send request with default options
		_, err = p.Send(context.Background(), "test", provider.DefaultRequestOptions)
		if err != nil {
			t.Fatalf("Send failed: %v", err)
		}

		// Verify metrics
		if monitor.requests != 2 { // Initial + tool completion
			t.Errorf("Expected 2 requests, got %d", monitor.requests)
		}
		if monitor.successes != 2 {
			t.Errorf("Expected 2 successes, got %d", monitor.successes)
		}
		if monitor.failures != 0 {
			t.Errorf("Expected 0 failures, got %d", monitor.failures)
		}

		expectedPrompt := 25     // 10 + 15
		expectedCompletion := 45 // 20 + 25
		expectedTotal := 70      // 30 + 40

		if monitor.promptTokens != expectedPrompt {
			t.Errorf("Expected %d prompt tokens, got %d", expectedPrompt, monitor.promptTokens)
		}
		if monitor.completionTokens != expectedCompletion {
			t.Errorf("Expected %d completion tokens, got %d", expectedCompletion, monitor.completionTokens)
		}
		if monitor.totalTokens != expectedTotal {
			t.Errorf("Expected %d total tokens, got %d", expectedTotal, monitor.totalTokens)
		}

		if len(monitor.latencies) != 2 {
			t.Errorf("Expected 2 latency records, got %d", len(monitor.latencies))
		}
	})
}
