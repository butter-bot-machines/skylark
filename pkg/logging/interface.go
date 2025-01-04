package logging

import "io"

// Level represents a logging level
type Level int

const (
	// Log levels in order of increasing severity
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// String returns the string representation of a log level
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger defines the interface for logging operations
type Logger interface {
	// Basic logging
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})

	// Context operations
	With(args ...interface{}) Logger
	WithGroup(name string) Logger

	// Level operations
	SetLevel(level Level)
	GetLevel() Level

	// Output operations
	SetOutput(w io.Writer)
	GetOutput() io.Writer
}

// Entry represents a log entry
type Entry struct {
	Level   Level
	Message string
	Args    []interface{}
}

// Error types for logging operations
var (
	ErrInvalidLevel  = Error{"invalid log level"}
	ErrInvalidOutput = Error{"invalid output writer"}
	ErrInvalidArgs   = Error{"invalid arguments"}
)

// Error represents a logging error
type Error struct {
	Message string
}

func (e Error) Error() string {
	return e.Message
}
