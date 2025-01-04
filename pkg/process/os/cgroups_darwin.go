package os

import "github.com/butter-bot-machines/skylark/pkg/process"

// applyMemoryLimit is a no-op on Darwin
func applyMemoryLimit(p *Process) error {
	// Memory limits not supported on Darwin
	if p.limits.MaxMemoryMB > 0 {
		return process.Error{Message: "memory limits not supported on Darwin"}
	}
	return nil
}

// cleanupMemoryLimit is a no-op on Darwin
func cleanupMemoryLimit(p *Process) error {
	return nil
}
