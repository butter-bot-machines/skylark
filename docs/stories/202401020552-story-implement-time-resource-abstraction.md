# Implement Time and Resource Abstraction

## Problem
Direct time and resource management makes testing difficult:

```go
// Direct timer usage
timer := time.NewTimer(limit)
time.AfterFunc(d.delay, callback)
time.After(1 * time.Second)

// Direct resource control
debug.SetMemoryLimit(limit)
debug.SetGCPercent(10)
runtime.GC()
runtime.LockOSThread()

// Fixed timeouts/delays
const apiTimeout = 30 * time.Second
MaxCPUTime: 50 * time.Millisecond
delay: 100 * time.Millisecond

// Timer state management
if timer, exists := d.timers[key]; exists {
    timer.Stop()
}
defer timer.Stop()
```

This means:
1. Tests are time-dependent
2. Tests are resource-dependent
3. Tests are slow
4. Tests are flaky

## Solution

After evaluating options for time mocking, we'll use benbjohnson/clock which provides:
- A well-tested Clock interface that mimics Go's time package
- Both real and mock implementations
- Proper handling of timers and tickers
- Avoidance of common pitfalls in time mocking

1. Create Time Interface wrapping benbjohnson/clock:
```go
// pkg/timing/interface.go
type Clock interface {
    // Time operations
    Now() time.Time
    Sleep(d time.Duration)
    After(d time.Duration) <-chan time.Time
    
    // Timer operations
    NewTimer(d time.Duration) Timer
    AfterFunc(d time.Duration, f func()) Timer
    
    // Ticker operations
    NewTicker(d time.Duration) Ticker
}

// Timer and Ticker interfaces match benbjohnson/clock
type Timer interface {
    C() <-chan time.Time
    Stop() bool
    Reset(d time.Duration) bool
}

type Ticker interface {
    C() <-chan time.Time
    Stop()
}

// New returns a real clock implementation
func New() Clock {
    return clock.New()
}

// NewMock returns a mock clock for testing
func NewMock() Clock {
    return clock.NewMock()
}
```

2. Create Resource Interface:
```go
// pkg/resources/interface.go
type ResourceController interface {
    // Memory management
    SetMemoryLimit(bytes int64) error
    GetMemoryUsage() int64
    ForceGC()
    
    // CPU management
    SetCPULimit(cores int) error
    GetCPUUsage() float64
    LockThread() error
    UnlockThread()
    
    // Profile management
    StartProfiling() error
    StopProfiling() error
}

type ResourceLimits struct {
    MaxMemory   int64
    MaxCPU      float64
    MaxThreads  int
    ProfileRate int
}
```

3. Example Usage:
```go
// Production code
type Worker struct {
    clock timing.Clock
    resources resources.ResourceController
}

func NewWorker() *Worker {
    return &Worker{
        clock: timing.New(), // Uses real clock
        resources: resources.New(),
    }
}

// Test code
func TestWorker(t *testing.T) {
    mock := timing.NewMock()
    w := &Worker{
        clock: mock,
        resources: resources.NewMock(),
    }
    
    // Control time in tests
    mock.Add(5 * time.Second)
}
```

## Benefits

1. Testing:
   - Control time flow using proven library
   - Simulate delays reliably
   - Mock resources
   - Fast execution

2. Production:
   - Same interface
   - No behavior changes
   - Better control
   - Better monitoring

3. Future:
   - Custom timing
   - Resource pools
   - Better profiling
   - Better debugging

## Implementation

1. Core Changes:
   - Add benbjohnson/clock dependency
   - Create timing package wrapping clock
   - Create resources package
   - Add interfaces
   - Add implementations

2. Component Updates:
   - Update debouncer to use Clock
   - Update worker to use Clock
   - Update processor to use Clock
   - Update sandbox to use ResourceController

3. Test Support:
   - Use mock clock from benbjohnson/clock
   - Add mock resources
   - Update test helpers
   - Add examples

## Acceptance Criteria

1. Functionality:
   - [ ] All time operations use Clock
   - [ ] All resource ops use Controller
   - [ ] Production behavior unchanged
   - [ ] Proper cleanup

2. Testing:
   - [ ] Tests control time via benbjohnson/clock
   - [ ] Tests control resources
   - [ ] Fast execution
   - [ ] No flakiness

3. Documentation:
   - [ ] Interface docs
   - [ ] Implementation docs
   - [ ] Test patterns
   - [ ] Examples

## Technical Notes

We chose benbjohnson/clock over implementing our own solution because:
1. It's a mature, well-tested library
2. It properly handles edge cases in time mocking
3. It closely mimics Go's time package behavior
4. It avoids common pitfalls like goroutine scheduling issues
5. It reduces maintenance burden vs custom implementation
