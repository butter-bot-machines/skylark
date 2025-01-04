package process

import (
	"io"
	"os"
	"time"
)

// Process represents a running or finished process
type Process interface {
	// Basic operations
	Start() error
	Wait() error
	Signal(sig os.Signal) error

	// IO operations
	SetStdin(io.Reader)
	SetStdout(io.Writer)
	SetStderr(io.Writer)

	// Resource operations
	SetLimits(ResourceLimits) error
	GetLimits() ResourceLimits

	// State
	ID() int
	Running() bool
	ExitCode() int
}

// Manager handles process creation and management
type Manager interface {
	// Process creation
	New(name string, args []string) Process
	Get(pid int) (Process, error)
	List() []Process

	// Resource management
	SetDefaultLimits(ResourceLimits)
	GetDefaultLimits() ResourceLimits
}

// ResourceLimits defines resource constraints for processes
type ResourceLimits struct {
	MaxCPUTime    time.Duration
	MaxMemoryMB   int64
	MaxFileSizeMB int64
	MaxFiles      int64
	MaxProcesses  int64
}

// Error types for process operations
var (
	ErrNotFound      = Error{"process not found"}
	ErrAlreadyExists = Error{"process already exists"}
	ErrNotRunning    = Error{"process not running"}
	ErrInvalidLimits = Error{"invalid resource limits"}
)

// Error represents a process error
type Error struct {
	Message string
}

func (e Error) Error() string {
	return e.Message
}
