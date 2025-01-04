package security

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/butter-bot-machines/skylark/pkg/config"
	"github.com/butter-bot-machines/skylark/pkg/security/types"
)

func TestKeyStore(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "keys.dat")

	// Generate test encryption key
	encKey := make([]byte, 32)
	for i := range encKey {
		encKey[i] = byte(i)
	}
	encodedKey := base64.StdEncoding.EncodeToString(encKey)

	// Create test config
	cfg := &config.Config{
		Security: types.SecurityConfig{
			EncryptionKey:  encodedKey,
			KeyStoragePath: keyPath,
		},
	}

	// Create key store
	store, err := NewKeyStore(cfg)
	if err != nil {
		t.Fatalf("Failed to create key store: %v", err)
	}

	// Test adding a key
	t.Run("add key", func(t *testing.T) {
		err := store.AddKey("test-key", "openai", "sk-test123", AccessRead|AccessWrite, nil)
		if err != nil {
			t.Errorf("Failed to add key: %v", err)
		}

		// Verify key was added
		value, err := store.GetKey("test-key", AccessRead)
		if err != nil {
			t.Errorf("Failed to get key: %v", err)
		}
		if value != "sk-test123" {
			t.Errorf("Got wrong key value: %s", value)
		}
	})

	// Test key expiration
	t.Run("key expiration", func(t *testing.T) {
		expiry := time.Now().Add(-time.Hour) // Expired 1 hour ago
		err := store.AddKey("expired-key", "openai", "sk-expired", AccessRead, &expiry)
		if err != nil {
			t.Errorf("Failed to add expired key: %v", err)
		}

		// Try to get expired key
		_, err = store.GetKey("expired-key", AccessRead)
		if err != ErrKeyExpired {
			t.Errorf("Expected ErrKeyExpired, got: %v", err)
		}
	})

	// Test access levels
	t.Run("access levels", func(t *testing.T) {
		err := store.AddKey("read-key", "openai", "sk-readonly", AccessRead, nil)
		if err != nil {
			t.Errorf("Failed to add read-only key: %v", err)
		}

		// Try to use key with insufficient access
		_, err = store.GetKey("read-key", AccessWrite)
		if err != ErrInvalidAccess {
			t.Errorf("Expected ErrInvalidAccess, got: %v", err)
		}

		// Try to use key with correct access
		_, err = store.GetKey("read-key", AccessRead)
		if err != nil {
			t.Errorf("Failed to get key with correct access: %v", err)
		}
	})

	// Test key rotation
	t.Run("key rotation", func(t *testing.T) {
		// Add initial key
		err := store.AddKey("rotate-key", "openai", "sk-old", AccessRead, nil)
		if err != nil {
			t.Errorf("Failed to add key for rotation: %v", err)
		}

		// Rotate key
		err = store.RotateKey("rotate-key", "sk-new")
		if err != nil {
			t.Errorf("Failed to rotate key: %v", err)
		}

		// Verify new value
		value, err := store.GetKey("rotate-key", AccessRead)
		if err != nil {
			t.Errorf("Failed to get rotated key: %v", err)
		}
		if value != "sk-new" {
			t.Errorf("Got wrong rotated key value: %s", value)
		}
	})

	// Test key removal
	t.Run("remove key", func(t *testing.T) {
		// Add key to remove
		err := store.AddKey("remove-key", "openai", "sk-remove", AccessRead, nil)
		if err != nil {
			t.Errorf("Failed to add key for removal: %v", err)
		}

		// Remove key
		err = store.RemoveKey("remove-key")
		if err != nil {
			t.Errorf("Failed to remove key: %v", err)
		}

		// Verify key is gone
		_, err = store.GetKey("remove-key", AccessRead)
		if err != ErrKeyNotFound {
			t.Errorf("Expected ErrKeyNotFound, got: %v", err)
		}
	})

	// Test persistence
	t.Run("persistence", func(t *testing.T) {
		// Add a key
		err := store.AddKey("persist-key", "openai", "sk-persist", AccessRead, nil)
		if err != nil {
			t.Errorf("Failed to add key for persistence test: %v", err)
		}

		// Create new store instance
		store2, err := NewKeyStore(cfg)
		if err != nil {
			t.Fatalf("Failed to create second key store: %v", err)
		}

		// Verify key exists in new instance
		value, err := store2.GetKey("persist-key", AccessRead)
		if err != nil {
			t.Errorf("Failed to get persisted key: %v", err)
		}
		if value != "sk-persist" {
			t.Errorf("Got wrong persisted key value: %s", value)
		}
	})

	// Test invalid keys
	t.Run("invalid keys", func(t *testing.T) {
		tests := []struct {
			name     string
			keyType  string
			value    string
			wantErr  error
		}{
			{"", "openai", "sk-test", ErrInvalidKey},
			{"empty-type", "", "sk-test", ErrInvalidKey},
			{"empty-value", "openai", "", ErrInvalidKey},
		}

		for _, tt := range tests {
			err := store.AddKey(tt.name, tt.keyType, tt.value, AccessRead, nil)
			if err != tt.wantErr {
				t.Errorf("AddKey(%q, %q, %q) error = %v, want %v",
					tt.name, tt.keyType, tt.value, err, tt.wantErr)
			}
		}
	})

	// Test file permissions
	t.Run("file permissions", func(t *testing.T) {
		info, err := os.Stat(keyPath)
		if err != nil {
			t.Fatalf("Failed to stat key file: %v", err)
		}

		mode := info.Mode()
		if mode&0077 != 0 {
			t.Errorf("Key file has wrong permissions: %v", mode)
		}
	})
}

func TestKeyStoreErrors(t *testing.T) {
	// Test invalid encryption key
	t.Run("invalid encryption key", func(t *testing.T) {
		cfg := &config.Config{
			Security: types.SecurityConfig{
				EncryptionKey:  "invalid-base64",
				KeyStoragePath: "test.dat",
			},
		}

		_, err := NewKeyStore(cfg)
		if err == nil {
			t.Error("Expected error for invalid encryption key")
		}
	})

	// Test corrupted storage
	t.Run("corrupted storage", func(t *testing.T) {
		tmpDir := t.TempDir()
		keyPath := filepath.Join(tmpDir, "corrupt.dat")

		// Write invalid data
		err := os.WriteFile(keyPath, []byte("corrupted"), 0600)
		if err != nil {
			t.Fatalf("Failed to write corrupted file: %v", err)
		}

		cfg := &config.Config{
			Security: types.SecurityConfig{
				EncryptionKey:  base64.StdEncoding.EncodeToString(make([]byte, 32)),
				KeyStoragePath: keyPath,
			},
		}

		_, err = NewKeyStore(cfg)
		if err == nil {
			t.Error("Expected error for corrupted storage")
		}
	})
}
