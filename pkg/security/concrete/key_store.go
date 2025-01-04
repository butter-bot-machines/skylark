package concrete

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/butter-bot-machines/skylark/pkg/config"
	"github.com/butter-bot-machines/skylark/pkg/security"
)

var (
	ErrKeyNotFound      = errors.New("key not found")
	ErrKeyExpired       = errors.New("key expired")
	ErrInvalidAccess    = errors.New("invalid access level")
	ErrInvalidKey       = errors.New("invalid key format")
	ErrStorageCorrupted = errors.New("key storage corrupted")
)

const (
	// Access level bits
	AccessRead   uint32 = 1 << iota // Read operations
	AccessWrite                     // Write operations
	AccessAdmin                     // Administrative operations
)

// Key represents an API key with metadata
type Key struct {
	Value      string    `json:"value"`
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	Created    time.Time `json:"created"`
	LastUsed   time.Time `json:"last_used"`
	Expiry     time.Time `json:"expiry,omitempty"`
	AccessMask uint32    `json:"access_mask"`
}

// keyStore implements security.KeyStore
type keyStore struct {
	mu       sync.RWMutex
	keys     map[string]Key
	filepath string
	cipher   cipher.AEAD
}

// NewKeyStore creates a new key store
func NewKeyStore(cfg *config.Config) (security.KeyStore, error) {
	// Decode encryption key
	encKey, err := base64.StdEncoding.DecodeString(cfg.Security.EncryptionKey)
	if err != nil || len(encKey) != 32 {
		return nil, fmt.Errorf("invalid encryption key: must be base64 encoded 32-byte key")
	}

	// Create AES cipher
	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM cipher mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Create storage directory if needed
	storageDir := filepath.Dir(cfg.Security.KeyStoragePath)
	if err := os.MkdirAll(storageDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	ks := &keyStore{
		keys:     make(map[string]Key),
		filepath: cfg.Security.KeyStoragePath,
		cipher:   gcm,
	}

	// Load existing keys
	if err := ks.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load keys: %w", err)
	}

	return ks, nil
}

// Get implements security.KeyStore
func (ks *keyStore) Get(name string) (string, error) {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	key, ok := ks.keys[name]
	if !ok {
		return "", ErrKeyNotFound
	}

	// Check expiry
	if !key.Expiry.IsZero() && time.Now().After(key.Expiry) {
		return "", ErrKeyExpired
	}

	// Update last used time
	key.LastUsed = time.Now()
	ks.keys[name] = key

	// Save changes
	if err := ks.save(); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to save key store: %v\n", err)
	}

	return key.Value, nil
}

// Set implements security.KeyStore
func (ks *keyStore) Set(name, value string) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	// Validate key
	if name == "" || value == "" {
		return ErrInvalidKey
	}

	// Create or update key entry
	key := Key{
		Value:      value,
		Name:       name,
		Type:       "generic",
		Created:    time.Now(),
		LastUsed:   time.Now(),
		AccessMask: AccessRead | AccessWrite,
	}

	// Add to store
	ks.keys[name] = key

	// Save changes
	return ks.save()
}

// Delete implements security.KeyStore
func (ks *keyStore) Delete(name string) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	if _, ok := ks.keys[name]; !ok {
		return ErrKeyNotFound
	}

	delete(ks.keys, name)
	return ks.save()
}

// List implements security.KeyStore
func (ks *keyStore) List() []string {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	names := make([]string, 0, len(ks.keys))
	for name := range ks.keys {
		names = append(names, name)
	}
	return names
}

// Close implements security.KeyStore
func (ks *keyStore) Close() error {
	return ks.save()
}

// Internal methods

func (ks *keyStore) load() error {
	// Read encrypted data
	data, err := os.ReadFile(ks.filepath)
	if err != nil {
		return err
	}

	// Data must be at least nonce size + 1
	if len(data) < ks.cipher.NonceSize()+1 {
		return ErrStorageCorrupted
	}

	// Extract nonce and ciphertext
	nonce := data[:ks.cipher.NonceSize()]
	ciphertext := data[ks.cipher.NonceSize():]

	// Decrypt data
	plaintext, err := ks.cipher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return fmt.Errorf("failed to decrypt: %w", err)
	}

	// Parse JSON
	if err := json.Unmarshal(plaintext, &ks.keys); err != nil {
		return fmt.Errorf("failed to parse: %w", err)
	}

	return nil
}

func (ks *keyStore) save() error {
	// Marshal to JSON
	plaintext, err := json.Marshal(ks.keys)
	if err != nil {
		return fmt.Errorf("failed to marshal: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, ks.cipher.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt data
	ciphertext := ks.cipher.Seal(nil, nonce, plaintext, nil)

	// Combine nonce and ciphertext
	data := make([]byte, 0, len(nonce)+len(ciphertext))
	data = append(data, nonce...)
	data = append(data, ciphertext...)

	// Write to temporary file first
	tmpFile := ks.filepath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Rename temporary file to actual file (atomic operation)
	if err := os.Rename(tmpFile, ks.filepath); err != nil {
		os.Remove(tmpFile) // Clean up on error
		return fmt.Errorf("failed to save: %w", err)
	}

	return nil
}
