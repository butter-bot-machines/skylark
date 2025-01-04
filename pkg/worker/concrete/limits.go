package concrete

import (
	"time"

	"github.com/butter-bot-machines/skylark/pkg/process"
)

// DefaultLimits returns default resource limits
func DefaultLimits() process.ResourceLimits {
	return process.ResourceLimits{
		MaxCPUTime:    30 * time.Second,
		MaxMemoryMB:   512,                // 512MB
		MaxFileSizeMB: 50,                 // 50MB
		MaxFiles:      100,
		MaxProcesses:  10,
	}
}
