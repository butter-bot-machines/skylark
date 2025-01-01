package security

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/butter-bot-machines/skylark/pkg/config"
)

func TestAuditLog(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	// Create test config
	cfg := &config.Config{
		Security: config.SecurityConfig{
			AuditLog: config.AuditLogConfig{
				Enabled:       true,
				Path:         logPath,
				RetentionDays: 30,
				Events:       []string{string(EventKeyAccess), string(EventFileAccess)},
			},
		},
	}

	// Create audit log
	auditLog, err := NewAuditLog(cfg)
	if err != nil {
		t.Fatalf("Failed to create audit log: %v", err)
	}
	defer auditLog.Close()

	// Test basic event logging
	t.Run("basic logging", func(t *testing.T) {
		err := auditLog.Log(
			EventKeyAccess,
			SeverityInfo,
			"test",
			"accessed key test-key",
			map[string]interface{}{"key": "test-key"},
		)
		if err != nil {
			t.Errorf("Failed to log event: %v", err)
		}

		// Force flush
		if err := auditLog.flush(); err != nil {
			t.Errorf("Failed to flush: %v", err)
		}

		// Read log file
		events := readLogEvents(t, logPath)
		if len(events) != 1 {
			t.Fatalf("Expected 1 event, got %d", len(events))
		}

		event := events[0]
		if event.Type != EventKeyAccess {
			t.Errorf("Wrong event type: %s", event.Type)
		}
		if event.Severity != SeverityInfo {
			t.Errorf("Wrong severity: %s", event.Severity)
		}
		if event.Source != "test" {
			t.Errorf("Wrong source: %s", event.Source)
		}
		if event.Details != "accessed key test-key" {
			t.Errorf("Wrong details: %s", event.Details)
		}
		if event.Metadata["key"] != "test-key" {
			t.Errorf("Wrong metadata: %v", event.Metadata)
		}
	})

	// Test event filtering
	t.Run("event filtering", func(t *testing.T) {
		// Log allowed event
		err := auditLog.Log(
			EventFileAccess,
			SeverityInfo,
			"test",
			"accessed file test.txt",
			nil,
		)
		if err != nil {
			t.Errorf("Failed to log allowed event: %v", err)
		}

		// Log filtered event
		err = auditLog.Log(
			EventCPULimit,
			SeverityWarning,
			"test",
			"CPU limit exceeded",
			nil,
		)
		if err != nil {
			t.Errorf("Failed to log filtered event: %v", err)
		}

		// Force flush
		if err := auditLog.flush(); err != nil {
			t.Errorf("Failed to flush: %v", err)
		}

		// Read log file
		events := readLogEvents(t, logPath)
		eventTypes := make([]EventType, len(events))
		for i, e := range events {
			eventTypes[i] = e.Type
		}

		// Should only see allowed events
		for _, eventType := range eventTypes {
			if eventType == EventCPULimit {
				t.Error("Found filtered event type in log")
			}
		}
	})

	// Test log rotation
	t.Run("log rotation", func(t *testing.T) {
		// Write some events
		for i := 0; i < 5; i++ {
			err := auditLog.Log(
				EventKeyAccess,
				SeverityInfo,
				"test",
				"test event",
				nil,
			)
			if err != nil {
				t.Errorf("Failed to log event: %v", err)
			}
		}

		// Force flush
		if err := auditLog.flush(); err != nil {
			t.Errorf("Failed to flush: %v", err)
		}

		// Rotate log
		if err := auditLog.Rotate(); err != nil {
			t.Errorf("Failed to rotate log: %v", err)
		}

		// Check that old log exists with timestamp
		files, err := os.ReadDir(tmpDir)
		if err != nil {
			t.Fatalf("Failed to read temp dir: %v", err)
		}

		var found bool
		for _, f := range files {
			if strings.HasPrefix(f.Name(), "audit.log.") {
				found = true
				break
			}
		}
		if !found {
			t.Error("No rotated log file found")
		}

		// Check that new log is empty
		events := readLogEvents(t, logPath)
		if len(events) != 0 {
			t.Errorf("Expected empty log after rotation, got %d events", len(events))
		}
	})

	// Test buffering
	t.Run("event buffering", func(t *testing.T) {
		// Write events but don't force flush
		for i := 0; i < 3; i++ {
			err := auditLog.Log(
				EventKeyAccess,
				SeverityInfo,
				"test",
				"buffered event",
				nil,
			)
			if err != nil {
				t.Errorf("Failed to log event: %v", err)
			}
		}

		// Events should be in buffer, not file
		events := readLogEvents(t, logPath)
		preFlushCount := len(events)

		// Force flush
		if err := auditLog.flush(); err != nil {
			t.Errorf("Failed to flush: %v", err)
		}

		// Check events are now in file
		events = readLogEvents(t, logPath)
		postFlushCount := len(events)

		if postFlushCount != preFlushCount+3 {
			t.Errorf("Expected %d events after flush, got %d", preFlushCount+3, postFlushCount)
		}
	})

	// Test auto-flush on buffer full
	t.Run("auto-flush", func(t *testing.T) {
		// Fill buffer
		for i := 0; i < 101; i++ { // Buffer size is 100
			err := auditLog.Log(
				EventKeyAccess,
				SeverityInfo,
				"test",
				"auto-flush test",
				nil,
			)
			if err != nil {
				t.Errorf("Failed to log event: %v", err)
			}
		}

		// Events should be automatically flushed
		events := readLogEvents(t, logPath)
		found := false
		for _, e := range events {
			if e.Details == "auto-flush test" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Auto-flush did not write events to file")
		}
	})

	// Test file permissions
	t.Run("file permissions", func(t *testing.T) {
		info, err := os.Stat(logPath)
		if err != nil {
			t.Fatalf("Failed to stat log file: %v", err)
		}

		mode := info.Mode()
		if mode&0077 != 0 {
			t.Errorf("Log file has wrong permissions: %v", mode)
		}
	})
}

func TestAuditLogErrors(t *testing.T) {
	// Test invalid path
	t.Run("invalid path", func(t *testing.T) {
		cfg := &config.Config{
			Security: config.SecurityConfig{
				AuditLog: config.AuditLogConfig{
					Enabled: true,
					Path:   "/nonexistent/path/audit.log",
				},
			},
		}

		_, err := NewAuditLog(cfg)
		if err == nil {
			t.Error("Expected error for invalid path")
		}
	})

	// Test disabled audit log
	t.Run("disabled audit log", func(t *testing.T) {
		cfg := &config.Config{
			Security: config.SecurityConfig{
				AuditLog: config.AuditLogConfig{
					Enabled: false,
					Path:   "audit.log",
				},
			},
		}

		log, err := NewAuditLog(cfg)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if log != nil {
			t.Error("Expected nil audit log when disabled")
		}
	})
}

// readLogEvents reads all events from a log file
func readLogEvents(t *testing.T, path string) []AuditEvent {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("Failed to open log file: %v", err)
	}
	defer file.Close()

	var events []AuditEvent
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var event AuditEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			t.Fatalf("Failed to parse event: %v", err)
		}
		events = append(events, event)
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	return events
}
