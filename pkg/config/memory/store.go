package memory

import (
	"sync"

	"github.com/butter-bot-machines/skylark/pkg/config"
)

// Store implements config.Store using in-memory storage
type Store struct {
	mu       sync.RWMutex
	data     map[string]interface{}
	validate config.ValidateFunc
}

// NewStore creates a new memory-backed config store
func NewStore(validate config.ValidateFunc) *Store {
	return &Store{
		data:     make(map[string]interface{}),
		validate: validate,
	}
}

// Load is a no-op for memory store since data is already in memory
func (s *Store) Load() error {
	if s.validate != nil {
		s.mu.RLock()
		err := s.validate(s.data)
		s.mu.RUnlock()
		return err
	}
	return nil
}

// Save is a no-op for memory store
func (s *Store) Save() error {
	return nil
}

// Reset clears all stored data
func (s *Store) Reset() error {
	s.mu.Lock()
	s.data = make(map[string]interface{})
	s.mu.Unlock()
	return nil
}

// Get retrieves a value by key
func (s *Store) Get(key string) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, ok := s.data[key]
	if !ok {
		return nil, config.ErrNotFound
	}
	return value, nil
}

// Set stores a value by key
func (s *Store) Set(key string, value interface{}) error {
	if value == nil {
		return config.ErrInvalidValue
	}

	s.mu.Lock()
	s.data[key] = value
	s.mu.Unlock()

	if s.validate != nil {
		s.mu.RLock()
		err := s.validate(s.data)
		s.mu.RUnlock()
		if err != nil {
			// Rollback on validation failure
			s.mu.Lock()
			delete(s.data, key)
			s.mu.Unlock()
			return err
		}
	}

	return nil
}

// Delete removes a value by key
func (s *Store) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.data[key]; !ok {
		return config.ErrNotFound
	}

	delete(s.data, key)
	return nil
}

// GetAll returns all stored key/value pairs
func (s *Store) GetAll() (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create a copy to prevent external modification
	result := make(map[string]interface{}, len(s.data))
	for k, v := range s.data {
		result[k] = v
	}
	return result, nil
}

// SetAll replaces all stored values
func (s *Store) SetAll(values map[string]interface{}) error {
	if values == nil {
		return config.ErrInvalidValue
	}

	// Validate before replacing
	if s.validate != nil {
		if err := s.validate(values); err != nil {
			return err
		}
	}

	s.mu.Lock()
	s.data = make(map[string]interface{}, len(values))
	for k, v := range values {
		s.data[k] = v
	}
	s.mu.Unlock()

	return nil
}

// Validate runs the validation function if set
func (s *Store) Validate() error {
	if s.validate == nil {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.validate(s.data)
}
