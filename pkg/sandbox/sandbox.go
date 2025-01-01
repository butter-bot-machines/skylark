package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const RLIMIT_NPROC = 6 // syscall.RLIMIT_NPROC on Linux

// ResourceLimits defines resource constraints for sandboxed processes
type ResourceLimits struct {
	MaxCPUTime    time.Duration // Maximum CPU time
	MaxMemoryMB   int64         // Maximum memory in MB
	MaxFileSizeMB int64         // Maximum file size in MB
	MaxFiles      int64         // Maximum number of open files
	MaxProcesses  int64         // Maximum number of processes
}

// DefaultLimits provides reasonable default resource limits
var DefaultLimits = ResourceLimits{
	MaxCPUTime:    30 * time.Second,
	MaxMemoryMB:   512,
	MaxFileSizeMB: 10,
	MaxFiles:      100,
	MaxProcesses:  10,
}

// NetworkPolicy defines network access rules
type NetworkPolicy struct {
	AllowOutbound bool     // Allow outbound connections
	AllowInbound  bool     // Allow inbound connections
	AllowedHosts  []string // List of allowed hostnames/IPs
	AllowedPorts  []int    // List of allowed ports
}

// Sandbox represents a sandboxed environment for tool execution
type Sandbox struct {
	WorkDir       string         // Working directory for the sandboxed process
	Limits        ResourceLimits // Resource limits
	Network       NetworkPolicy  // Network access policy
	AllowedPaths  []string      // List of paths accessible to the sandboxed process
	EnvWhitelist  []string      // List of allowed environment variables
	ToolVersion   string        // Version of the tool being executed
	CacheEnabled  bool          // Whether to cache results
	cacheDir      string        // Directory for caching results
}

// NewSandbox creates a new sandbox with the specified configuration
func NewSandbox(workDir string, limits *ResourceLimits, network *NetworkPolicy) (*Sandbox, error) {
	// Use default limits if none provided
	if limits == nil {
		limits = &DefaultLimits
	}

	// Create working directory if it doesn't exist
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create working directory: %w", err)
	}

	// Create cache directory
	cacheDir := filepath.Join(workDir, ".cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &Sandbox{
		WorkDir:  workDir,
		Limits:   *limits,
		Network:  *network,
		cacheDir: cacheDir,
	}, nil
}

// Execute runs a command in the sandbox with the specified limits
func (s *Sandbox) Execute(cmd *exec.Cmd) error {
	// Set working directory
	cmd.Dir = s.WorkDir

	// Set up process group for cleanup
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Set environment variables
	if len(s.EnvWhitelist) > 0 {
		// Always include PATH and basic environment
		basicEnv := []string{"PATH", "HOME", "USER", "SHELL"}
		s.EnvWhitelist = append(s.EnvWhitelist, basicEnv...)

		filteredEnv := make([]string, 0)
		for _, env := range os.Environ() {
			for _, allowed := range s.EnvWhitelist {
				if strings.HasPrefix(env, allowed+"=") {
					filteredEnv = append(filteredEnv, env)
					break
				}
			}
		}
		cmd.Env = filteredEnv
	} else {
		cmd.Env = os.Environ()
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Apply CPU time limit
	if s.Limits.MaxCPUTime > 0 {
		timer := time.AfterFunc(s.Limits.MaxCPUTime, func() {
			syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		})
		defer timer.Stop()
	}

	// Wait for command to complete
	return cmd.Wait()
}

// Cleanup performs cleanup after sandbox execution
func (s *Sandbox) Cleanup() error {
	// Remove temporary files
	pattern := filepath.Join(s.WorkDir, "tmp.*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to list temporary files: %w", err)
	}

	for _, match := range matches {
		if err := os.Remove(match); err != nil {
			return fmt.Errorf("failed to remove temporary file %s: %w", match, err)
		}
	}

	return nil
}

// GetCachedResult attempts to retrieve a cached result
func (s *Sandbox) GetCachedResult(key string) ([]byte, bool) {
	if !s.CacheEnabled {
		return nil, false
	}

	cacheFile := filepath.Join(s.cacheDir, key)
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, false
	}

	// Check if cache is still valid (1 hour)
	info, err := os.Stat(cacheFile)
	if err != nil {
		return nil, false
	}

	if time.Since(info.ModTime()) > time.Hour {
		os.Remove(cacheFile)
		return nil, false
	}

	return data, true
}

// SetCachedResult stores a result in the cache
func (s *Sandbox) SetCachedResult(key string, data []byte) error {
	if !s.CacheEnabled {
		return nil
	}

	cacheFile := filepath.Join(s.cacheDir, key)
	return os.WriteFile(cacheFile, data, 0644)
}

// VerifyToolVersion checks if the tool version is compatible
func (s *Sandbox) VerifyToolVersion(minVersion string) bool {
	if s.ToolVersion == "" || minVersion == "" {
		return true // Skip version check if either version is not specified
	}

	// Parse versions (assuming semantic versioning)
	current := parseVersion(s.ToolVersion)
	minimum := parseVersion(minVersion)

	// Compare versions
	for i := 0; i < 3; i++ {
		if current[i] < minimum[i] {
			return false
		}
		if current[i] > minimum[i] {
			return true
		}
	}

	return true
}

// parseVersion parses a semantic version string into components
func parseVersion(version string) [3]int {
	var components [3]int
	fmt.Sscanf(version, "%d.%d.%d", &components[0], &components[1], &components[2])
	return components
}

// applyResourceLimits applies resource limits to a running process
func (s *Sandbox) applyResourceLimits(pid int) error {
	// Apply memory limit
	if s.Limits.MaxMemoryMB > 0 {
		var rLimit syscall.Rlimit
		rLimit.Max = uint64(s.Limits.MaxMemoryMB * 1024 * 1024)
		rLimit.Cur = rLimit.Max
		if err := syscall.Setrlimit(syscall.RLIMIT_AS, &rLimit); err != nil {
			return fmt.Errorf("failed to set memory limit: %w", err)
		}
	}

	// Apply file size limit
	if s.Limits.MaxFileSizeMB > 0 {
		var rLimit syscall.Rlimit
		rLimit.Max = uint64(s.Limits.MaxFileSizeMB * 1024 * 1024)
		rLimit.Cur = rLimit.Max
		if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &rLimit); err != nil {
			return fmt.Errorf("failed to set file size limit: %w", err)
		}
	}

	// Apply open files limit
	if s.Limits.MaxFiles > 0 {
		var rLimit syscall.Rlimit
		rLimit.Max = uint64(s.Limits.MaxFiles)
		rLimit.Cur = rLimit.Max
		if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
			return fmt.Errorf("failed to set open files limit: %w", err)
		}
	}

	// Apply process limit
	if s.Limits.MaxProcesses > 0 {
		var rLimit syscall.Rlimit
		rLimit.Max = uint64(s.Limits.MaxProcesses)
		rLimit.Cur = rLimit.Max
		if err := syscall.Setrlimit(RLIMIT_NPROC, &rLimit); err != nil {
			return fmt.Errorf("failed to set process limit: %w", err)
		}
	}

	return nil
}
