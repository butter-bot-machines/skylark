package worker

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"time"
)

// ResourceLimits defines resource usage limits for jobs
type ResourceLimits struct {
	MaxMemory   int64         // Maximum memory in bytes
	MaxCPUTime  time.Duration // Maximum CPU time per job
	MaxFileSize int64         // Maximum file size in bytes
}

// DefaultLimits returns default resource limits
func DefaultLimits() ResourceLimits {
	return ResourceLimits{
		MaxMemory:   256 * 1024 * 1024, // 256MB - more restrictive for testing
		MaxCPUTime:  50 * time.Millisecond, // 50ms - even more restrictive for testing
		MaxFileSize: 10 * 1024 * 1024,   // 10MB
	}
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
func runWithCPULimit(ctx context.Context, limit time.Duration, fn func() error) error {
	done := make(chan error, 1)
	workCtx, cancel := context.WithTimeout(ctx, limit)
	defer cancel()

	// Save current GOMAXPROCS
	oldMaxProcs := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(oldMaxProcs)

	// Enable CPU profiling with high frequency
	runtime.SetCPUProfileRate(1000)
	defer runtime.SetCPUProfileRate(0)

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

		// Create a timer for CPU limit
		timer := time.NewTimer(limit)
		defer timer.Stop()

		// Run the function with timeout
		errChan := make(chan error, 1)
		go func() {
			errChan <- fn()
		}()

		select {
		case err := <-errChan:
			done <- err
		case <-timer.C:
			panic("CPU time limit exceeded")
		case <-workCtx.Done():
			panic("CPU time limit exceeded")
		}
	}()

	// Wait for completion or timeout
	select {
	case err := <-done:
		return err
	case <-workCtx.Done():
		return fmt.Errorf("CPU time limit exceeded")
	}
}
