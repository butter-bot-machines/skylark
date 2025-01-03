# Implement Process Abstraction

## Problem
Direct process management makes testing difficult:

```go
// Direct process creation
cmd := exec.Command("go", "build", "-o", binaryPath, mainFile)
cmd.Dir = toolPath
output, err := cmd.CombinedOutput()

// Direct process control
cmd.SysProcAttr = &syscall.SysProcAttr{
    Setpgid: true,
}
syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)

// Direct resource limits
syscall.Setrlimit(RLIMIT_NPROC, &rLimit)
syscall.Setrlimit(syscall.RLIMIT_FSIZE, &rLimit)
syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)

// Direct pipe management
stdin, err := cmd.StdinPipe()
stdout, err := cmd.StdoutPipe()
```

This means:
1. Tests need real processes
2. Tests need system privileges
3. Tests are slow
4. Tests are platform-dependent

## Solution

1. Create Process Interface:
```go
// pkg/process/interface.go
type Process interface {
    // Basic operations
    Start() error
    Wait() error
    Signal(sig os.Signal) error
    
    // IO operations
    SetStdin(io.Reader)
    SetStdout(io.Writer)
    SetStderr(io.Writer)
    
    // Resource operations
    SetLimits(ResourceLimits) error
    GetLimits() ResourceLimits
    
    // State
    ID() int
    Running() bool
    ExitCode() int
}

type Manager interface {
    // Process creation
    New(name string, args []string) Process
    Get(pid int) (Process, error)
    List() []Process
    
    // Resource management
    SetDefaultLimits(ResourceLimits)
    GetDefaultLimits() ResourceLimits
}

type ResourceLimits struct {
    MaxCPUTime    time.Duration
    MaxMemoryMB   int64
    MaxFileSizeMB int64
    MaxFiles      int64
    MaxProcesses  int64
}
```

2. Add Production Implementation:
```go
// pkg/process/os/manager.go
type OSProcessManager struct {
    defaultLimits ResourceLimits
    processes     map[int]*OSProcess
    mu           sync.RWMutex
}

type OSProcess struct {
    cmd     *exec.Cmd
    limits  ResourceLimits
    started bool
}

func (p *OSProcess) Start() error {
    if err := p.applyLimits(); err != nil {
        return err
    }
    return p.cmd.Start()
}

// ... implement other methods
```

3. Add Test Implementation:
```go
// pkg/process/memory/manager.go
type MemoryProcessManager struct {
    processes map[int]*MemoryProcess
    nextPID   int
    mu        sync.RWMutex
}

type MemoryProcess struct {
    id       int
    name     string
    args     []string
    running  bool
    exitCode int
    stdin    io.Reader
    stdout   io.Writer
}

func (p *MemoryProcess) Start() error {
    p.running = true
    return nil
}

// ... implement other methods
```

4. Update Components:
```go
// pkg/sandbox/sandbox.go
type Sandbox struct {
    procMgr process.Manager
    // ...
}

func New(cfg *config.Config, opts Options) (*Sandbox, error) {
    if opts.ProcessManager == nil {
        opts.ProcessManager = &process.OSProcessManager{}
    }
    return &Sandbox{
        procMgr: opts.ProcessManager,
    }, nil
}

// Use interface instead of direct operations
func (s *Sandbox) Execute(name string, args []string) error {
    proc := s.procMgr.New(name, args)
    proc.SetLimits(s.limits)
    
    if err := proc.Start(); err != nil {
        return fmt.Errorf("failed to start process: %w", err)
    }
    
    return proc.Wait()
}
```

## Benefits

1. Testing:
   - Use in-memory processes
   - No real processes needed
   - Fast execution
   - Platform-independent

2. Production:
   - Same interface
   - No behavior changes
   - Better resource control
   - Proper cleanup

3. Future:
   - Remote execution
   - Process pools
   - Better monitoring
   - Resource quotas

## Implementation

1. Core Changes:
   - Create process package
   - Add interfaces
   - Add implementations
   - Add tests

2. Component Updates:
   - Update sandbox
   - Update tool manager
   - Update resource limits
   - Update security

3. Test Updates:
   - Add test helpers
   - Update existing tests
   - Add examples
   - Verify coverage

## Acceptance Criteria

1. Functionality:
   - [ ] All process operations use interface
   - [ ] No direct exec/syscall usage
   - [ ] Production behavior unchanged
   - [ ] Proper resource management

2. Testing:
   - [ ] Tests use memory processes
   - [ ] No real processes in tests
   - [ ] Fast execution
   - [ ] Platform-independent

3. Documentation:
   - [ ] Interface docs
   - [ ] Implementation docs
   - [ ] Test patterns
   - [ ] Examples
