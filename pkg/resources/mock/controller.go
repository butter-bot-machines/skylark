package mock

import (
	"sync"
	"sync/atomic"

	"github.com/butter-bot-machines/skylark/pkg/resources"
)

// Controller implements resources.Controller for testing
type Controller struct {
	mu sync.RWMutex

	// Memory
	memoryLimit int64
	memoryUsage int64

	// CPU
	cpuLimit  float64
	cpuUsage  float64
	threadIDs map[int]bool

	// Profile
	profiling   bool
	profileRate int
}

// New creates a new mock resource controller
func New() *Controller {
	return &Controller{
		threadIDs: make(map[int]bool),
	}
}

// SetMemoryLimit sets the maximum memory limit
func (c *Controller) SetMemoryLimit(bytes int64) error {
	if bytes < 0 {
		return resources.ErrInvalidLimit
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.memoryUsage > bytes {
		return resources.ErrLimitExceeded
	}

	c.memoryLimit = bytes
	return nil
}

// GetMemoryUsage returns the current memory usage
func (c *Controller) GetMemoryUsage() int64 {
	return atomic.LoadInt64(&c.memoryUsage)
}

// ForceGC simulates garbage collection
func (c *Controller) ForceGC() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Simulate GC by reducing memory usage by 50%
	atomic.StoreInt64(&c.memoryUsage, c.memoryUsage/2)
}

// SetCPULimit sets the maximum CPU usage
func (c *Controller) SetCPULimit(cores int) error {
	if cores < 0 {
		return resources.ErrInvalidLimit
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.cpuLimit = float64(cores)
	return nil
}

// GetCPUUsage returns the current CPU usage
func (c *Controller) GetCPUUsage() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cpuUsage
}

// LockThread simulates thread locking
func (c *Controller) LockThread() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	id := len(c.threadIDs)
	if _, ok := c.threadIDs[id]; ok {
		return resources.ErrThreadLocked
	}

	c.threadIDs[id] = true
	return nil
}

// UnlockThread simulates thread unlocking
func (c *Controller) UnlockThread() {
	c.mu.Lock()
	defer c.mu.Unlock()

	id := len(c.threadIDs) - 1
	delete(c.threadIDs, id)
}

// StartProfiling starts memory profiling
func (c *Controller) StartProfiling() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.profiling {
		return resources.ErrProfileActive
	}

	c.profiling = true
	c.profileRate = 1
	return nil
}

// StopProfiling stops memory profiling
func (c *Controller) StopProfiling() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.profiling {
		return resources.ErrProfileInactive
	}

	c.profiling = false
	c.profileRate = 0
	return nil
}

// SimulateAllocation simulates memory allocation for testing
func (c *Controller) SimulateAllocation(bytes int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	newUsage := c.memoryUsage + bytes
	if c.memoryLimit > 0 && newUsage > c.memoryLimit {
		return resources.ErrLimitExceeded
	}

	atomic.StoreInt64(&c.memoryUsage, newUsage)
	return nil
}

// SimulateCPUUsage simulates CPU usage for testing
func (c *Controller) SimulateCPUUsage(cores float64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if cores < 0 {
		return resources.ErrInvalidLimit
	}

	if c.cpuLimit > 0 && cores > c.cpuLimit {
		return resources.ErrLimitExceeded
	}

	c.cpuUsage = cores
	return nil
}

// GetProfileRate returns the current profiling rate
func (c *Controller) GetProfileRate() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.profileRate
}

// IsThreadLocked returns whether a thread is locked
func (c *Controller) IsThreadLocked(id int) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.threadIDs[id]
}

// IsProfiling returns whether profiling is active
func (c *Controller) IsProfiling() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.profiling
}
