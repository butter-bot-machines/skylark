package security

import (
	"io"

	"github.com/butter-bot-machines/skylark/pkg/security/types"
)

// EventFilter defines criteria for querying events
type EventFilter interface {
	// MatchEvent returns true if an event matches the filter
	MatchEvent(*types.Event) bool
}

// EventStorage stores and retrieves events
type EventStorage interface {
	// Store stores an event
	Store(*types.Event) error

	// Query retrieves events matching a filter
	Query(EventFilter) ([]*types.Event, error)

	// Export exports events to a writer
	Export(io.Writer) error

	// Rotate rotates the storage (e.g., log rotation)
	Rotate() error

	// Close closes the storage
	Close() error
}

// AuditLogger logs security events
type AuditLogger interface {
	// Log logs a security event
	Log(types.EventType, types.Severity, string, string, map[string]interface{}) error

	// Query searches audit logs
	Query(EventFilter) ([]*types.Event, error)

	// Export exports logs
	Export(io.Writer) error

	// Rotate rotates the log file
	Rotate() error

	// Close closes the logger
	Close() error
}

// KeyStore manages security keys
type KeyStore interface {
	// Get retrieves a key by name
	Get(name string) (string, error)

	// Set stores a key
	Set(name, value string) error

	// Delete removes a key
	Delete(name string) error

	// List returns all key names
	List() []string

	// Close closes the store
	Close() error
}

// FileGuard controls file access
type FileGuard interface {
	// CheckRead verifies read access to a path
	CheckRead(path string) error

	// CheckWrite verifies write access to a path
	CheckWrite(path string) error

	// AddAllowedPath adds a path to allowed paths
	AddAllowedPath(path string) error

	// RemoveAllowedPath removes a path from allowed paths
	RemoveAllowedPath(path string)

	// Close closes the guard
	Close() error
}

// ResourceGuard controls resource usage
type ResourceGuard interface {
	// Check verifies resource usage
	Check(types.ResourceUsage) error

	// Update updates resource limits
	Update(types.ResourceLimits) error

	// Reset resets usage counters
	Reset()

	// Close closes the guard
	Close() error
}

// Manager coordinates security components
type Manager interface {
	// Logger returns the audit logger
	Logger() AuditLogger

	// Keys returns the key store
	Keys() KeyStore

	// Files returns the file guard
	Files() FileGuard

	// Resources returns the resource guard
	Resources() ResourceGuard

	// Close closes all components
	Close() error
}
