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
environment:
  log_level: debug
  log_file: app.log
model:
  provider: openai
  name: gpt-4
  parameters:
    max_tokens: 2048
    temperature: 0.7
tools:
  path: /usr/local/bin
  parameters:
    key1: value1
    key2: value2
assistants:
  default:
    name: Default Assistant
    model: gpt-4
    description: Default testing assistant
default_assistant: default
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
	model := manager.GetModelConfig()
	if model.Provider != "openai" {
		t.Errorf("Expected provider 'openai', got '%s'", model.Provider)
	}
	if model.Name != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got '%s'", model.Name)
	}

	// Test tool config
	tools := manager.GetToolConfig()
	if tools.Path != "/usr/local/bin" {
		t.Errorf("Expected path '/usr/local/bin', got '%s'", tools.Path)
	}
	if tools.Parameters["key1"] != "value1" {
		t.Errorf("Expected parameter key1='value1', got '%s'", tools.Parameters["key1"])
	}
	if tools.Parameters["key2"] != "value2" {
		t.Errorf("Expected parameter key2='value2', got '%s'", tools.Parameters["key2"])
	}

	// Test assistant config
	assistant, ok := manager.GetAssistantConfig("default")
	if !ok {
		t.Error("Failed to get default assistant config")
	}
	if assistant.Name != "Default Assistant" {
		t.Errorf("Expected assistant name 'Default Assistant', got '%s'", assistant.Name)
	}
}

func TestConfigSaving(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create a config
	config := &Config{
		Environment: EnvironmentConfig{
			LogLevel: "info",
			LogFile:  "test.log",
		},
		Model: ModelConfig{
			Provider: "test-provider",
			Name:     "test-model",
		},
	}

	// Create manager and save config
	manager := NewManager(tmpDir)
	manager.Set(config)
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
	loadedConfig := newManager.Get()
	if loadedConfig.Environment.LogLevel != config.Environment.LogLevel {
		t.Errorf("Expected log level '%s', got '%s'", config.Environment.LogLevel, loadedConfig.Environment.LogLevel)
	}
	if loadedConfig.Model.Provider != config.Model.Provider {
		t.Errorf("Expected provider '%s', got '%s'", config.Model.Provider, loadedConfig.Model.Provider)
	}
}

func TestDefaultConfig(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create manager with default config
	manager := NewManager(tmpDir)
	config := manager.Get()

	// Verify default values
	if config.Workers.Count != 0 {
		t.Errorf("Expected default worker count 0, got %d", config.Workers.Count)
	}
	if config.Workers.QueueSize != 0 {
		t.Errorf("Expected default queue size 0, got %d", config.Workers.QueueSize)
	}
}

func TestToolConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configData := []byte(`
tools:
  path: /custom/path
  parameters:
    api_key: test-key
    endpoint: test-endpoint
    timeout: "30s"
`)

	err := os.WriteFile(configPath, configData, 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load and verify tool config
	manager := NewManager(tmpDir)
	err = manager.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	toolConfig := manager.GetToolConfig()
	if toolConfig.Path != "/custom/path" {
		t.Errorf("Expected tool path '/custom/path', got '%s'", toolConfig.Path)
	}
	if toolConfig.Parameters["api_key"] != "test-key" {
		t.Errorf("Expected parameter api_key='test-key', got '%s'", toolConfig.Parameters["api_key"])
	}
	if toolConfig.Parameters["endpoint"] != "test-endpoint" {
		t.Errorf("Expected parameter endpoint='test-endpoint', got '%s'", toolConfig.Parameters["endpoint"])
	}
}

func TestAssistantConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configData := []byte(`
assistants:
  test:
    name: Test Assistant
    model: gpt-4
    description: Test assistant
default_assistant: test
`)

	err := os.WriteFile(configPath, configData, 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load and verify assistant config
	manager := NewManager(tmpDir)
	err = manager.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	defaultAssistant, ok := manager.GetDefaultAssistant()
	if !ok {
		t.Error("Failed to get default assistant")
	}
	if defaultAssistant.Name != "Test Assistant" {
		t.Errorf("Expected assistant name 'Test Assistant', got '%s'", defaultAssistant.Name)
	}

	assistant, ok := manager.GetAssistantConfig("test")
	if !ok {
		t.Error("Failed to get test assistant")
	}
	if assistant.Name != "Test Assistant" {
		t.Errorf("Expected assistant name 'Test Assistant', got '%s'", assistant.Name)
	}
}
