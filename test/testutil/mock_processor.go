package testutil

import (
	"github.com/butter-bot-machines/skylark/pkg/config"
	"github.com/butter-bot-machines/skylark/pkg/processor"
)

// NewMockProcessor creates a minimal processor for testing
func NewMockProcessor() (*processor.Processor, error) {
	// Create minimal config
	cfg := &config.Config{
		Environment: config.EnvironmentConfig{
			ConfigDir: "/tmp/test",
		},
		Models: map[string]map[string]config.ModelConfig{
			"openai": {
				"gpt-4": {
					APIKey: "test-key",
				},
			},
		},
	}

	return processor.New(cfg)
}
