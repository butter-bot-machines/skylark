package os

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/butter-bot-machines/skylark/pkg/process"
	"github.com/butter-bot-machines/skylark/pkg/timing"
)

// getCurrentMemoryUsage reads the current memory usage from cgroup stats
func getCurrentMemoryUsage(pid int) (int64, error) {
	cgroupPath := filepath.Join("/sys/fs/cgroup", "memory", "skylark-"+strconv.Itoa(pid))
	data, err := os.ReadFile(filepath.Join(cgroupPath, "memory.usage_in_bytes"))
	if err != nil {
		return 0, err
	}
	bytes, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		return 0, err
	}
	return bytes / (1024 * 1024), nil // Convert to MB
}

func TestProcess_MemoryLimit(t *testing.T) {
	// Skip if not running on Linux
	if _, err := os.Stat("/sys/fs/cgroup"); os.IsNotExist(err) {
		t.Skip("Cgroups not available - skipping memory limit tests")
	}

	mock := timing.NewMock()
	mgr := NewManager(mock)

	t.Run("Memory Limit Enforcement", func(t *testing.T) {
		// Create a memory-intensive process
		proc := mgr.New("python3", []string{"-c", `
import array
# Allocate memory in chunks
mem = []
while True:
    # Allocate 10MB at a time
    mem.append(array.array('b', [0] * (10 * 1024 * 1024)))
`})

		// Set a 50MB memory limit
		limits := process.ResourceLimits{
			MaxMemoryMB: 50,
		}
		if err := proc.SetLimits(limits); err != nil {
			t.Fatalf("SetLimits failed: %v", err)
		}

		// Start the process
		if err := proc.Start(); err != nil {
			t.Skipf("Could not start test process: %v", err)
		}

		// Wait for process to be killed by memory limit
		err := proc.Wait()
		if err == nil {
			t.Error("Process should have been killed by memory limit")
		}

		// Verify cgroup was cleaned up
		cgroupPath := filepath.Join("/sys/fs/cgroup", "memory", "skylark-"+strconv.Itoa(proc.ID()))
		if _, err := os.Stat(cgroupPath); !os.IsNotExist(err) {
			t.Error("Cgroup directory should have been cleaned up")
		}
	})

	t.Run("Within Memory Limit", func(t *testing.T) {
		// Create a process that stays within memory limit
		proc := mgr.New("python3", []string{"-c", `
import array
# Allocate 20MB once
mem = array.array('b', [0] * (20 * 1024 * 1024))
`})

		// Set a 50MB memory limit
		limits := process.ResourceLimits{
			MaxMemoryMB: 50,
		}
		if err := proc.SetLimits(limits); err != nil {
			t.Fatalf("SetLimits failed: %v", err)
		}

		// Start and wait for the process
		if err := proc.Start(); err != nil {
			t.Skipf("Could not start test process: %v", err)
		}

		// Process should complete normally
		if err := proc.Wait(); err != nil {
			t.Errorf("Process failed within memory limit: %v", err)
		}

		// Verify cgroup was cleaned up
		cgroupPath := filepath.Join("/sys/fs/cgroup", "memory", "skylark-"+strconv.Itoa(proc.ID()))
		if _, err := os.Stat(cgroupPath); !os.IsNotExist(err) {
			t.Error("Cgroup directory should have been cleaned up")
		}
	})

	t.Run("Memory Usage Tracking", func(t *testing.T) {
		proc := mgr.New("python3", []string{"-c", `
import array
import time
# Allocate 30MB and hold
mem = array.array('b', [0] * (30 * 1024 * 1024))
time.sleep(1)
`})

		// Set a high memory limit
		limits := process.ResourceLimits{
			MaxMemoryMB: 100,
		}
		if err := proc.SetLimits(limits); err != nil {
			t.Fatalf("SetLimits failed: %v", err)
		}

		// Start the process
		if err := proc.Start(); err != nil {
			t.Skipf("Could not start test process: %v", err)
		}

		// Let memory allocation happen
		time.Sleep(100 * time.Millisecond)

		// Check current memory usage
		usage, err := getCurrentMemoryUsage(proc.ID())
		if err != nil {
			t.Fatalf("Failed to get memory usage: %v", err)
		}

		// Should be around 30MB (allow some overhead)
		if usage < 25 || usage > 40 {
			t.Errorf("Expected memory usage around 30MB, got %dMB", usage)
		}

		// Wait for completion
		if err := proc.Wait(); err != nil {
			t.Errorf("Process failed: %v", err)
		}
	})
}

func TestProcess_CgroupCleanup(t *testing.T) {
	// Skip if not running on Linux
	if _, err := os.Stat("/sys/fs/cgroup"); os.IsNotExist(err) {
		t.Skip("Cgroups not available - skipping cleanup tests")
	}

	mock := timing.NewMock()
	mgr := NewManager(mock)

	t.Run("Cleanup After Normal Exit", func(t *testing.T) {
		proc := mgr.New("echo", []string{"test"})
		limits := process.ResourceLimits{
			MaxMemoryMB: 50,
		}
		proc.SetLimits(limits)
		proc.Start()
		proc.Wait()

		// Verify cgroup cleanup
		cgroupPath := filepath.Join("/sys/fs/cgroup", "memory", "skylark-"+strconv.Itoa(proc.ID()))
		if _, err := os.Stat(cgroupPath); !os.IsNotExist(err) {
			t.Error("Cgroup directory should have been cleaned up")
		}
	})

	t.Run("Cleanup After Kill", func(t *testing.T) {
		proc := mgr.New("sleep", []string{"10"})
		limits := process.ResourceLimits{
			MaxMemoryMB: 50,
		}
		proc.SetLimits(limits)
		proc.Start()
		proc.Signal(os.Kill)
		proc.Wait()

		// Verify cgroup cleanup
		cgroupPath := filepath.Join("/sys/fs/cgroup", "memory", "skylark-"+strconv.Itoa(proc.ID()))
		if _, err := os.Stat(cgroupPath); !os.IsNotExist(err) {
			t.Error("Cgroup directory should have been cleaned up")
		}
	})
}
