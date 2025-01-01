package errors

import (
	"fmt"
	"runtime"
	"strings"
)

// ErrorType represents the category of error
type ErrorType int

const (
	// Configuration errors
	ConfigError ErrorType = iota
	// Tool execution errors
	ToolError
	// Resource access errors
	ResourceError
	// Network errors
	NetworkError
	// System errors
	SystemError
	// Unknown errors
	UnknownError
)

// Error represents a categorized error with context
type Error struct {
	Type      ErrorType
	Message   string
	Cause     error
	Stack     []Frame
	Context   map[string]any
	Temporary bool
	Timeout   bool
}

// Frame represents a stack frame
type Frame struct {
	File     string
	Line     int
	Function string
}

// New creates a new error with type and message
func New(errType ErrorType, message string) *Error {
	return &Error{
		Type:    errType,
		Message: message,
		Stack:   captureStack(2),
		Context: make(map[string]any),
	}
}

// Wrap wraps an existing error with additional context
func Wrap(err error, message string) *Error {
	if err == nil {
		return nil
	}

	// If already wrapped, add context
	if e, ok := err.(*Error); ok {
		e.Message = fmt.Sprintf("%s: %s", message, e.Message)
		return e
	}

	// Create new wrapped error
	return &Error{
		Type:    UnknownError,
		Message: fmt.Sprintf("%s: %v", message, err),
		Cause:   err,
		Stack:   captureStack(1),
		Context: make(map[string]any),
	}
}

// WithContext adds context to the error
func (e *Error) WithContext(key string, value any) *Error {
	if e == nil {
		return nil
	}
	e.Context[key] = value
	return e
}

// WithType sets the error type
func (e *Error) WithType(errType ErrorType) *Error {
	if e == nil {
		return nil
	}
	e.Type = errType
	return e
}

// IsTemporary indicates if the error is temporary
func (e *Error) IsTemporary() bool {
	if e == nil {
		return false
	}
	return e.Temporary
}

// IsTimeout indicates if the error is a timeout
func (e *Error) IsTimeout() bool {
	if e == nil {
		return false
	}
	return e.Timeout
}

// Error implements the error interface
func (e *Error) Error() string {
	if e == nil {
		return ""
	}

	var b strings.Builder
	b.WriteString(e.Message)

	if len(e.Context) > 0 {
		b.WriteString(" [")
		first := true
		for k, v := range e.Context {
			if !first {
				b.WriteString(", ")
			}
			fmt.Fprintf(&b, "%s=%v", k, v)
			first = false
		}
		b.WriteString("]")
	}

	return b.String()
}

// Format implements fmt.Formatter for detailed error output
func (e *Error) Format(f fmt.State, c rune) {
	if e == nil {
		return
	}

	switch c {
	case 'v':
		if f.Flag('+') {
			// Detailed format with stack trace
			fmt.Fprintf(f, "%s\n", e.Error())
			if e.Cause != nil {
				fmt.Fprintf(f, "Caused by: %+v\n", e.Cause)
			}
			fmt.Fprintf(f, "Stack trace:\n")
			for _, frame := range e.Stack {
				fmt.Fprintf(f, "  %s:%d %s\n", frame.File, frame.Line, frame.Function)
			}
		} else {
			fmt.Fprintf(f, "%s", e.Error())
		}
	default:
		fmt.Fprintf(f, "%s", e.Error())
	}
}

// captureStack captures the current stack trace
func captureStack(skip int) []Frame {
	var frames []Frame
	for i := skip; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			break
		}

		// Get short file name
		shortFile := file
		if idx := strings.LastIndex(file, "/"); idx >= 0 {
			shortFile = file[idx+1:]
		}

		frames = append(frames, Frame{
			File:     shortFile,
			Line:     line,
			Function: fn.Name(),
		})

		// Limit stack depth
		if len(frames) >= 32 {
			break
		}
	}
	return frames
}

// Recover returns a function that can be used with defer to recover from panics
func Recover(errType ErrorType) func() error {
	return func() error {
		r := recover()
		if r == nil {
			return nil
		}

		var msg string
		switch v := r.(type) {
		case string:
			msg = v
		case error:
			msg = v.Error()
		default:
			msg = fmt.Sprintf("%v", v)
		}

		return New(errType, fmt.Sprintf("panic recovered: %s", msg)).
			WithContext("recovered", true)
	}
}

// Aggregate combines multiple errors into a single error
type Aggregate struct {
	errors []error
}

// NewAggregate creates a new error aggregate
func NewAggregate() *Aggregate {
	return &Aggregate{
		errors: make([]error, 0),
	}
}

// Add adds an error to the aggregate
func (a *Aggregate) Add(err error) {
	if err != nil {
		a.errors = append(a.errors, err)
	}
}

// HasErrors returns true if there are any errors
func (a *Aggregate) HasErrors() bool {
	return len(a.errors) > 0
}

// Error implements the error interface
func (a *Aggregate) Error() string {
	if !a.HasErrors() {
		return ""
	}

	if len(a.errors) == 1 {
		return a.errors[0].Error()
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%d errors occurred:\n", len(a.errors))
	for i, err := range a.errors {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "[%d] %v", i+1, err)
	}
	return b.String()
}

// Errors returns the slice of errors
func (a *Aggregate) Errors() []error {
	return a.errors
}
