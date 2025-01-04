package real

import (
	"runtime"
	"runtime/debug"
	"sync"

	"github.com/butter-bot-machines/skylark/pkg/resources"
)

// Controller implements resources.Controller using real system resources
type Controller struct {
	mu sync.RWMutex

	// Memory
	memoryLimit int64

	// CPU
	cpuLimit float64

	// Profile
	profiling bool
}

// New creates a new real resource controller
func New() *Controller {
	return &Controller{}
}

// SetMemoryLimit sets the maximum memory limit
func (c *Controller) SetMemoryLimit(bytes int64) error {
	if bytes < 0 {
		return resources.ErrInvalidLimit
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.memoryLimit = bytes
	debug.SetMemoryLimit(bytes)
	return nil
}

// GetMemoryUsage returns the current memory usage
func (c *Controller) GetMemoryUsage() int64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return int64(m.Alloc)
}

// ForceGC forces garbage collection
func (c *Controller) ForceGC() {
	runtime.GC()
}

// SetCPULimit sets the maximum CPU usage
func (c *Controller) SetCPULimit(cores int) error {
	if cores < 0 {
		return resources.ErrInvalidLimit
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.cpuLimit = float64(cores)
	runtime.GOMAXPROCS(cores)
	return nil
}

// GetCPUUsage returns the current CPU usage
// Note: This is an approximation based on GOMAXPROCS
func (c *Controller) GetCPUUsage() float64 {
	return float64(runtime.GOMAXPROCS(0))
}

// LockThread locks the calling goroutine to its current operating system thread
func (c *Controller) LockThread() error {
	runtime.LockOSThread()
	return nil
}

// UnlockThread unlocks the calling goroutine from its operating system thread
func (c *Controller) UnlockThread() {
	runtime.UnlockOSThread()
}

// StartProfiling starts memory profiling
func (c *Controller) StartProfiling() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.profiling {
		return resources.ErrProfileActive
	}

	// Set memory profiling rate (1 = profile all allocations)
	runtime.MemProfileRate = 1
	c.profiling = true
	return nil
}

// StopProfiling stops memory profiling
func (c *Controller) StopProfiling() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.profiling {
		return resources.ErrProfileInactive
	}

	// Disable memory profiling
	runtime.MemProfileRate = 0
	c.profiling = false
	return nil
}

// getMemoryLimit returns the current memory limit
func (c *Controller) getMemoryLimit() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.memoryLimit
}

// getCPULimit returns the current CPU limit
func (c *Controller) getCPULimit() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cpuLimit
}

// isProfiling returns whether profiling is active
func (c *Controller) isProfiling() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.profiling
}
