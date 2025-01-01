package sandbox

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestNewSandbox(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name      string
		limits    *ResourceLimits
		network   *NetworkPolicy
		wantError bool
	}{
		{
			name:      "default limits",
			limits:    nil,
			network:   &NetworkPolicy{},
			wantError: false,
		},
		{
			name: "custom limits",
			limits: &ResourceLimits{
				MaxCPUTime:    10 * time.Second,
				MaxMemoryMB:   256,
				MaxFileSizeMB: 5,
				MaxFiles:      50,
				MaxProcesses:  5,
			},
			network:   &NetworkPolicy{},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sandbox, err := NewSandbox(tempDir, tt.limits, tt.network)
			if (err != nil) != tt.wantError {
				t.Errorf("NewSandbox() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError {
				if sandbox.WorkDir != tempDir {
					t.Errorf("WorkDir = %v, want %v", sandbox.WorkDir, tempDir)
				}

				if tt.limits == nil {
					if sandbox.Limits != DefaultLimits {
						t.Error("Default limits not applied")
					}
				} else {
					if sandbox.Limits != *tt.limits {
						t.Error("Custom limits not applied")
					}
				}
			}
		})
	}
}

func TestSandboxExecution(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name      string
		cmd       string
		args      []string
		limits    ResourceLimits
		env       []string
		wantError bool
	}{
		{
			name: "simple command",
			cmd:  "echo",
			args: []string{"hello"},
			limits: ResourceLimits{
				MaxCPUTime: 1 * time.Second,
			},
			wantError: false,
		},
		{
			name: "timeout command",
			cmd:  "sleep",
			args: []string{"2"},
			limits: ResourceLimits{
				MaxCPUTime: 100 * time.Millisecond,
			},
			wantError: true,
		},
		{
			name: "environment filtering",
			cmd:  "env",
			limits: ResourceLimits{
				MaxCPUTime: 1 * time.Second,
			},
			env:       []string{"TEST_VAR=test"},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sandbox, err := NewSandbox(tempDir, &tt.limits, &NetworkPolicy{})
			if err != nil {
				t.Fatalf("Failed to create sandbox: %v", err)
			}

			sandbox.EnvWhitelist = tt.env

			cmd := exec.Command(tt.cmd, tt.args...)
			err = sandbox.Execute(cmd)
			if (err != nil) != tt.wantError {
				t.Errorf("Execute() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestVersionChecking(t *testing.T) {
	sandbox := &Sandbox{
		ToolVersion: "1.2.3",
	}

	tests := []struct {
		name       string
		minVersion string
		want       bool
	}{
		{
			name:       "exact match",
			minVersion: "1.2.3",
			want:       true,
		},
		{
			name:       "higher version",
			minVersion: "1.2.2",
			want:       true,
		},
		{
			name:       "lower version",
			minVersion: "1.2.4",
			want:       false,
		},
		{
			name:       "major version higher",
			minVersion: "0.9.9",
			want:       true,
		},
		{
			name:       "major version lower",
			minVersion: "2.0.0",
			want:       false,
		},
		{
			name:       "empty version",
			minVersion: "",
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sandbox.VerifyToolVersion(tt.minVersion); got != tt.want {
				t.Errorf("VerifyToolVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResultCaching(t *testing.T) {
	tempDir := t.TempDir()
	sandbox, err := NewSandbox(tempDir, nil, &NetworkPolicy{})
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}

	sandbox.CacheEnabled = true
	testData := []byte("test data")
	testKey := "test-key"

	// Test setting cache
	if err := sandbox.SetCachedResult(testKey, testData); err != nil {
		t.Errorf("SetCachedResult() error = %v", err)
	}

	// Test getting cache
	data, ok := sandbox.GetCachedResult(testKey)
	if !ok {
		t.Error("GetCachedResult() returned not ok, want ok")
	}
	if string(data) != string(testData) {
		t.Errorf("GetCachedResult() = %v, want %v", string(data), string(testData))
	}

	// Test cache expiration
	cacheFile := filepath.Join(sandbox.cacheDir, testKey)
	oldTime := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(cacheFile, oldTime, oldTime); err != nil {
		t.Fatalf("Failed to change cache file time: %v", err)
	}

	_, ok = sandbox.GetCachedResult(testKey)
	if ok {
		t.Error("GetCachedResult() returned ok for expired cache")
	}

	// Test disabled cache
	sandbox.CacheEnabled = false
	if err := sandbox.SetCachedResult("new-key", testData); err != nil {
		t.Error("SetCachedResult() returned error for disabled cache")
	}
	_, ok = sandbox.GetCachedResult("new-key")
	if ok {
		t.Error("GetCachedResult() returned ok for disabled cache")
	}
}

func TestCleanup(t *testing.T) {
	tempDir := t.TempDir()
	sandbox, err := NewSandbox(tempDir, nil, &NetworkPolicy{})
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}

	// Create some temporary files
	tempFiles := []string{
		filepath.Join(tempDir, "tmp.1"),
		filepath.Join(tempDir, "tmp.2"),
		filepath.Join(tempDir, "other.file"),
	}

	for _, file := range tempFiles {
		if err := os.WriteFile(file, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Run cleanup
	if err := sandbox.Cleanup(); err != nil {
		t.Errorf("Cleanup() error = %v", err)
	}

	// Check that tmp.* files are removed but other.file remains
	for _, file := range tempFiles {
		exists := true
		if _, err := os.Stat(file); os.IsNotExist(err) {
			exists = false
		}

		shouldExist := !filepath.HasPrefix(filepath.Base(file), "tmp.")
		if exists != shouldExist {
			t.Errorf("File %s exists = %v, want %v", file, exists, shouldExist)
		}
	}
}
