package config

import (
	"fmt"
	"time"

	"github.com/butter-bot-machines/skylark/pkg/security/types"
	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Version     string                     `yaml:"version"`
	Environment EnvironmentConfig          `yaml:"environment"`
	Models      map[string]ModelConfigSet  `yaml:"models"`
	Tools       map[string]ToolConfig      `yaml:"tools"`
	Workers     WorkerConfig               `yaml:"workers"`
	FileWatch   FileWatchConfig           `yaml:"file_watch"`
	WatchPaths  []string                  `yaml:"watch_paths"`
	Security    types.SecurityConfig       `yaml:"security"`
}

// EnvironmentConfig defines environment-specific settings
type EnvironmentConfig struct {
	LogLevel  string `yaml:"log_level"`
	LogFile   string `yaml:"log_file"`
	ConfigDir string `yaml:"-"` // Set at runtime
}

// ModelConfigSet groups model configurations by provider
type ModelConfigSet map[string]ModelConfig

// ModelConfig defines model-specific settings
type ModelConfig struct {
	APIKey      string  `yaml:"api_key"`
	Temperature float64 `yaml:"temperature"`
	MaxTokens   int     `yaml:"max_tokens"`
	TopP        float64 `yaml:"top_p"`
}

// ToolConfig defines tool-specific settings
type ToolConfig struct {
	Env map[string]string `yaml:"env"`
}

// WorkerConfig defines worker pool settings
type WorkerConfig struct {
	Count     int `yaml:"count"`
	QueueSize int `yaml:"queue_size"`
}

// FileWatchConfig defines file watching settings
type FileWatchConfig struct {
	DebounceDelay time.Duration `yaml:"debounce_delay"`
	MaxDelay      time.Duration `yaml:"max_delay"`
	Extensions    []string      `yaml:"extensions"`
}

// ParseConfig parses a configuration from YAML
func ParseConfig(data []byte) (*Config, error) {
	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	return config, nil
}

// Marshal marshals the configuration to YAML
func (c *Config) Marshal() ([]byte, error) {
	return yaml.Marshal(c)
}

// GetEnvironment returns the environment configuration
func (c *Config) GetEnvironment() EnvironmentConfig {
	return c.Environment
}

// GetModelConfig returns the model configuration for a provider and model
func (c *Config) GetModelConfig(provider, model string) (ModelConfig, bool) {
	if models, ok := c.Models[provider]; ok {
		if config, ok := models[model]; ok {
			return config, true
		}
	}
	return ModelConfig{}, false
}

// GetToolConfig returns the tool configuration for a tool name
func (c *Config) GetToolConfig(name string) (ToolConfig, bool) {
	if config, ok := c.Tools[name]; ok {
		return config, true
	}
	return ToolConfig{}, false
}

// GetToolEnv returns the environment variables for a tool
func (c *Config) GetToolEnv(name string) map[string]string {
	if config, ok := c.GetToolConfig(name); ok {
		return config.Env
	}
	return nil
}

// GetSecurityConfig returns the security configuration
func (c *Config) GetSecurityConfig() types.SecurityConfig {
	return c.Security
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Version == "" {
		return fmt.Errorf("%w: version required", ErrInvalidConfig)
	}

	// Validate model configurations
	for provider, models := range c.Models {
		for model, config := range models {
			if config.APIKey == "" {
				return fmt.Errorf("%w: API key required for model %s/%s", ErrInvalidConfig, provider, model)
			}
		}
	}

	return nil
}

// AsMap converts the configuration to a map
func (c *Config) AsMap() map[string]interface{} {
	data, _ := yaml.Marshal(c)
	var result map[string]interface{}
	yaml.Unmarshal(data, &result)
	return result
}

// FromMap updates the configuration from a map
func (c *Config) FromMap(data map[string]interface{}) error {
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal map: %w", err)
	}
	
	if err := yaml.Unmarshal(yamlData, c); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return nil
}
