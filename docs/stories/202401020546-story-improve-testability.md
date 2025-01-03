# Improve Integration Test Architecture

## Problem
The codebase has pervasive filesystem access and tightly coupled dependencies that make testing difficult:

1. Direct Filesystem Access In:
   - Processor (file reading/writing)
   - Assistant (prompt loading)
   - Config (configuration files)
   - Security (audit logs, keys)
   - Sandbox (file access control)

2. Tightly Coupled Dependencies:
   - Processor requires real provider
   - Assistant needs real filesystem
   - Config depends on file locations
   - Security needs real files
   - Tests use real temp directories

3. Global State and Resources:
   - Logger initialization in init()
   - Hardcoded resource limits
   - Fixed file paths
   - Network policies

The core issue is that basic operations (like marking a command as processed) require the entire system to be initialized with real files and dependencies.

## Investigation Findings

### Investigation Findings

1. File Operation Patterns:
   - Direct os.ReadFile/WriteFile calls
   - Hardcoded file paths
   - Real temp directories in tests
   - No filesystem abstraction
   - Mixed file and business logic

2. Test Challenges:
   - TestAssistantIntegration passes because it avoids files
   - TestCommandInvalidation fails due to full chain
   - Performance tests need real files
   - Security tests touch filesystem
   - Integration tests use temp dirs

3. Core Issues:
   - No filesystem abstraction
   - No dependency injection
   - Mixed concerns
   - Global state
   - Resource coupling

## Proposed Solution

1. Create Filesystem Interface
```go
// pkg/fs/interface.go
type FileSystem interface {
    Read(path string) ([]byte, error)
    Write(path string, content []byte) error)
    Exists(path string) bool
    MkdirAll(path string, perm os.FileMode) error
    Remove(path string) error
    Walk(root string, fn WalkFn) error
}

// Implementations
type OSFileSystem struct{} // Production
type MemoryFileSystem struct{} // Testing
type ReadOnlyFileSystem struct{} // Security
```

2. Add Dependency Injection
```go
// pkg/processor/processor.go
type Config struct {
    FileSystem FileSystem
    Provider   Provider
    Limits     ResourceLimits
}

// pkg/assistant/assistant.go
type Config struct {
    FileSystem FileSystem
    Provider   Provider
    Tools      ToolManager
}

// pkg/security/security.go
type Config struct {
    FileSystem FileSystem
    Audit      AuditConfig
}
```

3. Create Test Utilities
```go
// pkg/testing/fs.go
type TestFS struct {
    Files map[string][]byte
}

func NewTestFS(files map[string][]byte) *TestFS
func WithFile(name, content string) Option
func WithTempDir() Option

// Usage
fs := testing.NewTestFS(
    testing.WithFile("config.yaml", configData),
    testing.WithFile("prompt.md", promptData),
)
```

4. Update Components
```go
// pkg/processor/processor.go
type Processor struct {
    fs       FileSystem    // Filesystem access
    provider Provider      // Optional provider
    limits   Limits       // Optional limits
}

// pkg/assistant/assistant.go
type Assistant struct {
    fs      FileSystem   // Filesystem access
    prompt  string       // Cached prompt
    config  Config      // Runtime config
}
```

## Benefits

1. Simpler Testing:
   - In-memory filesystem for unit tests
   - No temp directories needed
   - Controlled test environments
   - Isolated component testing
   - Predictable behavior

2. Better Architecture:
   - Clear dependency boundaries
   - Explicit file operations
   - Configurable components
   - Testable security
   - Resource isolation

3. Production Improvements:
   - Filesystem abstraction
   - Better error handling
   - Security controls
   - Performance monitoring
   - Cleaner code

## Implementation Plan

1. Core Infrastructure:
   - Create filesystem interface
   - Implement OS filesystem
   - Implement memory filesystem
   - Add test utilities

2. Update Components:
   - Modify processor
   - Update assistant
   - Adapt security
   - Adjust config
   - Update sandbox

3. Testing Support:
   - Add test filesystem
   - Create test helpers
   - Update test suites
   - Add examples

4. Documentation:
   - Update architecture docs
   - Add testing guides
   - Document patterns
   - Provide examples

## Migration Strategy

1. Infrastructure (Week 1):
   - Add fs package
   - Create interfaces
   - Add test utilities
   - No production changes

2. Component Updates (Week 2-3):
   - One component at a time
   - Keep old code working
   - Update tests first
   - Gradual rollout

3. Testing Migration (Week 3-4):
   - Convert unit tests
   - Update integration tests
   - Add new test patterns
   - Verify coverage

4. Cleanup (Week 4):
   - Remove old patterns
   - Update documentation
   - Final testing
   - Release

## Acceptance Criteria

1. Functionality:
   - [ ] All tests pass with memory filesystem
   - [ ] No changes to production behavior
   - [ ] Security tests work with read-only filesystem
   - [ ] Performance tests run without temp files
   - [ ] Integration tests use test utilities

2. Architecture:
   - [ ] Clear filesystem abstraction
   - [ ] Proper dependency injection
   - [ ] No direct os package usage
   - [ ] Resource isolation
   - [ ] Better error handling

3. Testing:
   - [ ] Simpler test setup
   - [ ] No temp directories
   - [ ] Predictable test behavior
   - [ ] Good test coverage
   - [ ] Clear testing patterns

4. Documentation:
   - [ ] Updated architecture docs
   - [ ] Testing guidelines
   - [ ] Migration guide
   - [ ] Example patterns
