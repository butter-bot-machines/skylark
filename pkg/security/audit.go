package security

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/butter-bot-machines/skylark/pkg/config"
)

// EventType represents the type of security event
type EventType string

const (
	// Key management events
	EventKeyAccess    EventType = "key_access"
	EventKeyCreated   EventType = "key_created"
	EventKeyRotated   EventType = "key_rotated"
	EventKeyRemoved   EventType = "key_removed"
	EventKeyExpired   EventType = "key_expired"
	
	// File operation events
	EventFileAccess   EventType = "file_access"
	EventFileModified EventType = "file_modified"
	EventFileCreated  EventType = "file_created"
	EventFileRemoved  EventType = "file_removed"
	
	// Resource events
	EventMemoryLimit  EventType = "memory_limit"
	EventCPULimit     EventType = "cpu_limit"
	EventDiskLimit    EventType = "disk_limit"
	
	// Security events
	EventAuthFailure  EventType = "auth_failure"
	EventAccessDenied EventType = "access_denied"
	EventThreatDetected EventType = "threat_detected"
)

// Severity represents the severity level of a security event
type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
	SeverityCritical Severity = "critical"
)

// AuditEvent represents a security audit event
type AuditEvent struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Type      EventType `json:"type"`
	Severity  Severity  `json:"severity"`
	Source    string    `json:"source"`
	Details   string    `json:"details"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// AuditLog manages security event logging
type AuditLog struct {
	mu       sync.Mutex
	config   config.AuditLogConfig
	file     *os.File
	buffer   []*AuditEvent
	lastFlush time.Time
}

// NewAuditLog creates a new audit log
func NewAuditLog(cfg *config.Config) (*AuditLog, error) {
	if !cfg.Security.AuditLog.Enabled {
		return nil, nil // Audit logging disabled
	}

	// Create log directory if needed
	logDir := filepath.Dir(cfg.Security.AuditLog.Path)
	if err := os.MkdirAll(logDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create audit log directory: %w", err)
	}

	// Open log file in append mode
	file, err := os.OpenFile(
		cfg.Security.AuditLog.Path,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0600,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log: %w", err)
	}

	return &AuditLog{
		config:    cfg.Security.AuditLog,
		file:      file,
		buffer:    make([]*AuditEvent, 0, 100),
		lastFlush: time.Now(),
	}, nil
}

// Log records a security event
func (a *AuditLog) Log(eventType EventType, severity Severity, source, details string, metadata map[string]interface{}) error {
	if a == nil {
		return nil // Audit logging disabled
	}

	// Check if this event type should be logged
	if !a.shouldLog(eventType) {
		return nil
	}

	event := &AuditEvent{
		ID:        generateEventID(),
		Timestamp: time.Now(),
		Type:      eventType,
		Severity:  severity,
		Source:    source,
		Details:   details,
		Metadata:  metadata,
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Add to buffer
	a.buffer = append(a.buffer, event)

	// Flush if buffer is full or enough time has passed
	if len(a.buffer) >= 100 || time.Since(a.lastFlush) > 5*time.Second {
		return a.flush()
	}

	return nil
}

// shouldLog checks if an event type should be logged
func (a *AuditLog) shouldLog(eventType EventType) bool {
	if len(a.config.Events) == 0 {
		return true // Log everything if no specific events configured
	}
	for _, e := range a.config.Events {
		if EventType(e) == eventType {
			return true
		}
	}
	return false
}

// flush writes buffered events to disk
func (a *AuditLog) flush() error {
	if len(a.buffer) == 0 {
		return nil
	}

	// Convert events to JSON lines
	for _, event := range a.buffer {
		data, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal event: %w", err)
		}
		if _, err := a.file.Write(append(data, '\n')); err != nil {
			return fmt.Errorf("failed to write event: %w", err)
		}
	}

	// Clear buffer and update flush time
	a.buffer = a.buffer[:0]
	a.lastFlush = time.Now()

	return a.file.Sync()
}

// Close flushes remaining events and closes the log file
func (a *AuditLog) Close() error {
	if a == nil {
		return nil
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.flush(); err != nil {
		return err
	}
	return a.file.Close()
}

// Rotate rotates the audit log file
func (a *AuditLog) Rotate() error {
	if a == nil {
		return nil
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Flush pending events
	if err := a.flush(); err != nil {
		return err
	}

	// Close current file
	if err := a.file.Close(); err != nil {
		return fmt.Errorf("failed to close current log: %w", err)
	}

	// Rotate file (add timestamp to filename)
	timestamp := time.Now().Format("20060102-150405")
	rotatedPath := fmt.Sprintf("%s.%s", a.config.Path, timestamp)
	if err := os.Rename(a.config.Path, rotatedPath); err != nil {
		return fmt.Errorf("failed to rotate log: %w", err)
	}

	// Open new log file
	file, err := os.OpenFile(
		a.config.Path,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0600,
	)
	if err != nil {
		return fmt.Errorf("failed to open new log: %w", err)
	}

	a.file = file
	return nil
}

// generateEventID generates a unique event ID
func generateEventID() string {
	return fmt.Sprintf("%d-%x", time.Now().UnixNano(), time.Now().UnixNano()%1000000)
}
