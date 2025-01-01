package worker

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/butter-bot-machines/skylark/pkg/config"
)

// mockJob implements the Job interface for testing
type mockJob struct {
	processFunc func() error
	maxRetries  int
	onFailure   func(error)
}

func (j *mockJob) Process() error {
	if j.processFunc != nil {
		return j.processFunc()
	}
	return nil
}

func (j *mockJob) OnFailure(err error) {
	if j.onFailure != nil {
		j.onFailure(err)
	}
}

func (j *mockJob) MaxRetries() int {
	return j.maxRetries
}

func TestWorkerPool(t *testing.T) {
	cfg := &config.Config{
		Workers: config.WorkerConfig{
			Count:     2,
			QueueSize: 10,
		},
	}

	pool := NewPool(cfg)
	defer pool.Stop()

	// Test successful job processing
	t.Run("successful job", func(t *testing.T) {
		processed := false
		job := &mockJob{
			processFunc: func() error {
				processed = true
				return nil
			},
		}

		pool.Queue() <- job
		time.Sleep(100 * time.Millisecond)

		if !processed {
			t.Error("Job was not processed")
		}

		stats := pool.Stats()
		if stats.ProcessedJobs != 1 {
			t.Errorf("Expected 1 processed job, got %d", stats.ProcessedJobs)
		}
		if stats.FailedJobs != 0 {
			t.Errorf("Expected 0 failed jobs, got %d", stats.FailedJobs)
		}
	})

	// Test failed job processing
	t.Run("failed job", func(t *testing.T) {
		failureHandled := false
		job := &mockJob{
			processFunc: func() error {
				return errors.New("test error")
			},
			onFailure: func(err error) {
				failureHandled = true
			},
		}

		pool.Queue() <- job
		time.Sleep(100 * time.Millisecond)

		if !failureHandled {
			t.Error("Job failure was not handled")
		}

		stats := pool.Stats()
		if stats.FailedJobs != 1 {
			t.Errorf("Expected 1 failed job, got %d", stats.FailedJobs)
		}
	})

	// Test multiple jobs
	t.Run("multiple jobs", func(t *testing.T) {
		var processedCount uint64
		jobCount := 5

		for i := 0; i < jobCount; i++ {
			job := &mockJob{
				processFunc: func() error {
					atomic.AddUint64(&processedCount, 1)
					return nil
				},
			}
			pool.Queue() <- job
		}

		time.Sleep(200 * time.Millisecond)

		if atomic.LoadUint64(&processedCount) != uint64(jobCount) {
			t.Errorf("Expected %d processed jobs, got %d", jobCount, processedCount)
		}

		stats := pool.Stats()
		if stats.ProcessedJobs != uint64(jobCount)+2 { // +2 from previous tests
			t.Errorf("Expected %d processed jobs in stats, got %d", jobCount+2, stats.ProcessedJobs)
		}
	})
}

func TestWorkerPoolShutdown(t *testing.T) {
	cfg := &config.Config{
		Workers: config.WorkerConfig{
			Count:     2,
			QueueSize: 10,
		},
	}

	pool := NewPool(cfg)

	// Queue a job that takes some time
	completed := make(chan struct{})
	job := &mockJob{
		processFunc: func() error {
			time.Sleep(200 * time.Millisecond)
			close(completed)
			return nil
		},
	}

	pool.Queue() <- job

	// Stop the pool
	pool.Stop()

	// Check if job completed
	select {
	case <-completed:
		// Job completed successfully
	case <-time.After(500 * time.Millisecond):
		t.Error("Job did not complete before shutdown")
	}
}
