package memory

import (
	"bytes"
	"strings"
	"sync"
	"testing"

	"github.com/butter-bot-machines/skylark/pkg/logging"
)

func TestLogger_BasicOperations(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(logging.LevelDebug, &buf)

	// Test all log levels
	t.Run("Log Levels", func(t *testing.T) {
		logger.Debug("debug message", "key", "value")
		logger.Info("info message", "key", "value")
		logger.Warn("warn message", "key", "value")
		logger.Error("error message", "key", "value")

		entries := logger.GetEntries()
		if len(entries) != 4 {
			t.Errorf("Got %d entries, want 4", len(entries))
		}

		// Verify levels
		levels := []logging.Level{
			entries[0].Level,
			entries[1].Level,
			entries[2].Level,
			entries[3].Level,
		}
		expected := []logging.Level{
			logging.LevelDebug,
			logging.LevelInfo,
			logging.LevelWarn,
			logging.LevelError,
		}
		for i, level := range levels {
			if level != expected[i] {
				t.Errorf("Entry %d: got level %v, want %v", i, level, expected[i])
			}
		}

		// Verify output format
		output := buf.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")
		if len(lines) != 4 {
			t.Errorf("Got %d output lines, want 4", len(lines))
		}
		for _, line := range lines {
			if !strings.Contains(line, "key=value") {
				t.Errorf("Line missing attributes: %s", line)
			}
		}
	})
}

func TestLogger_LevelFiltering(t *testing.T) {
	logger := NewLogger(logging.LevelWarn, nil)

	// Messages below Warn should be filtered
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	entries := logger.GetEntries()
	if len(entries) != 2 {
		t.Errorf("Got %d entries, want 2", len(entries))
	}

	if entries[0].Level != logging.LevelWarn {
		t.Errorf("Got level %v, want WARN", entries[0].Level)
	}
	if entries[1].Level != logging.LevelError {
		t.Errorf("Got level %v, want ERROR", entries[1].Level)
	}
}

func TestLogger_ContextAndGroups(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(logging.LevelDebug, &buf)

	// Test With
	t.Run("With Context", func(t *testing.T) {
		contextLogger := logger.With("user", "test", "request", "123")
		contextLogger.Info("test message")

		entries := logger.GetEntries()
		if len(entries) != 1 {
			t.Errorf("Got %d entries, want 1", len(entries))
		}

		if len(entries[0].Attrs) != 4 {
			t.Errorf("Got %d attributes, want 4", len(entries[0].Attrs))
		}

		output := buf.String()
		if !strings.Contains(output, "user=test") || !strings.Contains(output, "request=123") {
			t.Errorf("Missing context in output: %s", output)
		}
	})

	// Test WithGroup
	t.Run("With Group", func(t *testing.T) {
		buf.Reset()
		groupLogger := logger.WithGroup("test-group").WithGroup("sub-group")
		groupLogger.Info("test message")

		entries := logger.GetEntries()
		if len(entries[1].Groups) != 2 {
			t.Errorf("Got %d groups, want 2", len(entries[1].Groups))
		}

		output := buf.String()
		if !strings.Contains(output, "[test-group][sub-group]") {
			t.Errorf("Missing groups in output: %s", output)
		}
	})

	// Test With and WithGroup together
	t.Run("With Context and Group", func(t *testing.T) {
		buf.Reset()
		contextAndGroupLogger := logger.
			With("user", "test").
			WithGroup("auth").
			With("role", "admin")

		contextAndGroupLogger.Info("test message")

		output := buf.String()
		if !strings.Contains(output, "[auth]") ||
			!strings.Contains(output, "user=test") ||
			!strings.Contains(output, "role=admin") {
			t.Errorf("Missing context or group in output: %s", output)
		}
	})
}

func TestLogger_Concurrency(t *testing.T) {
	logger := NewLogger(logging.LevelDebug, nil)
	var wg sync.WaitGroup
	workers := 10
	iterations := 100

	// Concurrent logging
	t.Run("Concurrent Logging", func(t *testing.T) {
		wg.Add(workers)
		for i := 0; i < workers; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					logger.Info("test message", "worker", id, "iteration", j)
				}
			}(i)
		}

		wg.Wait()
		entries := logger.GetEntries()
		expected := workers * iterations
		if len(entries) != expected {
			t.Errorf("Got %d entries, want %d", len(entries), expected)
		}

		// Verify entries are in chronological order
		for i := 1; i < len(entries); i++ {
			if entries[i].Time.Before(entries[i-1].Time) {
				t.Errorf("Entries not in chronological order at index %d", i)
			}
		}
	})

	// Concurrent level changes
	t.Run("Concurrent Level Changes", func(t *testing.T) {
		wg.Add(workers)
		for i := 0; i < workers; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					level := logging.Level(j % 4)
					logger.SetLevel(level)
					if got := logger.GetLevel(); got < logging.LevelDebug || got > logging.LevelError {
						t.Errorf("Invalid level: %v", got)
					}
				}
			}()
		}
		wg.Wait()
	})
}

func TestLogger_ErrorCases(t *testing.T) {
	logger := NewLogger(logging.LevelDebug, nil)

	// Test odd number of arguments
	t.Run("Odd Arguments", func(t *testing.T) {
		contextLogger := logger.With("key1", "value1", "key2")
		contextLogger.Info("test message")

		entries := logger.GetEntries()
		if len(entries) != 1 {
			t.Errorf("Got %d entries, want 1", len(entries))
		}

		// Should have added MISSING_VALUE
		attrs := entries[0].Attrs
		if len(attrs) != 4 {
			t.Errorf("Got %d attributes, want 4", len(attrs))
		}
		if attrs[3] != "MISSING_VALUE" {
			t.Errorf("Got %v, want MISSING_VALUE", attrs[3])
		}
	})

	// Test nil output writer
	t.Run("Nil Output", func(t *testing.T) {
		logger := NewLogger(logging.LevelDebug, nil)
		logger.Info("test message") // Should not panic
	})
}
