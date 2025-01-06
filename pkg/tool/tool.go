package tool

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/butter-bot-machines/skylark/internal/builtins"
	"github.com/butter-bot-machines/skylark/pkg/sandbox"
	"github.com/fsnotify/fsnotify"
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
	watcher  *fsnotify.Watcher
	mu       sync.RWMutex
}

// NewManager creates a new tool manager
func NewManager(basePath string) (*Manager, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	m := &Manager{
		tools:    make(map[string]*Tool),
		basePath: basePath,
		watcher:  watcher,
	}

	// Start watching for tool changes
	go m.watchTools()

	return m, nil
}

// InitBuiltinTools extracts and initializes builtin tools
func (m *Manager) InitBuiltinTools() error {
	// Extract currentDateTime source to .skai/tools
	data, err := builtins.GetToolSource("currentdatetime")
	if err != nil {
		return fmt.Errorf("failed to read embedded source: %w", err)
	}

	// Extract to .skai/tools like any other tool
    toolDir := filepath.Join(m.basePath, "currentdatetime")
	if err := os.MkdirAll(toolDir, 0755); err != nil {
		return fmt.Errorf("failed to create tool directory: %w", err)
	}

	mainFile := filepath.Join(toolDir, "main.go")
	if err := os.WriteFile(mainFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write source: %w", err)
	}

	// Let the standard tool manager handle the rest
	// Initial compilation
    if err := m.Compile("currentdatetime"); err != nil {
		return fmt.Errorf("failed to compile tool: %w", err)
	}

	if err := m.watcher.Add(toolDir); err != nil {
		return fmt.Errorf("failed to watch tool directory: %w", err)
	}

	return nil
}

// watchTools monitors tool source files for changes
func (m *Manager) watchTools() {
	for {
		select {
		case event, ok := <-m.watcher.Events:
			if !ok {
				return
			}
			// Only handle .go file changes
			if filepath.Ext(event.Name) != ".go" {
				continue
			}
			// Get tool name from path
			toolName := filepath.Base(filepath.Dir(event.Name))
			// Recompile tool
			if err := m.Compile(toolName); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to compile tool %s: %v\n", toolName, err)
			}
		case err, ok := <-m.watcher.Errors:
			if !ok {
				return
			}
			fmt.Fprintf(os.Stderr, "Watcher error: %v\n", err)
		}
	}
}

// Close stops the tool manager and cleans up resources
func (m *Manager) Close() error {
	return m.watcher.Close()
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
	m.mu.Lock()
	m.tools[name] = tool
	m.mu.Unlock()
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
	m.mu.Lock()
	if tool, exists := m.tools[name]; exists {
		tool.LastBuilt = time.Now()
	}
	m.mu.Unlock()

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
