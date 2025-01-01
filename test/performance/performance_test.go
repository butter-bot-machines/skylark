package performance

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/butter-bot-machines/skylark/pkg/config"
	"github.com/butter-bot-machines/skylark/pkg/worker"
)

// BenchmarkWorkerPool measures worker pool performance under load
func BenchmarkWorkerPool(b *testing.B) {
	// Disable CPU profiling
	b.SetParallelism(1)
	b.ReportAllocs()

	// Test with a single configuration
	cfg := &config.Config{
		Workers: config.WorkerConfig{
			Count:     4,
			QueueSize: 1000,
		},
	}

	pool := worker.NewPool(cfg)
	defer pool.Stop()

	var wg sync.WaitGroup
	jobQueue := pool.Queue()

	b.ResetTimer()
	b.StopTimer()

	// Run benchmark
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		b.StartTimer()
		jobQueue <- &benchmarkJob{
			id: i,
			onComplete: func() {
				wg.Done()
			},
		}
		wg.Wait()
		b.StopTimer()
	}
}

// BenchmarkFileProcessing measures file processing performance
func BenchmarkFileProcessing(b *testing.B) {
	// Disable CPU profiling
	b.SetParallelism(1)
	b.ReportAllocs()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "skylark-perf-*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	content := generateTestContent(100, "simple") // 100KB files
	for i := 0; i < 100; i++ {
		path := filepath.Join(tmpDir, fmt.Sprintf("test-%d.md", i))
		if err := os.WriteFile(path, content, 0644); err != nil {
			b.Fatalf("Failed to write test file: %v", err)
		}
	}

	cfg := &config.Config{
		Workers: config.WorkerConfig{
			Count:     runtime.NumCPU(),
			QueueSize: 1000,
		},
	}

	pool := worker.NewPool(cfg)
	defer pool.Stop()

	b.ResetTimer()
	b.StopTimer()

	// Run benchmark
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		jobQueue := pool.Queue()

		// Process all files
		b.StartTimer()
		for j := 0; j < 100; j++ {
			wg.Add(1)
			jobQueue <- &benchmarkJob{
				id: j,
				onComplete: func() {
					wg.Done()
				},
			}
		}
		wg.Wait()
		b.StopTimer()
	}
}

// TestWorkerPoolConcurrency verifies worker pool behavior under concurrent load
func TestWorkerPoolConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency test in short mode")
	}

	cfg := &config.Config{
		Workers: config.WorkerConfig{
			Count:     8,
			QueueSize: 1000,
		},
	}

	pool := worker.NewPool(cfg)
	defer pool.Stop()

	// Track completed jobs
	var completed uint64
	jobQueue := pool.Queue()

	// Launch multiple goroutines to queue jobs
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				jobQueue <- &benchmarkJob{
					id: j,
					onComplete: func() {
						atomic.AddUint64(&completed, 1)
					},
				}
			}
		}(i)
	}

	// Wait for all jobs to be queued
	wg.Wait()

	// Wait for all jobs to complete with timeout
	deadline := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			t.Fatal("Timeout waiting for jobs to complete")
		case <-ticker.C:
			if atomic.LoadUint64(&completed) == 1000 {
				return // All jobs completed
			}
		}
	}
}

// Helper functions and types

// generateTestContent creates test file content
func generateTestContent(sizeKB int, cmdType string) []byte {
	var content string
	switch cmdType {
	case "simple":
		content = fmt.Sprintf("# Test Document\n\n!command test-%d\n", sizeKB)
	case "multiple":
		content = "# Test Document\n\n"
		for i := 0; i < 10; i++ {
			content += fmt.Sprintf("!command test-%d\n", i)
		}
	case "references":
		content = `# Test Document

## Section 1
Content 1

## Section 2
Content 2

!command test with #Section 1# and #Section 2#
`
	}

	// Pad to reach desired size
	contentBytes := []byte(content)
	if len(contentBytes) < sizeKB*1024 {
		padding := make([]byte, sizeKB*1024-len(contentBytes))
		for i := range padding {
			padding[i] = byte('.')
		}
		contentBytes = append(contentBytes, padding...)
	}

	return contentBytes
}

// benchmarkJob implements the job.Job interface for performance testing
type benchmarkJob struct {
	id         int
	onComplete func()
}

func (j *benchmarkJob) Process() error {
	// Simulate some work
	time.Sleep(100 * time.Microsecond)
	j.onComplete()
	return nil
}

func (j *benchmarkJob) OnFailure(err error) {
	// No-op for benchmark
}

func (j *benchmarkJob) MaxRetries() int {
	return 0
}
