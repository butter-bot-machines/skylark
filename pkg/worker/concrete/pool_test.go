package concrete

import (
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/butter-bot-machines/skylark/pkg/config"
	"github.com/butter-bot-machines/skylark/pkg/logging"
	"github.com/butter-bot-machines/skylark/pkg/process"
	"github.com/butter-bot-machines/skylark/pkg/timing"
	"github.com/butter-bot-machines/skylark/pkg/worker"
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

// mockLogger implements logging.Logger for testing
type mockLogger struct {
	logging.Logger // Embed to get default implementations
}

func (l *mockLogger) Debug(msg string, args ...interface{}) {}
func (l *mockLogger) Info(msg string, args ...interface{})  {}
func (l *mockLogger) Warn(msg string, args ...interface{})  {}
func (l *mockLogger) Error(msg string, args ...interface{}) {}
func (l *mockLogger) With(args ...interface{}) logging.Logger {
	return l
}
func (l *mockLogger) WithGroup(name string) logging.Logger {
	return l
}

// mockProcess implements process.Process for testing
type mockProcess struct {
	process.Process // Embed to get default implementations
	limits          process.ResourceLimits
	started         bool
	done            chan struct{}
	err             error
	mu              sync.Mutex
}

func newMockProcess() *mockProcess {
	return &mockProcess{
		done: make(chan struct{}),
	}
}

func (p *mockProcess) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started {
		return errors.New("process already started")
	}
	p.started = true
	return nil
}

func (p *mockProcess) Wait() error {
	p.mu.Lock()
	if !p.started {
		p.mu.Unlock()
		return errors.New("process not started")
	}
	p.mu.Unlock()

	<-p.done
	return p.err
}

func (p *mockProcess) Signal(sig os.Signal) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.started {
		return errors.New("process not started")
	}
	select {
	case <-p.done:
		return nil
	default:
		close(p.done)
		return nil
	}
}

func (p *mockProcess) SetLimits(l process.ResourceLimits) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.limits = l
	return nil
}

// mockProcMgr implements process.Manager for testing
type mockProcMgr struct {
	process.Manager // Embed to get default implementations
	processes       []*mockProcess
	mu              sync.Mutex
	newFunc         func(name string, args []string) process.Process
	defaultLimits   process.ResourceLimits
}

func newMockProcMgr() *mockProcMgr {
	m := &mockProcMgr{}
	m.newFunc = func(name string, args []string) process.Process {
		m.mu.Lock()
		defer m.mu.Unlock()

		proc := newMockProcess()
		m.processes = append(m.processes, proc)
		return proc
	}
	return m
}

func (m *mockProcMgr) New(name string, args []string) process.Process {
	return m.newFunc(name, args)
}

func (m *mockProcMgr) SetDefaultLimits(limits process.ResourceLimits) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultLimits = limits
}

func (m *mockProcMgr) GetDefaultLimits() process.ResourceLimits {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.defaultLimits
}

// mockConfig implements config.Store for testing
type mockConfig struct {
	config.Store // Embed to get default implementations
}

func (c *mockConfig) Load() error                                { return nil }
func (c *mockConfig) Save() error                                { return nil }
func (c *mockConfig) Reset() error                               { return nil }
func (c *mockConfig) Get(key string) (interface{}, error)        { return nil, nil }
func (c *mockConfig) Set(key string, value interface{}) error    { return nil }
func (c *mockConfig) Delete(key string) error                    { return nil }
func (c *mockConfig) GetAll() (map[string]interface{}, error)    { return nil, nil }
func (c *mockConfig) SetAll(values map[string]interface{}) error { return nil }
func (c *mockConfig) Validate() error                            { return nil }

func TestWorkerPool(t *testing.T) {
	mock := timing.NewMock()
	logger := &mockLogger{}
	procMgr := newMockProcMgr()

	opts := worker.Options{
		Config:    &mockConfig{},
		Logger:    logger,
		ProcMgr:   procMgr,
		QueueSize: 10,
		Workers:   2,
	}

	pool, err := NewPool(opts)
	if err != nil {
		t.Fatalf("Failed to create worker pool: %v", err)
	}
	pool.(*poolImpl).WithClock(mock)
	defer pool.Stop()

	// Test successful job processing
	t.Run("successful job", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)
		processed := false
		jobProcessed := make(chan struct{})
		job := &mockJob{
			processFunc: func() error {
				processed = true
				wg.Done()
				close(jobProcessed)
				return nil
			},
		}

		pool.Queue() <- job
		wg.Wait()
		<-jobProcessed // Wait for job to be fully processed

		if !processed {
			t.Error("Job was not processed")
		}

		stats := pool.Stats()
		if stats.ProcessedJobs() != 1 {
			t.Errorf("Expected 1 processed job, got %d", stats.ProcessedJobs())
		}
		if stats.FailedJobs() != 0 {
			t.Errorf("Expected 0 failed jobs, got %d", stats.FailedJobs())
		}
	})

	// Test failed job processing
	t.Run("failed job", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)
		failureHandled := false
		jobProcessed := make(chan struct{})
		job := &mockJob{
			processFunc: func() error {
				return errors.New("test error")
			},
			onFailure: func(err error) {
				failureHandled = true
				wg.Done()
				close(jobProcessed)
			},
		}

		pool.Queue() <- job
		wg.Wait()
		<-jobProcessed // Wait for job to be fully processed

		if !failureHandled {
			t.Error("Job failure was not handled")
		}

		stats := pool.Stats()
		if stats.FailedJobs() != 1 {
			t.Errorf("Expected 1 failed job, got %d", stats.FailedJobs())
		}
	})

	// Test multiple jobs
	t.Run("multiple jobs", func(t *testing.T) {
		var processedCount uint64
		var wg sync.WaitGroup
		jobCount := 5
		wg.Add(jobCount)

		// Create channels to track job completion
		jobProcessed := make([]chan struct{}, jobCount)
		for i := range jobProcessed {
			jobProcessed[i] = make(chan struct{})
		}

		// Queue all jobs
		for i := 0; i < jobCount; i++ {
			i := i // Create new variable for closure
			job := &mockJob{
				processFunc: func() error {
					atomic.AddUint64(&processedCount, 1)
					wg.Done()
					close(jobProcessed[i])
					return nil
				},
			}
			pool.Queue() <- job
		}

		// Wait for all jobs to complete
		wg.Wait()

		// Wait for all jobs to be fully processed
		for i := 0; i < jobCount; i++ {
			<-jobProcessed[i]
		}

		// Wait for stats to update with retries
		deadline := time.Now().Add(30 * time.Second)
		var stats worker.Stats
		for time.Now().Before(deadline) {
			stats = pool.Stats()
			if stats.ProcessedJobs() == uint64(jobCount)+1 { // +1 from first test (second test failed)
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		if atomic.LoadUint64(&processedCount) != uint64(jobCount) {
			t.Errorf("Expected %d processed jobs, got %d", jobCount, processedCount)
		}

		if stats.ProcessedJobs() != uint64(jobCount)+1 { // +1 from first test (second test failed)
			t.Errorf("Expected %d processed jobs in stats, got %d", jobCount+1, stats.ProcessedJobs())
		}
	})
}

func TestWorkerPoolShutdown(t *testing.T) {
	mock := timing.NewMock()
	logger := &mockLogger{}
	procMgr := newMockProcMgr()

	opts := worker.Options{
		Config:    &mockConfig{},
		Logger:    logger,
		ProcMgr:   procMgr,
		QueueSize: 10,
		Workers:   2,
	}

	pool, err := NewPool(opts)
	if err != nil {
		t.Fatalf("Failed to create worker pool: %v", err)
	}
	pool.(*poolImpl).WithClock(mock)

	// Queue a job that takes some time
	var wg sync.WaitGroup
	wg.Add(1)
	jobProcessed := make(chan struct{})
	job := &mockJob{
		processFunc: func() error {
			defer wg.Done()
			close(jobProcessed)
			return nil
		},
	}

	pool.Queue() <- job
	wg.Wait()
	<-jobProcessed // Wait for job to be fully processed

	pool.Stop()
}

func TestWorkerPoolCPULimit(t *testing.T) {
	mock := timing.NewMock()
	logger := &mockLogger{}
	procMgr := newMockProcMgr()

	opts := worker.Options{
		Config:    &mockConfig{},
		Logger:    logger,
		ProcMgr:   procMgr,
		QueueSize: 1,
		Workers:   1,
	}

	pool, err := NewPool(opts)
	if err != nil {
		t.Fatalf("Failed to create worker pool: %v", err)
	}
	pool.(*poolImpl).WithClock(mock)
	defer pool.Stop()

	t.Run("cpu limit exceeded", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)
		failureHandled := false
		jobProcessed := make(chan struct{})

		job := &mockJob{
			processFunc: func() error {
				return errors.New("CPU time limit exceeded")
			},
			onFailure: func(err error) {
				if err != nil && err.Error() == "CPU time limit exceeded" {
					failureHandled = true
				}
				wg.Done()
				close(jobProcessed)
			},
		}

		pool.Queue() <- job
		wg.Wait()
		<-jobProcessed // Wait for job to be fully processed

		if !failureHandled {
			t.Error("CPU limit exceeded was not handled")
		}

		stats := pool.Stats()
		if stats.FailedJobs() != 1 {
			t.Errorf("Expected 1 failed job, got %d", stats.FailedJobs())
		}
	})

	t.Run("within cpu limit", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)
		processed := false
		jobProcessed := make(chan struct{})
		job := &mockJob{
			processFunc: func() error {
				processed = true
				wg.Done()
				close(jobProcessed)
				return nil
			},
		}

		pool.Queue() <- job
		wg.Wait()
		<-jobProcessed // Wait for job to be fully processed

		if !processed {
			t.Error("Job within CPU limit was not processed")
		}
	})
}
