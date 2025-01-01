package assistant

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParsePromptFile(t *testing.T) {
	validContent := `---
name: test-assistant
description: A test assistant
model: gpt-4
tools:
  - summarize
  - search
config:
  max_tokens: 1000
  temperature: 0.7
---
You are a helpful assistant that provides accurate and concise information.
`

	tests := []struct {
		name        string
		content     string
		wantError   bool
		wantModel   string
		wantPrompt  string
		wantTools   int
		wantMaxTokens int
	}{
		{
			name:        "valid prompt file",
			content:     validContent,
			wantError:   false,
			wantModel:   "gpt-4",
			wantPrompt:  "You are a helpful assistant that provides accurate and concise information.",
			wantTools:   2,
			wantMaxTokens: 1000,
		},
		{
			name: "missing required fields",
			content: `---
description: Missing required fields
---
Prompt content
`,
			wantError: true,
		},
		{
			name:      "invalid yaml",
			content:   "---\nname: [invalid yaml\n---\nContent",
			wantError: true,
		},
		{
			name:      "missing front-matter",
			content:   "No front-matter here",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assistant, err := parsePromptFile("test", []byte(tt.content))
			if (err != nil) != tt.wantError {
				t.Errorf("parsePromptFile() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError {
				if assistant.Model != tt.wantModel {
					t.Errorf("Model = %v, want %v", assistant.Model, tt.wantModel)
				}
				if assistant.Prompt != tt.wantPrompt {
					t.Errorf("Prompt = %v, want %v", assistant.Prompt, tt.wantPrompt)
				}
				if len(assistant.Tools) != tt.wantTools {
					t.Errorf("Tools length = %v, want %v", len(assistant.Tools), tt.wantTools)
				}
				if assistant.Config.MaxTokens != tt.wantMaxTokens {
					t.Errorf("MaxTokens = %v, want %v", assistant.Config.MaxTokens, tt.wantMaxTokens)
				}
			}
		})
	}
}

func TestAssistantManager(t *testing.T) {
	// Create temporary test directory
	tempDir := t.TempDir()
	assistantDir := filepath.Join(tempDir, "assistants", "test-assistant")
	err := os.MkdirAll(assistantDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create test prompt.md
	promptContent := `---
name: test-assistant
description: A test assistant
model: gpt-4
---
Test prompt content
`
	err = os.WriteFile(filepath.Join(assistantDir, "prompt.md"), []byte(promptContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test prompt.md: %v", err)
	}

	// Create test knowledge directory
	knowledgeDir := filepath.Join(assistantDir, "knowledge")
	err = os.MkdirAll(knowledgeDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create knowledge directory: %v", err)
	}

	// Create test knowledge file
	err = os.WriteFile(filepath.Join(knowledgeDir, "test.txt"), []byte("Test knowledge content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test knowledge file: %v", err)
	}

	manager := NewManager(tempDir)

	// Test loading assistant
	assistant, err := manager.Load("test-assistant")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if assistant.Name != "test-assistant" {
		t.Errorf("Assistant name = %v, want test-assistant", assistant.Name)
	}

	// Test loading knowledge
	knowledge, err := manager.LoadKnowledge("test-assistant")
	if err != nil {
		t.Fatalf("LoadKnowledge() error = %v", err)
	}

	if len(knowledge) != 1 {
		t.Errorf("Knowledge files = %v, want 1", len(knowledge))
	}

	if content := string(knowledge["test.txt"]); content != "Test knowledge content" {
		t.Errorf("Knowledge content = %v, want 'Test knowledge content'", content)
	}

	// Test getting cached assistant
	cachedAssistant, err := manager.GetAssistant("test-assistant")
	if err != nil {
		t.Fatalf("GetAssistant() error = %v", err)
	}

	if cachedAssistant != assistant {
		t.Error("GetAssistant() returned different instance than Load()")
	}
}

func TestConfigMerging(t *testing.T) {
	assistant := &Assistant{
		Config: Config{
			MaxTokens: 500,
			// Temperature and TopP not set
		},
	}

	globalConfig := Config{
		MaxTokens:   1000,
		Temperature: 0.7,
		TopP:        0.9,
	}

	merged := assistant.MergeConfig(globalConfig)

	if merged.MaxTokens != 500 {
		t.Errorf("MaxTokens = %v, want 500 (assistant value should override)", merged.MaxTokens)
	}

	if merged.Temperature != 0.7 {
		t.Errorf("Temperature = %v, want 0.7 (should use global default)", merged.Temperature)
	}

	if merged.TopP != 0.9 {
		t.Errorf("TopP = %v, want 0.9 (should use global default)", merged.TopP)
	}
}
