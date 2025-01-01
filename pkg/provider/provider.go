package provider

import "context"

// Provider defines the interface for model providers
type Provider interface {
	Send(ctx context.Context, prompt string) (*Response, error)
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
