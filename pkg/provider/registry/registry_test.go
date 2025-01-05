package registry

import (
	"context"
	"testing"

	"github.com/butter-bot-machines/skylark/pkg/provider"
)

type mockProvider struct {
	model string
}

func (m *mockProvider) Send(ctx context.Context, prompt string, opts *provider.RequestOptions) (*provider.Response, error) {
	return &provider.Response{Content: "mock response"}, nil
}

func (m *mockProvider) Close() error {
	return nil
}

func TestRegistry(t *testing.T) {
	tests := []struct {
		name            string
		modelSpec       string
		defaultProvider string
		wantProvider    string
		wantModel       string
		wantErr         bool
	}{
		{
			name:            "simple model name",
			modelSpec:       "gpt-4",
			defaultProvider: "openai",
			wantProvider:    "openai",
			wantModel:      "gpt-4",
			wantErr:        false,
		},
		{
			name:            "provider:model format",
			modelSpec:       "anthropic:claude-2",
			defaultProvider: "openai",
			wantProvider:    "anthropic",
			wantModel:      "claude-2",
			wantErr:        false,
		},
		{
			name:            "unknown provider",
			modelSpec:       "unknown:model",
			defaultProvider: "openai",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New()

			// Register test providers
			r.Register("openai", func(model string) (provider.Provider, error) {
				return &mockProvider{model: model}, nil
			})
			r.Register("anthropic", func(model string) (provider.Provider, error) {
				return &mockProvider{model: model}, nil
			})

			// Create provider for model spec
			p, err := r.CreateForModel(tt.modelSpec, tt.defaultProvider)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify provider and model
			mp, ok := p.(*mockProvider)
			if !ok {
				t.Fatal("expected mockProvider")
			}

			if mp.model != tt.wantModel {
				t.Errorf("model = %v, want %v", mp.model, tt.wantModel)
			}
		})
	}
}

func TestParseModelSpec(t *testing.T) {
	tests := []struct {
		spec         string
		wantProvider string
		wantModel    string
	}{
		{
			spec:         "gpt-4",
			wantProvider: "",
			wantModel:    "gpt-4",
		},
		{
			spec:         "openai:gpt-4",
			wantProvider: "openai",
			wantModel:    "gpt-4",
		},
		{
			spec:         "anthropic:claude-2",
			wantProvider: "anthropic",
			wantModel:    "claude-2",
		},
		{
			spec:         ":gpt-4",
			wantProvider: "",
			wantModel:    "gpt-4",
		},
		{
			spec:         "openai:",
			wantProvider: "openai",
			wantModel:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.spec, func(t *testing.T) {
			provider, model := ParseModelSpec(tt.spec)
			if provider != tt.wantProvider {
				t.Errorf("provider = %v, want %v", provider, tt.wantProvider)
			}
			if model != tt.wantModel {
				t.Errorf("model = %v, want %v", model, tt.wantModel)
			}
		})
	}
}
