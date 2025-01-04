package file

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/butter-bot-machines/skylark/pkg/config"
)

func TestStore_BasicOperations(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	store := NewStore(path, nil)

	// Test initial state
	t.Run("Initial State", func(t *testing.T) {
		if err := store.Load(); err != nil {
			t.Errorf("Load failed: %v", err)
		}

		data, err := store.GetAll()
		if err != nil {
			t.Errorf("GetAll failed: %v", err)
		}
		if len(data) != 0 {
			t.Errorf("Expected empty data, got %v", data)
		}
	})

	// Test basic operations
	t.Run("Basic Operations", func(t *testing.T) {
		// Set and get
		if err := store.Set("key1", "value1"); err != nil {
			t.Errorf("Set failed: %v", err)
		}

		value, err := store.Get("key1")
		if err != nil {
			t.Errorf("Get failed: %v", err)
		}
		if value != "value1" {
			t.Errorf("Got %v, want value1", value)
		}

		// Delete
		if err := store.Delete("key1"); err != nil {
			t.Errorf("Delete failed: %v", err)
		}

		_, err = store.Get("key1")
		if err != config.ErrNotFound {
			t.Errorf("Got error %v, want ErrNotFound", err)
		}
	})

	// Test persistence
	t.Run("Persistence", func(t *testing.T) {
		// Set some data
		if err := store.Set("key2", "value2"); err != nil {
			t.Errorf("Set failed: %v", err)
		}

		// Create new store with same path
		store2 := NewStore(path, nil)
		if err := store2.Load(); err != nil {
			t.Errorf("Load failed: %v", err)
		}

		// Verify data persisted
		value, err := store2.Get("key2")
		if err != nil {
			t.Errorf("Get failed: %v", err)
		}
		if value != "value2" {
			t.Errorf("Got %v, want value2", value)
		}
	})
}

func TestStore_BatchOperations(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	store := NewStore(path, nil)

	// Test SetAll and GetAll
	t.Run("Batch Operations", func(t *testing.T) {
		data := map[string]interface{}{
			"key1": "value1",
			"key2": 42,
			"key3": true,
		}

		if err := store.SetAll(data); err != nil {
			t.Errorf("SetAll failed: %v", err)
		}

		got, err := store.GetAll()
		if err != nil {
			t.Errorf("GetAll failed: %v", err)
		}

		if len(got) != len(data) {
			t.Errorf("Got %d items, want %d", len(got), len(data))
		}

		for k, v := range data {
			if got[k] != v {
				t.Errorf("For key %s, got %v, want %v", k, got[k], v)
			}
		}
	})

	// Test Reset
	t.Run("Reset", func(t *testing.T) {
		if err := store.Reset(); err != nil {
			t.Errorf("Reset failed: %v", err)
		}

		data, err := store.GetAll()
		if err != nil {
			t.Errorf("GetAll failed: %v", err)
		}
		if len(data) != 0 {
			t.Errorf("Expected empty data after reset, got %v", data)
		}
	})
}

func TestStore_Validation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	validate := func(data map[string]interface{}) error {
		if v, ok := data["invalid"]; ok && v == true {
			return config.ErrInvalidConfig
		}
		return nil
	}

	store := NewStore(path, validate)

	// Test validation on Set
	t.Run("Validation on Set", func(t *testing.T) {
		// Valid data should work
		if err := store.Set("valid", true); err != nil {
			t.Errorf("Set failed: %v", err)
		}

		// Invalid data should fail
		if err := store.Set("invalid", true); err != config.ErrInvalidConfig {
			t.Errorf("Got error %v, want ErrInvalidConfig", err)
		}

		// Verify invalid data was not stored
		_, err := store.Get("invalid")
		if err != config.ErrNotFound {
			t.Errorf("Got error %v, want ErrNotFound", err)
		}
	})

	// Test validation on SetAll
	t.Run("Validation on SetAll", func(t *testing.T) {
		validData := map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		}

		invalidData := map[string]interface{}{
			"key1":    "value1",
			"invalid": true,
		}

		// Valid data should work
		if err := store.SetAll(validData); err != nil {
			t.Errorf("SetAll failed: %v", err)
		}

		// Invalid data should fail
		if err := store.SetAll(invalidData); err != config.ErrInvalidConfig {
			t.Errorf("Got error %v, want ErrInvalidConfig", err)
		}
	})
}

func TestStore_Concurrency(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	store := NewStore(path, nil)

	// Test concurrent access
	t.Run("Concurrent Access", func(t *testing.T) {
		var wg sync.WaitGroup
		var mu sync.Mutex
		var errors []error

		workers := 10
		iterations := 100

		wg.Add(workers)
		for i := 0; i < workers; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					key := "key"
					value := "value"

					// Set
					if err := store.Set(key, value); err != nil {
						mu.Lock()
						errors = append(errors, err)
						mu.Unlock()
						return
					}

					// Get
					if _, err := store.Get(key); err != nil {
						mu.Lock()
						errors = append(errors, err)
						mu.Unlock()
						return
					}

					// GetAll
					if _, err := store.GetAll(); err != nil {
						mu.Lock()
						errors = append(errors, err)
						mu.Unlock()
						return
					}
				}
			}(i)
		}
		wg.Wait()

		// Check for errors after all goroutines complete
		if len(errors) > 0 {
			for _, err := range errors {
				t.Errorf("Concurrent operation failed: %v", err)
			}
		}
	})
}

func TestStore_ErrorCases(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	store := NewStore(path, nil)

	// Test invalid operations
	t.Run("Invalid Operations", func(t *testing.T) {
		// Get non-existent key
		if _, err := store.Get("nonexistent"); err != config.ErrNotFound {
			t.Errorf("Got error %v, want ErrNotFound", err)
		}

		// Delete non-existent key
		if err := store.Delete("nonexistent"); err != config.ErrNotFound {
			t.Errorf("Got error %v, want ErrNotFound", err)
		}

		// Set nil value
		if err := store.Set("key", nil); err != config.ErrInvalidValue {
			t.Errorf("Got error %v, want ErrInvalidValue", err)
		}

		// SetAll nil map
		if err := store.SetAll(nil); err != config.ErrInvalidValue {
			t.Errorf("Got error %v, want ErrInvalidValue", err)
		}
	})

	// Test file system errors
	t.Run("File System Errors", func(t *testing.T) {
		// Make directory read-only
		readOnlyDir := filepath.Join(dir, "readonly")
		if err := os.MkdirAll(readOnlyDir, 0444); err != nil {
			t.Fatalf("Failed to create read-only directory: %v", err)
		}

		readOnlyPath := filepath.Join(readOnlyDir, "config.yaml")
		readOnlyStore := NewStore(readOnlyPath, nil)

		// Save should fail
		if err := readOnlyStore.Save(); err == nil {
			t.Error("Save should fail with read-only directory")
		}
	})
}
