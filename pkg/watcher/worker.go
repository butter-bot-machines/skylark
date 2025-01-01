package watcher

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
)

// WorkerPool manages a pool of workers for processing file events
type WorkerPool struct {
	size       int32
	maxSize    int32
	active     int32
	workerChan chan struct{}
	wg         sync.WaitGroup
}

// NewWorkerPool creates a new worker pool with the specified size
func NewWorkerPool(size, maxSize int32) *WorkerPool {
	if size <= 0 {
		size = int32(runtime.NumCPU() * 2) // Default to num_cpu * 2
	}
	if maxSize <= 0 {
		maxSize = 32 // Maximum size as per plan
	}
	if size > maxSize {
		size = maxSize
	}

	return &WorkerPool{
		size:       size,
		maxSize:    maxSize,
		workerChan: make(chan struct{}, maxSize),
		active:     0,
	}
}

// Submit submits a task to the worker pool
func (p *WorkerPool) Submit(task func()) error {
	if atomic.LoadInt32(&p.active) >= p.maxSize {
		return fmt.Errorf("worker pool at maximum capacity")
	}

	p.wg.Add(1)
	atomic.AddInt32(&p.active, 1)

	go func() {
		defer func() {
			atomic.AddInt32(&p.active, -1)
			p.wg.Done()
		}()

		// Execute the task
		task()
	}()

	return nil
}

// Wait waits for all submitted tasks to complete
func (p *WorkerPool) Wait() {
	p.wg.Wait()
}

// Active returns the number of currently active workers
func (p *WorkerPool) Active() int32 {
	return atomic.LoadInt32(&p.active)
}

// Resize changes the size of the worker pool
func (p *WorkerPool) Resize(newSize int32) {
	if newSize <= 0 || newSize > p.maxSize {
		return
	}
	atomic.StoreInt32(&p.size, newSize)
}

// ProcessFileEvent processes a file event with the worker pool
func (w *FileWatcher) ProcessFileEvent(event FileEvent, processor func(FileEvent) error) error {
	if w.workerPool == nil {
		w.workerPool = NewWorkerPool(0, 32) // Use defaults
	}

	return w.workerPool.Submit(func() {
		if err := processor(event); err != nil {
			fmt.Printf("error processing file event: %v\n", err)
		}
	})
}
