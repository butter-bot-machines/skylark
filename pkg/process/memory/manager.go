package memory

import (
	"context"
	"io"
	"os"
	"sync"

	"github.com/butter-bot-machines/skylark/pkg/process"
	"github.com/butter-bot-machines/skylark/pkg/timing"
)

// Manager implements process.Manager for testing
type Manager struct {
	mu           sync.RWMutex
	processes    map[int]*Process
	nextPID      int
	defaultLimit process.ResourceLimits
	clock        timing.Clock
}

// NewManager creates a new memory process manager
func NewManager(clock timing.Clock) *Manager {
	if clock == nil {
		clock = timing.New()
	}
	return &Manager{
		processes: make(map[int]*Process),
		nextPID:   1,
		clock:     clock,
	}
}

// New creates a new memory process
func (m *Manager) New(name string, args []string) process.Process {
	m.mu.Lock()
	defer m.mu.Unlock()

	proc := &Process{
		id:      m.nextPID,
		name:    name,
		args:    args,
		limits:  m.defaultLimit,
		manager: m,
		clock:   m.clock,
	}
	m.processes[proc.id] = proc
	m.nextPID++

	return proc
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

// Process implements process.Process for testing
type Process struct {
	mu       sync.RWMutex
	manager  *Manager
	id       int
	name     string
	args     []string
	running  bool
	exitCode int
	limits   process.ResourceLimits
	stdin    io.Reader
	stdout   io.Writer
	stderr   io.Writer
	clock    timing.Clock
	cancel   context.CancelFunc // For CPU time limit
	timer    timing.Timer       // For CPU time limit
}

// Start marks the process as running
func (p *Process) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return process.ErrAlreadyExists
	}

	// Validate resource limits
	if err := validateLimits(p.limits); err != nil {
		return err
	}

	p.running = true

	// Set up CPU time limit
	if p.limits.MaxCPUTime > 0 {
		ctx := context.Background()
		ctx, p.cancel = context.WithCancel(ctx)
		p.timer = p.clock.AfterFunc(p.limits.MaxCPUTime, func() {
			p.mu.Lock()
			defer p.mu.Unlock()
			if p.running {
				p.running = false
				p.exitCode = -1 // Killed by CPU limit
			}
		})
	}

	return nil
}

// Wait simulates process completion
func (p *Process) Wait() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running && p.exitCode != -1 {
		return process.ErrNotRunning
	}

	// Clean up CPU time limit
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}
	if p.timer != nil {
		p.timer.Stop()
		p.timer = nil
	}

	// Return error if killed by CPU limit
	if p.exitCode == -1 {
		p.running = false
		return process.Error{"process killed: CPU time limit exceeded"}
	}

	p.running = false
	p.exitCode = 0
	return nil
}

// Signal handles process signals
func (p *Process) Signal(sig os.Signal) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return process.ErrNotRunning
	}

	// Simulate process termination on kill signals
	if sig == os.Kill {
		p.running = false
		p.exitCode = -1
	}

	return nil
}

// SetStdin sets the process stdin
func (p *Process) SetStdin(r io.Reader) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stdin = r
}

// SetStdout sets the process stdout
func (p *Process) SetStdout(w io.Writer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stdout = w
}

// SetStderr sets the process stderr
func (p *Process) SetStderr(w io.Writer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stderr = w
}

// SetLimits sets resource limits
func (p *Process) SetLimits(limits process.ResourceLimits) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
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
	return p.id
}

// Running returns whether the process is running
func (p *Process) Running() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// ExitCode returns the process exit code
func (p *Process) ExitCode() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.exitCode
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
