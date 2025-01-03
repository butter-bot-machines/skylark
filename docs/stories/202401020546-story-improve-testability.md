# Improve Integration Test Architecture

## Problem
The codebase has tightly coupled dependencies across multiple layers:

1. Command Processing Chain:
```go
// Processor creates everything internally
func New(cfg *config.Config) (*Processor, error) {
    toolMgr := tool.NewManager(...)
    p, err = openai.New("gpt-4", modelConfig)
    networkPolicy := &sandbox.NetworkPolicy{...}
    assistantMgr, err := assistant.NewManager(...)
}

// Assistant requires real filesystem
func NewManager(basePath string, ...) (*Manager, error) {
    promptPath := filepath.Join(basePath, "prompt.md")
    content, err := os.ReadFile(promptPath)
}

// Config requires .skai directory
func (m *Manager) Load() error {
    data, err := os.ReadFile(m.path)
    config.Environment.ConfigDir = filepath.Dir(m.path)
}

// Tool manager requires real compilation
func (m *Manager) Compile(name string) error {
    cmd := exec.Command("go", "build", "-o", binaryPath, mainFile)
    output, err := cmd.CombinedOutput()
}

// Tool execution needs real processes
func (t *Tool) Execute(input []byte, env map[string]string) error {
    cmd := exec.Command(binaryPath)
    stdin, err := cmd.StdinPipe()
    stdout, err := cmd.StdoutPipe()
    cmd.Env = append(os.Environ(), cmdEnv...)
}
```

2. System Dependencies:
```go
// Worker pool creates own limits
func NewPool(cfg *config.Config) *Pool {
    p := &Pool{
        limits: DefaultLimits(),
    }
}

// Watcher requires real filesystem
func New(cfg *config.Config, ...) (*Watcher, error) {
    fsWatcher, err := fsnotify.NewWatcher()
    absPath, err := filepath.Abs(path)
    err := fsWatcher.Add(absPath)
}

// Tool compilation needs go compiler
func Compile(name string) error {
    cmd := exec.Command("go", "build", ...)
    cmd.Dir = toolPath
}

// Tool execution needs system access
func Execute(input []byte) error {
    cmd.Env = append(os.Environ(), ...)
    stdin, _ := cmd.StdinPipe()
    stdout, _ := cmd.StdoutPipe()
}
```

3. Global State and Resources:
```go
// Global logger initialization
var logger *slog.Logger
func init() {
    logger = logging.NewLogger(...)
}

// Hardcoded network policy
networkPolicy := &sandbox.NetworkPolicy{
    AllowedHosts: []string{
        "api.openai.com",
    },
}

// Fixed resource limits
func DefaultLimits() ResourceLimits {
    return ResourceLimits{
        MaxMemory:  256 * 1024 * 1024,
        MaxCPUTime: 50 * time.Millisecond,
    }
}

// Direct environment access
func Execute(input []byte) error {
    if path := os.Getenv("PATH"); path != "" {
        cmdEnv = append(cmdEnv, "PATH="+path)
    }
    if value := os.Getenv(name); value != "" {
        cmdEnv = append(cmdEnv, name+"="+value)
    }
}
```

The result is that even simple operations require the entire system to be working:
```go
// To test this transformation:
"!command test" -> "-!command test"

// We need:
1. Working filesystem
   - .skai directory
   - config.yaml
   - prompt.md
   - tool definitions
   - go compiler
   - tool binaries

2. Working providers
   - OpenAI configuration
   - API key
   - Network access
   - Process execution
   - System environment

3. Working resources
   - CPU limits
   - Memory limits
   - Network policy
   - Sandbox setup
   - Process isolation
```

## Investigation Findings

### Investigation Findings

1. Successful Test Patterns:
```go
// TestAssistantIntegration passes because:
- Creates minimal test assistant
- Bypasses filesystem
- Direct job creation
- No provider needed
- No resource limits

// TestWorkerPool passes because:
- Uses simple job interface
- No file operations
- No provider needed
- Minimal configuration
```

2. Problematic Test Patterns:
```go
// TestCommandInvalidation fails because:
- Needs real filesystem
- Creates full processor chain
- Requires OpenAI config
- Uses resource limits
- Real file operations

// TestWatcherWorkerIntegration fails because:
- Needs fsnotify
- Real filesystem paths
- Full processor chain
- Resource limits
```

3. Core Issues:
   - Components create their dependencies
   - Direct filesystem operations
   - Hardcoded resource limits
   - Global state (loggers, policies)
   - Fixed provider (OpenAI)
   - No interface abstractions

## Proposed Solution

1. Add Core Interfaces:
```go
// pkg/fs/interface.go
type FileSystem interface {
    Read(path string) ([]byte, error)
    Write(path string, []byte) error
    Watch(path string) (<-chan Event, error)
}

// pkg/provider/interface.go
type Provider interface {
    Send(ctx context.Context, prompt string) (*Response, error)
}

// pkg/resource/interface.go
type ResourceManager interface {
    WithCPULimit(d time.Duration) ResourceManager
    WithMemoryLimit(bytes int64) ResourceManager
    Run(ctx context.Context, fn func() error) error
}

// pkg/tool/interface.go
type Compiler interface {
    Compile(src string) (binary []byte, err error)
}

type Executor interface {
    Execute(binary []byte, input []byte) (output []byte, err error)
}

type Environment interface {
    GetEnv(name string) string
    SetEnv(name, value string)
}
```

2. Add Component Options:
```go
// pkg/processor/options.go
type Options struct {
    FileSystem FileSystem    // Optional
    Provider   Provider      // Optional
    Resources  ResourceManager // Optional
    Logger     *slog.Logger   // Optional
}

// pkg/worker/options.go
type Options struct {
    Resources  ResourceManager
    QueueSize  int
    Workers    int
}

// pkg/watcher/options.go
type Options struct {
    FileSystem FileSystem
    Filter     func(string) bool
    Debounce   time.Duration
}
```

3. Add Test Implementations:
```go
// pkg/testing/fs.go
type MemoryFS struct {
    files map[string][]byte
    watch chan Event
}

// pkg/testing/provider.go
type TestProvider struct {
    Response string
}

// pkg/testing/resources.go
type NoopResources struct{}
```

## Benefits

1. Simpler Testing:
   - In-memory filesystem
   - No provider needed
   - No resource limits
   - No global state
   - Fast execution

2. Better Architecture:
   - Clear interfaces
   - Dependency injection
   - Resource isolation
   - Component boundaries
   - Flexible configuration

3. Production Improvements:
   - Multiple providers
   - Custom filesystems
   - Resource control
   - Better monitoring
   - Error handling

## Implementation Plan

1. Core Interfaces (Week 1):
   - Create fs package
   - Create provider package
   - Create resource package
   - Add interfaces
   - Add options

2. Component Updates (Week 2):
   - Update processor
   - Update worker
   - Update watcher
   - Update config
   - Update security

3. Test Support (Week 3):
   - Add memory filesystem
   - Add test provider
   - Add test resources
   - Update test helpers
   - Convert tests

4. Documentation (Week 4):
   - Architecture updates
   - Interface docs
   - Testing guide
   - Examples
   - Migration guide

## Migration Strategy

1. Infrastructure (Week 1):
   - Add interfaces
   - Keep existing code
   - Add options
   - No breaking changes

2. Components (Week 2-3):
   - One component at a time
   - Update tests first
   - Keep backwards compatibility
   - Gradual rollout

3. Testing (Week 3):
   - Add test implementations
   - Convert unit tests
   - Update integration tests
   - Verify coverage

4. Cleanup (Week 4):
   - Remove old patterns
   - Update docs
   - Final testing
   - Release

## Acceptance Criteria

1. Core Functionality:
   - [ ] All tests pass with in-memory filesystem
   - [ ] Tests run without OpenAI config
   - [ ] No resource limits needed for tests
   - [ ] No global state dependencies
   - [ ] Clear component boundaries

2. Test Improvements:
   - [ ] Simple test setup
   - [ ] Fast execution
   - [ ] No filesystem dependencies
   - [ ] No provider requirements
   - [ ] No resource limits

3. Architecture:
   - [ ] Clear interfaces
   - [ ] Proper dependency injection
   - [ ] Resource isolation
   - [ ] Component boundaries
   - [ ] No global state

4. Documentation:
   - [ ] Updated architecture docs
   - [ ] Interface documentation
   - [ ] Testing patterns
   - [ ] Migration guide
   - [ ] Examples
