package errors

import (
	"fmt"
	"io"
)

// Registry manages error types and their creation
type Registry interface {
	// Register adds a new error type
	Register(name string, code int) ErrorType

	// Get returns an error type by name
	Get(name string) (ErrorType, bool)

	// List returns all registered error types
	List() []ErrorType
}

// ErrorType represents a category of error
type ErrorType interface {
	// Name returns the error type name
	Name() string

	// Code returns the error type code
	Code() int

	// New creates a new error of this type
	New(msg string, args ...interface{}) Error

	// Wrap wraps an existing error
	Wrap(err error, msg string, args ...interface{}) Error
}

// StackProvider captures and manages stack traces
type StackProvider interface {
	// Capture captures the current stack trace
	Capture(skip int) StackTrace

	// Format formats a stack trace
	Format(st StackTrace) string
}

// StackTrace represents a captured stack trace
type StackTrace interface {
	// Frames returns the stack frames
	Frames() []Frame

	// String returns a formatted stack trace
	String() string
}

// Frame represents a stack frame
type Frame interface {
	// File returns the file name
	File() string

	// Line returns the line number
	Line() int

	// Function returns the function name
	Function() string

	// String returns a formatted frame
	String() string
}

// Formatter formats errors with optional context
type Formatter interface {
	// Format formats an error with optional stack trace
	Format(err error, includeStack bool) string

	// FormatWithContext formats an error with context
	FormatWithContext(err error, ctx map[string]interface{}) string
}

// PanicHandler handles panic recovery
type PanicHandler interface {
	// Handle handles a panic
	Handle(v interface{}) error

	// Recover returns a function that recovers from panics
	Recover() func() error
}

// Error represents an error with context and stack trace
type Error interface {
	error
	fmt.Formatter

	// WithContext adds context to the error
	WithContext(key string, value interface{}) Error

	// WithType sets the error type
	WithType(errType ErrorType) Error

	// IsTemporary indicates if the error is temporary
	IsTemporary() bool

	// IsTimeout indicates if the error is a timeout
	IsTimeout() bool

	// Stack returns the error's stack trace
	Stack() StackTrace

	// Context returns the error's context
	Context() map[string]interface{}

	// Cause returns the underlying cause
	Cause() error
}

// Aggregate represents multiple errors as one
type Aggregate interface {
	error
	io.Writer

	// Add adds an error to the aggregate
	Add(err error)

	// HasErrors returns true if there are any errors
	HasErrors() bool

	// Errors returns the slice of errors
	Errors() []error
}
