# Story: Implement Watch Command

## Background
The Watch command should monitor the project folder by convention, respecting .gitignore patterns. While most users will likely use a flat directory structure initially, we should support nested directories through command flags for future extensibility.

## Requirements

### Project Convention
- Watch command monitors project folder by convention
- No configuration needed in config.yaml
- Respect .gitignore patterns if present
- Filter for .md files only

### Command Line Interface
- Add --recursive flag (default: false)
- Add --exclude flag for additional exclusions
- Keep existing --timeout flag

### Implementation
- Remove watch settings from config.yaml
- Use debouncing for rapid changes
- Support nested directories when --recursive flag is used

## Technical Details

### Config Changes
Remove from config.yaml:
```yaml
file_watch:
  debounce_delay: "500ms"
  max_delay: "2s"
  extensions:
    - ".md"

watch_paths:
  - "."
```

### Command Interface
```bash
# Watch project folder (default)
skai watch

# Watch with timeout
skai watch --timeout 5m

# Watch recursively
skai watch --recursive

# Watch with additional exclusions
skai watch --exclude="output,temp"
```

### Dependencies
- pkg/watcher: Add .gitignore support
- pkg/cmd: Update Watch command flags
- pkg/config: Remove watch settings

## Acceptance Criteria
1. [ ] Watch command monitors project folder by convention
2. [ ] .gitignore patterns are respected
3. [ ] Config.yaml no longer contains watch settings
4. [ ] Changes are processed automatically on save
5. [ ] Only .md files trigger processing
6. [ ] Rapid changes are properly debounced
7. [ ] Recursive watching works when flag is used
8. [ ] Tests verify basic and recursive watching

## Impact Assessment

### File Changes (6 files)
1. pkg/cmd/cmd.go:
   - Remove watch config sections
   - Add new flags (--recursive, --exclude)
   - Update Watch command implementation

2. pkg/cmd/cmd_test.go:
   - Add TestCLIWatch test cases
   - Test flag handling
   - Test .gitignore respect

3. pkg/watcher/watcher.go:
   - Add .gitignore support
   - Update New() to not require config
   - Add recursive watching support

4. pkg/watcher/watcher_test.go:
   - Add .gitignore test cases
   - Add recursive watching tests

5. pkg/config/config.go:
   - Remove FileWatchConfig struct
   - Remove WatchPaths field

6. test/integration/integration_test.go:
   - Update TestWatcherWorkerIntegration
   - Add recursive test cases
   - Add .gitignore test cases
   - Complete TestEndToEnd

### Test Changes (15-20 new test cases)
1. TestCLIWatch (5-6 cases):
   - Basic watching
   - Flag handling
   - .gitignore respect

2. TestWatcher (5-6 cases):
   - Recursive watching
   - .gitignore patterns
   - Exclusion patterns

3. TestIntegration (5-8 cases):
   - Complete TestEndToEnd
   - Recursive scenarios
   - .gitignore scenarios

### Level of Effort: Medium
- 6 files to modify
- 15-20 new test cases
- Core functionality exists
- Main work in testing and .gitignore support

### Risk Assessment
- Low: Basic flat directory watching
- Medium: Recursive directory support
- Low: .gitignore integration
- Medium: Integration test reliability

## Related
- Vision.md: File watching requirements
- Configuration.spec.md: Config structure
