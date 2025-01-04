package concrete

import (
	"context"
	"strings"

	"github.com/butter-bot-machines/skylark/pkg/parser"
	"github.com/butter-bot-machines/skylark/pkg/provider"
)

// mockProvider implements provider.Provider for testing
type mockProvider struct {
	provider.Provider
}

func newMockProvider() provider.Provider {
	return &mockProvider{}
}

func (p *mockProvider) Process(cmd *parser.Command) (string, error) {
	// For testing, return raw response
	return "test", nil
}

func (p *mockProvider) Send(ctx context.Context, prompt string) (*provider.Response, error) {
	// Extract command from prompt
	lines := strings.Split(prompt, "\n")
	var command string
	for _, line := range lines {
		if strings.HasPrefix(line, "Command: ") {
			command = strings.TrimPrefix(line, "Command: ")
			break
		}
	}
	// Return raw command response
	return &provider.Response{
		Content: command,
	}, nil
}

func (p *mockProvider) Close() error {
	return nil
}
