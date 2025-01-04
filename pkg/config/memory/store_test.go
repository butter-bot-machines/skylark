package memory

import (
	"sync"
	"testing"

	"github.com/butter-bot-machines/skylark/pkg/config"
)

func TestStore_BasicOperations(t *testing.T) {
	store := NewStore(nil)

	// Test Set and Get
	t.Run("Set and Get", func(t *testing.T) {
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
	})

	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		if err := store.Delete("key1"); err != nil {
			t.Errorf("Delete failed: %v", err)
		}

		_, err := store.Get("key1")
		if err != config.ErrNotFound {
			t.Errorf("Got error %v, want ErrNotFound", err)
		}
	})

	// Test Reset
	t.Run("Reset", func(t *testing.T) {
		if err := store.Set("key2", "value2"); err != nil {
			t.Errorf("Set failed: %v", err)
		}

		if err := store.Reset(); err != nil {
			t.Errorf("Reset failed: %v", err)
		}

		_, err := store.Get("key2")
		if err != config.ErrNotFound {
			t.Errorf("Got error %v, want ErrNotFound", err)
		}
	})
}

func TestStore_BatchOperations(t *testing.T) {
	store := NewStore(nil)

	// Test SetAll and GetAll
	t.Run("SetAll and GetAll", func(t *testing.T) {
		input := map[string]interface{}{
			"key1": "value1",
			"key2": 42,
			"key3": true,
		}

		if err := store.SetAll(input); err != nil {
			t.Errorf("SetAll failed: %v", err)
		}

		output, err := store.GetAll()
		if err != nil {
			t.Errorf("GetAll failed: %v", err)
		}

		if len(output) != len(input) {
			t.Errorf("Got %d keys, want %d", len(output), len(input))
		}

		for k, v := range input {
			if output[k] != v {
				t.Errorf("For key %s, got %v, want %v", k, output[k], v)
			}
		}
	})
}

func TestStore_Validation(t *testing.T) {
	validator := func(data map[string]interface{}) error {
		if v, ok := data["invalid"]; ok && v == "value" {
			return config.ErrInvalidValue
		}
		return nil
	}

	store := NewStore(validator)

	// Test validation on Set
	t.Run("Validation on Set", func(t *testing.T) {
		if err := store.Set("valid", "value"); err != nil {
			t.Errorf("Set failed: %v", err)
		}

		if err := store.Set("invalid", "value"); err != config.ErrInvalidValue {
			t.Errorf("Got error %v, want ErrInvalidValue", err)
		}

		// Verify invalid value was not stored
		_, err := store.Get("invalid")
		if err != config.ErrNotFound {
			t.Errorf("Got error %v, want ErrNotFound", err)
		}
	})

	// Test validation on SetAll
	t.Run("Validation on SetAll", func(t *testing.T) {
		input := map[string]interface{}{
			"invalid": "value",
			"valid":   "value",
		}

		if err := store.SetAll(input); err != config.ErrInvalidValue {
			t.Errorf("Got error %v, want ErrInvalidValue", err)
		}

		// Verify no values were stored
		output, err := store.GetAll()
		if err != nil {
			t.Errorf("GetAll failed: %v", err)
		}

		if len(output) != 1 {
			t.Errorf("Got %d keys, want 1", len(output))
		}
	})
}

func TestStore_Concurrency(t *testing.T) {
	store := NewStore(nil)
	var wg sync.WaitGroup
	workers := 10
	iterations := 100

	// Concurrent Set and Get
	t.Run("Concurrent Set and Get", func(t *testing.T) {
		wg.Add(workers)
		for i := 0; i < workers; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					key := "key"
					value := "value"

					if err := store.Set(key, value); err != nil {
						t.Errorf("Set failed: %v", err)
					}

					_, err := store.Get(key)
					if err != nil {
						t.Errorf("Get failed: %v", err)
					}
				}
			}(i)
		}
		wg.Wait()
	})

	// Concurrent GetAll and SetAll
	t.Run("Concurrent GetAll and SetAll", func(t *testing.T) {
		wg.Add(workers)
		for i := 0; i < workers; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					input := map[string]interface{}{
						"key1": "value1",
						"key2": "value2",
					}

					if err := store.SetAll(input); err != nil {
						t.Errorf("SetAll failed: %v", err)
					}

					_, err := store.GetAll()
					if err != nil {
						t.Errorf("GetAll failed: %v", err)
					}
				}
			}(i)
		}
		wg.Wait()
	})
}

func TestStore_ErrorCases(t *testing.T) {
	store := NewStore(nil)

	// Test nil value
	t.Run("Nil Value", func(t *testing.T) {
		if err := store.Set("key", nil); err != config.ErrInvalidValue {
			t.Errorf("Got error %v, want ErrInvalidValue", err)
		}
	})

	// Test nil map in SetAll
	t.Run("Nil Map in SetAll", func(t *testing.T) {
		if err := store.SetAll(nil); err != config.ErrInvalidValue {
			t.Errorf("Got error %v, want ErrInvalidValue", err)
		}
	})

	// Test non-existent key
	t.Run("Non-existent Key", func(t *testing.T) {
		_, err := store.Get("nonexistent")
		if err != config.ErrNotFound {
			t.Errorf("Got error %v, want ErrNotFound", err)
		}
	})

	// Test delete non-existent key
	t.Run("Delete Non-existent Key", func(t *testing.T) {
		err := store.Delete("nonexistent")
		if err != config.ErrNotFound {
			t.Errorf("Got error %v, want ErrNotFound", err)
		}
	})
}
