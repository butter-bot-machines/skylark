package concrete

import (
	"fmt"
	"runtime"
	"strings"
	"sync"

	"github.com/butter-bot-machines/skylark/pkg/errors"
	"github.com/butter-bot-machines/skylark/pkg/logging"
)

// errorType implements errors.ErrorType
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

func (t *errorType) New(msg string, args ...interface{}) errors.Error {
	return &concreteError{
		errType: t,
		message: fmt.Sprintf(msg, args...),
		stack:   captureStackTrace(2),
		context: make(map[string]interface{}),
	}
}

func (t *errorType) Wrap(err error, msg string, args ...interface{}) errors.Error {
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

// concreteError implements errors.Error
type concreteError struct {
	errType   *errorType
	message   string
	cause     error
	stack     errors.StackTrace
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

func (e *concreteError) WithContext(key string, value interface{}) errors.Error {
	if e == nil {
		return nil
	}
	e.context[key] = value
	return e
}

func (e *concreteError) WithType(errType errors.ErrorType) errors.Error {
	if e == nil {
		return nil
	}
	e.errType = &errorType{
		name: errType.Name(),
		code: errType.Code(),
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

func (e *concreteError) Stack() errors.StackTrace {
	return e.stack
}

func (e *concreteError) Context() map[string]interface{} {
	return e.context
}

func (e *concreteError) Cause() error {
	return e.cause
}

// stackFrame implements errors.Frame
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

// stackTrace implements errors.StackTrace
type stackTrace struct {
	frames []errors.Frame
}

func (st *stackTrace) Frames() []errors.Frame {
	return st.frames
}

func (st *stackTrace) String() string {
	var b strings.Builder
	for _, frame := range st.frames {
		fmt.Fprintf(&b, "  %s\n", frame.String())
	}
	return b.String()
}

// captureStackTrace captures the current stack trace
func captureStackTrace(skip int) errors.StackTrace {
	var frames []errors.Frame
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

// errorAggregate implements errors.Aggregate
type errorAggregate struct {
	errs []error
}

func NewAggregate() errors.Aggregate {
	return &errorAggregate{
		errs: make([]error, 0),
	}
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

// Registry manages error types
type Registry struct {
	types map[string]*errorType
	mu    sync.RWMutex
}

// NewRegistry creates a new error type registry
func NewRegistry() *Registry {
	return &Registry{
		types: make(map[string]*errorType),
	}
}

// Register adds a new error type
func (r *Registry) Register(name string, code int) errors.ErrorType {
	r.mu.Lock()
	defer r.mu.Unlock()

	t := &errorType{
		name: name,
		code: code,
	}
	r.types[name] = t
	return t
}

// Get returns an error type by name
func (r *Registry) Get(name string) (errors.ErrorType, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, ok := r.types[name]
	return t, ok
}

// List returns all registered error types
func (r *Registry) List() []errors.ErrorType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]errors.ErrorType, 0, len(r.types))
	for _, t := range r.types {
		types = append(types, t)
	}
	return types
}

// PanicHandler handles panic recovery
type PanicHandler struct {
	registry *Registry
	logger   logging.Logger
}

// NewPanicHandler creates a new panic handler
func NewPanicHandler(registry *Registry, logger logging.Logger) *PanicHandler {
	return &PanicHandler{
		registry: registry,
		logger:   logger,
	}
}

// Handle handles a panic
func (h *PanicHandler) Handle(v interface{}) error {
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

	if err, ok := errType.New("panic recovered: %s", msg).(errors.Error); ok {
		return err
	}
	return nil
}

// Recover returns a function that recovers from panics
func (h *PanicHandler) Recover() func() error {
	return func() error {
		r := recover()
		if r == nil {
			return nil
		}
		return h.Handle(r)
	}
}
