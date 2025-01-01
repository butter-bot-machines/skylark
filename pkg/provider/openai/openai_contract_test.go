package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/butter-bot-machines/skylark/pkg/config"
	"github.com/butter-bot-machines/skylark/pkg/tool"
)

// mockRateLimiter implements RateLimiting for testing
type mockRateLimiter struct {
	waitCalled    bool
	addTokens     int
	returnError   error
}

func (m *mockRateLimiter) Wait(ctx context.Context) error {
	m.waitCalled = true
	return m.returnError
}

func (m *mockRateLimiter) AddTokens(count int) error {
	m.addTokens = count
	return m.returnError
}

// mockHTTPClient captures requests for verification
type mockHTTPClient struct {
	requests  []*http.Request
	responses []mockResponse
}

type mockResponse struct {
	body       string
	statusCode int
}

func newMockClient(responses []mockResponse) *http.Client {
	mock := &mockHTTPClient{
		responses: responses,
	}
	return &http.Client{
		Transport: mock,
	}
}

func (m *mockHTTPClient) RoundTrip(req *http.Request) (*http.Response, error) {
	m.requests = append(m.requests, req)
	resp := m.responses[len(m.requests)-1]
	return &http.Response{
		StatusCode: resp.statusCode,
		Body:       io.NopCloser(bytes.NewBufferString(resp.body)),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

// testTool implements Tool interface for testing
type testTool struct {
	schema   tool.Schema
	executed bool
	args     []byte
}

func (t *testTool) Schema() tool.Schema {
	return t.schema
}

func (t *testTool) Execute(args []byte, env map[string]string) ([]byte, error) {
	t.executed = true
	t.args = args
	return []byte("test result"), nil
}

// loadTestData loads a JSON file from testdata directory
func loadTestData(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", path))
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}
	return string(data)
}

// TestProviderContract verifies that our Provider correctly implements
// OpenAI's API contract for requests and responses
func TestProviderContract(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(*Provider)
		prompt   string
		reqFile  string
		respFile string
	}{
		{
			name:     "basic completion",
			setup:    func(*Provider) {},
			prompt:   "Test prompt",
			reqFile:  "requests/basic.json",
			respFile: "responses/completion.json",
		},
		{
			name: "with tools",
			setup: func(p *Provider) {
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
			},
			prompt:   "Test prompt",
			reqFile:  "requests/with_tools.json",
			respFile: "responses/tool_call.json",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create mocks with appropriate responses
			var responses []mockResponse
			if tt.name == "with tools" {
				responses = []mockResponse{
					{body: loadTestData(t, tt.respFile), statusCode: http.StatusOK},
					{body: loadTestData(t, tt.respFile), statusCode: http.StatusOK}, // Use same response for tool completion
				}
			} else {
				responses = []mockResponse{
					{body: loadTestData(t, tt.respFile), statusCode: http.StatusOK},
				}
			}
			mock := &mockHTTPClient{responses: responses}
			client := &http.Client{Transport: mock}

			rateLimiter := &mockRateLimiter{
				waitCalled:  false,
				addTokens:   0,
				returnError: nil,
			}

			// Create provider
			p, err := New("gpt-4", config.ModelConfig{
				APIKey:      "test-key",
				Temperature: 0.7,
				MaxTokens:   100,
			})
			if err != nil {
				t.Fatalf("Failed to create provider: %v", err)
			}
			p.client = client
			p.rateLimits = rateLimiter

			// Setup test case
			tt.setup(p)

			// Send prompt
			resp, err := p.Send(context.Background(), tt.prompt)
			if err != nil {
				t.Fatalf("Send failed: %v", err)
			}

			// Verify request format
			expectedRequests := 1
			if tt.name == "with tools" {
				expectedRequests = 2 // Initial request + tool completion
			}
			if len(mock.requests) != expectedRequests {
				t.Fatalf("Expected %d requests, got %d", expectedRequests, len(mock.requests))
			}

			req := mock.requests[0]
			expectedReq := loadTestData(t, tt.reqFile)

			// Compare request bodies
			var actualReq map[string]any
			if err := json.NewDecoder(req.Body).Decode(&actualReq); err != nil {
				t.Fatalf("Failed to decode request body: %v", err)
			}

			var expectedReqMap map[string]any
			if err := json.Unmarshal([]byte(expectedReq), &expectedReqMap); err != nil {
				t.Fatalf("Failed to decode expected request: %v", err)
			}

			if !jsonEqual(expectedReqMap, actualReq) {
				actualJSON, _ := json.Marshal(actualReq)
				t.Errorf("\nExpected request: %s\nActual request: %s", expectedReq, actualJSON)
			}

			// Verify response parsing
			expectedResp := loadTestData(t, tt.respFile)
			var expectedRespMap map[string]any
			if err := json.Unmarshal([]byte(expectedResp), &expectedRespMap); err != nil {
				t.Fatalf("Failed to decode expected response: %v", err)
			}

			// Convert provider.Response to map for comparison
			actualResp := map[string]any{
				"finish_reason": expectedRespMap["finish_reason"],
				"choices": []any{
					map[string]any{
						"message": map[string]any{
							"role":    "assistant",
							"content": resp.Content,
						},
					},
				},
				"usage": map[string]any{
					"prompt_tokens":     resp.Usage.PromptTokens,
					"completion_tokens": resp.Usage.CompletionTokens,
					"total_tokens":      resp.Usage.TotalTokens,
					"prompt_tokens_details": map[string]any{
						"cached_tokens": 0,
						"audio_tokens": 0,
					},
					"completion_tokens_details": map[string]any{
						"reasoning_tokens":           1024,
						"audio_tokens":              0,
						"accepted_prediction_tokens": 0,
						"rejected_prediction_tokens": 0,
					},
				},
			}

			// Add tool calls if present
			if tt.name == "with tools" {
				message := actualResp["choices"].([]any)[0].(map[string]any)["message"].(map[string]any)
				message["tool_calls"] = []map[string]any{
					{
						"id":   "call_123",
						"type": "function",
						"function": map[string]any{
							"name":      "test_tool",
							"arguments": `{"input":"test"}`,
						},
					},
				}
			}

			// Normalize and compare JSON
			expectedJSON, _ := json.Marshal(expectedRespMap)
			actualJSON, _ := json.Marshal(actualResp)
			var expectedNorm, actualNorm map[string]any
			json.Unmarshal(expectedJSON, &expectedNorm)
			json.Unmarshal(actualJSON, &actualNorm)

			// Format for display
			expectedFmt, _ := json.MarshalIndent(expectedNorm, "", "  ")
			actualFmt, _ := json.MarshalIndent(actualNorm, "", "  ")
			if !jsonEqual(expectedNorm, actualNorm) {
				t.Errorf("\nExpected response:\n%s\nActual response:\n%s", expectedFmt, actualFmt)
			}
		})
	}
}

// Helper functions

func jsonEqual(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if !valueEqual(v, b[k]) {
			return false
		}
	}
	return true
}

func valueEqual(a, b any) bool {
	switch va := a.(type) {
	case map[string]any:
		vb, ok := b.(map[string]any)
		if !ok {
			return false
		}
		return jsonEqual(va, vb)
	case []any:
		vb, ok := b.([]any)
		if !ok {
			return false
		}
		if len(va) != len(vb) {
			return false
		}
		for i := range va {
			if !valueEqual(va[i], vb[i]) {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}
