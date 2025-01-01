package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"time"
)

// Config represents the complete application configuration
type Config struct {
	Environment      EnvironmentConfig      `yaml:"environment"`
	Model           ModelConfig            `yaml:"model"`
	Tools           ToolConfig             `yaml:"tools"`
	Assistants      map[string]Assistant   `yaml:"assistants"`
	DefaultAssistant string                `yaml:"default_assistant"`
	Workers         WorkerConfig           `yaml:"workers"`
	WatchPaths      []string              `yaml:"watch_paths"`
	FileWatch       FileWatchConfig       `yaml:"file_watch"`
}

// EnvironmentConfig contains environment-specific settings
type EnvironmentConfig struct {
	LogLevel string `yaml:"log_level"`
	LogFile  string `yaml:"log_file"`
}

// ModelConfig contains model-specific settings
type ModelConfig struct {
	Provider          string                 `yaml:"provider"`
	Name             string                 `yaml:"name"`
	Parameters       map[string]interface{} `yaml:"parameters"`
	MaxTokens        int                    `yaml:"max_tokens"`
	Temperature      float64                `yaml:"temperature"`
	TopP             float64                `yaml:"top_p"`
	FrequencyPenalty float64                `yaml:"frequency_penalty"`
	PresencePenalty  float64                `yaml:"presence_penalty"`
}

// ToolConfig contains tool-specific settings
type ToolConfig struct {
	Path       string            `yaml:"path"`
	Parameters map[string]string `yaml:"parameters"`
}

// Assistant represents an assistant configuration
type Assistant struct {
	Name        string                 `yaml:"name"`
	Model       string                 `yaml:"model"`
	Parameters  map[string]interface{} `yaml:"parameters"`
	Description string                 `yaml:"description"`
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

// GetModelConfig returns the model configuration
func (c *Config) GetModelConfig() ModelConfig {
	return c.Model
}

// GetToolConfig returns the tool configuration
func (c *Config) GetToolConfig() ToolConfig {
	return c.Tools
}

// GetAssistantConfig returns the configuration for a specific assistant
func (c *Config) GetAssistantConfig(name string) (Assistant, bool) {
	assistant, ok := c.Assistants[name]
	return assistant, ok
}

// GetDefaultAssistant returns the default assistant configuration
func (c *Config) GetDefaultAssistant() (Assistant, bool) {
	if c.DefaultAssistant == "" {
		return Assistant{}, false
	}
	return c.GetAssistantConfig(c.DefaultAssistant)
}
