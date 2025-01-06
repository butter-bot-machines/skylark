package concrete

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/butter-bot-machines/skylark/pkg/config"
	"github.com/butter-bot-machines/skylark/pkg/security"
	"github.com/butter-bot-machines/skylark/pkg/security/types"
)

// auditLogger implements security.AuditLogger
type auditLogger struct {
	mu        sync.Mutex
	config    types.AuditLogConfig
	file      *os.File
	buffer    []*types.Event
	lastFlush time.Time
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(cfg *config.Config) (security.AuditLogger, error) {
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

	return &auditLogger{
		config:    cfg.Security.AuditLog,
		file:      file,
		buffer:    make([]*types.Event, 0, 100),
		lastFlush: time.Now(),
	}, nil
}

// Log implements security.AuditLogger
func (a *auditLogger) Log(eventType types.EventType, severity types.Severity, source, details string, metadata map[string]interface{}) error {
	if a == nil {
		return nil // Audit logging disabled
	}

	// Check if this event type should be logged
	if !a.shouldLog(eventType) {
		return nil
	}

	event := &types.Event{
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

// Query implements security.AuditLogger
func (a *auditLogger) Query(filter security.EventFilter) ([]*types.Event, error) {
	if a == nil {
		return nil, nil
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Flush pending events first
	if err := a.flush(); err != nil {
		return nil, err
	}

	// Read all events from file
	var events []*types.Event
	decoder := json.NewDecoder(a.file)
	for {
		var event types.Event
		if err := decoder.Decode(&event); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("failed to decode event: %w", err)
		}

		if filter == nil || filter.MatchEvent(&event) {
			events = append(events, &event)
		}
	}

	return events, nil
}

// Export implements security.AuditLogger
func (a *auditLogger) Export(w io.Writer) error {
	if a == nil {
		return nil
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Flush pending events first
	if err := a.flush(); err != nil {
		return err
	}

	// Copy log file to writer
	if _, err := a.file.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek log file: %w", err)
	}

	if _, err := io.Copy(w, a.file); err != nil {
		return fmt.Errorf("failed to export log: %w", err)
	}

	return nil
}

// Rotate implements security.AuditLogger
func (a *auditLogger) Rotate() error {
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

// Close implements security.AuditLogger
func (a *auditLogger) Close() error {
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

// Internal methods

func (a *auditLogger) shouldLog(eventType types.EventType) bool {
	if len(a.config.Events) == 0 {
		return true // Log everything if no specific events configured
	}
	for _, e := range a.config.Events {
		if types.EventType(e) == eventType {
			return true
		}
	}
	return false
}

func (a *auditLogger) flush() error {
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

// generateEventID generates a unique event ID
func generateEventID() string {
	return fmt.Sprintf("%d-%x", time.Now().UnixNano(), time.Now().UnixNano()%1000000)
}
