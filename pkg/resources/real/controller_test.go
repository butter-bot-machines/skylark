package real

import (
	"runtime"
	"sync"
	"testing"

	"github.com/butter-bot-machines/skylark/pkg/resources"
)

func TestController_MemoryManagement(t *testing.T) {
	c := New()

	// Test memory limits
	t.Run("Memory Limits", func(t *testing.T) {
		// Set valid limit
		if err := c.SetMemoryLimit(1 << 30); err != nil {
			t.Errorf("SetMemoryLimit failed: %v", err)
		}

		// Test invalid limit
		if err := c.SetMemoryLimit(-1); err != resources.ErrInvalidLimit {
			t.Errorf("Got error %v, want ErrInvalidLimit", err)
		}

		// Test memory usage
		usage := c.GetMemoryUsage()
		if usage <= 0 {
			t.Errorf("Got invalid memory usage: %d", usage)
		}
	})

	// Test garbage collection
	t.Run("Garbage Collection", func(t *testing.T) {
		// Allocate some memory
		data := make([]byte, 1<<20)
		runtime.KeepAlive(data)

		before := c.GetMemoryUsage()
		c.ForceGC()
		after := c.GetMemoryUsage()

		// Memory usage should be different after GC
		if before == after {
			t.Error("Memory usage unchanged after GC")
		}
	})
}

func TestController_CPUManagement(t *testing.T) {
	c := New()

	// Test CPU limits
	t.Run("CPU Limits", func(t *testing.T) {
		original := runtime.GOMAXPROCS(0)
		defer runtime.GOMAXPROCS(original)

		// Set valid limit
		cores := 2
		if err := c.SetCPULimit(cores); err != nil {
			t.Errorf("SetCPULimit failed: %v", err)
		}

		if got := runtime.GOMAXPROCS(0); got != cores {
			t.Errorf("Got GOMAXPROCS %d, want %d", got, cores)
		}

		// Test invalid limit
		if err := c.SetCPULimit(-1); err != resources.ErrInvalidLimit {
			t.Errorf("Got error %v, want ErrInvalidLimit", err)
		}

		// Test CPU usage
		usage := c.GetCPUUsage()
		if usage <= 0 {
			t.Errorf("Got invalid CPU usage: %f", usage)
		}
	})
}

func TestController_ThreadManagement(t *testing.T) {
	c := New()

	// Test thread locking
	t.Run("Thread Locking", func(t *testing.T) {
		// Lock thread
		if err := c.LockThread(); err != nil {
			t.Errorf("LockThread failed: %v", err)
		}

		// Unlock thread
		c.UnlockThread()

		// Lock and unlock again to ensure it's reusable
		if err := c.LockThread(); err != nil {
			t.Errorf("Second LockThread failed: %v", err)
		}
		c.UnlockThread()
	})
}

func TestController_ProfileManagement(t *testing.T) {
	c := New()

	// Test profiling
	t.Run("Profiling", func(t *testing.T) {
		// Start profiling
		if err := c.StartProfiling(); err != nil {
			t.Errorf("StartProfiling failed: %v", err)
		}

		if !c.isProfiling() {
			t.Error("Profiling not marked as active")
		}

		// Try starting again
		if err := c.StartProfiling(); err != resources.ErrProfileActive {
			t.Errorf("Got error %v, want ErrProfileActive", err)
		}

		// Stop profiling
		if err := c.StopProfiling(); err != nil {
			t.Errorf("StopProfiling failed: %v", err)
		}

		if c.isProfiling() {
			t.Error("Profiling still marked as active after stop")
		}

		// Try stopping again
		if err := c.StopProfiling(); err != resources.ErrProfileInactive {
			t.Errorf("Got error %v, want ErrProfileInactive", err)
		}
	})
}

func TestController_Concurrency(t *testing.T) {
	c := New()
	var wg sync.WaitGroup
	workers := 10
	iterations := 100

	// Test concurrent memory operations
	t.Run("Concurrent Memory", func(t *testing.T) {
		wg.Add(workers)
		for i := 0; i < workers; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					c.GetMemoryUsage()
					if j%10 == 0 {
						c.ForceGC()
					}
				}
			}(i)
		}
		wg.Wait()
	})

	// Test concurrent CPU operations
	t.Run("Concurrent CPU", func(t *testing.T) {
		wg.Add(workers)
		for i := 0; i < workers; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					c.GetCPUUsage()
				}
			}(i)
		}
		wg.Wait()
	})

	// Test concurrent thread operations
	t.Run("Concurrent Threads", func(t *testing.T) {
		wg.Add(workers)
		for i := 0; i < workers; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					c.LockThread()
					c.UnlockThread()
				}
			}(i)
		}
		wg.Wait()
	})

	// Test concurrent profile operations
	t.Run("Concurrent Profiling", func(t *testing.T) {
		wg.Add(workers)
		for i := 0; i < workers; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					if j%2 == 0 {
						c.StartProfiling()
					} else {
						c.StopProfiling()
					}
					c.isProfiling()
				}
			}(i)
		}
		wg.Wait()

		// Stop profiling at the end
		c.StopProfiling()
		if c.isProfiling() {
			t.Error("Profiling still active after test")
		}
	})
}
