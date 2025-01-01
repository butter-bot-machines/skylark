package integration

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/butter-bot-machines/skylark/pkg/config"
	"github.com/butter-bot-machines/skylark/pkg/watcher"
	"github.com/butter-bot-machines/skylark/pkg/worker"
	"github.com/butter-bot-machines/skylark/test/testutil"
)

// TestWatcherWorkerIntegration tests the integration between file watcher and worker pool
func TestWorkerPool(t *testing.T) {
	cfg := &config.Config{
		Workers: config.WorkerConfig{
			Count:     2,
			QueueSize: 10,
		},
	}

	pool := worker.NewPool(cfg)
	defer pool.Stop()

	// Create and queue a test job
	jobDone := make(chan struct{})
	jobQueue := pool.Queue()
	jobQueue <- &testJob{
		onProcess: func() error {
			close(jobDone)
			return nil
		},
	}

	// Wait for job completion with timeout
	select {
	case <-jobDone:
		// Job completed successfully
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for job completion")
	}

	// Verify job was processed
	stats := pool.Stats()
	if stats.ProcessedJobs != 1 {
		t.Errorf("Expected 1 processed job, got %d", stats.ProcessedJobs)
	}
}

// testJob implements the job.Job interface for testing
type testJob struct {
	onProcess  func() error
	onFailure  func(error)
	maxRetries int
}

func (j *testJob) Process() error {
	if j.onProcess != nil {
		return j.onProcess()
	}
	return nil
}

func (j *testJob) OnFailure(err error) {
	if j.onFailure != nil {
		j.onFailure(err)
	}
}

func (j *testJob) MaxRetries() int {
	return j.maxRetries
}

// TestWatcherWorkerIntegration verifies that file changes are properly processed
func TestWatcherWorkerIntegration(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "skylark-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test configuration
	cfg := &config.Config{
		WatchPaths: []string{tmpDir},
		Workers: config.WorkerConfig{
			Count:     2,
			QueueSize: 10,
		},
		FileWatch: config.FileWatchConfig{
			DebounceDelay: 100 * time.Millisecond,
			MaxDelay:      1 * time.Second,
		},
	}

	// Create worker pool
	pool := worker.NewPool(cfg)
	defer pool.Stop()

	// Create mock processor
	proc, err := testutil.NewMockProcessor()
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// Create and start file watcher
	w, err := watcher.New(cfg, pool.Queue(), proc)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer w.Stop()

	// Create a test markdown file
	testFile := filepath.Join(tmpDir, "test.md")
	content := []byte("# Test Document\n\n!command test\n")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Wait for file processing with timeout
	deadline := time.After(2 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			t.Fatal("Timeout waiting for file to be processed")
		case <-ticker.C:
			stats := pool.Stats()
			if stats.ProcessedJobs > 0 {
				return // Test passed
			}
		}
	}
}

// TestAssistantIntegration tests the integration between worker and assistant
func TestAssistantIntegration(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Workers: config.WorkerConfig{
			Count:     2,
			QueueSize: 10,
		},
	}

	// Create worker pool
	pool := worker.NewPool(cfg)
	defer pool.Stop()

	// Create test assistant
	assistant := &testAssistant{
		processedCommands: make(chan string, 1),
	}

	// Create and queue a command job
	jobQueue := pool.Queue()
	jobQueue <- &commandJob{
		command:   "!test hello world",
		assistant: assistant,
	}

	// Wait for command processing with timeout
	select {
	case cmd := <-assistant.processedCommands:
		if cmd != "hello world" {
			t.Errorf("Expected command 'hello world', got '%s'", cmd)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for command to be processed")
	}
}

// testAssistant implements a mock assistant for testing
type testAssistant struct {
	processedCommands chan string
}

func (a *testAssistant) ProcessCommand(cmd string) error {
	a.processedCommands <- cmd
	return nil
}

// commandJob implements the job.Job interface for testing command processing
type commandJob struct {
	command   string
	assistant *testAssistant
}

func (j *commandJob) Process() error {
	// Strip the command prefix and pass to assistant
	cmd := j.command[6:] // Remove "!test " including the space
	return j.assistant.ProcessCommand(cmd)
}

func (j *commandJob) OnFailure(err error) {
	// No-op for test
}

func (j *commandJob) MaxRetries() int {
	return 0
}

// TestEndToEnd tests the complete flow from file change to response
func TestEndToEnd(t *testing.T) {
	// TODO: Implement end-to-end integration test
	// This will test:
	// - File watching
	// - Command processing
	// - Assistant routing
	// - Tool execution
	// - Response writing
}
