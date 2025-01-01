package assistant

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Assistant represents a configured assistant with its prompt and tools
type Assistant struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Model       string   `yaml:"model"`
	Tools      []string `yaml:"tools,omitempty"`
	Prompt      string   `yaml:"-"` // Loaded from prompt.md content after front-matter
	Config      Config   `yaml:"config,omitempty"`
}

// Config represents assistant-specific configuration
type Config struct {
	MaxTokens   int     `yaml:"max_tokens,omitempty"`
	Temperature float64 `yaml:"temperature,omitempty"`
	TopP        float64 `yaml:"top_p,omitempty"`
}

// Manager handles loading and managing assistants
type Manager struct {
	assistants map[string]*Assistant
	basePath   string
}

// NewManager creates a new assistant manager
func NewManager(basePath string) *Manager {
	return &Manager{
		assistants: make(map[string]*Assistant),
		basePath:   basePath,
	}
}

// Load loads an assistant from the specified path
func (m *Manager) Load(name string) (*Assistant, error) {
	// Check if already loaded
	if assistant, exists := m.assistants[name]; exists {
		return assistant, nil
	}

	// Construct path to assistant directory
	assistantPath := filepath.Join(m.basePath, "assistants", name)
	promptPath := filepath.Join(assistantPath, "prompt.md")

	// Check if prompt.md exists
	if _, err := os.Stat(promptPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("assistant %s not found: %w", name, err)
	}

	// Read and parse prompt.md
	content, err := os.ReadFile(promptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read prompt.md: %w", err)
	}

	// Parse front-matter and content
	assistant, err := parsePromptFile(name, content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse prompt.md: %w", err)
	}

	// Store in cache
	m.assistants[name] = assistant
	return assistant, nil
}

// parsePromptFile parses the YAML front-matter and content from prompt.md
func parsePromptFile(name string, content []byte) (*Assistant, error) {
	// Split content into front-matter and prompt
	parts := strings.Split(string(content), "---\n")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid prompt.md format: missing YAML front-matter")
	}

	// Parse YAML front-matter
	assistant := &Assistant{Name: name}
	if err := yaml.Unmarshal([]byte(parts[1]), assistant); err != nil {
		return nil, fmt.Errorf("invalid YAML front-matter: %w", err)
	}

	// Store prompt content (everything after front-matter)
	assistant.Prompt = strings.TrimSpace(parts[2])

	// Validate required fields
	if err := assistant.validate(); err != nil {
		return nil, err
	}

	return assistant, nil
}

// validate checks that required fields are present
func (a *Assistant) validate() error {
	if a.Name == "" {
		return fmt.Errorf("assistant name is required")
	}
	if a.Model == "" {
		return fmt.Errorf("model is required")
	}
	if a.Prompt == "" {
		return fmt.Errorf("prompt content is required")
	}
	return nil
}

// LoadKnowledge loads knowledge files for an assistant
func (m *Manager) LoadKnowledge(name string) (map[string][]byte, error) {
	knowledgePath := filepath.Join(m.basePath, "assistants", name, "knowledge")
	if _, err := os.Stat(knowledgePath); os.IsNotExist(err) {
		return nil, nil // Knowledge directory is optional
	}

	knowledge := make(map[string][]byte)
	err := filepath.Walk(knowledgePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read knowledge file %s: %w", path, err)
		}

		// Store relative path as key
		relPath, err := filepath.Rel(knowledgePath, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}
		knowledge[relPath] = content
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to load knowledge directory: %w", err)
	}

	return knowledge, nil
}

// GetAssistant returns an assistant by name, loading it if necessary
func (m *Manager) GetAssistant(name string) (*Assistant, error) {
	// Try to get from cache first
	if assistant, exists := m.assistants[name]; exists {
		return assistant, nil
	}

	// Load if not found
	return m.Load(name)
}

// MergeConfig merges assistant-specific config with global defaults
func (a *Assistant) MergeConfig(globalConfig Config) Config {
	config := a.Config

	// Apply defaults for unset values
	if config.MaxTokens == 0 {
		config.MaxTokens = globalConfig.MaxTokens
	}
	if config.Temperature == 0 {
		config.Temperature = globalConfig.Temperature
	}
	if config.TopP == 0 {
		config.TopP = globalConfig.TopP
	}

	return config
}
