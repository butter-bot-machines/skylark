package config

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestConfig(t *testing.T) string {
	// Create temporary directory
	tempDir := t.TempDir()

	// Create valid config.yaml
	configContent := `version: "1.0"
environment:
  API_KEY: "${TEST_API_KEY}"
  REGION: "us-west-2"

model:
  provider: "openai"
  name: "gpt-4"
  max_tokens: 2000
  temperature: 0.7
  top_p: 0.9
  parameters:
    presence_penalty: 0.0
    frequency_penalty: 0.0

tools:
  max_timeout: 60
  environment:
    DEFAULT_TIMEOUT: "30"
    API_BASE_URL: "https://api.example.com"
  defaults:
    retry_count: 3
    retry_delay: 1000

assistants:
  default: "general"
  environment:
    MODEL_VERSION: "latest"
  parameters:
    max_context_size: 4000
    max_references: 10
`

	err := os.WriteFile(filepath.Join(tempDir, "config.yaml"), []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	return tempDir
}

func TestConfigLoading(t *testing.T) {
	basePath := setupTestConfig(t)
	manager := NewManager(basePath)

	// Set test environment variable
	os.Setenv("TEST_API_KEY", "test-key-123")
	defer os.Unsetenv("TEST_API_KEY")

	err := manager.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Test environment expansion
	if env := manager.GetEnvironment("API_KEY"); env != "test-key-123" {
		t.Errorf("Environment expansion failed, got %s, want test-key-123", env)
	}

	// Test model config
	modelConfig := manager.GetModelConfig()
	if modelConfig.Provider != "openai" {
		t.Errorf("Model provider = %s, want openai", modelConfig.Provider)
	}
	if modelConfig.MaxTokens != 2000 {
		t.Errorf("MaxTokens = %d, want 2000", modelConfig.MaxTokens)
	}

	// Test tool config
	toolConfig := manager.GetToolConfig("test-tool")
	if retryCount, ok := toolConfig["retry_count"].(int); !ok || retryCount != 3 {
		t.Errorf("Tool retry_count = %v, want 3", toolConfig["retry_count"])
	}

	// Test assistant config
	assistantConfig := manager.GetAssistantConfig("test-assistant")
	if maxContextSize, ok := assistantConfig["max_context_size"].(int); !ok || maxContextSize != 4000 {
		t.Errorf("Assistant max_context_size = %v, want 4000", assistantConfig["max_context_size"])
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		wantError   bool
		errorString string
	}{
		{
			name: "invalid version",
			config: `version: "2.0"
model:
  provider: "openai"
  name: "gpt-4"
  max_tokens: 2000
  temperature: 0.7
  top_p: 0.9
assistants:
  default: "general"`,
			wantError:   true,
			errorString: "unsupported configuration version: 2.0",
		},
		{
			name: "missing model provider",
			config: `version: "1.0"
model:
  name: "gpt-4"
  max_tokens: 2000
  temperature: 0.7
  top_p: 0.9
assistants:
  default: "general"`,
			wantError:   true,
			errorString: "model provider is required",
		},
		{
			name: "invalid temperature",
			config: `version: "1.0"
model:
  provider: "openai"
  name: "gpt-4"
  max_tokens: 2000
  temperature: 1.5
  top_p: 0.9
assistants:
  default: "general"`,
			wantError:   true,
			errorString: "temperature must be between 0 and 1",
		},
		{
			name: "missing default assistant",
			config: `version: "1.0"
model:
  provider: "openai"
  name: "gpt-4"
  max_tokens: 2000
  temperature: 0.7
  top_p: 0.9
assistants:
  environment:
    MODEL_VERSION: "latest"`,
			wantError:   true,
			errorString: "default assistant is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			err := os.WriteFile(filepath.Join(tempDir, "config.yaml"), []byte(tt.config), 0644)
			if err != nil {
				t.Fatalf("Failed to create test config file: %v", err)
			}

			manager := NewManager(tempDir)
			err = manager.Load()

			if (err != nil) != tt.wantError {
				t.Errorf("Load() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.wantError && err.Error() != tt.errorString {
				t.Errorf("Load() error = %v, want %v", err, tt.errorString)
			}
		})
	}
}

func TestEnvironmentResolution(t *testing.T) {
	basePath := setupTestConfig(t)
	manager := NewManager(basePath)

	// Set environment variables
	os.Setenv("TEST_API_KEY", "test-key-123")
	os.Setenv("CUSTOM_VAR", "custom-value")
	defer func() {
		os.Unsetenv("TEST_API_KEY")
		os.Unsetenv("CUSTOM_VAR")
	}()

	err := manager.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	tests := []struct {
		name     string
		key      string
		want     string
		fallback bool
	}{
		{
			name: "config environment",
			key:  "API_KEY",
			want: "test-key-123",
		},
		{
			name: "system environment",
			key:  "CUSTOM_VAR",
			want: "custom-value",
		},
		{
			name:     "nonexistent variable",
			key:      "NONEXISTENT_VAR",
			want:     "",
			fallback: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := manager.GetEnvironment(tt.key)
			if got != tt.want {
				t.Errorf("GetEnvironment(%s) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestToolConfiguration(t *testing.T) {
	basePath := setupTestConfig(t)
	manager := NewManager(basePath)

	err := manager.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Test tool-specific configuration
	toolConfig := manager.GetToolConfig("summarize")
	
	// Check defaults are applied
	if retryCount, ok := toolConfig["retry_count"].(int); !ok || retryCount != 3 {
		t.Errorf("Tool retry_count = %v, want 3", toolConfig["retry_count"])
	}

	// Check environment is properly prefixed
	env, ok := toolConfig["environment"].(map[string]string)
	if !ok {
		t.Error("Tool environment not found")
	} else {
		if timeout := env["DEFAULT_TIMEOUT"]; timeout != "30" {
			t.Errorf("Tool DEFAULT_TIMEOUT = %v, want 30", timeout)
		}
	}
}

func TestAssistantConfiguration(t *testing.T) {
	basePath := setupTestConfig(t)
	manager := NewManager(basePath)

	err := manager.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Test default assistant name
	if defaultAssistant := manager.GetDefaultAssistant(); defaultAssistant != "general" {
		t.Errorf("Default assistant = %s, want general", defaultAssistant)
	}

	// Test assistant-specific configuration
	assistantConfig := manager.GetAssistantConfig("researcher")
	
	// Check global parameters are applied
	if maxContextSize, ok := assistantConfig["max_context_size"].(int); !ok || maxContextSize != 4000 {
		t.Errorf("Assistant max_context_size = %v, want 4000", maxContextSize)
	}

	// Check environment is properly prefixed
	env, ok := assistantConfig["environment"].(map[string]string)
	if !ok {
		t.Error("Assistant environment not found")
	} else {
		if version := env["MODEL_VERSION"]; version != "latest" {
			t.Errorf("Assistant MODEL_VERSION = %v, want latest", version)
		}
	}
}
