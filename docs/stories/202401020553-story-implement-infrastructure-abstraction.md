# Implement Infrastructure Abstraction

## Problem
Direct infrastructure dependencies make testing difficult:

```go
// Direct config operations
data, err := os.ReadFile(m.path)
err := os.WriteFile(m.path, data, 0644)
err := os.MkdirAll(filepath.Dir(m.path), 0755)

// Fixed config paths
path:   filepath.Join(configDir, "config.yaml")
config.Environment.ConfigDir = filepath.Dir(m.path)

// Direct logging
Output:    os.Stdout
handler = slog.NewJSONHandler(opts.Output, handlerOpts)

// Global loggers
var logger *slog.Logger
func init() {
    logger = logging.NewLogger(...)
}
```

This means:
1. Tests need config files
2. Tests need writable paths
3. Tests pollute stdout
4. Tests have global state

## Solution

1. Create Config Interface:
```go
// pkg/config/interface.go
type Store interface {
    // Basic operations
    Load() error
    Save() error
    Reset() error
    
    // Value operations
    Get(key string) (interface{}, error)
    Set(key string, value interface{}) error
    Delete(key string) error
    
    // Batch operations
    GetAll() (map[string]interface{}, error)
    SetAll(values map[string]interface{}) error
    
    // Validation
    Validate() error
}

type Environment interface {
    // Environment access
    GetString(key string) string
    GetInt(key string) int
    GetBool(key string) bool
    GetDuration(key string) time.Duration
}
```

2. Create Logging Interface:
```go
// pkg/logging/interface.go
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

type Level int

const (
    LevelDebug Level = iota
    LevelInfo
    LevelWarn
    LevelError
)
```

3. Add Production Implementation:
```go
// pkg/config/file/store.go
type FileStore struct {
    fs       FileSystem
    path     string
    data     map[string]interface{}
    validate ValidateFunc
}

func (s *FileStore) Load() error {
    data, err := s.fs.Read(s.path)
    if err != nil {
        return err
    }
    return yaml.Unmarshal(data, &s.data)
}

// pkg/logging/slog/logger.go
type SlogLogger struct {
    logger *slog.Logger
    attrs  []interface{}
}

func (l *SlogLogger) Info(msg string, args ...interface{}) {
    l.logger.Info(msg, args...)
}
```

4. Add Test Implementation:
```go
// pkg/config/memory/store.go
type MemoryStore struct {
    data     map[string]interface{}
    validate ValidateFunc
}

func (s *MemoryStore) Load() error {
    return nil // Already in memory
}

// pkg/logging/memory/logger.go
type MemoryLogger struct {
    level  Level
    output []LogEntry
    mu     sync.Mutex
}

type LogEntry struct {
    Level   Level
    Message string
    Args    []interface{}
    Time    time.Time
}

func (l *MemoryLogger) Info(msg string, args ...interface{}) {
    l.mu.Lock()
    defer l.mu.Unlock()
    l.output = append(l.output, LogEntry{
        Level:   LevelInfo,
        Message: msg,
        Args:    args,
        Time:    time.Now(),
    })
}
```

5. Update Components:
```go
// pkg/processor/processor.go
type Processor struct {
    config config.Store
    logger logging.Logger
}

func New(opts Options) (*Processor, error) {
    if opts.Config == nil {
        opts.Config = file.NewStore(opts.ConfigPath)
    }
    if opts.Logger == nil {
        opts.Logger = slog.NewLogger(nil)
    }
    return &Processor{
        config: opts.Config,
        logger: opts.Logger,
    }, nil
}
```

## Benefits

1. Testing:
   - In-memory config
   - Captured logs
   - No file dependencies
   - No global state

2. Production:
   - Same interface
   - No behavior changes
   - Better validation
   - Better monitoring

3. Future:
   - Remote config
   - Log aggregation
   - Better validation
   - Better monitoring

## Implementation

1. Core Changes:
   - Create config interfaces
   - Create logging interfaces
   - Add implementations
   - Add tests

2. Component Updates:
   - Update processor
   - Update assistant
   - Update worker
   - Update security

3. Test Support:
   - Add memory store
   - Add memory logger
   - Update test helpers
   - Add examples

## Acceptance Criteria

1. Functionality:
   - [ ] All config ops use Store
   - [ ] All logging uses Logger
   - [ ] Production behavior unchanged
   - [ ] Proper validation

2. Testing:
   - [ ] Tests use memory store
   - [ ] Tests capture logs
   - [ ] No file dependencies
   - [ ] No global state

3. Documentation:
   - [ ] Interface docs
   - [ ] Implementation docs
   - [ ] Test patterns
   - [ ] Examples
