package concrete

import (
	"time"

	"github.com/butter-bot-machines/skylark/pkg/timing"
)

// ResourceLimits defines resource constraints for workers
type ResourceLimits struct {
	MaxCPUTime    time.Duration
	MaxMemory     int64
	MaxFileSize   int64
	MaxFiles      int
	MaxProcesses  int
	clock         timing.Clock
}

// WithClock returns a copy of limits with a custom clock
func (l ResourceLimits) WithClock(clock timing.Clock) ResourceLimits {
	l.clock = clock
	return l
}

// DefaultLimits returns default resource limits
func DefaultLimits() ResourceLimits {
	return ResourceLimits{
		MaxCPUTime:    30 * time.Second,
		MaxMemory:     512 * 1024 * 1024, // 512MB
		MaxFileSize:   50 * 1024 * 1024,  // 50MB
		MaxFiles:      100,
		MaxProcesses:  10,
		clock:         timing.New(),
	}
}
