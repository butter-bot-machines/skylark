# Implement Worker Pool Abstraction

## Problem
The worker pool currently has tight coupling:
- Creates own process manager
- Direct resource management
- Creates own job queue
- Direct processor access

This makes it difficult to:
1. Test worker behavior
2. Mock job processing
3. Control resource usage
4. Customize job handling

## Solution

1. Job Queue Interface:
```go
// pkg/worker/interface.go
type Queue interface {
    // Push adds a job to the queue
    Push(job Job) error
    
    // Pop removes and returns the next job
    Pop() (Job, error)
    
    // Len returns the current queue length
    Len() int
    
    // Close closes the queue
    Close() error
}

type Job interface {
    // Process executes the job
    Process() error
    
    // OnFailure handles job failure
    OnFailure(error)
    
    // MaxRetries returns max retry count
    MaxRetries() int
}
```

2. Worker Interface:
```go
type Worker interface {
    // Start starts the worker
    Start() error
    
    // Stop stops the worker
    Stop() error
    
    // ID returns the worker ID
    ID() int
    
    // Status returns worker status
    Status() WorkerStatus
}

type WorkerStatus struct {
    Running      bool
    CurrentJob   *JobInfo
    ProcessedJobs uint64
    FailedJobs   uint64
}
```

3. Resource Controller Interface:
```go
type ResourceController interface {
    // Check verifies resource availability
    Check(usage ResourceUsage) error
    
    // Acquire reserves resources
    Acquire(usage ResourceUsage) error
    
    // Release frees resources
    Release(usage ResourceUsage)
    
    // Status returns current usage
    Status() ResourceStatus
}

type ResourceUsage struct {
    Memory int64
    CPU    float64
    Files  int
}
```

4. Pool Manager Interface:
```go
type Manager interface {
    // Start starts the worker pool
    Start() error
    
    // Stop stops the worker pool
    Stop() error
    
    // Queue returns the job queue
    Queue() Queue
    
    // Status returns pool status
    Status() PoolStatus
}

type PoolStatus struct {
    Workers       int
    QueueSize     int
    ProcessedJobs uint64
    FailedJobs    uint64
    ActiveJobs    int
}
```

## Implementation

1. Create Interfaces:
```go
// pkg/worker/queue/queue.go
type channelQueue struct {
    jobs chan Job
    size int
    mu   sync.RWMutex
}

// pkg/worker/worker/worker.go
type basicWorker struct {
    id       int
    queue    Queue
    resource ResourceController
    stats    *WorkerStats
}

// pkg/worker/resource/controller.go
type basicController struct {
    limits ResourceLimits
    usage  ResourceUsage
    mu     sync.RWMutex
}

// pkg/worker/pool/manager.go
type basicManager struct {
    workers  []Worker
    queue    Queue
    resource ResourceController
    stats    *PoolStats
}
```

2. Create Mock Implementations:
```go
// pkg/worker/mock/queue.go
type mockQueue struct {
    jobs []Job
}

// pkg/worker/mock/worker.go
type mockWorker struct {
    id     int
    status WorkerStatus
}

// pkg/worker/mock/resource.go
type mockController struct {
    usage ResourceUsage
}

// pkg/worker/mock/manager.go
type mockManager struct {
    status PoolStatus
}
```

## Testing Strategy

1. Queue Tests:
   - Test job pushing
   - Test job popping
   - Test queue length
   - Test concurrent access

2. Worker Tests:
   - Test job processing
   - Test error handling
   - Test resource usage
   - Test state management

3. Resource Tests:
   - Test limit checking
   - Test resource tracking
   - Test concurrent usage
   - Test cleanup

4. Manager Tests:
   - Test pool lifecycle
   - Test worker management
   - Test job distribution
   - Test status reporting

## Migration Guide

1. Old Way:
```go
pool, err := worker.NewPool(worker.Options{
    Workers: 4,
    QueueSize: 100,
})
```

2. New Way:
```go
pool, err := worker.NewPool(worker.Options{
    Queue: queue.New(100),
    Resource: resource.NewController(limits),
    Workers: worker.NewWorkers(4),
})
```

## Benefits

1. Testing:
   - Mock job processing
   - Control resource usage
   - Verify worker behavior
   - Test error handling

2. Flexibility:
   - Custom queuing
   - Custom resource management
   - Custom worker behavior
   - Custom error handling

3. Monitoring:
   - Detailed status
   - Resource tracking
   - Error reporting
   - Performance metrics

## Success Criteria

1. Technical:
   - All interfaces defined
   - Mock implementations working
   - Tests passing
   - No direct coupling

2. Usability:
   - Clear documentation
   - Simple job creation
   - Easy resource control
   - Flexible configuration

3. Migration:
   - Existing code works
   - Clear upgrade path
   - No breaking changes
   - Good examples

## References

1. Related Stories:
   - [202501020546](202501020546-story-improve-testability.md)
   - [202501020548](202501020548-story-identify-coupling-patterns.md)
   - [202501020556](202501020556-analyze-core-coupling.md)
   - [202501020557](202501020557-implement-error-abstraction.md)
   - [202501020558](202501020558-implement-security-abstraction.md)

2. Documentation:
   - [Architecture](../architecture.md)
   - [Implementation Plan](implementation-plan.md)
   - [Dev Log](../dev_log.md)
