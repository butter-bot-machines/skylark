package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the root configuration structure
type Config struct {
	Version     string           `yaml:"version"`
	Environment map[string]string `yaml:"environment,omitempty"`
	Model       ModelConfig      `yaml:"model"`
	Tools       ToolsConfig      `yaml:"tools,omitempty"`
	Assistants  AssistantsConfig `yaml:"assistants"`
}

// ModelConfig represents AI model configuration
type ModelConfig struct {
	Provider    string            `yaml:"provider"`
	Name        string            `yaml:"name"`
	MaxTokens   int               `yaml:"max_tokens"`
	Temperature float64           `yaml:"temperature"`
	TopP        float64           `yaml:"top_p"`
	Parameters  map[string]interface{} `yaml:"parameters,omitempty"`
}

// ToolsConfig represents tool-related configuration
type ToolsConfig struct {
	MaxTimeout  int               `yaml:"max_timeout"`
	Environment map[string]string `yaml:"environment,omitempty"`
	Defaults    map[string]interface{} `yaml:"defaults,omitempty"`
}

// AssistantsConfig represents assistant-related configuration
type AssistantsConfig struct {
	Default     string            `yaml:"default"`
	Environment map[string]string `yaml:"environment,omitempty"`
	Parameters  map[string]interface{} `yaml:"parameters,omitempty"`
}

// Manager handles configuration loading and validation
type Manager struct {
	config    *Config
	basePath  string
	envValues map[string]string
}

// NewManager creates a new configuration manager
func NewManager(basePath string) *Manager {
	return &Manager{
		basePath:  basePath,
		envValues: make(map[string]string),
	}
}

// Load loads and validates the configuration file
func (m *Manager) Load() error {
	configPath := filepath.Join(m.basePath, "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate version
	if err := m.validateVersion(config.Version); err != nil {
		return err
	}

	// Expand environment variables
	if err := m.expandEnvironment(config); err != nil {
		return err
	}

	// Validate configuration
	if err := m.validate(config); err != nil {
		return err
	}

	m.config = config
	return nil
}

// validateVersion checks if the configuration version is supported
func (m *Manager) validateVersion(version string) error {
	supportedVersions := map[string]bool{
		"1.0": true,
	}

	if !supportedVersions[version] {
		return fmt.Errorf("unsupported configuration version: %s", version)
	}

	return nil
}

// expandEnvironment expands environment variables in configuration values
func (m *Manager) expandEnvironment(config *Config) error {
	// Process global environment
	for key, value := range config.Environment {
		expanded := os.ExpandEnv(value)
		m.envValues[key] = expanded
	}

	// Process tool environment
	for key, value := range config.Tools.Environment {
		expanded := os.ExpandEnv(value)
		m.envValues[fmt.Sprintf("TOOL_%s", key)] = expanded
	}

	// Process assistant environment
	for key, value := range config.Assistants.Environment {
		expanded := os.ExpandEnv(value)
		m.envValues[fmt.Sprintf("ASSISTANT_%s", key)] = expanded
	}

	return nil
}

// validate performs configuration validation
func (m *Manager) validate(config *Config) error {
	// Validate model configuration
	if config.Model.Provider == "" {
		return fmt.Errorf("model provider is required")
	}
	if config.Model.Name == "" {
		return fmt.Errorf("model name is required")
	}
	if config.Model.MaxTokens <= 0 {
		return fmt.Errorf("invalid max_tokens value: %d", config.Model.MaxTokens)
	}
	if config.Model.Temperature < 0 || config.Model.Temperature > 1 {
		return fmt.Errorf("temperature must be between 0 and 1")
	}
	if config.Model.TopP < 0 || config.Model.TopP > 1 {
		return fmt.Errorf("top_p must be between 0 and 1")
	}

	// Validate tool configuration
	if config.Tools.MaxTimeout <= 0 {
		config.Tools.MaxTimeout = 30 // Default timeout
	}
	if config.Tools.MaxTimeout > 300 {
		return fmt.Errorf("max_timeout cannot exceed 300 seconds")
	}

	// Validate assistant configuration
	if config.Assistants.Default == "" {
		return fmt.Errorf("default assistant is required")
	}

	return nil
}

// GetEnvironment returns the environment value for a key
func (m *Manager) GetEnvironment(key string) string {
	// Check configuration environment first
	if value, exists := m.envValues[key]; exists {
		return value
	}

	// Fall back to system environment
	return os.Getenv(key)
}

// GetModelConfig returns the model configuration
func (m *Manager) GetModelConfig() ModelConfig {
	return m.config.Model
}

// GetToolConfig returns tool-specific configuration
func (m *Manager) GetToolConfig(name string) map[string]interface{} {
	config := make(map[string]interface{})

	// Start with defaults
	if m.config != nil && m.config.Tools.Defaults != nil {
		for k, v := range m.config.Tools.Defaults {
			config[k] = v
		}
	}

	// Add tool environment
	env := make(map[string]string)
	
	// Add global tool environment
	if m.config != nil && m.config.Tools.Environment != nil {
		for k, v := range m.config.Tools.Environment {
			env[k] = v
		}
	}

	// Add tool-specific environment
	prefix := fmt.Sprintf("TOOL_%s_", strings.ToUpper(name))
	for k, v := range m.envValues {
		if strings.HasPrefix(k, prefix) {
			env[strings.TrimPrefix(k, prefix)] = v
		}
	}

	if len(env) > 0 {
		config["environment"] = env
	}

	return config
}

// GetAssistantConfig returns assistant-specific configuration
func (m *Manager) GetAssistantConfig(name string) map[string]interface{} {
	config := make(map[string]interface{})

	// Start with global parameters
	if m.config != nil && m.config.Assistants.Parameters != nil {
		for k, v := range m.config.Assistants.Parameters {
			config[k] = v
		}
	}

	// Add assistant environment
	env := make(map[string]string)

	// Add global assistant environment
	if m.config != nil && m.config.Assistants.Environment != nil {
		for k, v := range m.config.Assistants.Environment {
			env[k] = v
		}
	}

	// Add assistant-specific environment
	prefix := fmt.Sprintf("ASSISTANT_%s_", strings.ToUpper(name))
	for k, v := range m.envValues {
		if strings.HasPrefix(k, prefix) {
			env[strings.TrimPrefix(k, prefix)] = v
		}
	}

	if len(env) > 0 {
		config["environment"] = env
	}

	return config
}

// GetDefaultAssistant returns the name of the default assistant
func (m *Manager) GetDefaultAssistant() string {
	return m.config.Assistants.Default
}
