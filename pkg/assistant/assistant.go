package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/butter-bot-machines/skylark/pkg/logging"
	"github.com/butter-bot-machines/skylark/pkg/parser"
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
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Model       string            `yaml:"model"`
	Tools       []string          `yaml:"tools,omitempty"`
	Prompt      string            `yaml:"-"` // Loaded from prompt.md content
	toolMgr     toolManager       // Tool manager
	provider    provider.Provider // AI provider
	sandbox     *sandbox.Sandbox  // Tool sandbox
	logger      *slog.Logger      // Logger
}

// Manager handles loading and managing assistants
type Manager struct {
	assistants map[string]*Assistant
	basePath   string
	toolMgr    *tool.Manager
	provider   provider.Provider
	sandbox    *sandbox.Sandbox
	logger     *slog.Logger
}

// NewManager creates a new assistant manager
func NewManager(basePath string, toolMgr *tool.Manager, p provider.Provider, network *sandbox.NetworkPolicy) (*Manager, error) {
	// Create sandbox
	sb, err := sandbox.NewSandbox(filepath.Join(basePath, "tools"), &sandbox.DefaultLimits, network)
	if err != nil {
		return nil, fmt.Errorf("failed to create sandbox: %w", err)
	}

	return &Manager{
		assistants: make(map[string]*Assistant),
		basePath:   basePath,
		toolMgr:    toolMgr,
		provider:   p,
		sandbox:    sb,
		logger:     logging.NewLogger(&logging.Options{Level: slog.LevelDebug}),
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
	assistant.provider = m.provider
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
			return "", fmt.Errorf("tool execution failed: %w", err)
		}

		// Include tool result in context
		cmd.Text = fmt.Sprintf("%s\nTool result: %s", cmd.Text, result)
	}

	// Build context with any references
	ctx := context.Background()
	prompt := a.buildPrompt(cmd)

	// Get response from provider
	resp, err := a.provider.Send(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("provider error: %w", err)
	}

	// Handle tool calls if present
	if len(resp.ToolCalls) > 0 {
		// Execute each tool
		for _, call := range resp.ToolCalls {
			result, err := a.executeTool(call.Function.Name, call.Function.Arguments)
			if err != nil {
				return "", fmt.Errorf("tool execution failed: %w", err)
			}

			// Include tool result in context
			cmd.Text = fmt.Sprintf("%s\nTool '%s' result: %s",
				cmd.Text, call.Function.Name, result)
		}

		// Get final response with tool results
		prompt = a.buildPrompt(cmd)
		resp, err = a.provider.Send(ctx, prompt)
		if err != nil {
			return "", fmt.Errorf("provider error after tools: %w", err)
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

	// Prepare input
	inputJSON, err := json.Marshal(map[string]string{
		"content": input,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal input: %w", err)
	}

	// Validate input
	if err := tool.ValidateInput(inputJSON); err != nil {
		return "", fmt.Errorf("invalid tool input: %w", err)
	}

	// Execute in sandbox
	output, err := tool.Execute(inputJSON, nil, a.sandbox)
	if err != nil {
		return "", fmt.Errorf("tool execution failed: %w", err)
	}

	return string(output), nil
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
