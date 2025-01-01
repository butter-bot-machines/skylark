package security

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/butter-bot-machines/skylark/pkg/config"
)

var (
	ErrAccessDenied     = errors.New("access denied")
	ErrInvalidPath      = errors.New("invalid path")
	ErrFileTooLarge     = errors.New("file too large")
	ErrSymlinkDenied    = errors.New("symlink traversal not allowed")
	ErrBlockedPath      = errors.New("path is blocked")
)

// FileGuard manages file access controls
type FileGuard struct {
	mu            sync.RWMutex
	config        config.FilePermissionsConfig
	auditLog      *AuditLog
	allowedPaths  []string // Normalized absolute paths
	blockedPaths  []string // Normalized absolute paths
	maxFileSize   int64
	allowSymlinks bool
}

// NewFileGuard creates a new file access controller
func NewFileGuard(cfg *config.Config, auditLog *AuditLog) (*FileGuard, error) {
	guard := &FileGuard{
		auditLog:      auditLog,
		maxFileSize:   cfg.Security.FilePermissions.MaxFileSize,
		allowSymlinks: cfg.Security.FilePermissions.AllowSymlinks,
	}

	// Normalize and validate allowed paths
	for _, path := range cfg.Security.FilePermissions.AllowedPaths {
		// Check for invalid characters
		if strings.ContainsAny(path, "\x00\x7f") {
			return nil, fmt.Errorf("%w: allowed path contains invalid characters", ErrInvalidPath)
		}
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("invalid allowed path %s: %w", path, err)
		}
		guard.allowedPaths = append(guard.allowedPaths, filepath.Clean(absPath))
	}

	// Normalize and validate blocked paths
	for _, path := range cfg.Security.FilePermissions.BlockedPaths {
		// Check for invalid characters
		if strings.ContainsAny(path, "\x00\x7f") {
			return nil, fmt.Errorf("%w: blocked path contains invalid characters", ErrInvalidPath)
		}
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("invalid blocked path %s: %w", path, err)
		}
		guard.blockedPaths = append(guard.blockedPaths, filepath.Clean(absPath))
	}

	return guard, nil
}

// ValidateAccess checks if a file operation is allowed
func (g *FileGuard) ValidateAccess(path string, info os.FileInfo) error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Get absolute, clean path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidPath, err)
	}
	cleanPath := filepath.Clean(absPath)

	// Check if path is blocked
	for _, blocked := range g.blockedPaths {
		if isSubPath(cleanPath, blocked) {
			g.logAccessDenied(cleanPath, "path is blocked")
			return fmt.Errorf("%w: path is blocked", ErrBlockedPath)
		}
	}

	// Check if path is allowed
	allowed := false
	for _, allowedPath := range g.allowedPaths {
		if isSubPath(cleanPath, allowedPath) {
			allowed = true
			break
		}
	}
	if !allowed {
		g.logAccessDenied(cleanPath, "path not in allowed list")
		return fmt.Errorf("%w: path not in allowed list", ErrAccessDenied)
	}

	// Check symlinks
	if !g.allowSymlinks {
		isLink, err := isSymlink(cleanPath, info)
		if err != nil {
			return fmt.Errorf("failed to check symlink: %w", err)
		}
		if isLink {
			g.logAccessDenied(cleanPath, "symlinks not allowed")
			return fmt.Errorf("%w: symlinks not allowed", ErrSymlinkDenied)
		}
	}

	// Check file size
	if info != nil && !info.IsDir() && info.Size() > g.maxFileSize {
		g.logAccessDenied(cleanPath, fmt.Sprintf("file size %d exceeds limit %d", info.Size(), g.maxFileSize))
		return fmt.Errorf("%w: file size %d exceeds limit %d", ErrFileTooLarge, info.Size(), g.maxFileSize)
	}

	return nil
}

// ValidateWrite checks if a write operation is allowed
func (g *FileGuard) ValidateWrite(path string, size int64) error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Get absolute, clean path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidPath, err)
	}
	cleanPath := filepath.Clean(absPath)

	// Check if path is blocked
	for _, blocked := range g.blockedPaths {
		if isSubPath(cleanPath, blocked) {
			g.logAccessDenied(cleanPath, "path is blocked")
			return fmt.Errorf("%w: path is blocked", ErrBlockedPath)
		}
	}

	// Check if path is allowed
	allowed := false
	for _, allowedPath := range g.allowedPaths {
		if isSubPath(cleanPath, allowedPath) {
			allowed = true
			break
		}
	}
	if !allowed {
		g.logAccessDenied(cleanPath, "path not in allowed list")
		return fmt.Errorf("%w: path not in allowed list", ErrAccessDenied)
	}

	// Check file size
	if size > g.maxFileSize {
		g.logAccessDenied(cleanPath, fmt.Sprintf("write size %d exceeds limit %d", size, g.maxFileSize))
		return fmt.Errorf("%w: write size %d exceeds limit %d", ErrFileTooLarge, size, g.maxFileSize)
	}

	return nil
}

// AddAllowedPath adds a path to the allowed list
func (g *FileGuard) AddAllowedPath(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	cleanPath := filepath.Clean(absPath)

	g.mu.Lock()
	defer g.mu.Unlock()

	// Check if path is already blocked
	for _, blocked := range g.blockedPaths {
		if isSubPath(cleanPath, blocked) {
			return fmt.Errorf("%w: path is blocked", ErrBlockedPath)
		}
	}

	g.allowedPaths = append(g.allowedPaths, cleanPath)
	return nil
}

// AddBlockedPath adds a path to the blocked list
func (g *FileGuard) AddBlockedPath(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	cleanPath := filepath.Clean(absPath)

	g.mu.Lock()
	defer g.mu.Unlock()

	g.blockedPaths = append(g.blockedPaths, cleanPath)
	return nil
}

// isSubPath checks if child path is under parent path
func isSubPath(child, parent string) bool {
	childParts := strings.Split(filepath.Clean(child), string(filepath.Separator))
	parentParts := strings.Split(filepath.Clean(parent), string(filepath.Separator))

	if len(childParts) < len(parentParts) {
		return false
	}

	for i := range parentParts {
		if childParts[i] != parentParts[i] {
			return false
		}
	}

	return true
}

// isSymlink checks if a path is a symbolic link
func isSymlink(path string, info os.FileInfo) (bool, error) {
	if info == nil {
		var err error
		info, err = os.Lstat(path)
		if err != nil {
			return false, err
		}
	}
	return info.Mode()&os.ModeSymlink != 0, nil
}

// logAccessDenied logs an access denied event
func (g *FileGuard) logAccessDenied(path, reason string) {
	if g.auditLog == nil {
		return
	}

	g.auditLog.Log(
		EventAccessDenied,
		SeverityWarning,
		"file_guard",
		fmt.Sprintf("Access denied to %s: %s", path, reason),
		map[string]interface{}{
			"path":   path,
			"reason": reason,
		},
	)
}
