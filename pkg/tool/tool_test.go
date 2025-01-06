package tool

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/butter-bot-machines/skylark/pkg/sandbox"
)

func setupTestTool(t *testing.T, name string) string {
	// Create temporary directory
	tempDir := t.TempDir()
	toolDir := filepath.Join(tempDir, name)
	err := os.MkdirAll(toolDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create main.go with a simple tool implementation
	mainContent := `package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
)

type Input struct {
	Text string ` + "`json:\"text\"`" + `
}

type Output struct {
	Result string ` + "`json:\"result\"`" + `
}

func main() {
	usage := flag.Bool("usage", false, "Print JSON schema")
	health := flag.Bool("health", false, "Run health check")
	flag.Parse()

	if *usage {
		schema := ` + "`" + `{
			"schema": {
				"name": "test-tool",
				"description": "A test tool implementation",
				"parameters": {
					"type": "object",
					"properties": {
						"text": {
							"type": "string",
							"description": "Input text to process"
						}
					},
					"required": ["text"]
				}
			},
			"env": {
				"API_KEY": {
					"type": "string",
					"description": "API key for external service",
					"default": "test-key"
				},
				"TIMEOUT": {
					"type": "integer",
					"description": "Operation timeout in seconds",
					"default": 30
				}
			}
		}` + "`" + `
		fmt.Println(schema)
		return
	}

	if *health {
		status := map[string]interface{}{
			"status": true,
			"details": "All systems operational",
		}
		json.NewEncoder(os.Stdout).Encode(status)
		return
	}

	// Read input
	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}

	var input Input
	if err := json.Unmarshal(data, &input); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing input: %v\n", err)
		os.Exit(1)
	}

	// Process input using environment
	apiKey := os.Getenv("API_KEY")
	output := Output{
		Result: fmt.Sprintf("Processed with %s: %s", apiKey, input.Text),
	}

	// Write output
	json.NewEncoder(os.Stdout).Encode(output)
}
`

	err = os.WriteFile(filepath.Join(toolDir, "main.go"), []byte(mainContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create main.go: %v", err)
	}

	return tempDir
}

func TestToolManager(t *testing.T) {
	toolName := "test-tool"
	basePath := setupTestTool(t, toolName)
	manager, err := NewManager(basePath)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()

	// Test compilation
	err = manager.Compile(toolName)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	// Test loading
	tool, err := manager.LoadTool(toolName)
	if err != nil {
		t.Fatalf("LoadTool() error = %v", err)
	}

	if tool.Name != toolName {
		t.Errorf("Tool name = %v, want %v", tool.Name, toolName)
	}

	// Test schema loading
	params := tool.Schema.Schema.Parameters
	if params["type"] != "object" {
		t.Errorf("Schema type = %v, want object", params["type"])
	}

	required, ok := params["required"].([]interface{})
	if !ok || len(required) != 1 || required[0] != "text" {
		t.Errorf("Schema required fields = %v, want [text]", required)
	}

	// Test environment variables
	if len(tool.Schema.Env) != 2 {
		t.Errorf("Expected 2 environment variables, got %d", len(tool.Schema.Env))
	}

	apiKey, exists := tool.Schema.Env["API_KEY"]
	if !exists || apiKey.Type != "string" || apiKey.Default != "test-key" {
		t.Errorf("Invalid API_KEY environment spec: %+v", apiKey)
	}

	// Test tool execution with environment
	input := map[string]string{
		"text": "hello world",
	}
	inputJSON, _ := json.Marshal(input)

	env := map[string]string{
		"API_KEY": "test-execution-key",
	}

	// Create sandbox for test
	sb, err := sandbox.NewSandbox(basePath, &sandbox.DefaultLimits, &sandbox.NetworkPolicy{
		AllowOutbound: false,
		AllowInbound:  false,
	})
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}

	output, err := tool.Execute(inputJSON, env, sb)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var result map[string]string
	err = json.Unmarshal(output, &result)
	if err != nil {
		t.Fatalf("Failed to parse output: %v", err)
	}

	expectedResult := "Processed with test-execution-key: hello world"
	if result["result"] != expectedResult {
		t.Errorf("Execute() result = %v, want %v", result["result"], expectedResult)
	}
}

func TestToolValidation(t *testing.T) {
	toolName := "test-tool"
	basePath := setupTestTool(t, toolName)
	manager, err := NewManager(basePath)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()

	err = manager.Compile(toolName)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	tool, err := manager.LoadTool(toolName)
	if err != nil {
		t.Fatalf("LoadTool() error = %v", err)
	}

	tests := []struct {
		name      string
		input     map[string]interface{}
		wantError bool
	}{
		{
			name: "valid input",
			input: map[string]interface{}{
				"text": "hello world",
			},
			wantError: false,
		},
		{
			name: "missing required field",
			input: map[string]interface{}{
				"other": "value",
			},
			wantError: true,
		},
		{
			name:      "invalid json",
			input:     nil,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var input []byte
			var err error
			if tt.input != nil {
				input, err = json.Marshal(tt.input)
				if err != nil {
					t.Fatalf("Failed to marshal input: %v", err)
				}
			} else {
				input = []byte("invalid json")
			}

			err = tool.ValidateInput(input)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateInput() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestToolCaching(t *testing.T) {
	toolName := "test-tool"
	basePath := setupTestTool(t, toolName)
	manager, err := NewManager(basePath)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()

	// First load
	tool1, err := manager.LoadTool(toolName)
	if err != nil {
		t.Fatalf("First LoadTool() error = %v", err)
	}

	// Second load should return cached instance
	tool2, err := manager.LoadTool(toolName)
	if err != nil {
		t.Fatalf("Second LoadTool() error = %v", err)
	}

	if tool1 != tool2 {
		t.Error("Second load returned different instance")
	}

	// Compile should update LastBuilt
	time.Sleep(time.Millisecond) // Ensure time difference
	err = manager.Compile(toolName)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	if !tool1.LastBuilt.After(time.Time{}) {
		t.Error("LastBuilt not updated after compilation")
	}
}

func TestBuiltinTools(t *testing.T) {
	// Create test directory
	basePath := t.TempDir()
	
	// Create manager
	manager, err := NewManager(basePath)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()

	// Initialize builtin tools - this will extract and compile the tool
	err = manager.InitBuiltinTools()
	if err != nil {
		t.Fatalf("InitBuiltinTools() error = %v", err)
	}

	// Verify the tool was compiled successfully by checking if binary exists
	binaryPath := filepath.Join(basePath, "currentdatetime", "currentdatetime")
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Fatalf("Tool binary not created at %s", binaryPath)
	}

	// Test loading currentdatetime tool
    tool, err := manager.LoadTool("currentdatetime")
	if err != nil {
		t.Fatalf("LoadTool() error = %v", err)
	}

	// Test schema
    if tool.Schema.Schema.Name != "currentdatetime" {
        t.Errorf("Tool name = %v, want currentdatetime", tool.Schema.Schema.Name)
	}

	// Test execution
	input := map[string]string{
		"format": "2006-01-02",
	}
	inputJSON, _ := json.Marshal(input)

	// Create sandbox for test
	sb, err := sandbox.NewSandbox(basePath, &sandbox.DefaultLimits, &sandbox.NetworkPolicy{
		AllowOutbound: false,
		AllowInbound:  false,
	})
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}

	output, err := tool.Execute(inputJSON, nil, sb)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var result map[string]string
	err = json.Unmarshal(output, &result)
	if err != nil {
		t.Fatalf("Failed to parse output: %v", err)
	}

	// Verify date format
	_, err = time.Parse("2006-01-02", result["datetime"])
	if err != nil {
		t.Errorf("Invalid date format: %v", err)
	}
}
