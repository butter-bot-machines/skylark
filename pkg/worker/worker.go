package worker

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Task represents a unit of work to be processed
type Task struct {
	ID         string                 // Unique identifier
	Priority   int                    // Processing priority (higher = more important)
	Timeout    time.Duration         // Maximum execution time
	DependsOn  []string              // IDs of tasks that must complete first
	Execute    func() (any, error)   // Function to execute
	Result     chan TaskResult       // Channel for result delivery
	Status     TaskStatus            // Current status
	RetryCount int                   // Number of retries remaining
}

// TaskResult represents the outcome of a task execution
type TaskResult struct {
	TaskID string    // ID of the task
	Output any       // Task output data
	Error  error     // Error if task failed
	Time   time.Time // When the task completed
}

// TaskStatus represents the current state of a task
type TaskStatus int

const (
	TaskPending TaskStatus = iota
	TaskRunning
	TaskCompleted
	TaskFailed
)

// Pool manages a collection of workers
type Pool struct {
	workers    []*Worker
	tasks      chan *Task
	results    chan TaskResult
	pending    map[string]*Task
	completed  map[string]TaskResult
	maxRetries int
	stopped    bool
	mu         sync.RWMutex
	wg         sync.WaitGroup
}

// Worker represents a single worker in the pool
type Worker struct {
	id      int
	pool    *Pool
	tasks   chan *Task
	quit    chan bool
	metrics WorkerMetrics
}

// WorkerMetrics tracks worker performance
type WorkerMetrics struct {
	TasksProcessed int
	TotalTime      time.Duration
	Errors         int
	mu            sync.Mutex
}

// NewPool creates a new worker pool
func NewPool(size int, queueSize int, maxRetries int) *Pool {
	p := &Pool{
		workers:    make([]*Worker, size),
		tasks:      make(chan *Task, queueSize),
		results:    make(chan TaskResult, queueSize),
		pending:    make(map[string]*Task),
		completed:  make(map[string]TaskResult),
		maxRetries: maxRetries,
	}

	// Initialize workers
	for i := 0; i < size; i++ {
		w := &Worker{
			id:    i,
			pool:  p,
			tasks: make(chan *Task),
			quit:  make(chan bool),
		}
		p.workers[i] = w
		go w.start()
	}

	// Start task dispatcher
	go p.dispatch()

	return p
}

// Submit adds a task to the pool
func (p *Pool) Submit(task *Task) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if pool is stopped
	if p.stopped {
		return fmt.Errorf("worker pool is stopped")
	}

	// Validate task
	if task.ID == "" {
		return fmt.Errorf("task ID required")
	}
	if task.Execute == nil {
		return fmt.Errorf("task execution function required")
	}
	if task.Result == nil {
		task.Result = make(chan TaskResult, 1)
	}

	// Check dependencies
	for _, depID := range task.DependsOn {
		if _, ok := p.completed[depID]; !ok {
			return fmt.Errorf("dependency %s not completed", depID)
		}
	}

	// Set defaults
	if task.RetryCount == 0 {
		task.RetryCount = p.maxRetries
	}
	if task.Timeout == 0 {
		task.Timeout = 30 * time.Second
	}

	// Add to pending tasks
	p.pending[task.ID] = task

	// Submit to task queue
	select {
	case p.tasks <- task:
		return nil
	default:
		return fmt.Errorf("task queue full")
	}
}

// Wait blocks until all tasks are completed
func (p *Pool) Wait() {
	p.wg.Wait()
}

// Stop gracefully shuts down the pool
func (p *Pool) Stop() {
	p.mu.Lock()
	if p.stopped {
		p.mu.Unlock()
		return
	}
	p.stopped = true
	p.mu.Unlock()

	// Signal all workers to stop
	for _, w := range p.workers {
		close(w.quit)
	}

	// Wait for all workers to finish
	p.Wait()

	// Close channels
	close(p.tasks)
	close(p.results)

	// Close result channels for pending tasks
	p.mu.Lock()
	for _, task := range p.pending {
		close(task.Result)
	}
	p.mu.Unlock()
}

// GetResult retrieves the result for a task
func (p *Pool) GetResult(taskID string) (TaskResult, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result, ok := p.completed[taskID]
	return result, ok
}

// dispatch distributes tasks to workers
func (p *Pool) dispatch() {
	for task := range p.tasks {
		// Find available worker
		sent := false
		for _, w := range p.workers {
			select {
			case w.tasks <- task:
				sent = true
				break
			default:
				continue
			}
			if sent {
				break
			}
		}

		// If no worker available, requeue task
		if !sent {
			p.tasks <- task
		}
	}
}

// start begins the worker's processing loop
func (w *Worker) start() {
	for {
		select {
		case task := <-w.tasks:
			w.processTask(task)
		case <-w.quit:
			return
		}
	}
}

// processTask executes a task with timeout and retry logic
func (w *Worker) processTask(task *Task) {
	w.pool.wg.Add(1)
	defer w.pool.wg.Done()

	start := time.Now()
	var result TaskResult

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), task.Timeout)
	defer cancel()

	// Execute task with retry logic
	for retry := 0; retry <= task.RetryCount; retry++ {
		done := make(chan bool)
		go func() {
			output, err := task.Execute()
			result = TaskResult{
				TaskID: task.ID,
				Output: output,
				Error:  err,
				Time:   time.Now(),
			}
			done <- true
		}()

		select {
		case <-ctx.Done():
			result = TaskResult{
				TaskID: task.ID,
				Error:  ctx.Err(),
				Time:   time.Now(),
			}
			break
		case <-done:
			if result.Error == nil {
				break
			}
			if retry < task.RetryCount {
				time.Sleep(time.Second * time.Duration(retry+1))
				continue
			}
		}
		break
	}

	// Update metrics
	w.metrics.mu.Lock()
	w.metrics.TasksProcessed++
	w.metrics.TotalTime += time.Since(start)
	if result.Error != nil {
		w.metrics.Errors++
	}
	w.metrics.mu.Unlock()

	// Store result
	w.pool.mu.Lock()
	delete(w.pool.pending, task.ID)
	w.pool.completed[task.ID] = result
	w.pool.mu.Unlock()

	// Send result
	task.Result <- result
}
