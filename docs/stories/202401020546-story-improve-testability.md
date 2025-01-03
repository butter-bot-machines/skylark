# Improve Integration Test Architecture

## Problem
The processor package has tightly coupled dependencies that prevent isolated testing:

1. Processor.New Creates Everything:
```go
func New(cfg *config.Config) (*Processor, error) {
    // Creates own tool manager
    toolMgr := tool.NewManager(...)

    // Creates own provider (OpenAI only)
    p, err = openai.New("gpt-4", modelConfig)

    // Creates own network policy
    networkPolicy := &sandbox.NetworkPolicy{...}

    // Creates own assistant manager
    assistantMgr, err := assistant.NewManager(...)
}
```

2. No Way to Mock Dependencies:
   - Can't provide test provider
   - Can't provide test assistant
   - Can't bypass file operations
   - Can't disable network policy

3. Global State:
   - Logger initialized in init()
   - Network policy hardcoded
   - OpenAI provider assumed

The core issue is that simple operations (like marking a command as processed) require the entire dependency chain to be working:
```go
// Just to test this transformation:
"!command test" -> "-!command test"

// We need working:
- OpenAI provider
- Tool manager
- Assistant manager
- Network policy
- File operations
```

## Investigation Findings

### Investigation Findings

1. TestAssistantIntegration Passes Because:
```go
// Creates simple test assistant
assistant := &testAssistant{
    processedCommands: make(chan string, 1),
}

// Direct job creation
jobQueue <- &commandJob{
    command:   "!test hello world",
    assistant: assistant,
}
```

2. TestCommandInvalidation Fails Because:
```go
// Uses real processor
proc, err := testutil.NewMockProcessor()

// Mock processor isn't really a mock
func NewMockProcessor() (*processor.Processor, error) {
    // Still needs OpenAI config
    cfg := &config.Config{
        Models: map[string]map[string]config.ModelConfig{
            "openai": {
                "gpt-4": {
                    APIKey: "test-key",
                },
            },
        },
    }
    // Creates real processor
    return processor.New(cfg)
}
```

3. Core Issues:
   - No dependency injection
   - No interface abstractions
   - Components create dependencies
   - Global state in init()
   - Assumed OpenAI provider

## Proposed Solution

1. Add Dependency Injection:
```go
// pkg/processor/processor.go
type Options struct {
    Provider    provider.Provider    // Optional
    Assistant   assistant.Manager    // Optional
    Parser      parser.Parser       // Optional
    Logger      *slog.Logger        // Optional
}

func New(cfg *config.Config, opts Options) (*Processor, error) {
    p := &Processor{config: cfg}

    // Use provided or create default
    if opts.Provider != nil {
        p.provider = opts.Provider
    }

    if opts.Assistant != nil {
        p.assistants = opts.Assistant
    }

    if opts.Parser != nil {
        p.parser = opts.Parser
    }

    if opts.Logger != nil {
        p.logger = opts.Logger
    }

    return p, nil
}
```

2. Add Test Implementations:
```go
// test/testutil/mock_processor.go
type TestProvider struct {
    Response string
}

func (p *TestProvider) Send(ctx context.Context, prompt string) (*Response, error) {
    return &Response{Content: p.Response}, nil
}

type TestAssistant struct {
    Response string
}

func (a *TestAssistant) Process(cmd *parser.Command) (string, error) {
    return a.Response, nil
}

// Usage in tests
proc, err := processor.New(cfg, processor.Options{
    Provider:  &TestProvider{Response: "OK"},
    Assistant: &TestAssistant{Response: "OK"},
})
```

3. Split File Operations:
```go
// pkg/processor/processor.go
type FileProcessor interface {
    ProcessFile(path string) error
}

// Implementation that just marks commands
type MarkdownProcessor struct {
    parser parser.Parser
}

func (p *MarkdownProcessor) ProcessFile(path string) error {
    // Just handle ! -> -! transformation
    // No provider or assistant needed
}
```

## Benefits

1. Simpler Testing:
   - Can test command marking without provider
   - Can bypass assistant for simple tests
   - No need for OpenAI config
   - Clear test implementations
   - Isolated testing

2. Better Architecture:
   - Clear dependencies
   - Optional components
   - Interface-based design
   - No global state
   - Flexible configuration

3. Production Improvements:
   - Can swap providers
   - Better error handling
   - Configurable logging
   - Cleaner code
   - Easier maintenance

## Implementation Plan

1. Core Changes:
   - Add processor options
   - Create interfaces
   - Remove global logger
   - Split file operations

2. Test Support:
   - Add mock implementations
   - Create test helpers
   - Update existing tests
   - Add examples

3. Documentation:
   - Update processor docs
   - Add testing guide
   - Document patterns
   - Migration guide

## Migration Strategy

1. Phase 1 - Interfaces:
   - Add options struct
   - Create interfaces
   - Keep defaults
   - No breaking changes

2. Phase 2 - Tests:
   - Add mock implementations
   - Update test helpers
   - Convert existing tests
   - Add examples

3. Phase 3 - Cleanup:
   - Remove global state
   - Update documentation
   - Final testing
   - Release

## Acceptance Criteria

1. TestCommandInvalidation:
   - [ ] Passes without OpenAI config
   - [ ] Uses mock provider
   - [ ] No file operations
   - [ ] Clear test setup
   - [ ] Fast execution

2. Architecture:
   - [ ] Clear interfaces
   - [ ] Optional dependencies
   - [ ] No global state
   - [ ] Proper injection
   - [ ] Separated concerns

3. Testing:
   - [ ] Simple mock implementations
   - [ ] Easy test setup
   - [ ] Fast execution
   - [ ] Clear patterns
   - [ ] Good coverage

4. Documentation:
   - [ ] Interface docs
   - [ ] Testing guide
   - [ ] Migration steps
   - [ ] Examples
