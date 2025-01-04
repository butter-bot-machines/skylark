package os

import (
	"bytes"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/butter-bot-machines/skylark/pkg/process"
	"github.com/butter-bot-machines/skylark/pkg/timing"
)

func TestManager_BasicOperations(t *testing.T) {
	mock := timing.NewMock()
	mgr := NewManager(mock)

	// Test process creation
	t.Run("Process Creation", func(t *testing.T) {
		proc := mgr.New("test", []string{"-arg1", "-arg2"})
		if proc == nil {
			t.Error("New returned nil process")
		}

		// ID should be 0 before process starts
		if proc.ID() != 0 {
			t.Errorf("Got ID %d, want 0", proc.ID())
		}
	})

	// Test process retrieval
	t.Run("Process Retrieval", func(t *testing.T) {
		proc := mgr.New("test", []string{})
		if err := proc.Start(); err != nil {
			t.Skipf("Could not start test process: %v", err)
		}
		defer proc.Signal(os.Kill)

		pid := proc.ID()
		if pid == 0 {
			t.Error("Process ID should not be 0 after start")
		}

		got, err := mgr.Get(pid)
		if err != nil {
			t.Errorf("Get failed: %v", err)
		}
		if got == nil {
			t.Error("Get returned nil process")
		}

		// Test non-existent process
		if _, err := mgr.Get(999999); err != process.ErrNotFound {
			t.Errorf("Got error %v, want ErrNotFound", err)
		}
	})

	// Test process listing
	t.Run("Process Listing", func(t *testing.T) {
		procs := mgr.List()
		if len(procs) < 1 {
			t.Error("Expected at least one process")
		}
	})
}

func TestProcess_Lifecycle(t *testing.T) {
	mock := timing.NewMock()
	mgr := NewManager(mock)

	// Test initial state
	t.Run("Initial State", func(t *testing.T) {
		proc := mgr.New("test", []string{})
		if proc.Running() {
			t.Error("New process should not be running")
		}
		if code := proc.ExitCode(); code != 0 {
			t.Errorf("Got exit code %d, want 0", code)
		}
	})

	// Test start and wait
	t.Run("Start and Wait", func(t *testing.T) {
		proc := mgr.New("echo", []string{"test"})
		if err := proc.Start(); err != nil {
			t.Skipf("Could not start test process: %v", err)
		}
		if !proc.Running() {
			t.Error("Process should be running after Start")
		}

		// Test double start
		if err := proc.Start(); err != process.ErrAlreadyExists {
			t.Errorf("Got error %v, want ErrAlreadyExists", err)
		}

		if err := proc.Wait(); err != nil {
			t.Errorf("Wait failed: %v", err)
		}
		if proc.Running() {
			t.Error("Process should not be running after Wait")
		}
	})

	// Test IO redirection
	t.Run("IO Redirection", func(t *testing.T) {
		proc := mgr.New("echo", []string{"test"})
		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)

		proc.SetStdout(stdout)
		proc.SetStderr(stderr)

		if err := proc.Start(); err != nil {
			t.Skipf("Could not start test process: %v", err)
		}
		if err := proc.Wait(); err != nil {
			t.Errorf("Wait failed: %v", err)
		}

		if stdout.Len() == 0 {
			t.Error("Expected output from echo command")
		}
	})

	// Test resource limits
	t.Run("Resource Limits", func(t *testing.T) {
		proc := mgr.New("test", []string{})
		limits := process.ResourceLimits{
			MaxCPUTime:    time.Second,
			MaxMemoryMB:   100,
			MaxFileSizeMB: 10,
			MaxFiles:      100,
			MaxProcesses:  10,
		}

		if err := proc.SetLimits(limits); err != nil {
			t.Errorf("SetLimits failed: %v", err)
		}

		got := proc.GetLimits()
		if got != limits {
			t.Errorf("Got limits %+v, want %+v", got, limits)
		}
	})
}

func TestProcess_ResourceLimits(t *testing.T) {
	mock := timing.NewMock()
	mgr := NewManager(mock)

	// Test default limits
	t.Run("Default Limits", func(t *testing.T) {
		limits := process.ResourceLimits{
			MaxCPUTime:    time.Second,
			MaxMemoryMB:   100,
			MaxFileSizeMB: 10,
			MaxFiles:      100,
			MaxProcesses:  10,
		}

		mgr.SetDefaultLimits(limits)
		got := mgr.GetDefaultLimits()
		if got != limits {
			t.Errorf("Got limits %+v, want %+v", got, limits)
		}

		// New processes should inherit default limits
		proc := mgr.New("test", []string{})
		if got := proc.GetLimits(); got != limits {
			t.Errorf("Got limits %+v, want %+v", got, limits)
		}
	})

	// Test limit validation
	t.Run("Limit Validation", func(t *testing.T) {
		proc := mgr.New("test", []string{})

		invalidLimits := process.ResourceLimits{
			MaxCPUTime:    -1,
			MaxMemoryMB:   -1,
			MaxFileSizeMB: -1,
			MaxFiles:      -1,
			MaxProcesses:  -1,
		}

		if err := proc.SetLimits(invalidLimits); err != process.ErrInvalidLimits {
			t.Errorf("Got error %v, want ErrInvalidLimits", err)
		}
	})
}

func TestProcess_CPULimit(t *testing.T) {
	mock := timing.NewMock()
	mgr := NewManager(mock)

	t.Run("CPU Time Limit", func(t *testing.T) {
		proc := mgr.New("sleep", []string{"10"}) // Long-running process

		// Set a short CPU time limit
		limits := process.ResourceLimits{
			MaxCPUTime: 100 * time.Millisecond,
		}
		if err := proc.SetLimits(limits); err != nil {
			t.Fatalf("SetLimits failed: %v", err)
		}

		// Start the process
		if err := proc.Start(); err != nil {
			t.Skipf("Could not start test process: %v", err)
		}

		// Advance time past the CPU limit
		mock.Add(200 * time.Millisecond)

		// Process should be killed
		if err := proc.Wait(); err == nil {
			t.Error("Process should have been killed by CPU limit")
		}
	})

	t.Run("Within CPU Time Limit", func(t *testing.T) {
		proc := mgr.New("echo", []string{"test"}) // Quick process

		// Set a generous CPU time limit
		limits := process.ResourceLimits{
			MaxCPUTime: time.Second,
		}
		if err := proc.SetLimits(limits); err != nil {
			t.Fatalf("SetLimits failed: %v", err)
		}

		// Start and wait for the process
		if err := proc.Start(); err != nil {
			t.Skipf("Could not start test process: %v", err)
		}

		// Advance time a bit
		mock.Add(100 * time.Millisecond)

		// Process should complete normally
		if err := proc.Wait(); err != nil {
			t.Errorf("Process failed within CPU limit: %v", err)
		}
	})
}

func TestManager_Concurrency(t *testing.T) {
	mock := timing.NewMock()
	mgr := NewManager(mock)
	var wg sync.WaitGroup
	workers := 5
	iterations := 10

	t.Run("Concurrent Operations", func(t *testing.T) {
		wg.Add(workers)
		for i := 0; i < workers; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					// Create and start process
					proc := mgr.New("echo", []string{"test"})
					if err := proc.Start(); err != nil {
						continue // Skip if process start fails
					}

					// Get process by ID while it's running
					pid := proc.ID()
					if pid > 0 {
						if _, err := mgr.Get(pid); err != nil {
							t.Errorf("Get failed: %v", err)
						}

						// List processes while running
						if procs := mgr.List(); len(procs) < 1 {
							t.Error("List returned empty result")
						}
					}

					// Let process complete
					mock.Add(10 * time.Millisecond)
					proc.Wait()
				}
			}(i)
		}
		wg.Wait()
	})
}

func TestProcess_ErrorCases(t *testing.T) {
	mock := timing.NewMock()
	mgr := NewManager(mock)

	// Test operations on non-existent process
	t.Run("Non-existent Process", func(t *testing.T) {
		if _, err := mgr.Get(999); err != process.ErrNotFound {
			t.Errorf("Got error %v, want ErrNotFound", err)
		}
	})

	// Test invalid operations
	t.Run("Invalid Operations", func(t *testing.T) {
		proc := mgr.New("test", []string{})

		// Signal non-running process
		if err := proc.Signal(os.Kill); err != process.ErrNotRunning {
			t.Errorf("Got error %v, want ErrNotRunning", err)
		}

		// Wait on non-running process
		if err := proc.Wait(); err != process.ErrNotRunning {
			t.Errorf("Got error %v, want ErrNotRunning", err)
		}
	})

	// Test setting limits on running process
	t.Run("Running Process Limits", func(t *testing.T) {
		proc := mgr.New("echo", []string{"test"})
		if err := proc.Start(); err != nil {
			t.Skipf("Could not start test process: %v", err)
		}

		limits := process.ResourceLimits{
			MaxCPUTime:    time.Second,
			MaxMemoryMB:   100,
			MaxFileSizeMB: 10,
			MaxFiles:      100,
			MaxProcesses:  10,
		}

		if err := proc.SetLimits(limits); err == nil {
			t.Error("SetLimits should fail on running process")
		}
	})
}
