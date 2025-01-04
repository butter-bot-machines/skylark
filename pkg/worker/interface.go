package worker

import (
	"github.com/butter-bot-machines/skylark/pkg/config"
	"github.com/butter-bot-machines/skylark/pkg/job"
	"github.com/butter-bot-machines/skylark/pkg/logging"
	"github.com/butter-bot-machines/skylark/pkg/process"
)

// Stats tracks worker pool statistics
type Stats interface {
	// ProcessedJobs returns the number of successfully processed jobs
	ProcessedJobs() uint64

	// FailedJobs returns the number of failed jobs
	FailedJobs() uint64

	// QueuedJobs returns the number of currently queued jobs
	QueuedJobs() uint64
}

// Worker represents a single worker in the pool
type Worker interface {
	// ID returns the worker's unique identifier
	ID() int

	// Start begins processing jobs from the pool
	Start() error

	// Stop gracefully stops the worker
	Stop() error
}

// Pool represents a worker pool for processing jobs
type Pool interface {
	// Queue returns a channel for queueing jobs
	Queue() chan<- job.Job

	// Stats returns the current worker pool statistics
	Stats() Stats

	// Stop gracefully shuts down the worker pool
	Stop()
}

// Options configures a worker pool
type Options struct {
	Config    config.Store
	Logger    logging.Logger
	ProcMgr   process.Manager
	QueueSize int
	Workers   int
}

// Factory creates new worker pools
type Factory interface {
	// NewPool creates a new worker pool with the given options
	NewPool(opts Options) (Pool, error)
}
