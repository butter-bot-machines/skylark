package os

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/butter-bot-machines/skylark/pkg/process"
)

// cgroupsController manages cgroup operations
type cgroupsController struct {
	// Base path for cgroups filesystem
	basePath string
	// Version of cgroups (v1 or v2)
	version int
}

// newCgroupsController creates a new cgroups controller
func newCgroupsController() (*cgroupsController, error) {
	// Check for cgroups v2
	if _, err := os.Stat("/sys/fs/cgroup/cgroup.controllers"); err == nil {
		return &cgroupsController{
			basePath: "/sys/fs/cgroup",
			version:  2,
		}, nil
	}

	// Check for cgroups v1
	if _, err := os.Stat("/sys/fs/cgroup/memory"); err == nil {
		return &cgroupsController{
			basePath: "/sys/fs/cgroup/memory",
			version:  1,
		}, nil
	}

	return nil, fmt.Errorf("cgroups not available")
}

// setupMemoryLimit creates a cgroup and sets memory limit
func (c *cgroupsController) setupMemoryLimit(pid int, limitMB int64) error {
	cgroupPath := filepath.Join(c.basePath, fmt.Sprintf("skylark-%d", pid))

	// Create cgroup directory
	if err := os.MkdirAll(cgroupPath, 0755); err != nil {
		return fmt.Errorf("failed to create cgroup: %w", err)
	}

	// Convert MB to bytes
	limitBytes := limitMB * 1024 * 1024

	// Set memory limit based on cgroups version
	if c.version == 2 {
		// cgroups v2
		if err := os.WriteFile(
			filepath.Join(cgroupPath, "memory.max"),
			[]byte(strconv.FormatInt(limitBytes, 10)),
			0644,
		); err != nil {
			return fmt.Errorf("failed to set memory limit: %w", err)
		}
	} else {
		// cgroups v1
		if err := os.WriteFile(
			filepath.Join(cgroupPath, "memory.limit_in_bytes"),
			[]byte(strconv.FormatInt(limitBytes, 10)),
			0644,
		); err != nil {
			return fmt.Errorf("failed to set memory limit: %w", err)
		}

		// Also set memsw limit (memory+swap) to same value
		if err := os.WriteFile(
			filepath.Join(cgroupPath, "memory.memsw.limit_in_bytes"),
			[]byte(strconv.FormatInt(limitBytes, 10)),
			0644,
		); err != nil {
			// Some systems don't support swap limit, ignore error
			_ = err
		}
	}

	// Add process to cgroup
	if err := os.WriteFile(
		filepath.Join(cgroupPath, "cgroup.procs"),
		[]byte(strconv.Itoa(pid)),
		0644,
	); err != nil {
		return fmt.Errorf("failed to add process to cgroup: %w", err)
	}

	return nil
}

// getCurrentUsage gets current memory usage in MB
func (c *cgroupsController) getCurrentUsage(pid int) (int64, error) {
	cgroupPath := filepath.Join(c.basePath, fmt.Sprintf("skylark-%d", pid))

	var usageFile string
	if c.version == 2 {
		usageFile = "memory.current"
	} else {
		usageFile = "memory.usage_in_bytes"
	}

	data, err := os.ReadFile(filepath.Join(cgroupPath, usageFile))
	if err != nil {
		return 0, fmt.Errorf("failed to read memory usage: %w", err)
	}

	bytes, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse memory usage: %w", err)
	}

	return bytes / (1024 * 1024), nil // Convert to MB
}

// cleanup removes the cgroup
func (c *cgroupsController) cleanup(pid int) error {
	cgroupPath := filepath.Join(c.basePath, fmt.Sprintf("skylark-%d", pid))

	// Move processes to parent cgroup
	if err := os.WriteFile(
		filepath.Join(cgroupPath, "cgroup.procs"),
		[]byte("0"),
		0644,
	); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to move process to parent cgroup: %w", err)
		}
	}

	// Remove cgroup directory
	if err := os.RemoveAll(cgroupPath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove cgroup: %w", err)
		}
	}

	return nil
}

// applyMemoryLimit applies memory limit to a process
func applyMemoryLimit(p *Process) error {
	if p.limits.MaxMemoryMB <= 0 {
		return nil // No memory limit set
	}

	if p.cmd.Process == nil {
		return process.Error{"cannot apply memory limit on non-running process"}
	}

	controller, err := newCgroupsController()
	if err != nil {
		return fmt.Errorf("cgroups not available: %w", err)
	}

	if err := controller.setupMemoryLimit(p.cmd.Process.Pid, p.limits.MaxMemoryMB); err != nil {
		return fmt.Errorf("failed to set memory limit: %w", err)
	}

	return nil
}

// cleanupMemoryLimit cleans up cgroup resources
func cleanupMemoryLimit(p *Process) error {
	if p.cmd.Process == nil {
		return nil // Nothing to clean up
	}

	controller, err := newCgroupsController()
	if err != nil {
		return nil // Cgroups not available, nothing to clean up
	}

	return controller.cleanup(p.cmd.Process.Pid)
}
