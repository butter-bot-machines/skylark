package config

import "time"

// Store defines the interface for configuration storage and retrieval
type Store interface {
	// Basic operations
	Load() error
	Save() error
	Reset() error

	// Value operations
	Get(key string) (interface{}, error)
	Set(key string, value interface{}) error
	Delete(key string) error

	// Batch operations
	GetAll() (map[string]interface{}, error)
	SetAll(values map[string]interface{}) error

	// Validation
	Validate() error
}

// Environment defines the interface for environment variable access
type Environment interface {
	// Environment access
	GetString(key string) string
	GetInt(key string) int
	GetBool(key string) bool
	GetDuration(key string) time.Duration
}

// ValidateFunc defines a function type for custom validation
type ValidateFunc func(data map[string]interface{}) error

// Error types for config operations
var (
	ErrNotFound      = Error{"key not found"}
	ErrInvalidType   = Error{"invalid type"}
	ErrInvalidValue  = Error{"invalid value"}
	ErrInvalidConfig = Error{"invalid configuration"}
)

// Error represents a configuration error
type Error struct {
	Message string
}

func (e Error) Error() string {
	return e.Message
}
