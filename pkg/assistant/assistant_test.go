package assistant

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/butter-bot-machines/skylark/pkg/parser"
	"github.com/butter-bot-machines/skylark/pkg/provider"
	"github.com/butter-bot-machines/skylark/pkg/provider/registry"
	"github.com/butter-bot-machines/skylark/pkg/sandbox"
	"github.com/butter-bot-machines/skylark/pkg/tool"
)

// mockProvider implements provider.Provider for testing
type mockProvider struct {
	response      string
	err           error
	verifyOptions func(*provider.RequestOptions) error
}

func (m *mockProvider) Send(ctx context.Context, prompt string, opts *provider.RequestOptions) (*provider.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.verifyOptions != nil {
		if err := m.verifyOptions(opts); err != nil {
			return nil, err
		}
	}
	return &provider.Response{
		Content: m.response,
		Usage: provider.Usage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
	}, nil
}

func (m *mockProvider) Close() error {
	return nil
}

func TestAssistantModelConfig(t *testing.T) {
	tests := []struct {
		name        string
		modelConfig string
		wantModel   string
	}{
		{
			name:        "simple model name",
			modelConfig: "gpt-3.5-turbo",
			wantModel:   "gpt-3.5-turbo",
		},
		{
			name:        "provider:model format",
			modelConfig: "openai:gpt-4",
			wantModel:   "gpt-4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary test directory
			tempDir := t.TempDir()
			assistantDir := filepath.Join(tempDir, "test-assistant")
			err := os.MkdirAll(assistantDir, 0755)
			if err != nil {
				t.Fatalf("Failed to create test directory: %v", err)
			}

			// Create test prompt.md with specific model
			promptContent := fmt.Sprintf(`---
name: test-assistant
description: A test assistant
model: %s
---
Test prompt content
`, tt.modelConfig)
			err = os.WriteFile(filepath.Join(assistantDir, "prompt.md"), []byte(promptContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create test prompt.md: %v", err)
			}

			// Create mock provider that verifies model config
			mockProvider := &mockProvider{
				response: "Test response",
				verifyOptions: func(opts *provider.RequestOptions) error {
					if opts.Model != tt.wantModel {
						return fmt.Errorf("expected model %s, got %s", tt.wantModel, opts.Model)
					}
					return nil
				},
			}

			// Create provider registry
			reg := registry.New()
			reg.Register("openai", func(model string) (provider.Provider, error) {
				return mockProvider, nil
			})

			// Create tool manager
			toolManager, err := tool.NewManager(tempDir)
			if err != nil {
				t.Fatalf("NewManager() error = %v", err)
			}
			defer toolManager.Close()

			// Create manager
			manager, err := NewManager(tempDir, toolManager, reg, &sandbox.NetworkPolicy{}, "openai")
			if err != nil {
				t.Fatalf("NewManager() error = %v", err)
			}

			// Get assistant
			assistant, err := manager.Get("test-assistant")
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}

			// Process command - this should use the assistant's model config
			_, err = assistant.Process(&parser.Command{Text: "test"})
			if err != nil {
				t.Fatalf("Process() error = %v", err)
			}
		})
	}
}

func TestAssistantManager(t *testing.T) {
	// Create temporary test directory
	tempDir := t.TempDir()
	assistantDir := filepath.Join(tempDir, "test-assistant")
	err := os.MkdirAll(assistantDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create tools directory
	toolsDir := filepath.Join(tempDir, "tools")
	err = os.MkdirAll(toolsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create tools directory: %v", err)
	}

	// Create test prompt.md
	promptContent := `---
name: test-assistant
description: A test assistant
model: gpt-4
tools:
  - currentdatetime
---
Test prompt content
`
	err = os.WriteFile(filepath.Join(assistantDir, "prompt.md"), []byte(promptContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test prompt.md: %v", err)
	}

	// Create test tool
	toolDir := filepath.Join(toolsDir, "currentdatetime")
	err = os.MkdirAll(toolDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create tool directory: %v", err)
	}

	// Create mock tool source
	mainGo := `package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--usage" {
		fmt.Print(` + "`" + `{"schema":{"name":"currentdatetime","description":"Returns current date and time","parameters":{"type":"object","properties":{"format":{"type":"string","description":"Optional time format string"}}}},"env":{}}` + "`" + `)
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "--health" {
		fmt.Print(` + "`" + `{"status":true,"details":"healthy"}` + "`" + `)
		return
	}

	// Read input
	input, _ := io.ReadAll(os.Stdin)
	
	// Parse input
	var data struct {
		Format string ` + "`json:\"format\"`" + `
	}
	json.Unmarshal(input, &data)

	// Return mock result
	fmt.Printf(` + "`" + `{"datetime":"2025-01-05T10:00:00Z"}` + "`" + `)
}`

	err = os.WriteFile(filepath.Join(toolDir, "main.go"), []byte(mainGo), 0644)
	if err != nil {
		t.Fatalf("Failed to create test prompt.md: %v", err)
	}

	// Create mock provider
	mockProvider := &mockProvider{
		response: "Test response",
	}

	// Create provider registry
	reg := registry.New()
	reg.Register("openai", func(model string) (provider.Provider, error) {
		return mockProvider, nil
	})

	// Create tool manager
	toolMgr, err := tool.NewManager(toolsDir)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer toolMgr.Close()

	// Create network policy for sandbox
	networkPolicy := &sandbox.NetworkPolicy{
		AllowOutbound: false,
		AllowInbound:  false,
	}

	// Create manager with provider registry
	manager, err := NewManager(tempDir, toolMgr, reg, networkPolicy, "openai")
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	// Test getting assistant
	assistant, err := manager.Get("test-assistant")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if assistant.Name != "test-assistant" {
		t.Errorf("Assistant name = %v, want test-assistant", assistant.Name)
	}

	if len(assistant.Tools) != 1 || assistant.Tools[0] != "currentdatetime" {
		t.Errorf("Assistant tools = %v, want [currentdatetime]", assistant.Tools)
	}

	// Test processing command
	cmd := &parser.Command{
		Assistant: "test-assistant",
		Text:      "Test command",
	}

	response, err := assistant.Process(cmd)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if response != "Test response" {
		t.Errorf("Process() response = %v, want 'Test response'", response)
	}

	// Test tool usage
	cmd = &parser.Command{
		Assistant: "test-assistant",
		Text:      "use currentdatetime",
	}

	// Mock provider response for tool usage
	mockProvider.response = "The current time is 2025-01-05T10:00:00Z"

	response, err = assistant.Process(cmd)
	if err != nil {
		t.Fatalf("Process() with tool error = %v", err)
	}

	if response != "The current time is 2025-01-05T10:00:00Z" {
		t.Errorf("Process() with tool response = %v, want 'The current time is 2025-01-05T10:00:00Z'", response)
	}
}
