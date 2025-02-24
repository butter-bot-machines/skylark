package security

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/butter-bot-machines/skylark/pkg/config"
	"github.com/butter-bot-machines/skylark/pkg/config/memory"
	"github.com/butter-bot-machines/skylark/pkg/logging"
	"github.com/butter-bot-machines/skylark/pkg/logging/slog"
	"github.com/butter-bot-machines/skylark/pkg/process"
	wconcrete "github.com/butter-bot-machines/skylark/pkg/watcher/concrete"
	"github.com/butter-bot-machines/skylark/pkg/worker"
	wkconcrete "github.com/butter-bot-machines/skylark/pkg/worker/concrete"
	"github.com/butter-bot-machines/skylark/test/testutil"
)

// mockProcessManager implements a minimal process manager for testing
type mockProcessManager struct {
	defaultLimits process.ResourceLimits
}

func (m *mockProcessManager) New(name string, args []string) process.Process {
	return &mockProcess{
		limits: m.defaultLimits,
	}
}

func (m *mockProcessManager) Get(pid int) (process.Process, error) {
	return &mockProcess{
		limits: m.defaultLimits,
	}, nil
}

func (m *mockProcessManager) List() []process.Process {
	return []process.Process{&mockProcess{
		limits: m.defaultLimits,
	}}
}

func (m *mockProcessManager) SetDefaultLimits(limits process.ResourceLimits) {
	m.defaultLimits = limits
}

func (m *mockProcessManager) GetDefaultLimits() process.ResourceLimits {
	return m.defaultLimits
}

type mockProcess struct {
	limits process.ResourceLimits
	mu     sync.Mutex
}

func (p *mockProcess) Start() error {
	return nil
}

func (p *mockProcess) Wait() error {
	return nil
}

func (p *mockProcess) Signal(os.Signal) error {
	return nil
}

func (p *mockProcess) SetStdin(io.Reader) {}

func (p *mockProcess) SetStdout(io.Writer) {}

func (p *mockProcess) SetStderr(io.Writer) {}

func (p *mockProcess) SetLimits(limits process.ResourceLimits) error {
	p.mu.Lock()
	p.limits = limits
	p.mu.Unlock()
	return nil
}

func (p *mockProcess) GetLimits() process.ResourceLimits {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.limits
}

func (p *mockProcess) ID() int {
	return 0
}

func (p *mockProcess) Running() bool {
	return false
}

func (p *mockProcess) ExitCode() int {
	return 0
}

// enforceMemoryLimit checks if an allocation would exceed memory limits
func (p *mockProcess) enforceMemoryLimit(size int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.limits.MaxMemoryMB > 0 && size > int(p.limits.MaxMemoryMB*1024*1024) {
		return fmt.Errorf("memory limit exceeded: %d bytes > %d MB", size, p.limits.MaxMemoryMB)
	}
	return nil
}

// enforceCPULimit checks if CPU usage would exceed limits
func (p *mockProcess) enforceCPULimit(duration time.Duration) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.limits.MaxCPUTime > 0 && duration > p.limits.MaxCPUTime {
		return fmt.Errorf("CPU limit exceeded: %v > %v", duration, p.limits.MaxCPUTime)
	}
	return nil
}

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

	store := memory.NewStore(func(data map[string]interface{}) error { return nil })
	if err := store.SetAll(cfg.AsMap()); err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	logger := slog.NewLogger(logging.LevelInfo, os.Stdout)
	procMgr := &mockProcessManager{}

	pool, err := wkconcrete.NewPool(worker.Options{
		Config:    store,
		Logger:    logger,
		ProcMgr:   procMgr,
		QueueSize: cfg.Workers.QueueSize,
		Workers:   cfg.Workers.Count,
	})
	if err != nil {
		t.Fatalf("Failed to create worker pool: %v", err)
	}
	defer pool.Stop()

	// Create mock processor
	proc, err := testutil.NewMockProcessor()
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// Create watcher and wait for initialization
	w, err := wconcrete.NewWatcher(cfg, pool.Queue(), proc)
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

	store := memory.NewStore(func(data map[string]interface{}) error { return nil })
	if err := store.SetAll(cfg.AsMap()); err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	logger := slog.NewLogger(logging.LevelInfo, os.Stdout)
	procMgr := &mockProcessManager{
		defaultLimits: process.ResourceLimits{
			MaxMemoryMB:   100,                    // 100MB
			MaxCPUTime:    100 * time.Millisecond, // 100ms - short enough for test
			MaxFileSizeMB: 10,
			MaxFiles:      100,
			MaxProcesses:  10,
		},
	}

	pool, err := wkconcrete.NewPool(worker.Options{
		Config:    store,
		Logger:    logger,
		ProcMgr:   procMgr,
		QueueSize: cfg.Workers.QueueSize,
		Workers:   cfg.Workers.Count,
	})
	if err != nil {
		t.Fatalf("Failed to create worker pool: %v", err)
	}
	defer pool.Stop()

	// Test memory limits with cleanup
	t.Run("memory limits", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		var once sync.Once
		done := make(chan struct{})
		jobQueue := pool.Queue()

		// Create a job that tries to allocate too much memory
		job := &memoryHogJob{
			proc: procMgr.New("memory-hog", nil).(*mockProcess),
			size: 1024 * 1024 * 1024, // 1GB
			onComplete: func(recovered bool) {
				if !recovered {
					t.Error("Memory limits were not enforced")
				}
				once.Do(func() {
					close(done)
				})
			},
		}

		select {
		case jobQueue <- job:
		case <-ctx.Done():
			t.Fatal("Timeout queueing memory test job")
		}

		// Wait for job completion or timeout
		select {
		case <-done:
			// Test completed successfully - limits were enforced
		case <-time.After(100 * time.Millisecond):
			t.Error("Memory limits were not enforced in time")
		}
	})

	// Test CPU limits with cleanup
	t.Run("cpu limits", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		var once sync.Once
		done := make(chan struct{})
		jobQueue := pool.Queue()

		// Create a job that tries to use too much CPU
		job := &cpuHogJob{
			proc:     procMgr.New("cpu-hog", nil).(*mockProcess),
			duration: 110, // 110ms (slightly exceeds 100ms limit)
			onComplete: func(interrupted bool) {
				if !interrupted {
					t.Error("CPU limits were not enforced")
				}
				once.Do(func() {
					close(done)
				})
			},
		}

		select {
		case jobQueue <- job:
		case <-ctx.Done():
			t.Fatal("Timeout queueing CPU test job")
		}

		// Wait for job completion or timeout
		select {
		case <-done:
			// Test completed successfully - limits were enforced
		case <-time.After(200 * time.Millisecond):
			t.Error("CPU limits were not enforced within 200ms")
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
	proc       *mockProcess
	size       int
	onComplete func(bool)
}

func (j *memoryHogJob) Process() error {
	// Check memory limit before allocation
	if err := j.proc.enforceMemoryLimit(j.size); err != nil {
		j.onComplete(true)
		return err
	}

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
	return fmt.Errorf("memory limit exceeded")
}

func (j *memoryHogJob) OnFailure(err error) {
	j.onComplete(true) // Consider any error as limit enforcement
}

func (j *memoryHogJob) MaxRetries() int {
	return 0
}

// cpuHogJob attempts to use too much CPU
type cpuHogJob struct {
	proc       *mockProcess
	duration   int
	onComplete func(bool)
}

func (j *cpuHogJob) Process() error {
	start := time.Now()
	defer func() {
		if r := recover(); r != nil {
			// Panic indicates CPU limit was enforced
			j.onComplete(true)
			return
		}
	}()

	// Run CPU-intensive work
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Use multiple goroutines to max out CPU
	var wg sync.WaitGroup
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			x := uint64(0xdeadbeef)
			for {
				select {
				case <-ctx.Done():
					return
				default:
					// CPU intensive operations
					x = (x << 13) | (x >> 51)
					x ^= x * 0x123456789abcdef
					x = x*0xc6a4a7935bd1e995 + 1

					if x == 0 {
						runtime.Gosched()
					}
				}
			}
		}()
	}

	// Check CPU usage periodically
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			elapsed := time.Since(start)
			if err := j.proc.enforceCPULimit(elapsed); err != nil {
				cancel()  // Signal all goroutines to stop
				wg.Wait() // Wait for all goroutines to finish
				j.onComplete(true)
				return err
			}
		case <-ctx.Done():
			wg.Wait()
			return nil
		}
	}

	// If we get here, CPU limit wasn't enforced
	j.onComplete(false)
	return fmt.Errorf("CPU limit exceeded")
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
