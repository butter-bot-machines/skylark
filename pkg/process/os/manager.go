package os

import (
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"

	"github.com/butter-bot-machines/skylark/pkg/process"
	"github.com/butter-bot-machines/skylark/pkg/timing"
)

// Manager implements process.Manager using real OS processes
type Manager struct {
	mu           sync.RWMutex
	processes    map[int]*Process
	defaultLimit process.ResourceLimits
	clock        timing.Clock
}

// NewManager creates a new OS process manager
func NewManager(clock timing.Clock) *Manager {
	if clock == nil {
		clock = timing.New()
	}
	return &Manager{
		processes: make(map[int]*Process),
		clock:     clock,
	}
}

// New creates a new OS process
func (m *Manager) New(name string, args []string) process.Process {
	cmd := exec.Command(name, args...)
	return &Process{
		cmd:     cmd,
		limits:  m.defaultLimit,
		manager: m,
		clock:   m.clock,
	}
}

// Get retrieves a process by ID
func (m *Manager) Get(pid int) (process.Process, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if proc, ok := m.processes[pid]; ok {
		return proc, nil
	}
	return nil, process.ErrNotFound
}

// List returns all processes
func (m *Manager) List() []process.Process {
	m.mu.RLock()
	defer m.mu.RUnlock()

	procs := make([]process.Process, 0, len(m.processes))
	for _, p := range m.processes {
		procs = append(procs, p)
	}
	return procs
}

// SetDefaultLimits sets default resource limits
func (m *Manager) SetDefaultLimits(limits process.ResourceLimits) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultLimit = limits
}

// GetDefaultLimits returns default resource limits
func (m *Manager) GetDefaultLimits() process.ResourceLimits {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.defaultLimit
}

// Process implements process.Process using a real OS process
type Process struct {
	mu      sync.RWMutex
	cmd     *exec.Cmd
	limits  process.ResourceLimits
	manager *Manager
	clock   timing.Clock
	cancel  context.CancelFunc // For CPU time limit
}

// Start starts the process
func (p *Process) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd.Process != nil {
		return process.ErrAlreadyExists
	}

	// Validate resource limits
	if err := validateLimits(p.limits); err != nil {
		return err
	}

	// Apply basic resource limits
	if err := p.applyLimits(); err != nil {
		return err
	}

	// Set up CPU time limit
	if p.limits.MaxCPUTime > 0 {
		ctx := context.Background()
		ctx, p.cancel = context.WithCancel(ctx)
		timer := p.clock.AfterFunc(p.limits.MaxCPUTime, func() {
			p.mu.Lock()
			defer p.mu.Unlock()
			if p.cmd.Process != nil {
				p.cmd.Process.Kill()
			}
		})
		defer func() {
			if err := recover(); err != nil {
				timer.Stop()
				p.cancel()
			}
		}()
	}

	// Start the process
	if err := p.cmd.Start(); err != nil {
		return err
	}

	// Apply memory limit (requires running process)
	if err := applyMemoryLimit(p); err != nil {
		// Kill process if memory limit fails
		p.cmd.Process.Kill()
		return err
	}

	// Register process with manager after successful start
	p.manager.mu.Lock()
	p.manager.processes[p.cmd.Process.Pid] = p
	p.manager.mu.Unlock()

	return nil
}

// Wait waits for the process to complete
func (p *Process) Wait() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd.Process == nil {
		return process.ErrNotRunning
	}

	err := p.cmd.Wait()

	// Clean up CPU time limit
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}

	// Clean up memory limit
	if cleanupErr := cleanupMemoryLimit(p); cleanupErr != nil {
		// Log cleanup error but don't override process error
		_ = cleanupErr
	}

	// Remove process from manager after completion
	p.manager.mu.Lock()
	delete(p.manager.processes, p.cmd.Process.Pid)
	p.manager.mu.Unlock()

	return err
}

// Signal sends a signal to the process
func (p *Process) Signal(sig os.Signal) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd.Process == nil {
		return process.ErrNotRunning
	}

	return p.cmd.Process.Signal(sig)
}

// SetStdin sets the process stdin
func (p *Process) SetStdin(r io.Reader) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cmd.Stdin = r
}

// SetStdout sets the process stdout
func (p *Process) SetStdout(w io.Writer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cmd.Stdout = w
}

// SetStderr sets the process stderr
func (p *Process) SetStderr(w io.Writer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cmd.Stderr = w
}

// SetLimits sets resource limits
func (p *Process) SetLimits(limits process.ResourceLimits) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd.Process != nil {
		return process.Error{"cannot set limits on running process"}
	}

	if err := validateLimits(limits); err != nil {
		return err
	}
	p.limits = limits
	return nil
}

// GetLimits returns current resource limits
func (p *Process) GetLimits() process.ResourceLimits {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.limits
}

// ID returns the process ID
func (p *Process) ID() int {
	if p.cmd.Process != nil {
		return p.cmd.Process.Pid
	}
	return 0
}

// Running returns whether the process is running
func (p *Process) Running() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.cmd.Process != nil && p.cmd.ProcessState == nil
}

// ExitCode returns the process exit code
func (p *Process) ExitCode() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.cmd.ProcessState == nil {
		return 0
	}
	return p.cmd.ProcessState.ExitCode()
}

// validateLimits checks if resource limits are valid
func validateLimits(limits process.ResourceLimits) error {
	if limits.MaxCPUTime < 0 ||
		limits.MaxMemoryMB < 0 ||
		limits.MaxFileSizeMB < 0 ||
		limits.MaxFiles < 0 ||
		limits.MaxProcesses < 0 {
		return process.ErrInvalidLimits
	}
	return nil
}

// applyLimits applies resource limits to the process
func (p *Process) applyLimits() error {
	if p.cmd.Process != nil {
		return process.Error{"cannot apply limits on running process"}
	}

	// Set up process group for proper cleanup
	p.cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	var unsupported []string

	// Apply file descriptor limit
	if p.limits.MaxFiles > 0 {
		rlimit := &syscall.Rlimit{
			Cur: uint64(p.limits.MaxFiles),
			Max: uint64(p.limits.MaxFiles),
		}
		if err := syscall.Setrlimit(rlimitNOFILE, rlimit); err != nil {
			unsupported = append(unsupported, "max files")
		}
	}

	// Apply process limit
	if p.limits.MaxProcesses > 0 {
		rlimit := &syscall.Rlimit{
			Cur: uint64(p.limits.MaxProcesses),
			Max: uint64(p.limits.MaxProcesses),
		}
		if err := syscall.Setrlimit(rlimitNPROC, rlimit); err != nil {
			unsupported = append(unsupported, "max processes")
		}
	}

	// Apply file size limit
	if p.limits.MaxFileSizeMB > 0 {
		rlimit := &syscall.Rlimit{
			Cur: uint64(p.limits.MaxFileSizeMB * 1024 * 1024),
			Max: uint64(p.limits.MaxFileSizeMB * 1024 * 1024),
		}
		if err := syscall.Setrlimit(rlimitFSIZE, rlimit); err != nil {
			unsupported = append(unsupported, "max file size")
		}
	}

	// Handle non-memory limits first
	if len(unsupported) > 0 {
		return process.Error{Message: "unsupported limits: " + strings.Join(unsupported, ", ")}
	}

	// Memory limit is handled separately by cgroups/platform-specific code

	return nil
}
