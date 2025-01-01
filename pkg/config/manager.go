package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Manager handles configuration loading and management
type Manager struct {
	mu     sync.RWMutex
	config *Config
	path   string
}

// NewManager creates a new configuration manager with the config directory path
func NewManager(configDir string) *Manager {
	return &Manager{
		config: &Config{},
		path:   filepath.Join(configDir, "config.yaml"),
	}
}

// Load loads configuration from the specified path
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	config, err := ParseConfig(data)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Set runtime config values
	config.Environment.ConfigDir = filepath.Dir(m.path)

	m.config = config
	return nil
}

// Get returns the current configuration
func (m *Manager) Get() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// Set updates the current configuration
func (m *Manager) Set(config *Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config = config
}

// Save saves the current configuration to the specified path
func (m *Manager) Save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, err := m.config.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(m.path), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(m.path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetEnvironment returns the environment configuration
func (m *Manager) GetEnvironment() EnvironmentConfig {
	return m.Get().GetEnvironment()
}

// GetModelConfig returns the model configuration for a provider and model
func (m *Manager) GetModelConfig(provider, model string) (ModelConfig, bool) {
	return m.Get().GetModelConfig(provider, model)
}

// GetToolConfig returns the tool configuration for a tool name
func (m *Manager) GetToolConfig(name string) (ToolConfig, bool) {
	return m.Get().GetToolConfig(name)
}

// GetToolEnv returns the environment variables for a tool
func (m *Manager) GetToolEnv(name string) map[string]string {
	return m.Get().GetToolEnv(name)
}

// GetSecurityConfig returns the security configuration
func (m *Manager) GetSecurityConfig() SecurityConfig {
	return m.Get().GetSecurityConfig()
}

// Validate validates the current configuration
func (m *Manager) Validate() error {
	return m.Get().Validate()
}
