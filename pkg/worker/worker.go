package worker

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/butter-bot-machines/skylark/pkg/config"
	"github.com/butter-bot-machines/skylark/pkg/job"
	"github.com/butter-bot-machines/skylark/pkg/logging"
	"github.com/butter-bot-machines/skylark/pkg/process"
	"github.com/butter-bot-machines/skylark/pkg/timing"
)

// Pool represents a worker pool for processing jobs
type Pool struct {
	workers       []*worker
	jobQueue      chan job.Job
	done          chan struct{}
	wg            sync.WaitGroup
	stats         *Stats
	limits        ResourceLimits
	queueWrappers sync.WaitGroup // Track wrapper goroutines
	logger        logging.Logger
	procMgr       process.Manager
}

// Stats tracks worker pool statistics
type Stats struct {
	ProcessedJobs uint64
	FailedJobs    uint64
	QueuedJobs    uint64
}

type worker struct {
	id   int
	pool *Pool
}

// Options configures a worker pool
type Options struct {
	Config    config.Store
	Logger    logging.Logger
	ProcMgr   process.Manager
	QueueSize int
	Workers   int
}

// NewPool creates a new worker pool
func NewPool(opts Options) (*Pool, error) {
	if opts.Config == nil {
		return nil, fmt.Errorf("config store required")
	}
	if opts.Logger == nil {
		return nil, fmt.Errorf("logger required")
	}
	if opts.ProcMgr == nil {
		return nil, fmt.Errorf("process manager required")
	}

	p := &Pool{
		jobQueue: make(chan job.Job, opts.QueueSize),
		done:     make(chan struct{}),
		stats:    &Stats{},
		limits:   DefaultLimits(),
		logger:   opts.Logger.WithGroup("worker"),
		procMgr:  opts.ProcMgr,
	}

	p.workers = make([]*worker, opts.Workers)
	for i := 0; i < opts.Workers; i++ {
		w := &worker{
			id:   i,
			pool: p,
		}
		p.workers[i] = w
		p.wg.Add(1)
		go w.start()
	}

	p.logger.Info("worker pool started",
		"workers", opts.Workers,
		"queue_size", opts.QueueSize)

	return p, nil
}

// WithClock sets a custom clock for the worker pool
func (p *Pool) WithClock(clock timing.Clock) *Pool {
	p.limits = p.limits.WithClock(clock)
	return p
}

// Queue returns a channel for queueing jobs
func (p *Pool) Queue() chan<- job.Job {
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
				atomic.AddUint64(&p.stats.QueuedJobs, 1)
				p.logger.Debug("job queued",
					"queued_jobs", atomic.LoadUint64(&p.stats.QueuedJobs))

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
func (p *Pool) Stats() Stats {
	return Stats{
		ProcessedJobs: atomic.LoadUint64(&p.stats.ProcessedJobs),
		FailedJobs:    atomic.LoadUint64(&p.stats.FailedJobs),
		QueuedJobs:    atomic.LoadUint64(&p.stats.QueuedJobs),
	}
}

// Stop gracefully shuts down the worker pool
func (p *Pool) Stop() {
	p.logger.Info("stopping worker pool")
	close(p.done)          // Signal all goroutines to stop
	p.queueWrappers.Wait() // Wait for queue wrapper goroutines to finish
	close(p.jobQueue)      // Close the job queue
	p.wg.Wait()            // Wait for all workers to finish
	p.logger.Info("worker pool stopped")
}

func (w *worker) start() {
	defer w.pool.wg.Done()
	logger := w.pool.logger.WithGroup(fmt.Sprintf("worker-%d", w.id))
	logger.Info("worker started")

	for {
		select {
		case <-w.pool.done:
			logger.Info("worker stopping")
			return
		case job, ok := <-w.pool.jobQueue:
			if !ok {
				logger.Info("worker stopping (queue closed)")
				return
			}

			logger.Debug("processing job")

			// Create process with resource limits
			proc := w.pool.procMgr.New("worker", []string{fmt.Sprintf("%d", w.id)})
			if err := proc.SetLimits(process.ResourceLimits{
				MaxCPUTime:    w.pool.limits.MaxCPUTime,
				MaxMemoryMB:   w.pool.limits.MaxMemory / (1024 * 1024), // Convert bytes to MB
				MaxFileSizeMB: w.pool.limits.MaxFileSize / (1024 * 1024),
				MaxFiles:      w.pool.limits.MaxFiles,
				MaxProcesses:  w.pool.limits.MaxProcesses,
			}); err != nil {
				logger.Error("failed to set resource limits", "error", err)
				atomic.AddUint64(&w.pool.stats.FailedJobs, 1)
				job.OnFailure(err)
				continue
			}

			// Start the process
			if err := proc.Start(); err != nil {
				logger.Error("failed to start process", "error", err)
				atomic.AddUint64(&w.pool.stats.FailedJobs, 1)
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
				atomic.AddUint64(&w.pool.stats.FailedJobs, 1)
				job.OnFailure(jobErr)
			} else if waitErr != nil {
				// Process failed, check if it was a resource limit
				if strings.Contains(waitErr.Error(), "CPU limit exceeded") || strings.Contains(waitErr.Error(), "out of memory") {
					logger.Warn("job failed (resource limit)", "error", waitErr)
					atomic.AddUint64(&w.pool.stats.FailedJobs, 1)
					job.OnFailure(waitErr)
				}
			} else {
				logger.Debug("job and process completed successfully")
				atomic.AddUint64(&w.pool.stats.ProcessedJobs, 1)
				logger.Debug("stats updated",
					"processed_jobs", atomic.LoadUint64(&w.pool.stats.ProcessedJobs),
					"failed_jobs", atomic.LoadUint64(&w.pool.stats.FailedJobs))
			}

			// Decrement queued jobs counter
			atomic.AddUint64(&w.pool.stats.QueuedJobs, ^uint64(0))
			logger.Debug("queued jobs decremented",
				"queued_jobs", atomic.LoadUint64(&w.pool.stats.QueuedJobs))
		}
	}
}
