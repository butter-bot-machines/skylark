package tool

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupTestTool(t *testing.T, name string) string {
	// Create temporary directory
	tempDir := t.TempDir()
	toolDir := filepath.Join(tempDir, "tools", name)
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
			"input": {
				"type": "object",
				"properties": {
					"text": {
						"type": "string"
					}
				},
				"required": ["text"]
			},
			"output": {
				"type": "object",
				"properties": {
					"result": {
						"type": "string"
					}
				}
			}
		}` + "`" + `
		fmt.Println(schema)
		return
	}

	if *health {
		os.Exit(0)
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

	// Process input
	output := Output{
		Result: "Processed: " + input.Text,
	}

	// Write output
	result, err := json.Marshal(output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding output: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(result))
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
	manager := NewManager(basePath)

	// Test compilation
	err := manager.Compile(toolName)
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
	if tool.Schema.Input.Type != "object" {
		t.Errorf("Schema input type = %v, want object", tool.Schema.Input.Type)
	}

	if len(tool.Schema.Input.Required) != 1 || tool.Schema.Input.Required[0] != "text" {
		t.Errorf("Schema required fields = %v, want [text]", tool.Schema.Input.Required)
	}

	// Test tool execution
	input := map[string]string{
		"text": "hello world",
	}
	inputJSON, _ := json.Marshal(input)

	output, err := tool.Execute(inputJSON)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var result map[string]string
	err = json.Unmarshal(output, &result)
	if err != nil {
		t.Fatalf("Failed to parse output: %v", err)
	}

	expectedResult := "Processed: hello world"
	if result["result"] != expectedResult {
		t.Errorf("Execute() result = %v, want %v", result["result"], expectedResult)
	}
}

func TestToolValidation(t *testing.T) {
	toolName := "test-tool"
	basePath := setupTestTool(t, toolName)
	manager := NewManager(basePath)

	err := manager.Compile(toolName)
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
	manager := NewManager(basePath)

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
