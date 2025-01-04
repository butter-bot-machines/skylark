package slog

import (
	"bytes"
	"strings"
	"testing"

	"github.com/butter-bot-machines/skylark/pkg/logging"
	"github.com/butter-bot-machines/skylark/pkg/logging/slog/internal/testutil"
)

func TestLoggerWrapper_Levels(t *testing.T) {
	buf := new(bytes.Buffer)
	logger := NewLogger(logging.LevelInfo, buf)

	tests := []struct {
		name    string
		level   logging.Level
		logFunc func(string, ...interface{})
		want    bool // whether the message should be logged
	}{
		{
			name:    "Debug below Info",
			level:   logging.LevelDebug,
			logFunc: logger.Debug,
			want:    false,
		},
		{
			name:    "Info at Info",
			level:   logging.LevelInfo,
			logFunc: logger.Info,
			want:    true,
		},
		{
			name:    "Warn above Info",
			level:   logging.LevelWarn,
			logFunc: logger.Warn,
			want:    true,
		},
		{
			name:    "Error above Info",
			level:   logging.LevelError,
			logFunc: logger.Error,
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc("test message")

			if got := buf.Len() > 0; got != tt.want {
				t.Errorf("Message logged = %v, want %v", got, tt.want)
			}

			if tt.want && buf.Len() > 0 {
				entry := testutil.ParseLogEntry(t, buf.String())
				if entry.Message != "test message" {
					t.Errorf("Message = %v, want 'test message'", entry.Message)
				}
			}
		})
	}
}

func TestLoggerWrapper_Attributes(t *testing.T) {
	buf := new(bytes.Buffer)
	logger := NewLogger(logging.LevelInfo, buf)

	// Test With
	t.Run("With Attributes", func(t *testing.T) {
		logger := logger.With("key1", "value1", "key2", 42)
		buf.Reset()

		logger.Info("test message")
		entry := testutil.ParseLogEntry(t, buf.String())

		if entry.Attrs["key1"] != "value1" {
			t.Errorf("Attribute key1 = %v, want 'value1'", entry.Attrs["key1"])
		}
		if entry.Attrs["key2"] != float64(42) { // JSON numbers are float64
			t.Errorf("Attribute key2 = %v, want 42", entry.Attrs["key2"])
		}
	})

	// Test odd number of attributes
	t.Run("Odd Attributes", func(t *testing.T) {
		logger := logger.With("key1", "value1", "key2")
		buf.Reset()

		logger.Info("test message")
		entry := testutil.ParseLogEntry(t, buf.String())

		if entry.Attrs["key2"] != "MISSING_VALUE" {
			t.Errorf("Missing value = %v, want 'MISSING_VALUE'", entry.Attrs["key2"])
		}
	})
}

func TestLoggerWrapper_Groups(t *testing.T) {
	buf := new(bytes.Buffer)
	logger := NewLogger(logging.LevelInfo, buf)

	// Test WithGroup
	t.Run("With Group", func(t *testing.T) {
		logger := logger.WithGroup("group1").WithGroup("group2")
		buf.Reset()

		logger.Info("test message", "key", "value")

		if !strings.Contains(buf.String(), "group1") || !strings.Contains(buf.String(), "group2") {
			t.Errorf("Groups not found in output: %s", buf.String())
		}
	})
}

func TestLoggerWrapper_Output(t *testing.T) {
	buf1 := new(bytes.Buffer)
	logger := NewLogger(logging.LevelInfo, buf1)

	// Test initial output
	t.Run("Initial Output", func(t *testing.T) {
		logger.Info("test message")
		if buf1.Len() == 0 {
			t.Error("Expected output in buffer1")
		}
	})

	// Test changing output
	t.Run("Change Output", func(t *testing.T) {
		buf2 := new(bytes.Buffer)
		logger.SetOutput(buf2)
		buf1.Reset()

		logger.Info("test message")
		if buf1.Len() > 0 {
			t.Error("Expected no output in buffer1")
		}
		if buf2.Len() == 0 {
			t.Error("Expected output in buffer2")
		}
	})
}

func TestLoggerWrapper_LevelControl(t *testing.T) {
	buf := new(bytes.Buffer)
	logger := NewLogger(logging.LevelInfo, buf)

	// Test changing level
	t.Run("Change Level", func(t *testing.T) {
		logger.SetLevel(logging.LevelError)
		buf.Reset()

		logger.Info("info message")
		if buf.Len() > 0 {
			t.Error("Info message should not be logged at Error level")
		}

		logger.Error("error message")
		if buf.Len() == 0 {
			t.Error("Error message should be logged at Error level")
		}
	})

	// Test getting level
	t.Run("Get Level", func(t *testing.T) {
		if got := logger.GetLevel(); got != logging.LevelError {
			t.Errorf("GetLevel() = %v, want Error", got)
		}
	})
}

func TestLoggerWrapper_Initialization(t *testing.T) {
	// Test nil output defaults to stdout
	t.Run("Nil Output", func(t *testing.T) {
		logger := NewLogger(logging.LevelInfo, nil)
		if logger.GetOutput() == nil {
			t.Error("Output should default to stdout")
		}
	})

	// Test level conversion
	t.Run("Level Conversion", func(t *testing.T) {
		tests := []struct {
			level logging.Level
			want  string
		}{
			{logging.LevelDebug, "debug"},
			{logging.LevelInfo, "info"},
			{logging.LevelWarn, "warn"},
			{logging.LevelError, "error"},
		}

		for _, tt := range tests {
			t.Run(tt.level.String(), func(t *testing.T) {
				buf := new(bytes.Buffer)
				logger := NewLogger(tt.level, buf)
				logger.Info("test message")

				if buf.Len() > 0 {
					entry := testutil.ParseLogEntry(t, buf.String())
					if entry.Level != strings.ToLower(tt.want) {
						t.Errorf("Level = %v, want %v", entry.Level, strings.ToLower(tt.want))
					}
				}
			})
		}
	})
}
