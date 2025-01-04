package errors

// AsError attempts to convert an error to our Error interface
func AsError(err error) Error {
	if err == nil {
		return nil
	}
	if e, ok := err.(Error); ok {
		return e
	}
	if e, ok := err.(*concreteError); ok {
		return e
	}
	return nil
}

// GetType returns the error type if available
func GetType(err error) ErrorType {
	if e := AsError(err); e != nil {
		if ce, ok := e.(*concreteError); ok {
			return ce.errType
		}
	}
	return nil
}

// GetMessage returns the error message without context
func GetMessage(err error) string {
	if e := AsError(err); e != nil {
		if ce, ok := e.(*concreteError); ok {
			return ce.message
		}
	}
	if err != nil {
		return err.Error()
	}
	return ""
}

// GetStack returns the stack trace if available
func GetStack(err error) StackTrace {
	if e := AsError(err); e != nil {
		return e.Stack()
	}
	return nil
}

// GetContext returns the error context if available
func GetContext(err error) map[string]interface{} {
	if e := AsError(err); e != nil {
		return e.Context()
	}
	return nil
}

// GetCause returns the underlying cause if available
func GetCause(err error) error {
	if e := AsError(err); e != nil {
		return e.Cause()
	}
	return nil
}

// IsTemporary returns true if the error is temporary
func IsTemporary(err error) bool {
	if e := AsError(err); e != nil {
		if ce, ok := e.(*concreteError); ok {
			return ce.temporary
		}
	}
	return false
}

// IsTimeout returns true if the error is a timeout
func IsTimeout(err error) bool {
	if e := AsError(err); e != nil {
		if ce, ok := e.(*concreteError); ok {
			return ce.timeout
		}
	}
	return false
}

// WithContext adds context to an error
func WithContext(err error, key string, value interface{}) error {
	if e := AsError(err); e != nil {
		if ce, ok := e.(*concreteError); ok {
			ce.context[key] = value
			return ce
		}
	}
	return err
}

// WithType sets the error type
func WithType(err error, errType ErrorType) error {
	if e := AsError(err); e != nil {
		if ce, ok := e.(*concreteError); ok {
			ce.errType = errType.(*errorType)
			return ce
		}
	}
	return err
}

// SetTemporary marks an error as temporary
func SetTemporary(err error) error {
	if e := AsError(err); e != nil {
		if ce, ok := e.(*concreteError); ok {
			ce.temporary = true
			return ce
		}
	}
	return err
}

// SetTimeout marks an error as a timeout
func SetTimeout(err error) error {
	if e := AsError(err); e != nil {
		if ce, ok := e.(*concreteError); ok {
			ce.timeout = true
			return ce
		}
	}
	return err
}
