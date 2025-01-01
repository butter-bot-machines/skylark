package assistant

import (
	"context"

	"github.com/butter-bot-machines/skylark/pkg/provider"
)

// testProvider simulates an AI provider for testing
type testProvider struct {
	responses []provider.Response
	requests  []string
}

func (p *testProvider) Send(ctx context.Context, prompt string) (*provider.Response, error) {
	if len(p.responses) == 0 {
		return nil, &provider.Error{Code: provider.ErrServerError, Message: "no responses configured"}
	}
	resp := p.responses[0]
	p.responses = p.responses[1:]
	p.requests = append(p.requests, prompt)
	return &resp, resp.Error
}

func (p *testProvider) Close() error {
	return nil
}

// testFixtures provides common test data
var testFixtures = struct {
	basicResponse    provider.Response
	toolCallResponse provider.Response
	errorResponse    provider.Response
}{
	basicResponse: provider.Response{
		Content: "Hello! How can I help?",
		Usage: provider.Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	},
	toolCallResponse: provider.Response{
		Content: "Let me help summarize that",
		ToolCalls: []provider.ToolCall{
			{
				ID: "call_1",
				Function: provider.Function{
					Name:      "summarize",
					Arguments: `{"text":"test"}`,
				},
			},
		},
		Usage: provider.Usage{
			PromptTokens:     20,
			CompletionTokens: 10,
			TotalTokens:      30,
		},
	},
	errorResponse: provider.Response{
		Error: &provider.Error{
			Code:    provider.ErrServerError,
			Message: "simulated error",
		},
	},
}
