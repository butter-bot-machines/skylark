package resources

import "runtime"

// Controller defines the interface for resource management
type Controller interface {
	// Memory management
	SetMemoryLimit(bytes int64) error
	GetMemoryUsage() int64
	ForceGC()

	// CPU management
	SetCPULimit(cores int) error
	GetCPUUsage() float64
	LockThread() error
	UnlockThread()

	// Profile management
	StartProfiling() error
	StopProfiling() error
}

// Limits defines resource limits for a process
type Limits struct {
	MaxMemory   int64   // Maximum memory in bytes
	MaxCPU      float64 // Maximum CPU cores (1.0 = one core)
	MaxThreads  int     // Maximum number of OS threads
	ProfileRate int     // Memory profiling rate (1 = profile all allocations)
}

// Error types for resource operations
var (
	ErrInvalidLimit     = Error{"invalid resource limit"}
	ErrLimitExceeded    = Error{"resource limit exceeded"}
	ErrThreadLocked     = Error{"thread already locked"}
	ErrThreadNotLocked  = Error{"thread not locked"}
	ErrProfileActive    = Error{"profiling already active"}
	ErrProfileInactive  = Error{"profiling not active"}
	ErrUnsupported      = Error{"operation not supported"}
)

// Error represents a resource error
type Error struct {
	Message string
}

func (e Error) Error() string {
	return e.Message
}

// DefaultLimits returns the default resource limits
func DefaultLimits() Limits {
	return Limits{
		MaxMemory:   1 << 30,         // 1GB
		MaxCPU:      1.0,             // 1 core
		MaxThreads:  runtime.NumCPU(), // One thread per CPU
		ProfileRate: 0,               // No profiling
	}
}
