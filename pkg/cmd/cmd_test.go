package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIRun(t *testing.T) {
	cli := NewCLI()

	tests := []struct {
		name      string
		args      []string
		wantError bool
	}{
		{
			name:      "no arguments",
			args:      []string{},
			wantError: true,
		},
		{
			name:      "unknown command",
			args:      []string{"unknown"},
			wantError: true,
		},
		{
			name:      "version command",
			args:      []string{"version"},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cli.Run(tt.args)
			if (err != nil) != tt.wantError {
				t.Errorf("Run() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestCLIInit(t *testing.T) {
	cli := NewCLI()
	tempDir := t.TempDir()
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	// Change to temp directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	tests := []struct {
		name      string
		args      []string
		wantError bool
		check     func(t *testing.T, projectDir string)
	}{
		{
			name:      "no project name",
			args:      []string{},
			wantError: true,
		},
		{
			name:      "valid project name",
			args:      []string{"test-project"},
			wantError: false,
			check: func(t *testing.T, projectDir string) {
				// Check directory structure
				dirs := []string{
					".skai",
					".skai/assistants",
					".skai/assistants/default",
					".skai/assistants/default/knowledge",
					".skai/tools",
				}
				for _, dir := range dirs {
					path := filepath.Join(projectDir, dir)
					if _, err := os.Stat(path); os.IsNotExist(err) {
						t.Errorf("Directory %s not created", path)
					}
				}

				// Check config.yaml
				configPath := filepath.Join(projectDir, ".skai", "config.yaml")
				content, err := os.ReadFile(configPath)
				if err != nil {
					t.Errorf("Failed to read config.yaml: %v", err)
					return
				}
				if !strings.Contains(string(content), "version: \"1.0\"") {
					t.Error("config.yaml missing version")
				}

				// Check prompt.md
				promptPath := filepath.Join(projectDir, ".skai", "assistants", "default", "prompt.md")
				content, err = os.ReadFile(promptPath)
				if err != nil {
					t.Errorf("Failed to read prompt.md: %v", err)
					return
				}
				if !strings.Contains(string(content), "name: default") {
					t.Error("prompt.md missing name")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cli.Init(tt.args)
			if (err != nil) != tt.wantError {
				t.Errorf("Init() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError && tt.check != nil {
				tt.check(t, filepath.Join(tempDir, tt.args[0]))
			}
		})
	}
}

func TestFindSkaiDir(t *testing.T) {
	// Create temporary directory structure
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "project")
	skaiDir := filepath.Join(projectDir, ".skai")
	nestedDir := filepath.Join(projectDir, "nested", "dir")

	if err := os.MkdirAll(skaiDir, 0755); err != nil {
		t.Fatalf("Failed to create test directories: %v", err)
	}
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("Failed to create nested directories: %v", err)
	}

	// Save current directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	tests := []struct {
		name      string
		startDir  string
		wantError bool
	}{
		{
			name:      "in project root",
			startDir:  projectDir,
			wantError: false,
		},
		{
			name:      "in nested directory",
			startDir:  nestedDir,
			wantError: false,
		},
		{
			name:      "outside project",
			startDir:  tempDir,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := os.Chdir(tt.startDir); err != nil {
				t.Fatalf("Failed to change directory: %v", err)
			}

			dir, err := findSkaiDir()
			if (err != nil) != tt.wantError {
				t.Errorf("findSkaiDir() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError {
				want := filepath.Clean(skaiDir)
				got := filepath.Clean(dir)
				if got != want {
					t.Errorf("findSkaiDir() = %v, want %v", got, want)
				}
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	cli := NewCLI()
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "project")
	skaiDir := filepath.Join(projectDir, ".skai")

	// Create project structure
	if err := os.MkdirAll(skaiDir, 0755); err != nil {
		t.Fatalf("Failed to create test directories: %v", err)
	}

	// Create config.yaml
	configContent := `version: "1.0"
model:
  provider: "openai"
  name: "gpt-4"
  max_tokens: 2000
  temperature: 0.7
  top_p: 0.9
assistants:
  default: "default"
`
	if err := os.WriteFile(filepath.Join(skaiDir, "config.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config.yaml: %v", err)
	}

	// Save current directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	// Change to project directory
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Test loading configuration
	if err := cli.loadConfig(); err != nil {
		t.Errorf("loadConfig() error = %v", err)
	}

	if cli.config == nil {
		t.Error("loadConfig() did not set config")
	}
}