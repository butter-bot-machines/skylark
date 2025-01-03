# Implement Filesystem Abstraction

## Problem
Direct filesystem operations make testing difficult:

```go
// Direct os package usage
content, err := os.ReadFile(promptPath)
err := os.WriteFile(path, data, 0644)
err := os.MkdirAll(logDir, 0700)

// Fixed paths
configDir := filepath.Join(cfg.Environment.ConfigDir, "config.yaml")
promptPath := filepath.Join(basePath, "prompt.md")

// Direct file operations
file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
err := os.Rename(a.config.Path, rotatedPath)
```

This means:
1. Tests need real files
2. Tests are slow
3. Tests need permissions
4. Tests are fragile

## Solution

1. Create FileSystem Interface:
```go
// pkg/fs/interface.go
type FileSystem interface {
    // Basic operations
    Read(path string) ([]byte, error)
    Write(path string, []byte) error
    Remove(path string) error
    
    // Directory operations
    MkdirAll(path string, perm os.FileMode) error
    ReadDir(path string) ([]fs.DirEntry, error)
    
    // File operations
    OpenFile(path string, flag int, perm os.FileMode) (File, error)
    Rename(old, new string) error
    
    // Path operations
    Abs(path string) (string, error)
    Join(elem ...string) string
}

type File interface {
    io.ReadWriteCloser
    io.Seeker
    Name() string
    Sync() error
}
```

2. Add Production Implementation:
```go
// pkg/fs/os/filesystem.go
type OSFileSystem struct{}

func (fs *OSFileSystem) Read(path string) ([]byte, error) {
    return os.ReadFile(path)
}

func (fs *OSFileSystem) Write(path string, data []byte) error {
    return os.WriteFile(path, data, 0644)
}

// ... implement other methods
```

3. Add Test Implementation:
```go
// pkg/fs/memory/filesystem.go
type MemoryFileSystem struct {
    files  map[string][]byte
    perms  map[string]os.FileMode
    mu     sync.RWMutex
}

func (fs *MemoryFileSystem) Read(path string) ([]byte, error) {
    fs.mu.RLock()
    defer fs.mu.RUnlock()
    
    data, ok := fs.files[path]
    if !ok {
        return nil, os.ErrNotExist
    }
    return data, nil
}

// ... implement other methods
```

4. Update Components:
```go
// pkg/processor/processor.go
type Processor struct {
    fs FileSystem
    // ...
}

func New(cfg *config.Config, opts Options) (*Processor, error) {
    if opts.FileSystem == nil {
        opts.FileSystem = &fs.OSFileSystem{}
    }
    return &Processor{
        fs: opts.FileSystem,
    }, nil
}

// Use interface instead of direct operations
func (p *Processor) ProcessFile(path string) error {
    content, err := p.fs.Read(path)
    if err != nil {
        return fmt.Errorf("failed to read file: %w", err)
    }
    // ...
    return p.fs.Write(path, newContent)
}
```

## Benefits

1. Testing:
   - Use in-memory filesystem
   - No real files needed
   - Fast execution
   - Predictable behavior

2. Production:
   - Same interface
   - No behavior changes
   - Better error handling
   - Path manipulation

3. Future:
   - Remote filesystems
   - Caching layers
   - Monitoring
   - Access control

## Implementation

1. Core Changes:
   - Create fs package
   - Add interfaces
   - Add implementations
   - Add tests

2. Component Updates:
   - Update processor
   - Update assistant
   - Update config
   - Update security

3. Test Updates:
   - Add test helpers
   - Update existing tests
   - Add examples
   - Verify coverage

## Acceptance Criteria

1. Functionality:
   - [ ] All filesystem operations use interface
   - [ ] No direct os package usage
   - [ ] Production behavior unchanged
   - [ ] Proper error handling

2. Testing:
   - [ ] Tests use memory filesystem
   - [ ] No real files in tests
   - [ ] Fast execution
   - [ ] Good coverage

3. Documentation:
   - [ ] Interface docs
   - [ ] Implementation docs
   - [ ] Test patterns
   - [ ] Examples
