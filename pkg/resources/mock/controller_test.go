package mock

import (
	"sync"
	"testing"

	"github.com/butter-bot-machines/skylark/pkg/resources"
)

func TestController_MemoryManagement(t *testing.T) {
	c := New()

	// Test memory limits
	t.Run("Memory Limits", func(t *testing.T) {
		// Set valid limit
		if err := c.SetMemoryLimit(100); err != nil {
			t.Errorf("SetMemoryLimit failed: %v", err)
		}

		// Test invalid limit
		if err := c.SetMemoryLimit(-1); err != resources.ErrInvalidLimit {
			t.Errorf("Got error %v, want ErrInvalidLimit", err)
		}

		// Test allocation within limit
		if err := c.SimulateAllocation(50); err != nil {
			t.Errorf("SimulateAllocation failed: %v", err)
		}

		if usage := c.GetMemoryUsage(); usage != 50 {
			t.Errorf("Got usage %d, want 50", usage)
		}

		// Test allocation exceeding limit
		if err := c.SimulateAllocation(60); err != resources.ErrLimitExceeded {
			t.Errorf("Got error %v, want ErrLimitExceeded", err)
		}
	})

	// Test garbage collection
	t.Run("Garbage Collection", func(t *testing.T) {
		c.SimulateAllocation(100)
		before := c.GetMemoryUsage()
		c.ForceGC()
		after := c.GetMemoryUsage()

		if after >= before {
			t.Errorf("Memory usage not reduced after GC: before=%d, after=%d", before, after)
		}
	})
}

func TestController_CPUManagement(t *testing.T) {
	c := New()

	// Test CPU limits
	t.Run("CPU Limits", func(t *testing.T) {
		// Set valid limit
		if err := c.SetCPULimit(2); err != nil {
			t.Errorf("SetCPULimit failed: %v", err)
		}

		// Test invalid limit
		if err := c.SetCPULimit(-1); err != resources.ErrInvalidLimit {
			t.Errorf("Got error %v, want ErrInvalidLimit", err)
		}

		// Test usage within limit
		if err := c.SimulateCPUUsage(1.5); err != nil {
			t.Errorf("SimulateCPUUsage failed: %v", err)
		}

		if usage := c.GetCPUUsage(); usage != 1.5 {
			t.Errorf("Got usage %f, want 1.5", usage)
		}

		// Test usage exceeding limit
		if err := c.SimulateCPUUsage(2.5); err != resources.ErrLimitExceeded {
			t.Errorf("Got error %v, want ErrLimitExceeded", err)
		}
	})
}

func TestController_ThreadManagement(t *testing.T) {
	c := New()

	// Test thread locking
	t.Run("Thread Locking", func(t *testing.T) {
		// Lock first thread
		if err := c.LockThread(); err != nil {
			t.Errorf("LockThread failed: %v", err)
		}

		if !c.IsThreadLocked(0) {
			t.Error("Thread not marked as locked")
		}

		// Lock second thread
		if err := c.LockThread(); err != nil {
			t.Errorf("LockThread failed: %v", err)
		}

		// Unlock thread
		c.UnlockThread()
		if c.IsThreadLocked(1) {
			t.Error("Thread still marked as locked after unlock")
		}
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

		if !c.IsProfiling() {
			t.Error("Profiling not marked as active")
		}

		if rate := c.GetProfileRate(); rate != 1 {
			t.Errorf("Got profile rate %d, want 1", rate)
		}

		// Try starting again
		if err := c.StartProfiling(); err != resources.ErrProfileActive {
			t.Errorf("Got error %v, want ErrProfileActive", err)
		}

		// Stop profiling
		if err := c.StopProfiling(); err != nil {
			t.Errorf("StopProfiling failed: %v", err)
		}

		if c.IsProfiling() {
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
	c.SetMemoryLimit(1000)
	c.SetCPULimit(4)

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
					c.SimulateAllocation(1)
					c.GetMemoryUsage()
					if j%10 == 0 {
						c.ForceGC()
					}
				}
			}(i)
		}
		wg.Wait()

		if usage := c.GetMemoryUsage(); usage > 1000 {
			t.Errorf("Memory limit exceeded: got %d, limit 1000", usage)
		}
	})

	// Test concurrent CPU operations
	t.Run("Concurrent CPU", func(t *testing.T) {
		wg.Add(workers)
		for i := 0; i < workers; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					c.SimulateCPUUsage(float64(j%4) + 0.5)
					c.GetCPUUsage()
				}
			}(i)
		}
		wg.Wait()

		if usage := c.GetCPUUsage(); usage > 4 {
			t.Errorf("CPU limit exceeded: got %f, limit 4", usage)
		}
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

		// All threads should be unlocked
		for i := 0; i < workers; i++ {
			if c.IsThreadLocked(i) {
				t.Errorf("Thread %d still locked", i)
			}
		}
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
					c.IsProfiling()
					c.GetProfileRate()
				}
			}(i)
		}
		wg.Wait()

		// Stop profiling at the end
		c.StopProfiling()
		if c.IsProfiling() {
			t.Error("Profiling still active after test")
		}
	})
}
