package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/butter-bot-machines/skylark/pkg/config"
)

const Version = "0.1.0"

// CLI represents the command-line interface
type CLI struct {
	config *config.Manager
}

// NewCLI creates a new CLI instance
func NewCLI() *CLI {
	return &CLI{}
}

// Run executes the CLI with the given arguments
func (c *CLI) Run(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("expected 'init', 'watch', 'run' or 'version' subcommands")
	}

	switch args[0] {
	case "init":
		return c.Init(args[1:])
	case "watch":
		return c.Watch(args[1:])
	case "run":
		return c.RunOnce(args[1:])
	case "version":
		return c.Version(args[1:])
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

// Init initializes a new Skylark project
func (c *CLI) Init(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("project name required")
	}
	projectName := args[0]

	// Create project directory
	if err := os.MkdirAll(projectName, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Create .skai directory structure
	skaiDir := filepath.Join(projectName, ".skai")
	dirs := []string{
		filepath.Join(skaiDir, "assistants", "default"),
		filepath.Join(skaiDir, "assistants", "default", "knowledge"),
		filepath.Join(skaiDir, "tools"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create default config.yaml
	configContent := `version: "1.0"

environment:
  OPENAI_API_KEY: "${OPENAI_API_KEY}"

model:
  provider: "openai"
  name: "gpt-4"
  max_tokens: 2000
  temperature: 0.7
  top_p: 0.9

tools:
  max_timeout: 30
  environment:
    DEFAULT_TIMEOUT: "30"
  defaults:
    retry_count: 3
    retry_delay: 1000

assistants:
  default: "default"
  environment:
    MODEL_VERSION: "latest"
  parameters:
    max_context_size: 4000
    max_references: 10
`
	if err := os.WriteFile(filepath.Join(skaiDir, "config.yaml"), []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to create config.yaml: %w", err)
	}

	// Create default assistant prompt.md
	promptContent := `---
name: default
description: Default assistant for general tasks
model: gpt-4
---
You are a helpful assistant that provides accurate and concise information.

When processing commands, you should:
1. Understand the user's request thoroughly
2. Consider any provided context
3. Use available tools when appropriate
4. Provide clear, well-structured responses
`
	if err := os.WriteFile(filepath.Join(skaiDir, "assistants", "default", "prompt.md"), []byte(promptContent), 0644); err != nil {
		return fmt.Errorf("failed to create prompt.md: %w", err)
	}

	fmt.Printf("Initialized Skylark project in %s\n", projectName)
	return nil
}

// Watch starts watching for file changes
func (c *CLI) Watch(args []string) error {
	// Load configuration
	if err := c.loadConfig(); err != nil {
		return err
	}

	// TODO: Implement file watching
	return fmt.Errorf("watch command not implemented yet")
}

// RunOnce processes files once without watching
func (c *CLI) RunOnce(args []string) error {
	// Load configuration
	if err := c.loadConfig(); err != nil {
		return err
	}

	// TODO: Implement one-time processing
	return fmt.Errorf("run command not implemented yet")
}

// Version displays version information
func (c *CLI) Version(args []string) error {
	fmt.Printf("Skylark version %s\n", Version)
	return nil
}

// loadConfig loads and validates the configuration
func (c *CLI) loadConfig() error {
	// Find .skai directory
	dir, err := findSkaiDir()
	if err != nil {
		return err
	}

	// Load configuration
	c.config = config.NewManager(dir)
	if err := c.config.Load(); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	return nil
}

// findSkaiDir finds the nearest .skai directory
func findSkaiDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, ".skai")); err == nil {
			return filepath.Join(dir, ".skai"), nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf(".skai directory not found")
		}
		dir = parent
	}
}
