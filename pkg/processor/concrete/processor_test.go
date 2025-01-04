package concrete

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/butter-bot-machines/skylark/pkg/config"
	"github.com/butter-bot-machines/skylark/pkg/parser"
	"github.com/butter-bot-machines/skylark/pkg/processor"
)

func TestProcessor(t *testing.T) {
	// Create test directories
	configDir := t.TempDir()
	assistantDir := filepath.Join(configDir, "assistants", "test")
	if err := os.MkdirAll(assistantDir, 0755); err != nil {
		t.Fatalf("Failed to create assistant directory: %v", err)
	}

	// Create test prompt file
	promptFile := filepath.Join(assistantDir, "prompt.md")
	promptContent := `---
name: Test Assistant
description: Assistant for testing
model: gpt-4
---

Test prompt`
	if err := os.WriteFile(promptFile, []byte(promptContent), 0644); err != nil {
		t.Fatalf("Failed to create prompt file: %v", err)
	}

	// Create test config
	cfg := &config.Config{
		Environment: config.EnvironmentConfig{
			ConfigDir: configDir,
		},
		Models: map[string]config.ModelConfigSet{
			"openai": {
				"gpt-4": config.ModelConfig{
					APIKey:      "test-key",
					Temperature: 0.7,
					MaxTokens:   2000,
					TopP:        1.0,
				},
			},
		},
	}

	// Create processor
	proc, err := NewProcessor(cfg)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	t.Run("process command", func(t *testing.T) {
		cmd := &parser.Command{
			Original:  "!test command",
			Assistant: "test",
			Text:      "command",
		}

		response, err := proc.Process(cmd)
		if err != nil {
			t.Errorf("Failed to process command: %v", err)
		}
		expected := "command"
		if response != expected {
			t.Errorf("Expected response %q, got %q", expected, response)
		}
	})

	t.Run("process file", func(t *testing.T) {
		// Create test file
		testFile := filepath.Join(t.TempDir(), "test.md")
		content := "# Test\n!test command\n"
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Process file
		err := proc.ProcessFile(testFile)
		if err != nil {
			t.Errorf("Failed to process file: %v", err)
		}

		// Verify file was updated
		updated, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read updated file: %v", err)
		}

		// File should end with newline
		if string(updated[len(updated)-1]) != "\n" {
			t.Error("File should end with newline")
		}
	})

	t.Run("process directory", func(t *testing.T) {
		// Create test directory
		testDir := t.TempDir()

		// Create test files
		files := []struct {
			name    string
			content string
		}{
			{"test1.md", "# Test 1\n!test command 1\n"},
			{"test2.md", "# Test 2\n!test command 2\n"},
			{"test.txt", "Not a markdown file"},
		}

		for _, f := range files {
			path := filepath.Join(testDir, f.name)
			if err := os.WriteFile(path, []byte(f.content), 0644); err != nil {
				t.Fatalf("Failed to create test file %s: %v", f.name, err)
			}
		}

		// Process directory
		err := proc.ProcessDirectory(testDir)
		if err != nil {
			t.Errorf("Failed to process directory: %v", err)
		}

		// Verify only markdown files were processed
		for _, f := range files {
			path := filepath.Join(testDir, f.name)
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("Failed to read file %s: %v", f.name, err)
			}

			if filepath.Ext(f.name) == ".md" {
				// Markdown files should end with newline
				if string(content[len(content)-1]) != "\n" {
					t.Errorf("File %s should end with newline", f.name)
				}
			} else {
				// Non-markdown files should be unchanged
				if string(content) != f.content {
					t.Errorf("File %s should be unchanged", f.name)
				}
			}
		}
	})

	t.Run("handle response", func(t *testing.T) {
		cmd := &parser.Command{
			Original:  "!test command",
			Assistant: "test",
			Text:      "command",
		}

		// Test valid response
		err := proc.HandleResponse(cmd, "test response")
		if err != nil {
			t.Errorf("Failed to handle valid response: %v", err)
		}

		// Test nil command
		err = proc.HandleResponse(nil, "test response")
		if err == nil {
			t.Error("Expected error for nil command")
		}

		// Test empty response
		err = proc.HandleResponse(cmd, "")
		if err == nil {
			t.Error("Expected error for empty response")
		}
	})

	t.Run("update file", func(t *testing.T) {
		// Create test file
		testFile := filepath.Join(t.TempDir(), "test.md")
		content := "# Test\n!test command\nSome text\n!another command\n"
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Create responses
		responses := []processor.Response{
			{
				Command: &parser.Command{
					Original:  "!test command",
					Assistant: "test",
					Text:      "command",
				},
				Response: "command",
			},
			{
				Command: &parser.Command{
					Original:  "!another command",
					Assistant: "test",
					Text:      "another command",
				},
				Response: "another command",
			},
		}

		// Update file
		err := proc.UpdateFile(testFile, responses)
		if err != nil {
			t.Errorf("Failed to update file: %v", err)
		}

		// Verify file was updated correctly
		updated, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read updated file: %v", err)
		}

		expected := "# Test\n-!test command\n\ncommand\n\nSome text\n-!another command\n\nanother command\n"
		if string(updated) != expected {
			t.Errorf("File content mismatch\nExpected:\n%s\nGot:\n%s", expected, string(updated))
		}
	})

	t.Run("get process manager", func(t *testing.T) {
		mgr := proc.GetProcessManager()
		if mgr == nil {
			t.Error("Expected non-nil process manager")
		}
	})
}

func TestProcessorErrors(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		_, err := NewProcessor(nil)
		if err == nil {
			t.Error("Expected error for nil config")
		}
	})

	t.Run("invalid file", func(t *testing.T) {
		cfg := &config.Config{
			Environment: config.EnvironmentConfig{
				ConfigDir: t.TempDir(),
			},
			Models: map[string]config.ModelConfigSet{
				"openai": {
					"gpt-4": config.ModelConfig{
						APIKey:      "test-key",
						Temperature: 0.7,
						MaxTokens:   2000,
						TopP:        1.0,
					},
				},
			},
		}

		proc, err := NewProcessor(cfg)
		if err != nil {
			t.Fatalf("Failed to create processor: %v", err)
		}

		err = proc.ProcessFile("/nonexistent/file.md")
		if err == nil {
			t.Error("Expected error for nonexistent file")
		}
	})

	t.Run("invalid directory", func(t *testing.T) {
		cfg := &config.Config{
			Environment: config.EnvironmentConfig{
				ConfigDir: t.TempDir(),
			},
			Models: map[string]config.ModelConfigSet{
				"openai": {
					"gpt-4": config.ModelConfig{
						APIKey:      "test-key",
						Temperature: 0.7,
						MaxTokens:   2000,
						TopP:        1.0,
					},
				},
			},
		}

		proc, err := NewProcessor(cfg)
		if err != nil {
			t.Fatalf("Failed to create processor: %v", err)
		}

		err = proc.ProcessDirectory("/nonexistent/dir")
		if err == nil {
			t.Error("Expected error for nonexistent directory")
		}
	})
}
