package worker

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/butter-bot-machines/skylark/pkg/timing"
)

// ResourceLimits defines resource usage limits for jobs
type ResourceLimits struct {
	MaxMemory    int64         // Maximum memory in bytes
	MaxCPUTime   time.Duration // Maximum CPU time per job
	MaxFileSize  int64         // Maximum file size in bytes
	MaxFiles     int64         // Maximum number of open files
	MaxProcesses int64         // Maximum number of processes
	clock        timing.Clock  // Clock for timing operations
}

// DefaultLimits returns default resource limits
func DefaultLimits() ResourceLimits {
	return ResourceLimits{
		MaxMemory:    256 * 1024 * 1024,    // 256MB - more restrictive for testing
		MaxCPUTime:   50 * time.Millisecond, // 50ms - even more restrictive for testing
		MaxFileSize:  10 * 1024 * 1024,      // 10MB
		MaxFiles:     100,                   // 100 files
		MaxProcesses: 10,                    // 10 processes
		clock:       timing.New(),           // Use real clock by default
	}
}

// WithClock returns a copy of ResourceLimits with the specified clock
func (l ResourceLimits) WithClock(clock timing.Clock) ResourceLimits {
	l.clock = clock
	return l
}

// enforceMemoryLimit sets up memory limit enforcement
func enforceMemoryLimit(limit int64) {
	// Set soft memory limit
	debug.SetMemoryLimit(limit)
	
	// Set GOGC to trigger GC more frequently
	debug.SetGCPercent(10)
	
	// Force immediate GC
	runtime.GC()
	debug.FreeOSMemory()
}

// runWithCPULimit runs a function with CPU time limit
func runWithCPULimit(ctx context.Context, limits ResourceLimits, fn func() error) error {
	done := make(chan error, 1)

	// Save current GOMAXPROCS
	oldMaxProcs := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(oldMaxProcs)

	// Run the work in a dedicated goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("CPU limit exceeded: %v", r)
				return
			}
		}()

		// Lock goroutine to OS thread
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		// Run the function
		done <- fn()
	}()

	// Wait for completion or timeout
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("CPU time limit exceeded")
	}
}
