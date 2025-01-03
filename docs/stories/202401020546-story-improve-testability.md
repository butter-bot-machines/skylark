# Improve Integration Test Architecture

## Problem
Integration tests are failing because simple file processing tests are forced to use the full dependency chain. For example, TestCommandInvalidation only needs to verify that commands are properly marked as processed (! -> -!), but it's required to:

1. Create a real processor with:
   - OpenAI provider configuration
   - File system access
   - Resource limits
   - Network policies

2. Set up the complete pipeline:
   - Worker pool with CPU limits
   - File watcher with real filesystem
   - Assistant manager with prompt loading
   - Tool manager with sandbox

3. Deal with global state:
   - Logger initialization
   - Resource management
   - Network policies

The core issue is that the architecture doesn't allow testing individual concerns in isolation. Every test must use the full dependency chain, even when testing simple mechanics.

## Investigation Findings

### Test Analysis
TestAssistantIntegration passes because it:
- Uses a simple mock assistant
- Doesn't touch filesystem
- Bypasses provider chain

TestCommandInvalidation fails because it:
- Can't isolate file processing logic
- Must initialize full dependency chain
- Hits real provider and resource limits

### Architecture Issues
1. Command Processing:
   - File operations mixed with command processing
   - No way to process commands without provider
   - Can't bypass resource limits

2. Component Coupling:
   - Processor creates its own dependencies
   - Assistant requires real filesystem
   - Worker enforces limits on all jobs

3. Testing Patterns:
   - No clear interfaces for mocking
   - No separation of concerns
   - All-or-nothing dependency chain

## Proposed Solution

1. Separate Command Processing
```go
// pkg/processor/command/processor.go
type CommandProcessor interface {
    Process(cmd *Command) (string, error)
}

// Implementations
type AICommandProcessor struct {
    provider Provider
    limits   ResourceLimits
}

type TestCommandProcessor struct {
    response string // Fixed response for testing
}
```

2. Extract File Operations
```go
// pkg/processor/file/processor.go
type FileProcessor interface {
    ProcessFile(path string) error
}

// Implementation that handles command marking
type MarkdownProcessor struct {
    fs       FileSystem
    commands CommandProcessor
}
```

3. Decouple Components
```go
// pkg/processor/processor.go
type ProcessorConfig struct {
    FileSystem    FileSystem        // Interface for file operations
    Commands      CommandProcessor  // Interface for command processing
    Limits        ResourceLimits    // Optional resource limits
    Network       NetworkPolicy     // Optional network policy
}

func New(cfg ProcessorConfig) *Processor {
    // Use provided components or create defaults
}
```

4. Testing Support
```go
// pkg/processor/testing/processor.go
// Ready-to-use test implementations
var (
    NoopProcessor = &TestCommandProcessor{response: ""}
    MarkProcessor = &TestCommandProcessor{response: "OK"}
    MemoryFS = NewMemoryFileSystem()
)

// Example test setup
processor := NewMarkdownProcessor(
    MemoryFS,
    MarkProcessor,
)
```

## Benefits
1. Tests can focus on what they're testing:
   - File operations without provider
   - Command processing without filesystem
   - Resource limits only when needed

2. Clear testing patterns:
   - Use TestCommandProcessor for simple tests
   - Use MemoryFS for filesystem tests
   - Mix and match components as needed

3. Better production code:
   - Clear component boundaries
   - Explicit dependencies
   - Configurable behavior

## Implementation Plan
1. Create command processing interface
2. Extract file operations
3. Update processor configuration
4. Add testing implementations
5. Update existing tests
6. Document testing patterns

## Migration Strategy
1. Create new interfaces alongside existing code
2. Gradually migrate components to use interfaces
3. Update tests to use new patterns
4. Remove old implementations

## Acceptance Criteria
- [ ] TestCommandInvalidation passes without provider
- [ ] TestWatcherWorkerIntegration uses MemoryFS
- [ ] No changes to production behavior
- [ ] Clear documentation for test patterns
- [ ] Existing code paths still work
- [ ] Easy to add new test implementations
