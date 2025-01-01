package tool

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/butter-bot-machines/skylark/pkg/sandbox"
)

// Tool represents a compiled tool binary and its metadata
type Tool struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Version     string    `json:"version"`
	LastBuilt   time.Time `json:"last_built"`
	Description string    `json:"description"`
	Schema      Schema    `json:"schema"`
}

// Schema represents the tool's schema and environment requirements
type Schema struct {
	Schema struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		Parameters  map[string]interface{} `json:"parameters"`
	} `json:"schema"`
	Env map[string]EnvVar `json:"env"`
}

// EnvVar represents an environment variable requirement
type EnvVar struct {
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Default     interface{} `json:"default,omitempty"`
}

// Manager handles tool compilation and execution
type Manager struct {
	tools    map[string]*Tool
	basePath string
}

// NewManager creates a new tool manager
func NewManager(basePath string) *Manager {
	return &Manager{
		tools:    make(map[string]*Tool),
		basePath: basePath,
	}
}

// LoadTool loads a tool from the specified directory
func (m *Manager) LoadTool(name string) (*Tool, error) {
	// Check if already loaded
	if tool, exists := m.tools[name]; exists {
		return tool, nil
	}

	toolPath := filepath.Join(m.basePath, name)
	mainFile := filepath.Join(toolPath, "main.go")

	// Check if main.go exists
	if _, err := os.Stat(mainFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("tool %s not found: %w", name, err)
	}

	// Create tool instance
	tool := &Tool{
		Name: name,
		Path: toolPath,
	}

	// Compile the tool first
	if err := m.Compile(name); err != nil {
		return nil, fmt.Errorf("failed to compile tool: %w", err)
	}

	// Load schema from --usage
	if err := tool.loadSchema(); err != nil {
		return nil, fmt.Errorf("failed to load schema: %w", err)
	}

	// Check health
	if err := tool.checkHealth(); err != nil {
		return nil, fmt.Errorf("health check failed: %w", err)
	}

	// Store in cache
	m.tools[name] = tool
	return tool, nil
}

// Compile compiles the tool's source code
func (m *Manager) Compile(name string) error {
	toolPath := filepath.Join(m.basePath, name)
	mainFile := filepath.Join(toolPath, "main.go")
	binaryPath := filepath.Join(toolPath, name)

	cmd := exec.Command("go", "build", "-o", binaryPath, mainFile)
	cmd.Dir = toolPath // Set working directory to tool path

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("compilation failed: %s: %w", output, err)
	}

	// Update tool metadata if loaded
	if tool, exists := m.tools[name]; exists {
		tool.LastBuilt = time.Now()
	}

	return nil
}

// loadSchema executes the tool with --usage flag to get JSON schema
func (t *Tool) loadSchema() error {
	binaryPath := filepath.Join(t.Path, t.Name)
	cmd := exec.Command(binaryPath, "--usage")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get usage: %w", err)
	}

	if err := json.Unmarshal(output, &t.Schema); err != nil {
		return fmt.Errorf("invalid schema format: %w", err)
	}

	return nil
}

// checkHealth executes the tool with --health flag
func (t *Tool) checkHealth() error {
	binaryPath := filepath.Join(t.Path, t.Name)
	cmd := exec.Command(binaryPath, "--health")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	var status struct {
		Status  bool   `json:"status"`
		Details string `json:"details"`
	}
	if err := json.Unmarshal(output, &status); err != nil {
		return fmt.Errorf("invalid health check response: %w", err)
	}

	if !status.Status {
		return fmt.Errorf("tool unhealthy: %s", status.Details)
	}
	return nil
}

// Execute runs the tool with the provided input and environment
func (t *Tool) Execute(input []byte, env map[string]string, sb *sandbox.Sandbox) ([]byte, error) {
	binaryPath := filepath.Join(t.Path, t.Name)
	cmd := exec.Command(binaryPath)

	// Build environment from schema
	cmdEnv := make([]string, 0, len(t.Schema.Env)+1)
	
	// Add PATH for binary execution
	if path := os.Getenv("PATH"); path != "" {
		cmdEnv = append(cmdEnv, "PATH="+path)
	}
	for name, spec := range t.Schema.Env {
		// Try config value first
		if value, ok := env[name]; ok {
			fmt.Printf("Using config value for %s: %s\n", name, value)
			cmdEnv = append(cmdEnv, fmt.Sprintf("%s=%s", name, value))
			continue
		}

		// Fall back to current environment
		if value := os.Getenv(name); value != "" {
			fmt.Printf("Using env value for %s: %s\n", name, value)
			cmdEnv = append(cmdEnv, fmt.Sprintf("%s=%s", name, value))
			continue
		}

		// Use default if available
		if spec.Default != nil {
			fmt.Printf("Using default value for %s: %v\n", name, spec.Default)
			cmdEnv = append(cmdEnv, fmt.Sprintf("%s=%v", name, spec.Default))
		}
	}

	fmt.Printf("Final env: %v\n", cmdEnv)
	cmd.Env = cmdEnv

	// Set up pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Create channel to signal stdin write completion
	done := make(chan error)

	// Write input in goroutine
	go func() {
		_, err := stdin.Write(input)
		stdin.Close()
		done <- err
	}()

	// Start reading output before executing
	outputCh := make(chan []byte)
	errCh := make(chan error)
	go func() {
		output, err := io.ReadAll(stdout)
		if err != nil {
			errCh <- fmt.Errorf("failed to read output: %w", err)
			return
		}
		outputCh <- output
	}()

	// Wait for stdin write to complete
	if err := <-done; err != nil {
		return nil, fmt.Errorf("failed to write input: %w", err)
	}

	// Execute in sandbox
	if err := sb.Execute(cmd); err != nil {
		return nil, fmt.Errorf("tool execution failed: %w", err)
	}

	// Get output or error
	select {
	case err := <-errCh:
		return nil, err
	case output := <-outputCh:
		return output, nil
	}
}

// ValidateInput checks if the input matches the tool's schema
func (t *Tool) ValidateInput(input []byte) error {
	var data map[string]interface{}
	if err := json.Unmarshal(input, &data); err != nil {
		return fmt.Errorf("invalid JSON input: %w", err)
	}

	params := t.Schema.Schema.Parameters
	if params["type"] != "object" {
		return fmt.Errorf("invalid schema: root must be object type")
	}

	if _, ok := params["properties"].(map[string]interface{}); !ok {
		return fmt.Errorf("invalid schema: missing properties")
	}

	if required, ok := params["required"].([]interface{}); ok {
		for _, field := range required {
			fieldName, ok := field.(string)
			if !ok {
				return fmt.Errorf("invalid required field type: %v", field)
			}
			if _, exists := data[fieldName]; !exists {
				return fmt.Errorf("missing required field: %s", fieldName)
			}
		}
	}

	// TODO: Add more thorough schema validation
	return nil
}
