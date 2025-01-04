package testutil

import (
	"github.com/butter-bot-machines/skylark/pkg/config"
	"github.com/butter-bot-machines/skylark/pkg/processor"
	"github.com/butter-bot-machines/skylark/pkg/processor/concrete"
)

// NewMockProcessor creates a new processor for testing
func NewMockProcessor() (processor.ProcessManager, error) {
	// Create minimal config with test key to trigger mock provider
	cfg := &config.Config{
		Environment: config.EnvironmentConfig{
			ConfigDir: "/tmp/test",
		},
		Models: map[string]config.ModelConfigSet{
			"openai": {
				"gpt-4": config.ModelConfig{
					APIKey: "test-key", // Special key that triggers mock provider
				},
			},
		},
	}

	// Create processor with mock config
	return concrete.NewProcessor(cfg)
}
