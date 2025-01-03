# Apply Interface Design Patterns

## Problem
Our current stories propose interfaces that may not follow Go best practices:

1. Large Interfaces:
```go
// Too many methods in one interface
type FileSystem interface {
    Read(path string) ([]byte, error)
    Write(path string, []byte) error
    Remove(path string) error
    MkdirAll(path string, perm os.FileMode) error
    ReadDir(path string) ([]fs.DirEntry, error)
    OpenFile(path string, flag int, perm os.FileMode) (File, error)
    Rename(old, new string) error
    Abs(path string) (string, error)
    Join(elem ...string) string
}
```

2. Premature Abstraction:
```go
// Exposing interfaces before need
type ResourceController interface {
    SetMemoryLimit(bytes int64) error
    GetMemoryUsage() int64
    ForceGC()
    SetCPULimit(cores int) error
    GetCPUUsage() float64
}
```

3. Mixed Responsibilities:
```go
// Mixing different concerns
type Provider interface {
    Send(ctx context.Context, prompt string) (*Response, error)
    SetRateLimiter(RateLimiter)
    GetRateLimiter() RateLimiter
}
```

## Solution

1. Split Large Interfaces:
```go
// pkg/fs/interface.go
type Reader interface {
    Read(path string) ([]byte, error)
}

type Writer interface {
    Write(path string, []byte) error
}

type Directory interface {
    MkdirAll(path string, perm os.FileMode) error
    ReadDir(path string) ([]fs.DirEntry, error)
}

// Compose when needed
type FileSystem interface {
    Reader
    Writer
    Directory
}
```

2. Focus on Behavior:
```go
// pkg/process/interface.go
type Limiter interface {
    Limit(pid int, limits ResourceLimits) error
}

type Monitor interface {
    Usage(pid int) (ResourceUsage, error)
}

type Controller interface {
    Start(cmd *Command) (Process, error)
    Signal(pid int, sig os.Signal) error
}
```

3. Consumer-Driven Design:
```go
// pkg/provider/openai/provider.go
type rateLimited interface {
    Wait(ctx context.Context) error
    AddTokens(count int) error
}

type httpClient interface {
    Do(req *http.Request) (*http.Response, error)
}

type Provider struct {
    client     httpClient
    rateLimits rateLimited
}
```

## Updates Needed

1. Filesystem Story:
   - Split into focused interfaces
   - Keep implementation details private
   - Add specific test interfaces

2. Process Story:
   - Separate process control from limits
   - Focus on core behaviors
   - Simplify test implementations

3. Provider Story:
   - Move interfaces to consumers
   - Split rate limiting concerns
   - Simplify HTTP abstraction

4. Time/Resource Story:
   - Focus on core timing needs
   - Separate resource concerns
   - Simplify test implementations

5. Infrastructure Story:
   - Split config and logging
   - Focus on core behaviors
   - Keep implementation details private

## Benefits

1. Better Testing:
   - Smaller interfaces are easier to mock
   - Focused behaviors are easier to verify
   - Clear boundaries improve isolation

2. Better Maintenance:
   - Smaller interfaces are easier to implement
   - Clear responsibilities improve understanding
   - Private details allow refactoring

3. Better Design:
   - Natural interface evolution
   - Clear component boundaries
   - Flexible composition

## Implementation

1. Review Stories:
   - Update interface definitions
   - Split large interfaces
   - Move interfaces to consumers

2. Update Tests:
   - Simplify test implementations
   - Focus on behavior verification
   - Remove implementation details

3. Update Components:
   - Apply interface segregation
   - Use composition where needed
   - Keep implementation details private

## Acceptance Criteria

1. Interface Design:
   - [ ] Small, focused interfaces
   - [ ] Clear behavior names
   - [ ] Consumer-driven design
   - [ ] Proper encapsulation

2. Testing:
   - [ ] Simple mock implementations
   - [ ] Clear behavior verification
   - [ ] No implementation details
   - [ ] Good isolation

3. Documentation:
   - [ ] Clear interface purposes
   - [ ] Usage examples
   - [ ] Testing patterns
   - [ ] Design rationale
