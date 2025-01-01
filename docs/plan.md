# Skylark Implementation Plan

Skylark is a Go-based system that transforms Markdown documents through AI-powered commands. At its core, it watches for file changes and processes inline commands that begin with `!`, routing them through assistants that can leverage specialized tools. The system is built around a simple, file-based architecture where all configuration and extensions live in a `.skai/` directory.

## Phase 1: Core Infrastructure

The foundation of Skylark centers on file monitoring and command processing. This phase establishes the core system that watches Markdown files, detects commands, and manages the basic assistant infrastructure. The focus is on creating a reliable pipeline from file changes to command detection to assistant routing.

### 1.1 Project Setup

- Initialize Go module with dependency management
- Set up project structure following Go standards
- Establish testing framework (stdlib):
  - Test fixtures for markdown files:
    * Command parsing tests (assistant names, context)
    * Reference resolution tests (block types, matching)
    * Integration test scenarios
  - Test structure:
    * Unit tests in *_test.go files
    * Test data in testdata/ directory
    * Shared test helpers
  - Success criteria:
    * All test cases pass
    * Coverage meets targets
    * Edge cases handled

### 1.2 File Watcher System
- Implement basic file system monitoring
  * Use fsnotify for file change detection
  * Filter for .md extension
  * Handle file deletion/renaming
- Create change event handling system
  * Debounce rapid changes:
    - Default delay: 500ms
    - Configurable via config.yaml
    - Maximum delay: 2000ms
  * Queue system for file modifications:
    - In-memory queue with disk backup
    - Maximum queue size: 1000 events
    - FIFO processing order
  * File state cache:
    - LRU cache with 1000 entry limit
    - Stores file hash and last modified time
    - Purge strategy: LRU with 1 hour TTL
- Develop concurrent processing pipeline
  * Worker pool:
    - Default size: num_cpu * 2
    - Configurable via config.yaml
    - Maximum size: 32 workers
  * File locking mechanism:
    - Advisory locks using flock
    - Timeout after 5 seconds
    - Retry 3 times with exponential backoff

### 1.3 Command Parser
- Build command parser with specific regex patterns:
  - Command line: `^!(?:([a-zA-Z-]+)\s+)?(.+)$`
  - Reference: `#\s*([^#\n]+?)(?:\s*#|$)`
- Implement assistant name resolution:
  - Case-insensitive matching
  - Simple lowercase normalization (preserve format)
  - Default assistant fallback
- Add content reference parsing:
  - Flexible header matching:
    * Partial matches allowed
    * Case/whitespace/punctuation insensitive
    * Multiple matches supported
  - Block type handling:
    * Headers with hierarchy
    * Lists as individual items
    * Complete paragraphs/quotes/tables/code
  - Context rules:
    * Current section content
    * Parent header hierarchy
    * Sibling section content

- Create context extraction system:
  * Section boundary detection algorithm:
    1. Start from command line
    2. Include all content until next header of same or higher level
    3. Include parent header if within nested structure
    4. Maximum content size of 4000 chars per section
  * Parent/sibling header inclusion rules:
    1. Include immediate parent header
    2. Include previous sibling header if exists
    3. Include next sibling header if exists
    4. Stop at document root or section boundaries
  * Reference content assembly:
    1. Process references in order of appearance
    2. Deduplicate overlapping sections
    3. Maintain original document order
    4. Total context limit of 8000 chars
  * Multiple command handling:
    1. Process commands in document order
    2. Maximum 10 commands per file
    3. Parallel processing when independent
    4. Sequential for dependent commands

### 1.4 Assistant Manager
- Create assistant loading system:
  - YAML front-matter parsing with strict schema
  - Prompt content extraction
  - Knowledge directory handling
- Implement configuration validation:
  - Model parameter validation
  - Tool reference checking
  - Required field verification

- Build inheritance system:
  * Parameter resolution order:
    1. Assistant-specific config in prompt.md
    2. Provider defaults in config.yaml
    3. Global defaults in config.yaml
    4. System-wide fallbacks
  * Tool description inheritance:
    1. Assistant-specific description
    2. Tool's default description
    3. Generated description from schema
  * Model configuration merging:
    1. Override single values
    2. Deep merge for nested objects
    3. Array replacement (no merging)
  * Knowledge directory structure:
    ```
    knowledge/
    ├── index.json       # Directory manifest
    ├── concepts/        # Core domain concepts
    ├── examples/        # Usage examples
    └── references/      # External references
    ```

## Phase 2: Integration Layer
With the core pipeline established, this phase focuses on the components that give Skylark its power: tools and AI integration. Tools are Go programs that follow a strict interface with --usage and --health endpoints, while assistants are configured through YAML front-matter and markdown content. This phase connects these pieces into a cohesive system.
### 2.1 Tool System Foundation
- Create tool compilation system:
  - File change detection for .go files
  - Incremental compilation
  - Binary versioning
- Implement tool interface:
  - --usage JSON schema validation
  - --health status checking
  - Standard error format
- Add environment management:
  - Variable resolution order:
    1. Config file
    2. Tool defaults
    3. System environment
  - Secret handling
  - Environment isolation

- Build execution pipeline:
  * Process spawning with timeout:
    - Default timeout: 30 seconds
    - Configurable per tool in config.yaml
    - Maximum timeout: 300 seconds
  * Environment resolution order:
    1. Tool-specific vars in config.yaml
    2. Global vars in config.yaml
    3. Tool --usage defaults
    4. System environment
  * Output sanitization:
    1. Remove ANSI escape sequences
    2. Convert line endings to \n
    3. UTF-8 encoding validation
    4. Maximum output size: 1MB
  * stdin/stdout protocol:
    ```json
    {
      "input": {
        "content": string,
        "metadata": object
      }
    }
    ```
    ```json
    {
      "output": {
        "result": any,
        "error": string|null
      }
    }
    ```

### 2.2 Configuration Management

- Implement config.yaml parsing:
  - Version validation
  - Schema enforcement
  - Environment expansion
- Create environment resolution:
  - Hierarchical lookup
  - Default value handling
  - Override mechanics
- Add model configuration:
  - Provider-specific settings
  - Parameter validation
  - Token limit handling
- Build dependency system:
  - Tool dependency graph
  - Circular reference detection
  - Version compatibility

### 2.3 AI Integration

- Add provider interface:
  - OpenAI API integration
  - Rate limiting
  - Error handling
- Create prompt assembly:
  - Template rendering
  - Context injection
  - Tool availability
- Build response handling:
  - Output formatting:
    ```markdown
    !command text

    > Generated response
    ```
  - Error formatting:
    ```markdown
    !command text

    > Error: description
    ```
  - Multi-line handling

### 2.4 Basic CLI

- Create command structure:
  - init: Project initialization
  - watch: Start file watcher
  - run: One-time processing
  - version: Version info
- Implement project initialization:
  - Directory structure creation
  - Default assistant setup
  - Sample tool installation
- Add configuration validation:
  - Config file verification
  - Assistant validation
  - Tool health checks

## Phase 3: Advanced Features

Building on the foundation and integration layers, this phase implements Skylark's more sophisticated features. This includes the Emmet-inspired reference system for including document context, concurrent command processing, and a robust tool execution environment. These features transform Skylark from a basic command processor into a powerful document augmentation system.

### 3.1 Context Management

- Implement reference resolution:
  - Markdown parsing rules
  - Header hierarchy tracking
  - Content boundary detection
- Create context assembly:
  - Reference deduplication
  - Content ordering
  - Context limitation
- Add token management:
  - Count estimation
  - Dynamic truncation
  - Priority-based inclusion

### 3.2 Tool Enhancement

- Add sandboxing:
  - Process isolation
  - Resource limits
  - Network restrictions
- Implement versioning:
  - Binary versioning
  - Compatibility checking
  - Update mechanism
- Create dependency handling:
  - Tool chaining
  - Error propagation
  - Result caching

### 3.3 Concurrent Processing

- Implement worker pool:
  - Size configuration
  - Priority queuing
  - Resource allocation
- Add rate limiting:
  - Per-model limits
  - Tool execution limits
  - Adaptive throttling
- Create queue management:
  - Command batching
  - Priority scheduling
  - Failure handling

### 3.4 Error Handling & Recovery

- Implement error types:
  - Configuration errors
  - Runtime errors
  - Tool errors
  - AI provider errors
- Add recovery strategies:
  - Retry logic
  - Fallback options
  - State recovery

### 3.5 Logging

- Implement structured logging with slog:
  - Log levels (Debug, Info, Warn, Error)
  - Source code information
  - JSON and text output formats
  - Context and attribute support
- Add configuration options:
  - Output destination
  - Log level control
  - Format selection
  - Source location

## Phase 4: Polish & Documentation

The final phase focuses on hardening Skylark for production use. This includes comprehensive testing, security measures for tool execution, proper error handling with user-friendly messages, and thorough documentation. The goal is to ensure Skylark is both powerful for advanced users and approachable for newcomers.

### 4.1 Testing & Validation

- Implement unit test suite
- Add integration tests
- Create performance tests
- Build security tests
- Develop stress testing

### 4.2 Security Hardening

- Implement input validation
- Add API key management
- Create file access controls
- Build security audit system
- Develop threat monitoring

### 4.3 Documentation

- Create user documentation
- Add developer guides
- Build API documentation
- Create example collection
- Write troubleshooting guide

### 4.4 Performance Optimization

- Implement performance profiling
- Add caching strategies
- Create memory optimization
- Build response time improvement
- Develop resource usage monitoring

## Technical Details

### Core Components

#### File Watcher

```go
type FileWatcher struct {
    watchedPaths map[string]bool
    eventQueue   chan FileEvent
    processor    *CommandProcessor
}

type FileEvent struct {
    Path    string
    Type    EventType // Created, Modified, Deleted
    Content []byte
}
```

#### Command Processor

```go
type CommandProcessor struct {
    assistantMgr *AssistantManager
    toolMgr      *ToolManager
    parser       *CommandParser
}

type Command struct {
    Assistant string
    Prompt    string
    References []Reference
}
```

#### Assistant Manager

```go
type AssistantManager struct {
    assistants map[string]*Assistant
    config     *Config
    cache      *Cache
}

type Assistant struct {
    Name        string
    Config      AssistantConfig
    Tools       []Tool
    Prompt      string
}
```

#### Tool System

```go
type ToolManager struct {
    tools       map[string]*Tool
    compiler    *Compiler
    envManager  *EnvManager
}

type Tool struct {
    Name     string
    Schema   ToolSchema
    Binary   string
    Health   bool
}
```

### Key Interfaces

#### Assistant Interface

```go
type AssistantInterface interface {
    Load(path string) error
    Process(cmd Command) (Response, error)
    ValidateConfig() error
}
```

#### Tool Interface

```go
type ToolInterface interface {
    Compile() error
    CheckHealth() bool
    Execute(input []byte) ([]byte, error)
    ValidateSchema() error
}
```

#### AI Provider Interface

```go
type AIProvider interface {
    GenerateResponse(prompt string, config Config) (string, error)
    ValidateTokens(text string) (int, error)
    GetModelLimits() ModelLimits
}
```

## Implementation Notes

### Error Handling Strategy

- Use custom error types for different failure scenarios
- Implement graceful degradation
- Provide clear user feedback
- Log detailed error information
- Enable error recovery where possible

### Performance Considerations

- Implement efficient file watching
- Use goroutines for concurrent processing
- Cache compiled tools and configurations
- Optimize context assembly
- Monitor resource usage

### Security Measures

- Validate all user input
- Sandbox tool execution
- Secure API key storage
- Implement access controls
- Monitor security events

### Testing Strategy

- Unit tests for core components
- Integration tests for system flow
- Performance benchmarks
- Security testing
- User acceptance testing
