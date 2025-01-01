package assistant

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/butter-bot-machines/skylark/pkg/parser"
	"github.com/butter-bot-machines/skylark/pkg/provider"
	"github.com/butter-bot-machines/skylark/pkg/sandbox"
	"github.com/butter-bot-machines/skylark/pkg/tool"
)

// mockProvider implements provider.Provider for testing
type mockProvider struct {
	response string
	err      error
}

func (m *mockProvider) Send(ctx context.Context, prompt string) (*provider.Response, error) {
	if m.err != nil {
		return nil, m.err
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
  - summarize
---
Test prompt content
`
	err = os.WriteFile(filepath.Join(assistantDir, "prompt.md"), []byte(promptContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test prompt.md: %v", err)
	}

	// Create test tool
	toolDir := filepath.Join(toolsDir, "summarize")
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
		fmt.Print(` + "`" + `{"schema":{"name":"summarize","description":"Summarizes text","parameters":{"type":"object","properties":{"content":{"type":"string","description":"Text to summarize"}},"required":["content"]}},"env":{}}` + "`" + `)
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
		Content string ` + "`json:\"content\"`" + `
	}
	json.Unmarshal(input, &data)

	// Return mock result
	fmt.Printf(` + "`" + `{"result":"Summary of: %s"}` + "`" + `, data.Content)
}`

	err = os.WriteFile(filepath.Join(toolDir, "main.go"), []byte(mainGo), 0644)
	if err != nil {
		t.Fatalf("Failed to create test prompt.md: %v", err)
	}

	// Create mock provider
	mockProvider := &mockProvider{
		response: "Test response",
	}

	// Create tool manager
	toolMgr := tool.NewManager(toolsDir)

	// Create network policy for sandbox
	networkPolicy := &sandbox.NetworkPolicy{
		AllowOutbound: false,
		AllowInbound:  false,
	}

	// Create manager with network policy
	manager, err := NewManager(tempDir, toolMgr, mockProvider, networkPolicy)
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

	if len(assistant.Tools) != 1 || assistant.Tools[0] != "summarize" {
		t.Errorf("Assistant tools = %v, want [summarize]", assistant.Tools)
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
		Text:      "use summarize Test input",
	}

	// Mock provider response for tool usage
	mockProvider.response = "Summary: Test input summarized"

	response, err = assistant.Process(cmd)
	if err != nil {
		t.Fatalf("Process() with tool error = %v", err)
	}

	if response != "Summary: Test input summarized" {
		t.Errorf("Process() with tool response = %v, want 'Summary: Test input summarized'", response)
	}
}
