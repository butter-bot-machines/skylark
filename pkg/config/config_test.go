package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigLoading(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configData := []byte(`
version: "1.0"
environment:
  log_level: debug
  log_file: app.log
models:
  openai:
    gpt-4:
      api_key: sk-test-key
      temperature: 0.7
      max_tokens: 2048
    gpt-3.5-turbo:
      api_key: sk-test-key-2
      temperature: 0.5
      max_tokens: 1000
tools:
  summarize:
    env:
      API_KEY: sum-test-key
      MAX_LENGTH: "100"
  web_search:
    env:
      API_KEY: search-test-key
      TIMEOUT: "30s"
workers:
  count: 4
  queue_size: 100
watch_paths:
  - /path/to/watch1
  - /path/to/watch2
file_watch:
  debounce_delay: 100ms
  max_delay: 1s
  extensions:
    - .md
    - .txt
`)

	err := os.WriteFile(configPath, configData, 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Create config manager
	manager := NewManager(tmpDir)
	err = manager.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test environment config
	env := manager.GetEnvironment()
	if env.LogLevel != "debug" {
		t.Errorf("Expected log level 'debug', got '%s'", env.LogLevel)
	}
	if env.LogFile != "app.log" {
		t.Errorf("Expected log file 'app.log', got '%s'", env.LogFile)
	}

	// Test model config
	model, ok := manager.GetModelConfig("openai", "gpt-4")
	if !ok {
		t.Fatal("Failed to get model config")
	}
	if model.APIKey != "sk-test-key" {
		t.Errorf("Expected API key 'sk-test-key', got '%s'", model.APIKey)
	}
	if model.Temperature != 0.7 {
		t.Errorf("Expected temperature 0.7, got %f", model.Temperature)
	}
	if model.MaxTokens != 2048 {
		t.Errorf("Expected max tokens 2048, got %d", model.MaxTokens)
	}

	// Test tool config
	tool, ok := manager.GetToolConfig("summarize")
	if !ok {
		t.Fatal("Failed to get tool config")
	}
	if tool.Env["API_KEY"] != "sum-test-key" {
		t.Errorf("Expected API key 'sum-test-key', got '%s'", tool.Env["API_KEY"])
	}
	if tool.Env["MAX_LENGTH"] != "100" {
		t.Errorf("Expected MAX_LENGTH '100', got '%s'", tool.Env["MAX_LENGTH"])
	}

	// Test worker config
	cfg := manager.GetConfig()
	if cfg.Workers.Count != 4 {
		t.Errorf("Expected worker count 4, got %d", cfg.Workers.Count)
	}
	if cfg.Workers.QueueSize != 100 {
		t.Errorf("Expected queue size 100, got %d", cfg.Workers.QueueSize)
	}
}

func TestConfigSaving(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create a config
	config := &Config{
		Version: "1.0",
		Environment: EnvironmentConfig{
			LogLevel: "info",
			LogFile:  "test.log",
		},
		Models: map[string]ModelConfigSet{
			"openai": {
				"gpt-4": {
					APIKey:      "sk-test",
					Temperature: 0.7,
					MaxTokens:   2048,
				},
			},
		},
		Tools: map[string]ToolConfig{
			"summarize": {
				Env: map[string]string{
					"API_KEY": "test-key",
				},
			},
		},
	}

	// Create manager and save config
	manager := NewManager(tmpDir)
	manager.SetConfig(config)
	err := manager.Save()
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Create new manager and load saved config
	newManager := NewManager(tmpDir)
	err = newManager.Load()
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	// Verify loaded config matches original
	loadedConfig := newManager.GetConfig()
	if loadedConfig.Version != config.Version {
		t.Errorf("Expected version '%s', got '%s'", config.Version, loadedConfig.Version)
	}

	// Check model config
	model, ok := loadedConfig.Models["openai"]["gpt-4"]
	if !ok {
		t.Fatal("Failed to get model config")
	}
	if model.APIKey != "sk-test" {
		t.Errorf("Expected API key 'sk-test', got '%s'", model.APIKey)
	}

	// Check tool config
	tool, ok := loadedConfig.Tools["summarize"]
	if !ok {
		t.Fatal("Failed to get tool config")
	}
	if tool.Env["API_KEY"] != "test-key" {
		t.Errorf("Expected API key 'test-key', got '%s'", tool.Env["API_KEY"])
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Version: "1.0",
				Models: map[string]ModelConfigSet{
					"openai": {
						"gpt-4": {
							APIKey: "sk-test",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing version",
			config: &Config{
				Models: map[string]ModelConfigSet{
					"openai": {
						"gpt-4": {
							APIKey: "sk-test",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing API key",
			config: &Config{
				Version: "1.0",
				Models: map[string]ModelConfigSet{
					"openai": {
						"gpt-4": {},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
