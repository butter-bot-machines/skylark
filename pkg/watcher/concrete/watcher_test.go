package concrete

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/butter-bot-machines/skylark/pkg/config"
	"github.com/butter-bot-machines/skylark/pkg/job"
	"github.com/butter-bot-machines/skylark/pkg/parser"
	"github.com/butter-bot-machines/skylark/pkg/process"
	"github.com/butter-bot-machines/skylark/pkg/processor"
)

// mockProcessManager implements process.Manager for testing
type mockProcessManager struct {
	process.Manager
}

func (m *mockProcessManager) New(name string, args []string) process.Process {
	return nil
}

// mockProcessor implements processor.ProcessManager for testing
type mockProcessor struct {
	processFunc func(string) error
	procMgr     process.Manager
}

func (p *mockProcessor) Process(cmd *parser.Command) (string, error) {
	return "", nil
}

func (p *mockProcessor) ProcessFile(path string) error {
	if p.processFunc != nil {
		return p.processFunc(path)
	}
	return nil
}

func (p *mockProcessor) ProcessDirectory(dir string) error {
	return nil
}

func (p *mockProcessor) HandleResponse(cmd *parser.Command, response string) error {
	return nil
}

func (p *mockProcessor) UpdateFile(path string, responses []processor.Response) error {
	return nil
}

func (p *mockProcessor) GetProcessManager() process.Manager {
	return p.procMgr
}

func TestWatcher(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()

	// Create test files
	testFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create job queue and processor
	jobQueue := make(chan job.Job, 10)
	proc := &mockProcessor{
		procMgr: &mockProcessManager{},
	}

	// Create config
	cfg := &config.Config{
		WatchPaths: []string{tmpDir},
		FileWatch: config.FileWatchConfig{
			DebounceDelay: 100 * time.Millisecond,
			MaxDelay:      time.Second,
		},
	}

	// Create watcher
	w, err := NewWatcher(cfg, jobQueue, proc)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer w.Stop()

	t.Run("file creation", func(t *testing.T) {
		newFile := filepath.Join(tmpDir, "new.md")
		var wg sync.WaitGroup
		wg.Add(1)

		// Start job consumer
		go func() {
			defer wg.Done()
			select {
			case j := <-jobQueue:
				if j == nil {
					t.Error("Received nil job")
				}
			case <-time.After(time.Second):
				t.Error("Timed out waiting for job")
			}
		}()

		// Create new file
		if err := os.WriteFile(newFile, []byte("new content"), 0644); err != nil {
			t.Fatalf("Failed to create new file: %v", err)
		}

		wg.Wait()
	})

	t.Run("file modification", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)

		// Start job consumer
		go func() {
			defer wg.Done()
			select {
			case j := <-jobQueue:
				if j == nil {
					t.Error("Received nil job")
				}
			case <-time.After(time.Second):
				t.Error("Timed out waiting for job")
			}
		}()

		// Modify existing file
		if err := os.WriteFile(testFile, []byte("modified content"), 0644); err != nil {
			t.Fatalf("Failed to modify file: %v", err)
		}

		wg.Wait()
	})

	t.Run("non-markdown file", func(t *testing.T) {
		nonMdFile := filepath.Join(tmpDir, "test.txt")
		jobReceived := false

		// Start job consumer
		go func() {
			select {
			case <-jobQueue:
				jobReceived = true
			case <-time.After(200 * time.Millisecond):
				// No job should be received
			}
		}()

		// Create non-markdown file
		if err := os.WriteFile(nonMdFile, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create non-markdown file: %v", err)
		}

		if jobReceived {
			t.Error("Received job for non-markdown file")
		}
	})

	t.Run("debouncing", func(t *testing.T) {
		var jobCount int
		var mu sync.Mutex
		var wg sync.WaitGroup
		wg.Add(1)

		// Start job consumer
		go func() {
			defer wg.Done()
			timeout := time.After(500 * time.Millisecond)
			for {
				select {
				case <-jobQueue:
					mu.Lock()
					jobCount++
					mu.Unlock()
				case <-timeout:
					return
				}
			}
		}()

		// Rapidly modify file multiple times
		for i := 0; i < 5; i++ {
			if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
				t.Fatalf("Failed to modify file: %v", err)
			}
			time.Sleep(10 * time.Millisecond)
		}

		wg.Wait()

		mu.Lock()
		if jobCount > 2 { // Allow for some variation due to timing
			t.Errorf("Expected debounced events, got %d jobs", jobCount)
		}
		mu.Unlock()
	})
}

func TestWatcherErrors(t *testing.T) {
	t.Run("invalid path", func(t *testing.T) {
		cfg := &config.Config{
			WatchPaths: []string{"/nonexistent/path"},
		}
		jobQueue := make(chan job.Job)
		proc := &mockProcessor{
			procMgr: &mockProcessManager{},
		}

		_, err := NewWatcher(cfg, jobQueue, proc)
		if err == nil {
			t.Error("Expected error for invalid path")
		}
	})

	t.Run("nil config", func(t *testing.T) {
		jobQueue := make(chan job.Job)
		proc := &mockProcessor{
			procMgr: &mockProcessManager{},
		}

		_, err := NewWatcher(nil, jobQueue, proc)
		if err == nil {
			t.Error("Expected error for nil config")
		}
	})

	t.Run("nil job queue", func(t *testing.T) {
		cfg := &config.Config{
			WatchPaths: []string{"."},
		}
		proc := &mockProcessor{
			procMgr: &mockProcessManager{},
		}

		_, err := NewWatcher(cfg, nil, proc)
		if err == nil {
			t.Error("Expected error for nil job queue")
		}
	})

	t.Run("nil processor", func(t *testing.T) {
		cfg := &config.Config{
			WatchPaths: []string{"."},
		}
		jobQueue := make(chan job.Job)

		_, err := NewWatcher(cfg, jobQueue, nil)
		if err == nil {
			t.Error("Expected error for nil processor")
		}
	})
}
