# Story: Implement Watch Command and Component Integration (✓ Completed)

## Status
Completed on January 1, 2024 at 02:20
- Implemented watch command with timeout
- Connected watcher, queue, and worker pool
- Added graceful shutdown
- Added comprehensive testing support

## Context
Currently, we have several key components implemented but not connected:
1. File watcher (implemented but not connected to queue)
2. Job queue (implemented but not receiving events)
3. Worker pool (connected to processor but not receiving jobs)

This creates a gap in our continuous processing pipeline, where file changes aren't automatically processed.

## Goal
Implement the watch command and connect all components to enable continuous file monitoring and processing.

## Requirements
1. Watch Command Implementation:
   - Start file watcher with configured paths
   - Handle file change events
   - Support graceful shutdown
   - Show real-time status

2. Component Integration:
   - Connect file watcher to job queue
   - Connect job queue to worker pool
   - Connect worker pool to processor
   - Ensure proper error propagation

3. Event Flow:
   - File changes trigger watcher events
   - Events get queued as jobs
   - Workers process jobs through processor
   - Results update files

4. Resource Management:
   - Debounce rapid file changes
   - Control queue size
   - Manage worker concurrency
   - Handle backpressure

## Implementation Notes

1. Testing:
   - Add timeout parameter for testing: `skylark watch --timeout 5s`
   - Allows automated testing without hanging
   - Default to no timeout for normal use
   - Use time.After for timeout implementation

## Technical Changes

1. Watch Command:
```go
func (c *CLI) Watch(args []string) error {
    // Create components
    watcher := watcher.New(c.config)
    queue := worker.NewQueue(c.config)
    pool := worker.NewPool(c.config)
    processor := processor.New(c.config)

    // Connect components
    watcher.OnChange(func(event FileEvent) {
        queue.Push(NewFileJob(event))
    })

    pool.OnJob(func(job Job) error {
        return processor.ProcessFile(job.Path)
    })

    // Start watching
    return watcher.Start()
}
```

2. Event Handling:
```go
type FileEvent struct {
    Path      string
    Operation Op
    Timestamp time.Time
}

func (w *Watcher) handleEvent(event FileEvent) {
    // Debounce events
    if w.shouldDebounce(event) {
        return
    }

    // Create and queue job
    job := worker.NewFileJob(event.Path)
    w.queue.Push(job)
}
```

3. Job Processing:
```go
func (p *Pool) processJob(job Job) {
    // Apply resource limits
    ctx := context.WithTimeout(context.Background(), p.config.JobTimeout)

    // Process through pipeline
    if err := p.processor.ProcessFile(job.Path); err != nil {
        job.OnFailure(err)
        return
    }

    job.OnSuccess()
}
```

## Success Criteria
1. Watch Command Works:
```bash
$ skylark watch
Watching for changes...
Processing test.md...
Successfully processed test.md
```

2. File Changes Processed:
```bash
$ echo "!help What can you do?" >> test.md
Processing test.md...
Successfully processed test.md
```

3. Resource Management:
- Multiple rapid saves only trigger one processing
- Queue doesn't overflow
- Workers stay within limits

4. Error Handling:
```bash
$ skylark watch
Watching for changes...
Error processing bad.md: invalid command format
Continuing to watch...
```

## Non-Goals
1. Changing existing component implementations
2. Modifying file update logic
3. Changing configuration format
4. Adding new commands

## Testing Plan
1. Unit Tests:
   - Watch command setup
   - Event handling
   - Component integration
   - Error scenarios

2. Integration Tests:
   - End-to-end file processing
   - Multiple file changes
   - Resource limit verification
   - Error propagation

3. Performance Tests:
   - Rapid file changes
   - Queue behavior
   - Worker scaling
   - Memory usage

## Risks
1. Race conditions in event handling
2. Memory leaks from long-running process
3. File system edge cases
4. Resource exhaustion

## Future Considerations
1. Watch path configuration
2. Event filtering improvements
3. Job prioritization
4. Status UI improvements

## Acceptance Criteria
1. Command Usage:
```bash
$ skylark watch
Watching for changes...
[2024-01-01 02:03:00] Processing: test.md
[2024-01-01 02:03:01] Success: test.md
[2024-01-01 02:03:05] Error: bad.md (invalid command)
```

2. Resource Management:
- Debounce works (test with rapid saves)
- Queue size stays within limits
- Worker count respects config

3. Error Handling:
- Individual file errors don't stop watching
- Resource limits enforced
- Clear error messages

4. Logging:
```
time=2024-01-01T02:03:00Z level=INFO msg="started watching" paths=1
time=2024-01-01T02:03:01Z level=DEBUG msg="file changed" path=test.md
time=2024-01-01T02:03:01Z level=DEBUG msg="queued job" path=test.md
time=2024-01-01T02:03:01Z level=INFO msg="processed file" path=test.md
