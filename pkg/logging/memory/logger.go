package memory

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/butter-bot-machines/skylark/pkg/logging"
)

// Logger implements logging.Logger with in-memory storage
type Logger struct {
	mu       sync.RWMutex
	level    logging.Level
	output   io.Writer
	entries  *[]LogEntry // Pointer to share entries between loggers
	attrs    []interface{}
	groups   []string
}

// LogEntry represents a stored log entry
type LogEntry struct {
	Time    time.Time
	Level   logging.Level
	Message string
	Args    []interface{}
	Attrs   []interface{}
	Groups  []string
}

// NewLogger creates a new memory logger
func NewLogger(level logging.Level, output io.Writer) *Logger {
	entries := make([]LogEntry, 0)
	return &Logger{
		level:   level,
		output:  output,
		entries: &entries,
		attrs:   make([]interface{}, 0),
		groups:  make([]string, 0),
	}
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, args ...interface{}) {
	l.log(logging.LevelDebug, msg, args...)
}

// Info logs an info message
func (l *Logger) Info(msg string, args ...interface{}) {
	l.log(logging.LevelInfo, msg, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, args ...interface{}) {
	l.log(logging.LevelWarn, msg, args...)
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...interface{}) {
	l.log(logging.LevelError, msg, args...)
}

// With returns a new logger with additional attributes
func (l *Logger) With(args ...interface{}) logging.Logger {
	if len(args)%2 != 0 {
		args = append(args, "MISSING_VALUE")
	}

	newLogger := &Logger{
		level:   l.level,
		output:  l.output,
		entries: l.entries, // Share entries pointer
		groups:  append([]string{}, l.groups...),
	}

	newLogger.attrs = make([]interface{}, len(l.attrs)+len(args))
	copy(newLogger.attrs, l.attrs)
	copy(newLogger.attrs[len(l.attrs):], args)

	return newLogger
}

// WithGroup returns a new logger with an additional group
func (l *Logger) WithGroup(name string) logging.Logger {
	return &Logger{
		level:   l.level,
		output:  l.output,
		entries: l.entries, // Share entries pointer
		attrs:   append([]interface{}{}, l.attrs...),
		groups:  append(append([]string{}, l.groups...), name),
	}
}

// SetLevel sets the minimum log level
func (l *Logger) SetLevel(level logging.Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// GetLevel returns the current log level
func (l *Logger) GetLevel() logging.Level {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

// SetOutput sets the output writer
func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = w
}

// GetOutput returns the current output writer
func (l *Logger) GetOutput() io.Writer {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.output
}

// GetEntries returns all stored log entries
func (l *Logger) GetEntries() []LogEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	// Return a copy to prevent external modification
	entries := make([]LogEntry, len(*l.entries))
	copy(entries, *l.entries)
	return entries
}

// log handles the actual logging
func (l *Logger) log(level logging.Level, msg string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if level < l.level {
		return
	}

	entry := LogEntry{
		Time:    time.Now(),
		Level:   level,
		Message: msg,
		Args:    args,
		Attrs:   append([]interface{}{}, l.attrs...),
		Groups:  append([]string{}, l.groups...),
	}

	*l.entries = append(*l.entries, entry)

	if l.output != nil {
		// Format: TIME [LEVEL] [GROUP1][GROUP2]... MESSAGE key1=value1 key2=value2 ...
		groups := ""
		for _, g := range l.groups {
			groups += fmt.Sprintf("[%s]", g)
		}

		attrs := ""
		for i := 0; i < len(l.attrs); i += 2 {
			if i+1 < len(l.attrs) {
				attrs += fmt.Sprintf(" %v=%v", l.attrs[i], l.attrs[i+1])
			}
		}

		for i := 0; i < len(args); i += 2 {
			if i+1 < len(args) {
				attrs += fmt.Sprintf(" %v=%v", args[i], args[i+1])
			}
		}

		fmt.Fprintf(l.output, "%s [%s] %s%s%s\n",
			entry.Time.Format("2006-01-02T15:04:05.000"),
			level.String(),
			groups,
			msg,
			attrs,
		)
	}
}
