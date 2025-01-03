package worker

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/butter-bot-machines/skylark/pkg/config"
	"github.com/butter-bot-machines/skylark/pkg/job"
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

// NewPool creates a new worker pool
func NewPool(cfg *config.Config) *Pool {
	p := &Pool{
		jobQueue: make(chan job.Job, cfg.Workers.QueueSize),
		done:     make(chan struct{}),
		stats:    &Stats{},
		limits:   DefaultLimits(),
	}

	p.workers = make([]*worker, cfg.Workers.Count)
	for i := 0; i < cfg.Workers.Count; i++ {
		w := &worker{
			id:   i,
			pool: p,
		}
		p.workers[i] = w
		p.wg.Add(1)
		go w.start()
	}

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

// IncrementStats atomically increments a stats counter
func IncrementStats(p *Pool, counter *uint64) {
	atomic.AddUint64(counter, 1)
}

// GetStats atomically reads a stats counter
func GetStats(p *Pool, counter *uint64) uint64 {
	return atomic.LoadUint64(counter)
}

// Stop gracefully shuts down the worker pool
func (p *Pool) Stop() {
	close(p.done)           // Signal all goroutines to stop
	p.queueWrappers.Wait()  // Wait for queue wrapper goroutines to finish
	close(p.jobQueue)       // Close the job queue
	p.wg.Wait()            // Wait for all workers to finish
}

func (w *worker) start() {
	defer w.pool.wg.Done()

	// Set memory limit for this worker
	enforceMemoryLimit(w.pool.limits.MaxMemory)

	for {
		select {
		case <-w.pool.done:
			return
		case job, ok := <-w.pool.jobQueue:
			if !ok {
				return
			}

			// Create context with timeout for job execution
			ctx, cancel := context.WithTimeout(context.Background(), w.pool.limits.MaxCPUTime)

			// Run job with resource limits
			err := runWithCPULimit(ctx, w.pool.limits.MaxCPUTime, func() error {
				// Set memory limit for this goroutine
				enforceMemoryLimit(w.pool.limits.MaxMemory)
				return job.Process()
			})

			// Check if error is from resource limit
			if err != nil && (strings.Contains(err.Error(), "CPU limit exceeded") || strings.Contains(err.Error(), "out of memory")) {
				job.OnFailure(err)
				atomic.AddUint64(&w.pool.stats.FailedJobs, 1)
				continue
			}

			// Clean up
			cancel()

			if err != nil {
				atomic.AddUint64(&w.pool.stats.FailedJobs, 1)
				job.OnFailure(err)
			}
			atomic.AddUint64(&w.pool.stats.ProcessedJobs, 1)
			// Decrement queued jobs counter
			atomic.AddUint64(&w.pool.stats.QueuedJobs, ^uint64(0)) // This is equivalent to subtracting 1
		}
	}
}
