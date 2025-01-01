package watcher

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestNewWorkerPool(t *testing.T) {
	tests := []struct {
		name        string
		size        int32
		maxSize     int32
		wantSize    int32
		wantMaxSize int32
	}{
		{
			name:        "default values",
			size:        0,
			maxSize:     0,
			wantSize:    int32(8), // Assuming 4 CPU cores (4 * 2)
			wantMaxSize: 32,
		},
		{
			name:        "custom values within limits",
			size:        16,
			maxSize:     32,
			wantSize:    16,
			wantMaxSize: 32,
		},
		{
			name:        "size exceeds maxSize",
			size:        40,
			maxSize:     32,
			wantSize:    32,
			wantMaxSize: 32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewWorkerPool(tt.size, tt.maxSize)
			if pool == nil {
				t.Fatal("Expected non-nil WorkerPool")
			}

			if pool.maxSize != tt.wantMaxSize {
				t.Errorf("Expected maxSize %d, got %d", tt.wantMaxSize, pool.maxSize)
			}

			if pool.size > pool.maxSize {
				t.Errorf("Pool size %d exceeds maxSize %d", pool.size, pool.maxSize)
			}
		})
	}
}

func TestWorkerPoolSubmit(t *testing.T) {
	pool := NewWorkerPool(2, 4)
	var counter int32

	// Submit tasks up to capacity
	for i := 0; i < 4; i++ {
		err := pool.Submit(func() {
			atomic.AddInt32(&counter, 1)
			time.Sleep(10 * time.Millisecond)
		})
		if err != nil {
			t.Errorf("Failed to submit task %d: %v", i, err)
		}
	}

	// Try to submit one more task (should fail)
	err := pool.Submit(func() {
		atomic.AddInt32(&counter, 1)
	})
	if err == nil {
		t.Error("Expected error when submitting task beyond capacity")
	}

	// Wait for all tasks to complete
	pool.Wait()

	// Check that all submitted tasks were executed
	if counter != 4 {
		t.Errorf("Expected 4 tasks to complete, got %d", counter)
	}
}

func TestWorkerPoolResize(t *testing.T) {
	pool := NewWorkerPool(2, 8)

	// Test resizing within limits
	pool.Resize(4)
	if pool.size != 4 {
		t.Errorf("Expected size 4, got %d", pool.size)
	}

	// Test resizing beyond maxSize
	pool.Resize(10)
	if pool.size != 4 {
		t.Errorf("Expected size to remain 4, got %d", pool.size)
	}

	// Test resizing to invalid value
	pool.Resize(-1)
	if pool.size != 4 {
		t.Errorf("Expected size to remain 4, got %d", pool.size)
	}
}

func TestFileWatcherWithWorkerPool(t *testing.T) {
	w, err := NewFileWatcher(500 * time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to create FileWatcher: %v", err)
	}
	defer w.Stop()

	var processedCount int32
	processor := func(event FileEvent) error {
		atomic.AddInt32(&processedCount, 1)
		return nil
	}

	// Create multiple events
	events := []FileEvent{
		{Path: "test1.md", Type: Modified, Content: []byte("test1")},
		{Path: "test2.md", Type: Modified, Content: []byte("test2")},
		{Path: "test3.md", Type: Modified, Content: []byte("test3")},
	}

	// Process events
	for _, event := range events {
		err := w.ProcessFileEvent(event, processor)
		if err != nil {
			t.Errorf("Failed to process event: %v", err)
		}
	}

	// Wait a bit for processing to complete
	time.Sleep(100 * time.Millisecond)

	// Check that all events were processed
	if atomic.LoadInt32(&processedCount) != int32(len(events)) {
		t.Errorf("Expected %d events to be processed, got %d", len(events), processedCount)
	}
}
