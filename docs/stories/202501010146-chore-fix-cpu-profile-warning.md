# Chore: Fix CPU Profile Warning

## Context
When running the `skylark run` command, we see a runtime warning:
```
runtime: cannot set cpu profile rate until previous profile has finished.
```
This warning suggests a potential issue with CPU profiling in our resource management code.

## Goal
Fix the CPU profile warning to ensure clean runtime operation and proper resource management.

## Requirements
1. Identify source of CPU profile warning
2. Fix the issue without compromising resource limits
3. Ensure worker pool performance is maintained

## Technical Investigation
1. Check worker pool resource limits:
   - CPU time limits
   - Profile rate settings
   - Resource enforcement code

2. Check potential causes:
   - Multiple profile attempts
   - Profile cleanup issues
   - Resource limit implementation

## Success Criteria
1. No runtime warnings during command execution
2. Resource limits still properly enforced
3. Worker pool performance unchanged

## Non-Goals
1. Changing resource limit functionality
2. Modifying worker pool architecture
3. Adding new profiling features

## Testing Plan
1. Run command with different file counts
2. Verify resource limits still work
3. Check performance metrics
4. Test error scenarios

## Risks
1. Resource limits might be affected
2. Performance impact
3. Hidden dependencies

## Acceptance Criteria
1. Clean output:
```bash
$ skylark run
Processing files: 23 queued
[===>    ] 7/23 files processed
Successfully processed 23 files
```

2. Resource Management:
- CPU limits still enforced
- Memory limits maintained
- No runtime warnings
