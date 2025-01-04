package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/butter-bot-machines/skylark/pkg/security/types"
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

// GetConfig returns the current configuration
func (m *Manager) GetConfig() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// SetConfig updates the current configuration
func (m *Manager) SetConfig(config *Config) {
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
	return m.GetConfig().GetEnvironment()
}

// GetModelConfig returns the model configuration for a provider and model
func (m *Manager) GetModelConfig(provider, model string) (ModelConfig, bool) {
	return m.GetConfig().GetModelConfig(provider, model)
}

// GetToolConfig returns the tool configuration for a tool name
func (m *Manager) GetToolConfig(name string) (ToolConfig, bool) {
	return m.GetConfig().GetToolConfig(name)
}

// GetToolEnv returns the environment variables for a tool
func (m *Manager) GetToolEnv(name string) map[string]string {
	return m.GetConfig().GetToolEnv(name)
}

// GetSecurityConfig returns the security configuration
func (m *Manager) GetSecurityConfig() types.SecurityConfig {
	return m.GetConfig().GetSecurityConfig()
}

// Validate validates the current configuration
func (m *Manager) Validate() error {
	return m.GetConfig().Validate()
}

// Reset resets the configuration to default values
func (m *Manager) Reset() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config = &Config{}
	return nil
}

// Get gets a configuration value by key
func (m *Manager) Get(key string) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Split key into parts for nested access
	parts := strings.Split(key, ".")
	current := m.config.AsMap()
	
	for i, part := range parts[:len(parts)-1] {
		next, ok := current[part].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, strings.Join(parts[:i+1], "."))
		}
		current = next
	}
	
	value, ok := current[parts[len(parts)-1]]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, key)
	}
	return value, nil
}

// Set sets a configuration value by key
func (m *Manager) Set(key string, value interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Split key into parts for nested access
	parts := strings.Split(key, ".")
	current := m.config.AsMap()
	
	// Navigate to parent of target key
	for _, part := range parts[:len(parts)-1] {
		next, ok := current[part].(map[string]interface{})
		if !ok {
			next = make(map[string]interface{})
			current[part] = next
		}
		current = next
	}
	
	// Set value
	current[parts[len(parts)-1]] = value
	
	// Update config from map
	if err := m.config.FromMap(current); err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}
	
	return nil
}

// Delete deletes a configuration value by key
func (m *Manager) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Split key into parts for nested access
	parts := strings.Split(key, ".")
	current := m.config.AsMap()
	
	// Navigate to parent of target key
	for i, part := range parts[:len(parts)-1] {
		next, ok := current[part].(map[string]interface{})
		if !ok {
			return fmt.Errorf("%w: %s", ErrNotFound, strings.Join(parts[:i+1], "."))
		}
		current = next
	}
	
	// Delete key
	delete(current, parts[len(parts)-1])
	
	// Update config from map
	if err := m.config.FromMap(current); err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}
	
	return nil
}

// GetAll returns all configuration values as a map
func (m *Manager) GetAll() (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.AsMap(), nil
}

// SetAll sets all configuration values from a map
func (m *Manager) SetAll(values map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if err := m.config.FromMap(values); err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}
	return nil
}
