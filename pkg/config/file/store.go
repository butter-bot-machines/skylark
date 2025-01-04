package file

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/butter-bot-machines/skylark/pkg/config"
	"gopkg.in/yaml.v3"
)

// Store implements config.Store using file storage
type Store struct {
	mu       sync.RWMutex
	path     string
	data     map[string]interface{}
	validate config.ValidateFunc
}

// NewStore creates a new file-backed config store
func NewStore(path string, validate config.ValidateFunc) *Store {
	return &Store{
		path:     path,
		data:     make(map[string]interface{}),
		validate: validate,
	}
}

// Load reads configuration from file
func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			// Initialize with empty data if file doesn't exist
			s.data = make(map[string]interface{})
			return nil
		}
		return err
	}

	if err := yaml.Unmarshal(data, &s.data); err != nil {
		return err
	}

	if s.validate != nil {
		if err := s.validate(s.data); err != nil {
			return err
		}
	}

	return nil
}

// Save writes configuration to file
func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.validate != nil {
		if err := s.validate(s.data); err != nil {
			return err
		}
	}

	data, err := yaml.Marshal(s.data)
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(s.path, data, 0644)
}

// Reset clears all stored data
func (s *Store) Reset() error {
	s.mu.Lock()
	s.data = make(map[string]interface{})
	s.mu.Unlock()

	return s.Save()
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

	if s.validate != nil {
		err := s.validate(s.data)
		if err != nil {
			// Rollback on validation failure
			delete(s.data, key)
			s.mu.Unlock()
			return err
		}
	}

	s.mu.Unlock()
	return s.Save()
}

// Delete removes a value by key
func (s *Store) Delete(key string) error {
	s.mu.Lock()
	if _, ok := s.data[key]; !ok {
		s.mu.Unlock()
		return config.ErrNotFound
	}

	delete(s.data, key)
	s.mu.Unlock()

	return s.Save()
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

	return s.Save()
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
