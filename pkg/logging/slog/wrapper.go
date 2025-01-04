package slog

import (
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/butter-bot-machines/skylark/pkg/logging"
)

// LoggerWrapper wraps slog.Logger to implement logging.Logger
type LoggerWrapper struct {
	*slog.Logger
	level  logging.Level
	output io.Writer
}

// NewLogger creates a new logger with the given level and output
func NewLogger(level logging.Level, output io.Writer) logging.Logger {
	if output == nil {
		output = os.Stdout
	}

	opts := &slog.HandlerOptions{
		Level: levelToSlog(level),
	}
	handler := slog.NewJSONHandler(output, opts).WithAttrs([]slog.Attr{
		slog.String("level", strings.ToLower(level.String())),
	})
	logger := slog.New(handler)

	return &LoggerWrapper{
		Logger: logger,
		level:  level,
		output: output,
	}
}

// NewLoggerWrapper creates a new wrapped slog logger
func NewLoggerWrapper(logger *slog.Logger, level logging.Level, output io.Writer) *LoggerWrapper {
	return &LoggerWrapper{
		Logger: logger,
		level:  level,
		output: output,
	}
}

// GetLevel returns the current log level
func (l *LoggerWrapper) GetLevel() logging.Level {
	return l.level
}

// SetLevel sets the log level
func (l *LoggerWrapper) SetLevel(level logging.Level) {
	l.level = level
	opts := &slog.HandlerOptions{
		Level: levelToSlog(l.level),
	}
	handler := slog.NewJSONHandler(l.output, opts).WithAttrs([]slog.Attr{
		slog.String("level", strings.ToLower(level.String())),
	})
	l.Logger = slog.New(handler)
}

// GetOutput returns the current output writer
func (l *LoggerWrapper) GetOutput() io.Writer {
	return l.output
}

// SetOutput sets the output writer
func (l *LoggerWrapper) SetOutput(w io.Writer) {
	l.output = w
	opts := &slog.HandlerOptions{
		Level: levelToSlog(l.level),
	}
	handler := slog.NewJSONHandler(w, opts).WithAttrs([]slog.Attr{
		slog.String("level", strings.ToLower(l.level.String())),
	})
	l.Logger = slog.New(handler)
}

// With returns a new logger with the given attributes
func (l *LoggerWrapper) With(args ...interface{}) logging.Logger {
	if len(args)%2 != 0 {
		args = append(args, "MISSING_VALUE")
	}

	// Convert args to slog.Attr
	attrs := make([]any, 0, len(args)/2)
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			attrs = append(attrs, slog.Any(args[i].(string), args[i+1]))
		}
	}

	return &LoggerWrapper{
		Logger: l.Logger.With(attrs...),
		level:  l.level,
		output: l.output,
	}
}

// WithGroup returns a new logger with the given group
func (l *LoggerWrapper) WithGroup(name string) logging.Logger {
	return &LoggerWrapper{
		Logger: l.Logger.WithGroup(name),
		level:  l.level,
		output: l.output,
	}
}

// Debug logs a debug message
func (l *LoggerWrapper) Debug(msg string, args ...interface{}) {
	if logging.LevelDebug >= l.level {
		l.log(slog.LevelDebug, msg, args...)
	}
}

// Info logs an info message
func (l *LoggerWrapper) Info(msg string, args ...interface{}) {
	if logging.LevelInfo >= l.level {
		l.log(slog.LevelInfo, msg, args...)
	}
}

// Warn logs a warning message
func (l *LoggerWrapper) Warn(msg string, args ...interface{}) {
	if logging.LevelWarn >= l.level {
		l.log(slog.LevelWarn, msg, args...)
	}
}

// Error logs an error message
func (l *LoggerWrapper) Error(msg string, args ...interface{}) {
	if logging.LevelError >= l.level {
		l.log(slog.LevelError, msg, args...)
	}
}

// log handles the actual logging
func (l *LoggerWrapper) log(level slog.Level, msg string, args ...interface{}) {
	// Convert args to slog.Attr
	attrs := make([]slog.Attr, 0, len(args)/2)
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			key := args[i].(string)
			value := args[i+1]
			attrs = append(attrs, slog.Any(key, value))
		}
	}

	// Log with attributes
	l.Logger.LogAttrs(nil, level, msg, attrs...)
}
