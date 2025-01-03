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

1. Create Time Interface:
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

type Timer interface {
    C() <-chan time.Time
    Stop() bool
    Reset(d time.Duration) bool
}

type Ticker interface {
    C() <-chan time.Time
    Stop()
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

3. Add Production Implementation:
```go
// pkg/timing/real/clock.go
type RealClock struct{}

func (c *RealClock) Now() time.Time {
    return time.Now()
}

func (c *RealClock) NewTimer(d time.Duration) Timer {
    return &realTimer{time.NewTimer(d)}
}

// pkg/resources/real/controller.go
type RealController struct {
    limits ResourceLimits
}

func (c *RealController) SetMemoryLimit(bytes int64) error {
    debug.SetMemoryLimit(bytes)
    return nil
}
```

4. Add Test Implementation:
```go
// pkg/timing/mock/clock.go
type MockClock struct {
    now    time.Time
    timers []*mockTimer
    mu     sync.Mutex
}

func (c *MockClock) Advance(d time.Duration) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    c.now = c.now.Add(d)
    for _, t := range c.timers {
        t.tryFire(c.now)
    }
}

// pkg/resources/mock/controller.go
type MockController struct {
    memory   int64
    cpu      float64
    threads  int
    profiling bool
}

func (c *MockController) SetMemoryLimit(bytes int64) error {
    c.memory = bytes
    return nil
}
```

5. Update Components:
```go
// pkg/watcher/debouncer.go
type Debouncer struct {
    clock  timing.Clock
    timers map[string]timing.Timer
}

func (d *Debouncer) Debounce(key string, callback func()) {
    if timer, exists := d.timers[key]; exists {
        timer.Stop()
    }
    
    d.timers[key] = d.clock.AfterFunc(d.delay, callback)
}

// pkg/worker/limits.go
type Worker struct {
    resources resources.ResourceController
    limits    ResourceLimits
}

func (w *Worker) enforceResourceLimits() error {
    if err := w.resources.SetMemoryLimit(w.limits.MaxMemory); err != nil {
        return err
    }
    return w.resources.SetCPULimit(w.limits.MaxCPU)
}
```

## Benefits

1. Testing:
   - Control time flow
   - Simulate delays
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
   - Create timing package
   - Create resources package
   - Add interfaces
   - Add implementations

2. Component Updates:
   - Update debouncer
   - Update worker
   - Update processor
   - Update sandbox

3. Test Support:
   - Add mock clock
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
   - [ ] Tests control time
   - [ ] Tests control resources
   - [ ] Fast execution
   - [ ] No flakiness

3. Documentation:
   - [ ] Interface docs
   - [ ] Implementation docs
   - [ ] Test patterns
   - [ ] Examples
