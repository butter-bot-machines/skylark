package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/butter-bot-machines/skylark/pkg/logging"
	"github.com/butter-bot-machines/skylark/pkg/parser"
	"github.com/butter-bot-machines/skylark/pkg/provider/registry"
	"github.com/butter-bot-machines/skylark/pkg/provider"
	"github.com/butter-bot-machines/skylark/pkg/sandbox"
	"github.com/butter-bot-machines/skylark/pkg/tool"
	"gopkg.in/yaml.v3"
)

// toolManager defines what we need from a tool manager
type toolManager interface {
	LoadTool(name string) (*tool.Tool, error)
}

// Assistant represents a configured assistant
type Assistant struct {
	Name            string            `yaml:"name"`
	Description     string            `yaml:"description"`
	Model           string            `yaml:"model"`
	Tools           []string          `yaml:"tools,omitempty"`
	Prompt          string            `yaml:"-"` // Loaded from prompt.md content
	toolMgr         toolManager       // Tool manager
	providers       *registry.Registry // Provider registry
	defaultProvider string            // Default provider name
	sandbox         *sandbox.Sandbox  // Tool sandbox
	logger          *slog.Logger      // Logger
}

// Manager handles loading and managing assistants
type Manager struct {
	assistants      map[string]*Assistant
	basePath        string
	toolMgr         *tool.Manager
	providers       *registry.Registry
	defaultProvider string
	sandbox         *sandbox.Sandbox
	logger          *slog.Logger
}

// NewManager creates a new assistant manager
func NewManager(basePath string, toolMgr *tool.Manager, providers *registry.Registry, network *sandbox.NetworkPolicy, defaultProvider string) (*Manager, error) {
	// Create sandbox
	sb, err := sandbox.NewSandbox(filepath.Join(basePath, "tools"), &sandbox.DefaultLimits, network)
	if err != nil {
		return nil, fmt.Errorf("failed to create sandbox: %w", err)
	}

	return &Manager{
		assistants:      make(map[string]*Assistant),
		basePath:        basePath,
		toolMgr:         toolMgr,
		providers:       providers,
		defaultProvider: defaultProvider,
		sandbox:         sb,
		logger:         logging.NewLogger(&logging.Options{Level: slog.LevelDebug}),
	}, nil
}

// Get returns an assistant by name, loading it if necessary
func (m *Manager) Get(name string) (*Assistant, error) {
	// Check if already loaded
	if assistant, exists := m.assistants[name]; exists {
		return assistant, nil
	}

	// Load assistant
	assistant, err := m.loadAssistant(name)
	if err != nil {
		return nil, err
	}

	// Initialize assistant components
	assistant.toolMgr = m.toolMgr
	assistant.providers = m.providers
	assistant.defaultProvider = m.defaultProvider
	assistant.sandbox = m.sandbox
	assistant.logger = m.logger

	// Cache for future use
	m.assistants[name] = assistant
	return assistant, nil
}

// loadAssistant loads an assistant from its prompt.md file
func (m *Manager) loadAssistant(name string) (*Assistant, error) {
	promptPath := filepath.Join(m.basePath, name, "prompt.md")
	content, err := os.ReadFile(promptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read prompt.md: %w", err)
	}

	// Split front matter and prompt content
	parts := strings.Split(string(content), "---\n")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid prompt.md format: missing YAML front matter")
	}

	// Parse front matter
	assistant := &Assistant{Name: name}
	if err := yaml.Unmarshal([]byte(parts[1]), assistant); err != nil {
		return nil, fmt.Errorf("invalid YAML front matter: %w", err)
	}

	// Store prompt content
	assistant.Prompt = strings.TrimSpace(parts[2])

	return assistant, nil
}

// Process processes a command using this assistant
func (a *Assistant) Process(cmd *parser.Command) (string, error) {
	a.logger.Debug("processing command",
		"assistant", a.Name,
		"command", cmd.Text)

	// Check for tool usage in command
	toolName, toolInput := a.parseToolUsage(cmd.Text)
	if toolName != "" {
		// Execute tool
		result, err := a.executeTool(toolName, toolInput)
		if err != nil {
			return "", err // Don't wrap error to allow proper error propagation
		}

		// Include tool result in context
		cmd.Text = fmt.Sprintf("%s\nTool result: %s", cmd.Text, result)
	}

	// Build context with any references
	ctx := context.Background()
	prompt := a.buildPrompt(cmd)

	// Get provider for this assistant's model
	p, err := a.providers.CreateForModel(a.Model, a.defaultProvider)
	if err != nil {
		return "", fmt.Errorf("failed to create provider: %w", err)
	}
	defer p.Close()

	// Get model name without provider prefix
	_, modelName := registry.ParseModelSpec(a.Model)

	// Build request options from assistant config
	opts := &provider.RequestOptions{
		Model:       modelName,
		Temperature: 0.7,  // Default temperature
		MaxTokens:   2000, // Default max tokens
	}

	// Get response from provider
	resp, err := p.Send(ctx, prompt, opts)
	if err != nil {
		return "", fmt.Errorf("provider error: %w", err)
	}
	if resp.Error != nil {
		return "", fmt.Errorf("provider error: %v", resp.Error)
	}

	// Handle tool calls if present
	if len(resp.ToolCalls) > 0 {
		// Execute each tool
		for _, call := range resp.ToolCalls {
			result, err := a.executeTool(call.Function.Name, call.Function.Arguments)
			if err != nil {
				return "", err // Don't wrap error to allow proper error propagation
			}

			// Include tool result in context
			cmd.Text = fmt.Sprintf("%s\nTool '%s' result: %s",
				cmd.Text, call.Function.Name, result)
		}

		// Get final response with tool results
		prompt = a.buildPrompt(cmd)
		resp, err = p.Send(ctx, prompt, opts)
		if err != nil {
			return "", fmt.Errorf("provider error after tools: %w", err)
		}
		if resp.Error != nil {
			return "", fmt.Errorf("provider error after tools: %v", resp.Error)
		}
	}

	return resp.Content, nil
}

// parseToolUsage checks if a command wants to use a tool
func (a *Assistant) parseToolUsage(text string) (string, string) {
	// Simple parsing for now - look for "use <tool>" pattern
	if strings.HasPrefix(strings.ToLower(text), "use ") {
		parts := strings.SplitN(text[4:], " ", 2)
		if len(parts) == 2 {
			return parts[0], parts[1]
		}
		return parts[0], "" // Allow tool usage without arguments
	}
	return "", ""
}

// executeTool runs a tool in the sandbox
func (a *Assistant) executeTool(name string, input string) (string, error) {
	// Get tool
	tool, err := a.toolMgr.LoadTool(name)
	if err != nil {
		return "", fmt.Errorf("failed to load tool: %w", err)
	}

	// Prepare input JSON
	var inputJSON []byte
	if input == "" {
		// Empty input becomes empty object
		inputJSON = []byte("{}")
	} else {
		// Input must be valid JSON
		if !json.Valid([]byte(input)) {
			return "", fmt.Errorf("invalid JSON input: %s", input)
		}
		inputJSON = []byte(input)
	}

	// Validate input
	if err := tool.ValidateInput(inputJSON); err != nil {
		return "", fmt.Errorf("invalid tool input: %w", err)
	}

	// Execute in sandbox
	output, err := tool.Execute(inputJSON, nil, a.sandbox)
	if err != nil {
		return "", err // Don't wrap error to allow proper error propagation
	}

	// Validate output is JSON
	var prettyOutput bytes.Buffer
	if err := json.Indent(&prettyOutput, output, "", "  "); err != nil {
		// Not JSON, return as-is
		return string(output), nil
	}

	return prettyOutput.String(), nil
}

// buildPrompt creates the full prompt with context
func (a *Assistant) buildPrompt(cmd *parser.Command) string {
	var b strings.Builder

	// Add system prompt
	b.WriteString(a.Prompt)
	b.WriteString("\n\n")

	// Add available tools
	if len(a.Tools) > 0 {
		b.WriteString("Available tools:\n")
		for _, tool := range a.Tools {
			b.WriteString(fmt.Sprintf("- %s\n", tool))
		}
		b.WriteString("\n")
	}

	// Add command and any references
	b.WriteString("Command: ")
	b.WriteString(cmd.Text)
	b.WriteString("\n")

	return b.String()
}
