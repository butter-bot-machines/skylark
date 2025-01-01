package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"time"
)

// Config represents the complete application configuration
type Config struct {
	Version     string                              `yaml:"version"`
	Environment EnvironmentConfig                   `yaml:"environment"`
	Models      map[string]map[string]ModelConfig   `yaml:"models"`  // provider -> model -> config
	Tools       map[string]ToolConfig               `yaml:"tools"`   // name -> config
	Workers     WorkerConfig                        `yaml:"workers"`
	WatchPaths  []string                           `yaml:"watch_paths"`
	FileWatch   FileWatchConfig                    `yaml:"file_watch"`
	Security    SecurityConfig                      `yaml:"security"`
}

// EnvironmentConfig contains environment-specific settings
type EnvironmentConfig struct {
	LogLevel  string `yaml:"log_level"`
	LogFile   string `yaml:"log_file"`
	ConfigDir string `yaml:"-"` // Set at runtime, not from config file
}

// ModelConfig contains model-specific settings
type ModelConfig struct {
	APIKey           string  `yaml:"api_key"`
	Temperature      float64 `yaml:"temperature"`
	MaxTokens        int     `yaml:"max_tokens"`
	TopP             float64 `yaml:"top_p,omitempty"`
	FrequencyPenalty float64 `yaml:"frequency_penalty,omitempty"`
	PresencePenalty  float64 `yaml:"presence_penalty,omitempty"`
}

// ToolConfig contains tool-specific settings
type ToolConfig struct {
	Env map[string]string `yaml:"env"` // environment variables
}

// WorkerConfig contains worker pool settings
type WorkerConfig struct {
	Count     int `yaml:"count"`
	QueueSize int `yaml:"queue_size"`
}

// FileWatchConfig contains file watcher settings
type FileWatchConfig struct {
	DebounceDelay time.Duration `yaml:"debounce_delay"`
	MaxDelay      time.Duration `yaml:"max_delay"`
	Extensions    []string      `yaml:"extensions"`
}

// ParseConfig parses configuration from YAML data
func ParseConfig(data []byte) (*Config, error) {
	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	return config, nil
}

// Marshal converts the configuration to YAML
func (c *Config) Marshal() ([]byte, error) {
	return yaml.Marshal(c)
}

// GetEnvironment returns the environment configuration
func (c *Config) GetEnvironment() EnvironmentConfig {
	return c.Environment
}

// GetModelConfig returns the model configuration for a provider and model
func (c *Config) GetModelConfig(provider, model string) (ModelConfig, bool) {
	if providerModels, ok := c.Models[provider]; ok {
		if modelConfig, ok := providerModels[model]; ok {
			return modelConfig, true
		}
	}
	return ModelConfig{}, false
}

// GetToolConfig returns the tool configuration for a tool name
func (c *Config) GetToolConfig(name string) (ToolConfig, bool) {
	config, ok := c.Tools[name]
	return config, ok
}

// GetToolEnv returns the environment variables for a tool
func (c *Config) GetToolEnv(name string) map[string]string {
	if config, ok := c.Tools[name]; ok {
		return config.Env
	}
	return nil
}

// GetSecurityConfig returns the security configuration
func (c *Config) GetSecurityConfig() SecurityConfig {
	return c.Security
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Version == "" {
		return fmt.Errorf("version is required")
	}
	
	// Validate models configuration
	for provider, models := range c.Models {
		if len(models) == 0 {
			return fmt.Errorf("no models configured for provider %s", provider)
		}
		for model, config := range models {
			if config.APIKey == "" {
				return fmt.Errorf("API key required for %s:%s", provider, model)
			}
		}
	}
	
	return nil
}
