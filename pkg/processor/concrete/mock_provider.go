package concrete

import (
	"context"

	"github.com/butter-bot-machines/skylark/pkg/provider"
)

// mockProvider simulates an AI provider for testing
type mockProvider struct {
	response string
}

func (p *mockProvider) Send(ctx context.Context, prompt string, opts *provider.RequestOptions) (*provider.Response, error) {
	return &provider.Response{
		Content: p.response,
		Usage: provider.Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}, nil
}

func (p *mockProvider) Close() error {
	return nil
}

// newMockProvider creates a new mock provider for testing
func newMockProvider() provider.Provider {
	return &mockProvider{
		response: "command",
	}
}
