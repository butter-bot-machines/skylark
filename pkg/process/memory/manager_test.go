package memory

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

		if proc.ID() != 1 {
			t.Errorf("Got ID %d, want 1", proc.ID())
		}
	})

	// Test process retrieval
	t.Run("Process Retrieval", func(t *testing.T) {
		proc, err := mgr.Get(1)
		if err != nil {
			t.Errorf("Get failed: %v", err)
		}
		if proc == nil {
			t.Error("Get returned nil process")
		}

		// Test non-existent process
		if _, err := mgr.Get(999); err != process.ErrNotFound {
			t.Errorf("Got error %v, want ErrNotFound", err)
		}
	})

	// Test process listing
	t.Run("Process Listing", func(t *testing.T) {
		procs := mgr.List()
		if len(procs) != 1 {
			t.Errorf("Got %d processes, want 1", len(procs))
		}
	})
}

func TestProcess_Lifecycle(t *testing.T) {
	mock := timing.NewMock()
	mgr := NewManager(mock)
	proc := mgr.New("test", []string{"-arg"})

	// Test initial state
	t.Run("Initial State", func(t *testing.T) {
		if proc.Running() {
			t.Error("New process should not be running")
		}
		if code := proc.ExitCode(); code != 0 {
			t.Errorf("Got exit code %d, want 0", code)
		}
	})

	// Test start
	t.Run("Start", func(t *testing.T) {
		if err := proc.Start(); err != nil {
			t.Errorf("Start failed: %v", err)
		}
		if !proc.Running() {
			t.Error("Process should be running after Start")
		}

		// Test double start
		if err := proc.Start(); err != process.ErrAlreadyExists {
			t.Errorf("Got error %v, want ErrAlreadyExists", err)
		}
	})

	// Test wait
	t.Run("Wait", func(t *testing.T) {
		if err := proc.Wait(); err != nil {
			t.Errorf("Wait failed: %v", err)
		}
		if proc.Running() {
			t.Error("Process should not be running after Wait")
		}

		// Test wait on stopped process
		if err := proc.Wait(); err != process.ErrNotRunning {
			t.Errorf("Got error %v, want ErrNotRunning", err)
		}
	})

	// Test signal
	t.Run("Signal", func(t *testing.T) {
		proc := mgr.New("test", []string{"-arg"})
		proc.Start()

		if err := proc.Signal(os.Kill); err != nil {
			t.Errorf("Signal failed: %v", err)
		}
		if proc.Running() {
			t.Error("Process should not be running after Kill signal")
		}
		if code := proc.ExitCode(); code != -1 {
			t.Errorf("Got exit code %d, want -1", code)
		}
	})
}

func TestProcess_IO(t *testing.T) {
	mock := timing.NewMock()
	mgr := NewManager(mock)
	proc := mgr.New("test", []string{"-arg"})

	// Test IO redirection
	t.Run("IO Redirection", func(t *testing.T) {
		stdin := bytes.NewBufferString("input")
		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)

		proc.SetStdin(stdin)
		proc.SetStdout(stdout)
		proc.SetStderr(stderr)

		// Start and wait to simulate process execution
		proc.Start()
		mock.Add(10 * time.Millisecond)
		proc.Wait()
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

	// Test setting limits on running process
	t.Run("Running Process Limits", func(t *testing.T) {
		proc := mgr.New("test", []string{})
		proc.Start()

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

func TestProcess_CPULimit(t *testing.T) {
	mock := timing.NewMock()
	mgr := NewManager(mock)

	// Test CPU time limit exceeded
	t.Run("CPU Time Limit Exceeded", func(t *testing.T) {
		proc := mgr.New("test", []string{})
		limits := process.ResourceLimits{
			MaxCPUTime: 100 * time.Millisecond,
		}
		if err := proc.SetLimits(limits); err != nil {
			t.Fatalf("SetLimits failed: %v", err)
		}

		if err := proc.Start(); err != nil {
			t.Fatalf("Start failed: %v", err)
		}

		// Advance time past CPU limit
		mock.Add(200 * time.Millisecond)

		// Process should be killed
		err := proc.Wait()
		if err == nil {
			t.Error("Wait should return error for CPU limit exceeded")
		}
		if err.Error() != "process killed: CPU time limit exceeded" {
			t.Errorf("Got error %v, want 'process killed: CPU time limit exceeded'", err)
		}
		if proc.Running() {
			t.Error("Process should not be running after CPU limit exceeded")
		}
		if code := proc.ExitCode(); code != -1 {
			t.Errorf("Got exit code %d, want -1", code)
		}
	})

	// Test process completes within CPU limit
	t.Run("Within CPU Time Limit", func(t *testing.T) {
		proc := mgr.New("test", []string{})
		limits := process.ResourceLimits{
			MaxCPUTime: 100 * time.Millisecond,
		}
		if err := proc.SetLimits(limits); err != nil {
			t.Fatalf("SetLimits failed: %v", err)
		}

		if err := proc.Start(); err != nil {
			t.Fatalf("Start failed: %v", err)
		}

		// Advance time within CPU limit
		mock.Add(50 * time.Millisecond)

		// Process should complete normally
		if err := proc.Wait(); err != nil {
			t.Errorf("Wait failed: %v", err)
		}
		if proc.Running() {
			t.Error("Process should not be running after completion")
		}
		if code := proc.ExitCode(); code != 0 {
			t.Errorf("Got exit code %d, want 0", code)
		}
	})
}

func TestManager_Concurrency(t *testing.T) {
	mock := timing.NewMock()
	mgr := NewManager(mock)
	var wg sync.WaitGroup
	workers := 10
	iterations := 100

	// Test concurrent process creation and management
	t.Run("Concurrent Operations", func(t *testing.T) {
		wg.Add(workers)
		for i := 0; i < workers; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					// Create and manage process
					proc := mgr.New("test", []string{"-arg"})
					if err := proc.Start(); err != nil {
						t.Errorf("Start failed: %v", err)
					}
					mock.Add(10 * time.Millisecond)
					if err := proc.Wait(); err != nil {
						t.Errorf("Wait failed: %v", err)
					}

					// Get process by ID
					if _, err := mgr.Get(proc.ID()); err != nil {
						t.Errorf("Get failed: %v", err)
					}

					// List processes
					if procs := mgr.List(); len(procs) < 1 {
						t.Error("List returned empty result")
					}
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

		// Start process twice
		proc.Start()
		if err := proc.Start(); err != process.ErrAlreadyExists {
			t.Errorf("Got error %v, want ErrAlreadyExists", err)
		}
	})
}
