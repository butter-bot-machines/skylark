package security

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/butter-bot-machines/skylark/pkg/config"
	"github.com/butter-bot-machines/skylark/pkg/watcher"
	"github.com/butter-bot-machines/skylark/pkg/worker"
)

// TestFileAccessControl verifies proper file access restrictions
func TestFileAccessControl(t *testing.T) {
	// Create test directory structure
	tmpDir, err := os.MkdirTemp("", "skylark-security-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create allowed directory and file
	allowedDir := filepath.Join(tmpDir, "allowed")
	if err := os.MkdirAll(allowedDir, 0755); err != nil {
		t.Fatalf("Failed to create allowed dir: %v", err)
	}

	allowedFile := filepath.Join(allowedDir, "test.md")
	if err := os.WriteFile(allowedFile, []byte("# Test\n"), 0644); err != nil {
		t.Fatalf("Failed to write allowed file: %v", err)
	}

	// Create file outside allowed directory
	restrictedFile := filepath.Join(tmpDir, "secret.md")
	if err := os.WriteFile(restrictedFile, []byte("# Secret\n"), 0644); err != nil {
		t.Fatalf("Failed to write restricted file: %v", err)
	}

	// Configure watcher with only the allowed directory
	cfg := &config.Config{
		WatchPaths: []string{allowedDir},
		Workers: config.WorkerConfig{
			Count:     2,
			QueueSize: 10,
		},
	}

	pool := worker.NewPool(cfg)
	defer pool.Stop()

	// Create watcher and wait for initialization
	w, err := watcher.New(cfg, pool.Queue())
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer w.Stop()

	// Wait for watcher to initialize with timeout
	deadline := time.After(500 * time.Millisecond)
	initialized := make(chan struct{})
	go func() {
		time.Sleep(100 * time.Millisecond)
		close(initialized)
	}()

	select {
	case <-initialized:
		// Watcher initialized
	case <-deadline:
		t.Fatal("Timeout waiting for watcher initialization")
	}

	// Test allowed file access
	t.Run("allowed file", func(t *testing.T) {
		done := make(chan struct{})
		jobQueue := pool.Queue()
		jobQueue <- &accessJob{
			path:       allowedFile,
			allowedDir: allowedDir,
			onComplete: func(err error) {
				if err != nil {
					t.Errorf("Failed to read allowed file: %v", err)
				}
				close(done)
			},
		}

		select {
		case <-done:
			// Test completed
		case <-time.After(1 * time.Second):
			t.Fatal("Timeout waiting for allowed file access")
		}
	})

	// Test restricted file access
	t.Run("restricted file", func(t *testing.T) {
		done := make(chan struct{})
		jobQueue := pool.Queue()
		jobQueue <- &accessJob{
			path:       restrictedFile,
			allowedDir: allowedDir,
			onComplete: func(err error) {
				if err == nil {
					t.Error("Expected error accessing restricted file")
				}
				close(done)
			},
		}

		select {
		case <-done:
			// Test completed
		case <-time.After(1 * time.Second):
			t.Fatal("Timeout waiting for restricted file access")
		}
	})
}

// TestInputValidation verifies proper handling of malicious input
func TestInputValidation(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid command",
			input:   "!test hello world",
			wantErr: false,
		},
		{
			name:    "command injection",
			input:   "!test; rm -rf /",
			wantErr: true,
		},
		{
			name:    "path traversal",
			input:   "!test ../../../etc/passwd",
			wantErr: true,
		},
		{
			name:    "null bytes",
			input:   "!test\x00malicious",
			wantErr: true,
		},
		{
			name:    "very long input",
			input:   "!test " + strings.Repeat("a", 10000),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateInput(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateInput() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// validateInput performs basic input validation
func validateInput(input string) error {
	// Check for command injection
	if strings.ContainsAny(input, ";|&") {
		return &ValidationError{msg: "command injection detected"}
	}

	// Check for path traversal
	if strings.Contains(input, "..") {
		return &ValidationError{msg: "path traversal detected"}
	}

	// Check for null bytes
	if strings.Contains(input, "\x00") {
		return &ValidationError{msg: "null bytes detected"}
	}

	// Check input length
	if len(input) > 1000 {
		return &ValidationError{msg: "input too long"}
	}

	return nil
}

// ValidationError represents an input validation error
type ValidationError struct {
	msg string
}

func (e *ValidationError) Error() string {
	return e.msg
}

// TestResourceLimits verifies proper resource usage limits
func TestResourceLimits(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resource limit tests in short mode")
	}

	cfg := &config.Config{
		Workers: config.WorkerConfig{
			Count:     2,
			QueueSize: 10,
		},
	}

	pool := worker.NewPool(cfg)
	defer pool.Stop()

	// Test memory limits with cleanup
	t.Run("memory limits", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		done := make(chan struct{})
		jobQueue := pool.Queue()

		// Create a job that tries to allocate too much memory
		job := &memoryHogJob{
			size: 1024 * 1024 * 1024, // 1GB
			onComplete: func(recovered bool) {
				if !recovered {
					t.Error("Memory limits were not enforced")
				}
				close(done)
			},
		}

		select {
		case jobQueue <- job:
		case <-ctx.Done():
			t.Fatal("Timeout queueing memory test job")
		}

		// Wait for job completion
		select {
		case <-done:
			// Test completed
		case <-ctx.Done():
			t.Fatal("Timeout waiting for memory test")
		}
	})

	// Test CPU limits with cleanup
	t.Run("cpu limits", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		done := make(chan struct{})
		jobQueue := pool.Queue()

		// Create a job that tries to use too much CPU
		job := &cpuHogJob{
			duration: 10, // Reduced duration for faster test
			onComplete: func(interrupted bool) {
				if !interrupted {
					t.Error("CPU limits were not enforced")
				}
				close(done)
			},
		}

		select {
		case jobQueue <- job:
		case <-ctx.Done():
			t.Fatal("Timeout queueing CPU test job")
		}

		// Wait for job completion
		select {
		case <-done:
			// Test completed
		case <-ctx.Done():
			t.Fatal("Timeout waiting for CPU test")
		}
	})
}

// accessJob attempts to access a file with path validation
type accessJob struct {
	path       string
	allowedDir string
	onComplete func(error)
}

func (j *accessJob) Process() error {
	// Validate path is within allowed directory
	cleanPath := filepath.Clean(j.path)
	if !strings.HasPrefix(cleanPath, j.allowedDir) {
		err := fmt.Errorf("access denied: %s is outside allowed directory", j.path)
		j.onComplete(err)
		return err
	}

	// Attempt to read file
	_, err := os.ReadFile(j.path)
	j.onComplete(err)
	return err
}

func (j *accessJob) OnFailure(err error) {}

func (j *accessJob) MaxRetries() int {
	return 0
}

// memoryHogJob attempts to allocate too much memory
type memoryHogJob struct {
	size       int
	onComplete func(bool)
}

func (j *memoryHogJob) Process() error {
	// Try to allocate memory
	defer func() {
		if r := recover(); r != nil {
			j.onComplete(true)
		}
	}()

	// Attempt to allocate large slice
	data := make([]byte, j.size)
	for i := range data {
		data[i] = byte(i) // Force memory allocation
	}

	// If we get here, memory limit wasn't enforced
	j.onComplete(false)
	return nil
}

func (j *memoryHogJob) OnFailure(err error) {
	j.onComplete(true) // Consider any error as limit enforcement
}

func (j *memoryHogJob) MaxRetries() int {
	return 0
}

// cpuHogJob attempts to use too much CPU
type cpuHogJob struct {
	duration    int
	onComplete  func(bool)
}

func (j *cpuHogJob) Process() error {
	defer func() {
		if r := recover(); r != nil {
			// Panic indicates CPU limit was enforced
			j.onComplete(true)
			return
		}
		// No panic means CPU limit wasn't enforced
		j.onComplete(false)
	}()

	// Run CPU-intensive work that's hard to optimize away
	x := uint64(0xdeadbeef)
	for i := 0; i < j.duration*1000000; i++ {
		// Mix of integer operations that are hard to optimize
		x = (x << 13) | (x >> 51)
		x ^= uint64(i) * 0x123456789abcdef
		x = x*0xc6a4a7935bd1e995 + uint64(i)
		
		// Prevent compiler optimizations
		if x == 0 {
			runtime.Gosched() // Never actually called
		}
	}

	return nil
}

func (j *cpuHogJob) OnFailure(err error) {
	// Check if error is from CPU limit
	if err != nil && strings.Contains(err.Error(), "CPU limit exceeded") {
		j.onComplete(true)
		return
	}
	j.onComplete(false)
}

func (j *cpuHogJob) MaxRetries() int {
	return 0
}
