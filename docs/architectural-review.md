# Architectural Review

## Code Quality Assessment

### Test Coverage and Quality
- Comprehensive test suite across packages
- Most packages pass tests successfully
- Some cgroup-related tests skipped due to permission issues (expected behavior)

### Critical Issues

1. **Race Conditions**:
   - In pkg/timing/clock_test.go:
     - Data race in TestClock_AfterFunc between test goroutine and timer callback
     - Concurrent read/write access to shared data
   - In test/performance/performance_test.go:
     - Data race in TestWorkerPoolConcurrency
     - Multiple goroutines concurrently calling SetDefaultLimits on mockProcessManager
     - Race condition in worker pool initialization

2. **Code Style Issues**:
   - Unkeyed struct fields in:
     - pkg/process/memory/manager.go
     - pkg/process/os/manager.go
     - pkg/process/os/cgroups_linux.go
   - Unreachable code in:
     - pkg/errors/errors_test.go
     - test/security/security_test.go

## Tight Coupling Assessment

This section identifies problematic tight coupling ("new is glue") between packages in the Skylark codebase.

### Critical Coupling Issues

#### 1. pkg/processor/concrete/processor.go
Most concerning package with tight coupling to:
```go
import (
    "github.com/butter-bot-machines/skylark/pkg/assistant"
    "github.com/butter-bot-machines/skylark/pkg/config"
    "github.com/butter-bot-machines/skylark/pkg/logging"
    "github.com/butter-bot-machines/skylark/pkg/parser"
    "github.com/butter-bot-machines/skylark/pkg/process"
    procesos "github.com/butter-bot-machines/skylark/pkg/process/os"
    "github.com/butter-bot-machines/skylark/pkg/provider"
    "github.com/butter-bot-machines/skylark/pkg/provider/openai"
    "github.com/butter-bot-machines/skylark/pkg/provider/registry"
    "github.com/butter-bot-machines/skylark/pkg/sandbox"
    "github.com/butter-bot-machines/skylark/pkg/tool"
)
```
**Issue**: Directly imports and instantiates concrete implementations from multiple packages

#### 2. pkg/cmd/cmd.go
Tightly coupled to concrete implementations:
```go
import (
    "github.com/butter-bot-machines/skylark/pkg/processor/concrete"
    wconcrete "github.com/butter-bot-machines/skylark/pkg/watcher/concrete"
    wkconcrete "github.com/butter-bot-machines/skylark/pkg/worker/concrete"
)
```
**Issue**: Directly imports concrete packages rather than interfaces

#### 3. pkg/assistant/assistant.go
Heavy coupling to multiple subsystems:
```go
import (
    "github.com/butter-bot-machines/skylark/pkg/parser"
    "github.com/butter-bot-machines/skylark/pkg/provider/registry"
    "github.com/butter-bot-machines/skylark/pkg/provider"
    "github.com/butter-bot-machines/skylark/pkg/sandbox"
    "github.com/butter-bot-machines/skylark/pkg/tool"
)
```
**Issue**: Direct dependencies on multiple implementation packages

### Architectural Violations

1. **Concrete Implementation Dependencies**
   - pkg/processor/concrete imports pkg/process/os directly
   - pkg/cmd imports concrete implementations from multiple packages
   - pkg/assistant imports provider/registry directly

2. **Cross-Layer Violations**
   - Infrastructure packages (logging, config) imported everywhere
   - Security implementations directly coupled to config
   - Process management directly coupled to OS-specific code

3. **Direct Provider Implementation**
   - OpenAI provider directly instantiated in processor
   - Provider registry directly used in assistant
   - No abstraction layer for provider creation

4. **Global State**
   - pkg/processor/concrete: `var logger *slog.Logger`
   - pkg/parser: `var logger *slog.Logger`
   - pkg/provider/openai: `var apiURL = "https://api.openai.com/v1/chat/completions"`
   - pkg/sandbox: `var DefaultLimits = ResourceLimits{}`
   - pkg/provider: `var DefaultRequestOptions = &RequestOptions{}`

5. **Hidden Dependencies**
   - Multiple packages using global loggers
   - Default options/limits as package globals
   - Hardcoded external service URLs
   - Regular expressions as package globals

6. **Init Function Coupling**
   - pkg/processor/concrete: init() creates global logger
   - pkg/parser: init() likely creates global logger
   - Hidden initialization of global state
   - No control over logger configuration

7. **Network Policy Coupling**
   - Hardcoded allowed hosts: "api.openai.com"
   - Direct coupling to external services
   - Network policy created inside processor package
   - No configuration abstraction for network policies

8. **Interface Design Issues**
   - pkg/assistant: `toolManager interface` defined in implementation package
   - pkg/provider/openai: `Tool interface` tightly coupled to OpenAI implementation
   - pkg/security/interface: `AuditLogger interface` exposes concrete types
   - Interfaces accepting concrete types rather than interfaces
   - Some interfaces defined alongside implementations

9. **Interface Segregation Violations**
   - pkg/processor: Large `ProcessManager` interface (4 methods)
   - pkg/security: `Manager` interface combines multiple concerns
   - pkg/fs: `FS` interface combines read/write operations
   - pkg/worker: `Pool` interface has mixed responsibilities

### Clean Examples

1. **pkg/tool/tool.go**
   - Clean imports through interfaces
   - Uses builtins package appropriately
   - Proper abstraction layers

2. **pkg/fs/memory/fs.go**
   - No direct concrete dependencies
   - Interface-based design
   - Clean separation of concerns

### Recommendations

1. **Immediate Actions**
   - Fix identified race conditions in timing and worker packages
   - Use keyed fields in struct literals across process package
   - Remove unreachable code in test files
   - Move concrete implementations to separate packages (e.g., pkg/processor/concrete -> pkg/internal/processor)
   - Create factory interfaces for component creation
   - Use dependency injection for cross-package dependencies

2. **Structural Changes**
   - Create provider factory system
   - Abstract OS-specific implementations
   - Implement proper dependency injection container
   - Address thread safety in timing package
   - Improve worker pool concurrency handling

3. **Package Organization**
   - Move concrete implementations to internal/
   - Use interfaces at package boundaries
   - Create factory packages for instantiation
