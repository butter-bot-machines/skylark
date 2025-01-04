package concrete

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/butter-bot-machines/skylark/pkg/job"
	"github.com/butter-bot-machines/skylark/pkg/logging"
	"github.com/butter-bot-machines/skylark/pkg/process"
	"github.com/butter-bot-machines/skylark/pkg/timing"
	"github.com/butter-bot-machines/skylark/pkg/worker"
)

// poolStats implements worker.Stats
type poolStats struct {
	processedJobs uint64
	failedJobs    uint64
	queuedJobs    uint64
}

func (s *poolStats) ProcessedJobs() uint64 {
	return atomic.LoadUint64(&s.processedJobs)
}

func (s *poolStats) FailedJobs() uint64 {
	return atomic.LoadUint64(&s.failedJobs)
}

func (s *poolStats) QueuedJobs() uint64 {
	return atomic.LoadUint64(&s.queuedJobs)
}

// workerImpl implements worker.Worker
type workerImpl struct {
	id   int
	pool *poolImpl
}

func (w *workerImpl) ID() int {
	return w.id
}

func (w *workerImpl) Start() error {
	defer w.pool.wg.Done()
	logger := w.pool.logger.WithGroup(fmt.Sprintf("worker-%d", w.id))
	logger.Info("worker started")

	for {
		select {
		case <-w.pool.done:
			logger.Info("worker stopping")
			return nil
		case job, ok := <-w.pool.jobQueue:
			if !ok {
				logger.Info("worker stopping (queue closed)")
				return nil
			}

			logger.Debug("processing job")

			// Create process with resource limits
			proc := w.pool.procMgr.New("worker", []string{fmt.Sprintf("%d", w.id)})
			if err := proc.SetLimits(process.ResourceLimits{
				MaxCPUTime:    w.pool.limits.MaxCPUTime,
				MaxMemoryMB:   int64(w.pool.limits.MaxMemory / (1024 * 1024)), // Convert bytes to MB
				MaxFileSizeMB: int64(w.pool.limits.MaxFileSize / (1024 * 1024)),
				MaxFiles:      int64(w.pool.limits.MaxFiles),
				MaxProcesses:  int64(w.pool.limits.MaxProcesses),
			}); err != nil {
				logger.Error("failed to set resource limits", "error", err)
				atomic.AddUint64(&w.pool.stats.failedJobs, 1)
				job.OnFailure(err)
				continue
			}

			// Start the process
			if err := proc.Start(); err != nil {
				logger.Error("failed to start process", "error", err)
				atomic.AddUint64(&w.pool.stats.failedJobs, 1)
				job.OnFailure(err)
				continue
			}

			// Run the job
			logger.Debug("running job")
			jobErr := job.Process()
			logger.Debug("job process returned", "error", jobErr)

			// Signal process completion
			logger.Debug("signaling process completion")
			if err := proc.Signal(os.Interrupt); err != nil {
				logger.Error("failed to signal process", "error", err)
			}

			// Wait for process completion
			logger.Debug("waiting for process completion")
			waitErr := proc.Wait()
			logger.Debug("process wait returned", "error", waitErr)

			// Handle errors and update stats
			if jobErr != nil {
				logger.Error("job failed", "error", jobErr)
				atomic.AddUint64(&w.pool.stats.failedJobs, 1)
				job.OnFailure(jobErr)
			} else if waitErr != nil {
				// Process failed, check if it was a resource limit
				if strings.Contains(waitErr.Error(), "CPU limit exceeded") || strings.Contains(waitErr.Error(), "out of memory") {
					logger.Warn("job failed (resource limit)", "error", waitErr)
					atomic.AddUint64(&w.pool.stats.failedJobs, 1)
					job.OnFailure(waitErr)
				}
			} else {
				logger.Debug("job and process completed successfully")
				atomic.AddUint64(&w.pool.stats.processedJobs, 1)
				logger.Debug("stats updated",
					"processed_jobs", atomic.LoadUint64(&w.pool.stats.processedJobs),
					"failed_jobs", atomic.LoadUint64(&w.pool.stats.failedJobs))
			}

			// Decrement queued jobs counter
			atomic.AddUint64(&w.pool.stats.queuedJobs, ^uint64(0))
			logger.Debug("queued jobs decremented",
				"queued_jobs", atomic.LoadUint64(&w.pool.stats.queuedJobs))
		}
	}
}

func (w *workerImpl) Stop() error {
	return nil // Stop is handled by pool
}

// poolImpl implements worker.Pool
type poolImpl struct {
	workers       []*workerImpl
	jobQueue      chan job.Job
	done          chan struct{}
	wg            sync.WaitGroup
	stats         *poolStats
	limits        ResourceLimits
	queueWrappers sync.WaitGroup
	logger        logging.Logger
	procMgr       process.Manager
}

// NewPool creates a new worker pool
func NewPool(opts worker.Options) (worker.Pool, error) {
	if opts.Config == nil {
		return nil, fmt.Errorf("config store required")
	}
	if opts.Logger == nil {
		return nil, fmt.Errorf("logger required")
	}
	if opts.ProcMgr == nil {
		return nil, fmt.Errorf("process manager required")
	}

	p := &poolImpl{
		jobQueue: make(chan job.Job, opts.QueueSize),
		done:     make(chan struct{}),
		stats:    &poolStats{},
		limits:   DefaultLimits(),
		logger:   opts.Logger.WithGroup("worker"),
		procMgr:  opts.ProcMgr,
	}

	p.workers = make([]*workerImpl, opts.Workers)
	for i := 0; i < opts.Workers; i++ {
		w := &workerImpl{
			id:   i,
			pool: p,
		}
		p.workers[i] = w
		p.wg.Add(1)
		go w.Start()
	}

	p.logger.Info("worker pool started",
		"workers", opts.Workers,
		"queue_size", opts.QueueSize)

	return p, nil
}

// WithClock sets a custom clock for the worker pool
func (p *poolImpl) WithClock(clock timing.Clock) worker.Pool {
	p.limits = p.limits.WithClock(clock)
	return p
}

// Queue returns a channel for queueing jobs
func (p *poolImpl) Queue() chan<- job.Job {
	// Create a buffered channel with same capacity as jobQueue
	ch := make(chan job.Job, cap(p.jobQueue))
	p.queueWrappers.Add(1)
	go func() {
		defer p.queueWrappers.Done()
		defer close(ch)
		for {
			select {
			case <-p.done:
				return
			case j, ok := <-ch:
				if !ok {
					return
				}
				atomic.AddUint64(&p.stats.queuedJobs, 1)
				p.logger.Debug("job queued",
					"queued_jobs", atomic.LoadUint64(&p.stats.queuedJobs))

				// Try to send the job, but give up if pool is shutting down
				select {
				case <-p.done:
					return
				case p.jobQueue <- j:
				}
			}
		}
	}()
	return ch
}

// Stats returns the current worker pool statistics
func (p *poolImpl) Stats() worker.Stats {
	return p.stats
}

// Stop gracefully shuts down the worker pool
func (p *poolImpl) Stop() {
	p.logger.Info("stopping worker pool")
	close(p.done)          // Signal all goroutines to stop
	p.queueWrappers.Wait() // Wait for queue wrapper goroutines to finish
	close(p.jobQueue)      // Close the job queue
	p.wg.Wait()           // Wait for all workers to finish
	p.logger.Info("worker pool stopped")
}
