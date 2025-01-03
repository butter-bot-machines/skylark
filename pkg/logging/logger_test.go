package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestLogLevels(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(&Options{
		Level:  slog.LevelInfo,
		Output: buf,
	})

	// Debug should not be logged
	logger.Debug("debug message")
	if buf.Len() > 0 {
		t.Error("Debug message was logged when level is Info")
	}

	// Info should be logged
	buf.Reset()
	logger.Info("info message")
	if !strings.Contains(buf.String(), "info message") {
		t.Error("Info message was not logged correctly")
	}
}

func TestStructuredLogging(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(&Options{
		Level:  slog.LevelDebug,
		Output: buf,
	})

	// Test with attributes
	logger.With("key", "value").Info("test message")
	if !strings.Contains(buf.String(), "key") || !strings.Contains(buf.String(), "value") {
		t.Error("Attributes were not logged correctly")
	}

	// Test with groups
	buf.Reset()
	logger.WithGroup("group1").With(
		"key1", "value1",
		"key2", 42,
	).Info("test message")

	output := buf.String()
	if !strings.Contains(output, "group1") ||
		!strings.Contains(output, "key1") ||
		!strings.Contains(output, "value1") ||
		!strings.Contains(output, "key2") {
		t.Error("Group attributes were not logged correctly")
	}
}

func TestJSONFormatting(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(&Options{
		Level:  slog.LevelInfo,
		Output: buf,
		JSON:   true,
	})

	logger.With("key", "value").Info("test message")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Errorf("Failed to parse JSON: %v", err)
	}

	if entry["msg"] != "test message" {
		t.Error("Message not correctly encoded in JSON")
	}

	if entry["key"] != "value" {
		t.Error("Attributes not correctly encoded in JSON")
	}
}

func TestContextLogging(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(&Options{
		Level:  slog.LevelInfo,
		Output: buf,
	})

	ctx := context.Background()
	ctx = context.WithValue(ctx, "request_id", "123")

	logger.LogAttrs(ctx, slog.LevelInfo, "test message",
		slog.String("context_key", "context_value"),
	)

	output := buf.String()
	if !strings.Contains(output, "context_key") ||
		!strings.Contains(output, "context_value") {
		t.Error("Context attributes not logged correctly")
	}
}

func TestLoggerChaining(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(&Options{
		Level:  slog.LevelInfo,
		Output: buf,
	})

	// Chain multiple operations
	logger.With(
		"field1", "value1",
		"field2", "value2",
	).WithGroup("group").With(
		"field3", "value3",
	).Info("test message")

	output := buf.String()
	expected := []string{"field1", "value1", "field2", "value2", "group", "field3", "value3"}
	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("Expected %s in output", exp)
		}
	}
}

func TestSourceLocation(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(&Options{
		Level:     slog.LevelInfo,
		Output:    buf,
		AddSource: true,
	})

	logger.Info("test message")
	output := buf.String()
	t.Logf("Log output: %s", output)

	if !strings.Contains(output, "source=") {
		t.Error("Source location not included in log")
	}

	// Verify shortened source path
	if strings.Contains(output, "/") {
		t.Error("Source path is not shortened")
	}
}

func TestHelperFunctions(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(&Options{
		Level:  slog.LevelInfo,
		Output: buf,
	})

	// Test WithAttrs
	logger = WithAttrs(logger,
		"service", "test",
		"version", "1.0",
	)
	logger.Info("test message")
	output := buf.String()
	if !strings.Contains(output, "service=test") ||
		!strings.Contains(output, "version=1.0") {
		t.Error("WithAttrs did not add attributes correctly")
	}

	// Test WithGroup
	buf.Reset()
	logger = WithGroup(logger, "request")
	logger.Info("test message",
		"id", "123",
		"method", "GET",
	)
	output = buf.String()
	if !strings.Contains(output, "request.id=123") ||
		!strings.Contains(output, "request.method=GET") {
		t.Error("WithGroup did not group attributes correctly")
	}
}

func TestShortenedSourcePathsInTextFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(&Options{
		Level:     slog.LevelInfo,
		Output:    buf,
		AddSource: true,
	})

	logger.Info("test message")
	output := buf.String()
	t.Logf("Log output: %s", output)

	if !strings.Contains(output, "source=") {
		t.Error("Source location not included in log")
	}

	// Verify shortened source path
	if strings.Contains(output, "/") {
		t.Error("Source path is not shortened")
	}
}

func TestShortenedSourcePathsInJSONFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(&Options{
		Level:     slog.LevelInfo,
		Output:    buf,
		AddSource: true,
		JSON:      true,
	})

	logger.Info("test message")
	output := buf.String()
	t.Logf("Log output: %s", output)

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Errorf("Failed to parse JSON: %v", err)
	}

	if entry["source"] == nil {
		t.Error("Source location not included in log")
	}

	// Verify shortened source path
	source := entry["source"].(map[string]interface{})
	if strings.Contains(source["file"].(string), "/") {
		t.Error("Source path is not shortened")
	}
}
