package provider

import "context"

// RequestOptions contains configuration options for a single request
type RequestOptions struct {
	Model       string  // Model to use for this request
	Temperature float64 // Temperature setting for this request
	MaxTokens   int     // Max tokens for this request
}

// DefaultRequestOptions provides commonly used request settings for testing
var DefaultRequestOptions = &RequestOptions{
	Model:       "gpt-4",
	Temperature: 0.7,
	MaxTokens:   100,
}

// Provider defines the interface for model providers
type Provider interface {
	Send(ctx context.Context, prompt string, opts *RequestOptions) (*Response, error)
	Close() error
}

// Response represents a model's response
type Response struct {
	Content   string
	Usage     Usage
	Error     error
	ToolCalls []ToolCall
}

// ToolCall represents a request to execute a tool
type ToolCall struct {
	ID       string
	Function Function
}

// Function represents a tool function to execute
type Function struct {
	Name      string
	Arguments string
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// Error represents a provider error
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

// Common error codes
const (
	ErrRateLimit      = "rate_limit_exceeded"
	ErrInvalidInput   = "invalid_input"
	ErrServerError    = "server_error"
	ErrTimeout        = "timeout"
	ErrAuthentication = "authentication_error"
)

// Factory creates a new provider instance
type Factory interface {
	Create() (Provider, error)
}
