package errors

import (
	"fmt"
	"strings"
	"testing"
)

func TestErrorCreation(t *testing.T) {
	// Test basic error creation
	err := New(ConfigError, "configuration error")
	if err.Type != ConfigError {
		t.Errorf("Error type = %v, want %v", err.Type, ConfigError)
	}
	if err.Message != "configuration error" {
		t.Errorf("Error message = %v, want %v", err.Message, "configuration error")
	}
	if len(err.Stack) == 0 {
		t.Error("Stack trace not captured")
	}

	// Test error wrapping
	cause := fmt.Errorf("original error")
	wrapped := Wrap(cause, "wrapped error")
	if !strings.Contains(wrapped.Error(), "wrapped error") {
		t.Error("Wrapped error missing wrapper message")
	}
	if !strings.Contains(wrapped.Error(), "original error") {
		t.Error("Wrapped error missing original message")
	}

	// Test nil handling
	if Wrap(nil, "wrapper") != nil {
		t.Error("Wrapping nil error should return nil")
	}
}

func TestErrorContext(t *testing.T) {
	err := New(ToolError, "tool error").
		WithContext("tool", "example").
		WithContext("status", 500)

	// Test context values
	if err.Context["tool"] != "example" {
		t.Error("Context value not set correctly")
	}
	if err.Context["status"] != 500 {
		t.Error("Context value not set correctly")
	}

	// Test error string contains context
	errStr := err.Error()
	if !strings.Contains(errStr, "tool=example") {
		t.Error("Error string missing context")
	}
	if !strings.Contains(errStr, "status=500") {
		t.Error("Error string missing context")
	}

	// Test chaining
	err = err.WithType(NetworkError)
	if err.Type != NetworkError {
		t.Error("Error type not updated")
	}
}

func TestStackTrace(t *testing.T) {
	err := New(SystemError, "system error")

	// Verify stack frames
	if len(err.Stack) == 0 {
		t.Fatal("No stack frames captured")
	}

	// Check first frame
	frame := err.Stack[0]
	if frame.File != "errors_test.go" {
		t.Errorf("File = %v, want errors_test.go", frame.File)
	}
	if !strings.Contains(frame.Function, "github.com/butter-bot-machines/skylark/pkg/errors.TestStackTrace") {
		t.Errorf("Function = %v, want to contain TestStackTrace", frame.Function)
	}
	if frame.Line == 0 {
		t.Error("Stack frame missing line number")
	}
}

func TestErrorFormatting(t *testing.T) {
	err := New(ResourceError, "resource error").
		WithContext("resource", "database")

	// Test simple format
	simple := fmt.Sprintf("%s", err)
	if !strings.Contains(simple, "resource error") {
		t.Error("Simple format missing error message")
	}

	// Test verbose format
	verbose := fmt.Sprintf("%+v", err)
	if !strings.Contains(verbose, "Stack trace:") {
		t.Error("Verbose format missing stack trace")
	}
	if !strings.Contains(verbose, "errors_test.go") {
		t.Error("Verbose format missing file info")
	}
}

func TestPanicRecovery(t *testing.T) {
	// Test panic with string
	err := func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = New(SystemError, fmt.Sprintf("panic recovered: %v", r))
			}
		}()
		panic("test panic")
	}()
	if err == nil {
		t.Error("Expected error from recovered panic")
	}
	if !strings.Contains(err.Error(), "test panic") {
		t.Error("Error message does not contain panic message")
	}

	// Test panic with error
	err = func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = New(SystemError, fmt.Sprintf("panic recovered: %v", r))
			}
		}()
		panic(fmt.Errorf("error panic"))
	}()
	if err == nil {
		t.Error("Expected error from recovered panic")
	}
	if !strings.Contains(err.Error(), "error panic") {
		t.Error("Error message does not contain panic message")
	}

	// Test no panic
	err = func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = New(SystemError, fmt.Sprintf("panic recovered: %v", r))
			}
		}()
		return nil
	}()
	if err != nil {
		t.Error("Expected nil error when no panic")
	}
}

func TestErrorAggregation(t *testing.T) {
	agg := NewAggregate()

	// Test empty aggregate
	if agg.HasErrors() {
		t.Error("New aggregate should have no errors")
	}
	if agg.Error() != "" {
		t.Error("Empty aggregate should return empty string")
	}

	// Add single error
	err1 := New(ConfigError, "error one")
	agg.Add(err1)
	if !agg.HasErrors() {
		t.Error("Aggregate should have errors")
	}
	if !strings.Contains(agg.Error(), "error one") {
		t.Error("Aggregate string missing error")
	}

	// Add multiple errors
	err2 := New(ToolError, "error two")
	agg.Add(err2)
	errStr := agg.Error()
	if !strings.Contains(errStr, "2 errors occurred") {
		t.Error("Multiple error message incorrect")
	}
	if !strings.Contains(errStr, "[1]") || !strings.Contains(errStr, "[2]") {
		t.Error("Error numbering incorrect")
	}

	// Check error slice
	errors := agg.Errors()
	if len(errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(errors))
	}
}

func TestErrorBehavior(t *testing.T) {
	// Test temporary error
	tempErr := New(NetworkError, "network error")
	tempErr.Temporary = true
	if !tempErr.IsTemporary() {
		t.Error("Error should be temporary")
	}

	// Test timeout error
	timeoutErr := New(NetworkError, "timeout error")
	timeoutErr.Timeout = true
	if !timeoutErr.IsTimeout() {
		t.Error("Error should be timeout")
	}

	// Test nil error behavior
	var nilErr *Error
	if nilErr.IsTemporary() {
		t.Error("Nil error should not be temporary")
	}
	if nilErr.IsTimeout() {
		t.Error("Nil error should not be timeout")
	}
	if nilErr.Error() != "" {
		t.Error("Nil error should return empty string")
	}
}

func TestErrorTypeString(t *testing.T) {
	tests := []struct {
		errType ErrorType
		want    string
	}{
		{ConfigError, "configuration error"},
		{ToolError, "tool error"},
		{ResourceError, "resource error"},
		{NetworkError, "network error"},
		{SystemError, "system error"},
		{UnknownError, "unknown error"},
	}

	for _, tt := range tests {
		err := New(tt.errType, tt.want)
		if !strings.Contains(err.Error(), tt.want) {
			t.Errorf("Error string = %v, want %v", err.Error(), tt.want)
		}
	}
}
