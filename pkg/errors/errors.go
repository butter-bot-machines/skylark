package errors

import (
	"fmt"
	"runtime"
	"strings"
	"sync"

	"github.com/butter-bot-machines/skylark/pkg/logging"
)

// Global registry for error types
var (
	globalRegistry = NewRegistry()

	// Standard error types
	ConfigError   = globalRegistry.Register("ConfigError", 1)
	ToolError     = globalRegistry.Register("ToolError", 2)
	ResourceError = globalRegistry.Register("ResourceError", 3)
	NetworkError  = globalRegistry.Register("NetworkError", 4)
	SystemError   = globalRegistry.Register("SystemError", 5)
	UnknownError  = globalRegistry.Register("UnknownError", 6)
)

// New creates a new error with type and message
func New(errType ErrorType, msg string, args ...interface{}) Error {
	if t, ok := errType.(*errorType); ok {
		return &concreteError{
			errType: t,
			message: fmt.Sprintf(msg, args...),
			stack:   captureStackTrace(2),
			context: make(map[string]interface{}),
		}
	}
	// Fallback to unknown error type
	return &concreteError{
		errType: UnknownError.(*errorType),
		message: fmt.Sprintf(msg, args...),
		stack:   captureStackTrace(2),
		context: make(map[string]interface{}),
	}
}

// Wrap wraps an existing error with additional context
func Wrap(err error, msg string, args ...interface{}) Error {
	if err == nil {
		return nil
	}

	// If already wrapped, add context
	if e, ok := err.(*concreteError); ok {
		e.message = fmt.Sprintf(msg, args...) + ": " + e.message
		return e
	}

	// Create new wrapped error
	return &concreteError{
		message: fmt.Sprintf(msg, args...) + ": " + err.Error(),
		cause:   err,
		stack:   captureStackTrace(1),
		context: make(map[string]interface{}),
	}
}

// NewRegistry creates a new error type registry
func NewRegistry() Registry {
	return &registry{
		types: make(map[string]*errorType),
	}
}

// NewPanicHandler creates a new panic handler
func NewPanicHandler(reg Registry, logger logging.Logger) PanicHandler {
	return &panicHandler{
		registry: reg,
		logger:   logger,
	}
}

// NewAggregate creates a new error aggregate
func NewAggregate() Aggregate {
	return &errorAggregate{
		errs: make([]error, 0),
	}
}

// Internal implementations

type errorType struct {
	name string
	code int
}

func (t *errorType) Name() string {
	return t.name
}

func (t *errorType) Code() int {
	return t.code
}

func (t *errorType) New(msg string, args ...interface{}) Error {
	return &concreteError{
		errType: t,
		message: fmt.Sprintf(msg, args...),
		stack:   captureStackTrace(3),
		context: make(map[string]interface{}),
	}
}

func (t *errorType) Wrap(err error, msg string, args ...interface{}) Error {
	if err == nil {
		return nil
	}

	// If already wrapped, add context
	if e, ok := err.(*concreteError); ok {
		e.message = fmt.Sprintf(msg, args...) + ": " + e.message
		return e
	}

	// Create new wrapped error
	return &concreteError{
		errType: t,
		message: fmt.Sprintf(msg, args...) + ": " + err.Error(),
		cause:   err,
		stack:   captureStackTrace(1),
		context: make(map[string]interface{}),
	}
}

type concreteError struct {
	errType   *errorType
	message   string
	cause     error
	stack     StackTrace
	context   map[string]interface{}
	temporary bool
	timeout   bool
}

func (e *concreteError) Error() string {
	if e == nil {
		return ""
	}

	var b strings.Builder
	b.WriteString(e.message)

	if len(e.context) > 0 {
		b.WriteString(" [")
		first := true
		for k, v := range e.context {
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

func (e *concreteError) Format(f fmt.State, c rune) {
	if e == nil {
		return
	}

	switch c {
	case 'v':
		if f.Flag('+') {
			// Detailed format with stack trace
			fmt.Fprintf(f, "%s\n", e.Error())
			if e.cause != nil {
				fmt.Fprintf(f, "Caused by: %+v\n", e.cause)
			}
			fmt.Fprintf(f, "Stack trace:\n%s", e.stack.String())
		} else {
			fmt.Fprintf(f, "%s", e.Error())
		}
	default:
		fmt.Fprintf(f, "%s", e.Error())
	}
}

func (e *concreteError) WithContext(key string, value interface{}) Error {
	if e == nil {
		return nil
	}
	e.context[key] = value
	return e
}

func (e *concreteError) WithType(errType ErrorType) Error {
	if e == nil {
		return nil
	}
	if t, ok := errType.(*errorType); ok {
		e.errType = t
	}
	return e
}

func (e *concreteError) IsTemporary() bool {
	if e == nil {
		return false
	}
	return e.temporary
}

func (e *concreteError) IsTimeout() bool {
	if e == nil {
		return false
	}
	return e.timeout
}

func (e *concreteError) Stack() StackTrace {
	return e.stack
}

func (e *concreteError) Context() map[string]interface{} {
	return e.context
}

func (e *concreteError) Cause() error {
	return e.cause
}

type stackFrame struct {
	file     string
	line     int
	function string
}

func (f *stackFrame) File() string {
	return f.file
}

func (f *stackFrame) Line() int {
	return f.line
}

func (f *stackFrame) Function() string {
	return f.function
}

func (f *stackFrame) String() string {
	return fmt.Sprintf("%s:%d %s", f.file, f.line, f.function)
}

type stackTrace struct {
	frames []Frame
}

func (st *stackTrace) Frames() []Frame {
	return st.frames
}

func (st *stackTrace) String() string {
	var b strings.Builder
	for _, frame := range st.frames {
		fmt.Fprintf(&b, "  %s\n", frame.String())
	}
	return b.String()
}

func captureStackTrace(skip int) StackTrace {
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

		frames = append(frames, &stackFrame{
			file:     shortFile,
			line:     line,
			function: fn.Name(),
		})

		// Limit stack depth
		if len(frames) >= 32 {
			break
		}
	}
	return &stackTrace{frames: frames}
}

type errorAggregate struct {
	errs []error
}

func (a *errorAggregate) Write(p []byte) (n int, err error) {
	a.Add(fmt.Errorf("%s", p))
	return len(p), nil
}

func (a *errorAggregate) Add(err error) {
	if err != nil {
		a.errs = append(a.errs, err)
	}
}

func (a *errorAggregate) HasErrors() bool {
	return len(a.errs) > 0
}

func (a *errorAggregate) Error() string {
	if !a.HasErrors() {
		return ""
	}

	if len(a.errs) == 1 {
		return a.errs[0].Error()
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%d errors occurred:\n", len(a.errs))
	for i, err := range a.errs {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "[%d] %v", i+1, err)
	}
	return b.String()
}

func (a *errorAggregate) Errors() []error {
	return a.errs
}

type registry struct {
	types map[string]*errorType
	mu    sync.RWMutex
}

func (r *registry) Register(name string, code int) ErrorType {
	r.mu.Lock()
	defer r.mu.Unlock()

	t := &errorType{
		name: name,
		code: code,
	}
	r.types[name] = t
	return t
}

func (r *registry) Get(name string) (ErrorType, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, ok := r.types[name]
	return t, ok
}

func (r *registry) List() []ErrorType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]ErrorType, 0, len(r.types))
	for _, t := range r.types {
		types = append(types, t)
	}
	return types
}

type panicHandler struct {
	registry Registry
	logger   logging.Logger
}

func (h *panicHandler) Handle(v interface{}) error {
	errType, ok := h.registry.Get("PanicError")
	if !ok {
		errType = h.registry.Register("PanicError", 500)
	}

	var msg string
	switch v := v.(type) {
	case string:
		msg = v
	case error:
		msg = v.Error()
	default:
		msg = fmt.Sprintf("%v", v)
	}

	if h.logger != nil {
		h.logger.Error("panic recovered", map[string]interface{}{
			"error": msg,
		})
	}

	return &concreteError{
		errType: errType.(*errorType),
		message: fmt.Sprintf("panic recovered: %s", msg),
		stack:   captureStackTrace(1),
		context: map[string]interface{}{
			"recovered": true,
		},
	}
}

func (h *panicHandler) Recover() func() error {
	return func() error {
		r := recover()
		if r == nil {
			return nil
		}
		return h.Handle(r)
	}
}
