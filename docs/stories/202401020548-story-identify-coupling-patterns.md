# Identify Core Coupling Patterns

## Problem
After investigating the tightly coupled dependencies, we've identified several core patterns that make testing difficult:

1. Components Create Dependencies:
```go
// Instead of accepting dependencies, components create them
func New(cfg *config.Config) (*Processor, error) {
    toolMgr := tool.NewManager(...)      // Creates tool manager
    p, err = openai.New("gpt-4", cfg)    // Creates provider
    assistantMgr, err := NewManager(...) // Creates assistant
}
```

2. Direct System Access:
```go
// Direct filesystem operations
content, err := os.ReadFile(promptPath)
err := os.WriteFile(path, data, 0644)

// Direct process management
cmd := exec.Command("go", "build", ...)
syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)

// Direct network calls
resp, err := http.Post(apiURL, "application/json", body)
```

3. Global State:
```go
// Global loggers
var logger *slog.Logger
func init() {
    logger = logging.NewLogger(...)
}

// Fixed resource limits
var DefaultLimits = ResourceLimits{
    MaxMemory: 256 * 1024 * 1024,
    MaxCPUTime: 50 * time.Millisecond,
}
```

4. Fixed Implementations:
```go
// Hardcoded provider
p, err = openai.New("gpt-4", modelConfig)

// Fixed filesystem
content, err := os.ReadFile(path)

// Fixed network policy
networkPolicy := &sandbox.NetworkPolicy{
    AllowedHosts: []string{"api.openai.com"},
}
```

## Impact
These patterns mean that even simple operations require the entire system:

1. To test command marking (!cmd -> -!cmd):
   - Real filesystem
   - OpenAI provider
   - Process execution
   - System privileges

2. To test file watching:
   - Real filesystem
   - Process management
   - Resource limits
   - Network access

## Solution
Break these patterns by:

1. Dependency Injection:
```go
// Accept dependencies instead of creating them
func New(cfg *config.Config, opts Options) (*Processor, error) {
    opts.Provider    // Provided externally
    opts.FileSystem  // Provided externally
    opts.Resources  // Provided externally
}
```

2. Interface Abstraction:
```go
// Abstract system operations
type FileSystem interface {
    Read(path string) ([]byte, error)
    Write(path string, []byte) error)
}

type ProcessManager interface {
    Start(cmd *Command) (Process, error)
    Kill(pid int) error
}

type Provider interface {
    Send(ctx context.Context, prompt string) (*Response, error)
}
```

3. Configuration Injection:
```go
// Accept configuration instead of hardcoding
type Options struct {
    ResourceLimits ResourceLimits
    NetworkPolicy NetworkPolicy
    LogConfig     LogConfig
}
```

## Benefits

1. Testability:
   - Mock dependencies
   - Control system access
   - Isolate components
   - Fast execution

2. Flexibility:
   - Swap implementations
   - Configure behavior
   - Extend functionality
   - Better reuse

## Next Steps

1. Create follow-up stories for:
   - Filesystem abstraction
   - Process management
   - Provider system
   - Resource control

2. Focus on:
   - Interface design
   - Test patterns
   - Migration path
   - Documentation
