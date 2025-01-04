package testutil

import (
	"context"

	"github.com/butter-bot-machines/skylark/pkg/provider"
)

// MockProvider implements a test provider that returns predefined responses
type MockProvider struct {
	response string
}

// NewMockProvider creates a new mock provider
func NewMockProvider() *MockProvider {
	return &MockProvider{
		response: "Test response",
	}
}

// Send implements the provider.Provider interface
func (p *MockProvider) Send(ctx context.Context, prompt string) (*provider.Response, error) {
	return &provider.Response{
		Content: p.response,
		Usage: provider.Usage{
			PromptTokens:     10,
			CompletionTokens: 10,
			TotalTokens:      20,
		},
	}, nil
}

// Close implements the provider.Provider interface
func (p *MockProvider) Close() error {
	return nil
}
