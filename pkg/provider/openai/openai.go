package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/butter-bot-machines/skylark/pkg/config"
	"github.com/butter-bot-machines/skylark/pkg/provider"
	"github.com/butter-bot-machines/skylark/pkg/tool"
)

// RateLimiting defines the interface for rate limiting requests
type RateLimiting interface {
	// Wait blocks until a request can be made
	Wait(ctx context.Context) error
	// AddTokens records token usage and checks limits
	AddTokens(count int) error
}

// Tool defines the interface for tools used by the OpenAI provider
type Tool interface {
	// Schema returns the tool's schema for function parameters
	Schema() tool.Schema
	// Execute runs the tool with given args and env
	Execute(args []byte, env map[string]string) ([]byte, error)
}

const apiTimeout = 30 * time.Second

var apiURL = "https://api.openai.com/v1/chat/completions"

// Response types for parsing OpenAI API responses
type Response struct {
	Choices []struct {
		Message struct {
			Content   string `json:"content"`
			ToolCalls []struct {
				ID       string `json:"id"`
				Function struct {
					Name      string          `json:"name"`
					Arguments json.RawMessage `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls,omitempty"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// Options configures the OpenAI provider
type Options struct {
	// HTTPClient for making requests (optional)
	HTTPClient provider.HTTPClient
	// RateLimiter for controlling request rates (optional)
	RateLimiter RateLimiting
	// Monitor for tracking metrics (optional)
	Monitor provider.Monitor
}

// Provider implements the provider interface for OpenAI
type Provider struct {
	client     provider.HTTPClient
	config     config.ModelConfig
	model      string
	tools      map[string]Tool
	rateLimits RateLimiting
	monitor    provider.Monitor
	mu         sync.RWMutex
}

// New creates a new OpenAI provider
func New(model string, cfg config.ModelConfig, opts Options) (*Provider, error) {
	if cfg.APIKey == "" {
		return nil, &provider.Error{
			Code:    provider.ErrAuthentication,
			Message: "OpenAI API key is required",
		}
	}

	// Use provided client or create default
	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{
			Timeout: apiTimeout,
		}
	}

	// Use provided rate limiter or create default
	rateLimiter := opts.RateLimiter
	if rateLimiter == nil {
		rateLimiter = NewRateLimiter(RateLimitConfig{
			RequestsPerMinute: 3,
			TokensPerMinute:   1000,
		})
	}

	return &Provider{
		client:     client,
		config:     cfg,
		model:      model,
		tools:      make(map[string]Tool),
		rateLimits: rateLimiter,
		monitor:    opts.Monitor,
	}, nil
}

// Send sends a prompt to OpenAI and returns the response
func (p *Provider) Send(ctx context.Context, prompt string) (*provider.Response, error) {
	start := time.Now()
	success := false
	defer func() {
		if p.monitor != nil {
			p.monitor.RecordRequest(success)
			p.monitor.RecordLatency(time.Since(start).Seconds())
		}
	}()

	// Check context and rate limits
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if err := p.rateLimits.Wait(ctx); err != nil {
		return nil, err
	}

	// Build request
	req := map[string]any{
		"model": p.model,
		"messages": []map[string]any{{
			"role":    "user",
			"content": prompt,
		}},
		"temperature": p.config.Temperature,
		"max_tokens":  p.config.MaxTokens,
	}

	// Add tools if available
	p.mu.RLock()
	tools := make([]map[string]any, 0, len(p.tools))
	for name, t := range p.tools {
		schema := t.Schema()
		tools = append(tools, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        name,
				"description": schema.Schema.Description,
				"parameters":  schema.Schema.Parameters,
			},
		})
	}
	if len(tools) > 0 {
		req["tools"] = tools
	}
	p.mu.RUnlock()

	// Send request
	resp, err := p.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	// Update rate limits and metrics for initial response
	if err := p.rateLimits.AddTokens(resp.Usage.TotalTokens); err != nil {
		return nil, err
	}

	if p.monitor != nil {
		p.monitor.RecordTokens(
			resp.Usage.PromptTokens,
			resp.Usage.CompletionTokens,
			resp.Usage.TotalTokens,
		)
	}

	// Handle tool calls if present
	if len(resp.Choices[0].Message.ToolCalls) > 0 {
		success = true // Mark initial request as successful
		return p.handleToolCalls(ctx, resp, req)
	}

	success = true // Mark request as successful
	return &provider.Response{
		Content: resp.Choices[0].Message.Content,
		Usage: provider.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}, nil
}

// Close implements provider.Provider
func (p *Provider) Close() error {
	if closer, ok := p.client.(interface{ CloseIdleConnections() }); ok {
		closer.CloseIdleConnections()
	}
	return nil
}

// RegisterTool registers a tool with the provider
func (p *Provider) RegisterTool(name string, t Tool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.tools[name] = t
}

// handleToolCalls processes tool calls in the response
func (p *Provider) handleToolCalls(
	ctx context.Context,
	resp *Response,
	req map[string]any,
) (*provider.Response, error) {
	start := time.Now()
	success := false
	defer func() {
		if p.monitor != nil {
			p.monitor.RecordRequest(success)
			p.monitor.RecordLatency(time.Since(start).Seconds())
		}
	}()
	// Build new request with updated messages and tools
	newReq := map[string]any{
		"model":       req["model"],
		"messages":    req["messages"],
		"temperature": req["temperature"],
		"max_tokens":  req["max_tokens"],
	}

	// Add tools if available
	p.mu.RLock()
	tools := make([]map[string]any, 0, len(p.tools))
	for name, t := range p.tools {
		schema := t.Schema()
		tools = append(tools, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        name,
				"description": schema.Schema.Description,
				"parameters":  schema.Schema.Parameters,
			},
		})
	}
	if len(tools) > 0 {
		newReq["tools"] = tools
	}
	p.mu.RUnlock()

	// Add assistant's message with tool calls
	messages := newReq["messages"].([]map[string]any)
	messages = append(messages, map[string]any{
		"role":       "assistant",
		"content":    resp.Choices[0].Message.Content,
		"tool_calls": resp.Choices[0].Message.ToolCalls,
	})

	// Process each tool call
	for _, call := range resp.Choices[0].Message.ToolCalls {
		// Get tool
		p.mu.RLock()
		tool, ok := p.tools[call.Function.Name]
		p.mu.RUnlock()
		if !ok {
			return nil, &provider.Error{
				Code:    provider.ErrInvalidInput,
				Message: fmt.Sprintf("unknown tool: %s", call.Function.Name),
			}
		}

		// Execute tool
		result, err := tool.Execute([]byte(call.Function.Arguments), nil)
		if err != nil {
			return nil, &provider.Error{
				Code:    provider.ErrServerError,
				Message: fmt.Sprintf("tool execution failed: %v", err),
			}
		}

		// Add tool result
		messages = append(messages, map[string]any{
			"role":         "tool",
			"content":      string(result),
			"tool_call_id": call.ID,
		})
	}
	newReq["messages"] = messages

	// Get final response
	resp, err := p.doRequest(ctx, newReq)
	if err != nil {
		return nil, err
	}

	// Update rate limits and metrics
	if err := p.rateLimits.AddTokens(resp.Usage.TotalTokens); err != nil {
		return nil, err
	}

	if p.monitor != nil {
		p.monitor.RecordTokens(
			resp.Usage.PromptTokens,
			resp.Usage.CompletionTokens,
			resp.Usage.TotalTokens,
		)
	}

	success = true // Mark tool call request as successful
	return &provider.Response{
		Content: resp.Choices[0].Message.Content,
		Usage: provider.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}, nil
}

// doRequest sends a request to the OpenAI API
func (p *Provider) doRequest(ctx context.Context, req map[string]any) (*Response, error) {
	// Marshal request
	body, err := json.Marshal(req)
	if err != nil {
		return nil, &provider.Error{
			Code:    provider.ErrInvalidInput,
			Message: fmt.Sprintf("failed to marshal request: %v", err),
		}
	}

	// Create request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, &provider.Error{
			Code:    provider.ErrServerError,
			Message: fmt.Sprintf("failed to create request: %v", err),
		}
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	// Send request
	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &provider.Error{
			Code:    provider.ErrServerError,
			Message: fmt.Sprintf("request failed: %v", err),
		}
	}
	defer httpResp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, &provider.Error{
			Code:    provider.ErrServerError,
			Message: fmt.Sprintf("failed to read response: %v", err),
		}
	}

	// Check status code
	if httpResp.StatusCode != http.StatusOK {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    string `json:"code"`
			} `json:"error"`
		}
		if err := json.Unmarshal(respBody, &errResp); err != nil {
			return nil, &provider.Error{
				Code:    provider.ErrServerError,
				Message: fmt.Sprintf("request failed with status %d", httpResp.StatusCode),
			}
		}
		return nil, &provider.Error{
			Code:    p.mapErrorCode(errResp.Error.Code),
			Message: errResp.Error.Message,
		}
	}

	// Parse response
	var resp Response
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, &provider.Error{
			Code:    provider.ErrServerError,
			Message: fmt.Sprintf("failed to parse response: %v", err),
		}
	}

	return &resp, nil
}

// mapErrorCode maps OpenAI error codes to provider error codes
func (p *Provider) mapErrorCode(code string) string {
	switch code {
	case "rate_limit_exceeded", "rate_limit_error":
		return provider.ErrRateLimit
	case "invalid_request_error":
		return provider.ErrInvalidInput
	case "server_error":
		return provider.ErrServerError
	case "context_length_exceeded":
		return provider.ErrInvalidInput
	default:
		return provider.ErrServerError
	}
}
