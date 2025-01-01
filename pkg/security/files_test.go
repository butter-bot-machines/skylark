package security

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/butter-bot-machines/skylark/pkg/config"
)

func TestFileGuard(t *testing.T) {
	// Create temporary test directory structure
	tmpDir := t.TempDir()

	// Create test directories
	allowedDir := filepath.Join(tmpDir, "allowed")
	blockedDir := filepath.Join(tmpDir, "blocked")
	if err := os.MkdirAll(allowedDir, 0755); err != nil {
		t.Fatalf("Failed to create test directories: %v", err)
	}
	if err := os.MkdirAll(blockedDir, 0755); err != nil {
		t.Fatalf("Failed to create test directories: %v", err)
	}

	// Create test files
	allowedFile := filepath.Join(allowedDir, "test.txt")
	if err := os.WriteFile(allowedFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	blockedFile := filepath.Join(blockedDir, "secret.txt")
	if err := os.WriteFile(blockedFile, []byte("secret"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create symlink for testing
	symlinkPath := filepath.Join(allowedDir, "link.txt")
	if err := os.Symlink(allowedFile, symlinkPath); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	// Create test config
	cfg := &config.Config{
		Security: config.SecurityConfig{
			FilePermissions: config.FilePermissionsConfig{
				AllowedPaths:  []string{allowedDir},
				BlockedPaths:  []string{blockedDir},
				AllowSymlinks: false,
				MaxFileSize:   1024,
			},
		},
	}

	// Create audit log for testing
	auditLog, err := NewAuditLog(cfg)
	if err != nil {
		t.Fatalf("Failed to create audit log: %v", err)
	}
	defer auditLog.Close()

	// Create file guard
	guard, err := NewFileGuard(cfg, auditLog)
	if err != nil {
		t.Fatalf("Failed to create file guard: %v", err)
	}

	// Test allowed path access
	t.Run("allowed path", func(t *testing.T) {
		info, err := os.Stat(allowedFile)
		if err != nil {
			t.Fatalf("Failed to stat file: %v", err)
		}

		err = guard.ValidateAccess(allowedFile, info)
		if err != nil {
			t.Errorf("Expected access to be allowed, got error: %v", err)
		}
	})

	// Test blocked path access
	t.Run("blocked path", func(t *testing.T) {
		info, err := os.Stat(blockedFile)
		if err != nil {
			t.Fatalf("Failed to stat file: %v", err)
		}

		err = guard.ValidateAccess(blockedFile, info)
		if err == nil {
			t.Error("Expected access to be denied")
		} else if !errors.Is(err, ErrBlockedPath) {
			t.Errorf("Expected ErrBlockedPath, got: %v", err)
		}
	})

	// Test symlink access
	t.Run("symlink access", func(t *testing.T) {
		info, err := os.Lstat(symlinkPath)
		if err != nil {
			t.Fatalf("Failed to stat symlink: %v", err)
		}

		err = guard.ValidateAccess(symlinkPath, info)
		if err == nil {
			t.Error("Expected symlink access to be denied")
		} else if !errors.Is(err, ErrSymlinkDenied) {
			t.Errorf("Expected ErrSymlinkDenied, got: %v", err)
		}

		// Test with symlinks allowed
		guard.allowSymlinks = true
		err = guard.ValidateAccess(symlinkPath, info)
		if err != nil {
			t.Errorf("Expected symlink access to be allowed, got: %v", err)
		}
	})

	// Test file size limit
	t.Run("file size limit", func(t *testing.T) {
		// Create large file
		largeFile := filepath.Join(allowedDir, "large.txt")
		data := make([]byte, 2048) // Larger than maxFileSize
		if err := os.WriteFile(largeFile, data, 0644); err != nil {
			t.Fatalf("Failed to create large file: %v", err)
		}

		info, err := os.Stat(largeFile)
		if err != nil {
			t.Fatalf("Failed to stat file: %v", err)
		}

		err = guard.ValidateAccess(largeFile, info)
		if err == nil {
			t.Error("Expected access to be denied due to file size")
		} else if !errors.Is(err, ErrFileTooLarge) {
			t.Errorf("Expected ErrFileTooLarge, got: %v", err)
		}
	})

	// Test write validation
	t.Run("write validation", func(t *testing.T) {
		// Test allowed write
		err := guard.ValidateWrite(filepath.Join(allowedDir, "new.txt"), 100)
		if err != nil {
			t.Errorf("Expected write to be allowed, got: %v", err)
		}

		// Test blocked write
		err = guard.ValidateWrite(filepath.Join(blockedDir, "new.txt"), 100)
		if err == nil {
			t.Error("Expected write to be denied")
		} else if !errors.Is(err, ErrBlockedPath) {
			t.Errorf("Expected ErrBlockedPath, got: %v", err)
		}

		// Test write size limit
		err = guard.ValidateWrite(filepath.Join(allowedDir, "large.txt"), 2048)
		if err == nil {
			t.Error("Expected write to be denied due to size")
		} else if !errors.Is(err, ErrFileTooLarge) {
			t.Errorf("Expected ErrFileTooLarge, got: %v", err)
		}
	})

	// Test adding allowed/blocked paths
	t.Run("add paths", func(t *testing.T) {
		newAllowed := filepath.Join(tmpDir, "new-allowed")
		if err := os.MkdirAll(newAllowed, 0755); err != nil {
			t.Fatalf("Failed to create test directory: %v", err)
		}

		// Add allowed path
		err := guard.AddAllowedPath(newAllowed)
		if err != nil {
			t.Errorf("Failed to add allowed path: %v", err)
		}

		// Verify access is allowed
		err = guard.ValidateAccess(filepath.Join(newAllowed, "test.txt"), nil)
		if err != nil {
			t.Errorf("Expected access to be allowed, got: %v", err)
		}

		// Add blocked path
		newBlocked := filepath.Join(tmpDir, "new-blocked")
		err = guard.AddBlockedPath(newBlocked)
		if err != nil {
			t.Errorf("Failed to add blocked path: %v", err)
		}

		// Verify access is denied
		err = guard.ValidateAccess(filepath.Join(newBlocked, "test.txt"), nil)
		if err == nil {
			t.Error("Expected access to be denied")
		} else if !errors.Is(err, ErrBlockedPath) {
			t.Errorf("Expected ErrBlockedPath, got: %v", err)
		}
	})

	// Test path traversal attempts
	t.Run("path traversal", func(t *testing.T) {
		traversalPaths := []string{
			filepath.Join(allowedDir, "..", "blocked", "secret.txt"),
			filepath.Join(allowedDir, "subdir", "..", "..", "blocked", "secret.txt"),
			filepath.Join(allowedDir, ".."+string(filepath.Separator)+"blocked"),
		}

		for _, path := range traversalPaths {
			err := guard.ValidateAccess(path, nil)
			if err == nil {
				t.Errorf("Expected path traversal to be denied for: %s", path)
			}
		}
	})
}

func TestFileGuardErrors(t *testing.T) {
	// Test invalid config paths
	t.Run("invalid config paths", func(t *testing.T) {
		cfg := &config.Config{
			Security: config.SecurityConfig{
				FilePermissions: config.FilePermissionsConfig{
					AllowedPaths: []string{string([]byte{0x7f})}, // Invalid path character (DEL)
				},
			},
		}

		_, err := NewFileGuard(cfg, nil)
		if err == nil {
			t.Error("Expected error for invalid allowed path")
		}

		cfg.Security.FilePermissions.AllowedPaths = nil
		cfg.Security.FilePermissions.BlockedPaths = []string{string([]byte{0x7f})}
		_, err = NewFileGuard(cfg, nil)
		if err == nil {
			t.Error("Expected error for invalid blocked path")
		}
	})

	// Test non-existent paths
	t.Run("non-existent paths", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.Config{
			Security: config.SecurityConfig{
				FilePermissions: config.FilePermissionsConfig{
					AllowedPaths: []string{tmpDir},
					MaxFileSize:  1024,
				},
			},
		}

		guard, err := NewFileGuard(cfg, nil)
		if err != nil {
			t.Fatalf("Failed to create file guard: %v", err)
		}

		nonExistentPath := filepath.Join(tmpDir, "nonexistent.txt")
		err = guard.ValidateAccess(nonExistentPath, nil)
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
	})
}
