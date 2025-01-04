package testutil

import (
	"encoding/json"
	"testing"
)

// LogEntry represents a parsed JSON log entry
type LogEntry struct {
	Time    string                 `json:"time"`
	Level   string                 `json:"level"`
	Message string                 `json:"msg"`
	Attrs   map[string]interface{} `json:"-"`
}

// ParseLogEntry parses a JSON log entry from a string
func ParseLogEntry(t *testing.T, line string) LogEntry {
	t.Helper()

	var entry LogEntry
	var raw map[string]interface{}

	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		t.Fatalf("Failed to parse log entry: %v", err)
	}

	entry.Time = raw["time"].(string)
	entry.Level = raw["level"].(string)
	entry.Message = raw["msg"].(string)

	// Extract attributes (everything except time, level, msg)
	entry.Attrs = make(map[string]interface{})
	for k, v := range raw {
		if k != "time" && k != "level" && k != "msg" {
			entry.Attrs[k] = v
		}
	}

	return entry
}
