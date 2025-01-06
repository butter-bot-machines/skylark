# Story: Implement Command Invalidation (✓ Completed)

## Status
Completed on January 2, 2025 at 18:06
- Implemented command invalidation by replacing `!` with `-!`
- Updated command parsing to ignore `-!` prefixed commands
- Modified response insertion to convert `!` to `-!` for processed commands
- Ensured proper handling of whitespace and indentation
- Added test cases to verify command invalidation behavior

## Background
Currently, skylark enters an infinite loop when processing commands because it cannot distinguish between fresh commands and already-processed ones. When the processor updates a file with a command response, the file watcher detects this change and reprocesses the file, creating an infinite cycle.

## Requirements

### Command Invalidation
- Processor should mark processed commands by replacing `!` with `-!`
- Processor should ignore commands that start with `-!`
- This change should be made when inserting the response
- Original command text should be preserved, only the prefix changes

### Implementation
- Ensure command parsing ignores `-!` prefixed commands
- Modify response insertion to convert `!` to `-!` for processed commands
- Ensure proper handling of whitespace and indentation
- Maintain command history in the file

## Technical Details

### Command Processing Changes
Before:
```markdown
!command
[response]
```

After:
```markdown
-!command
[response]
```

### Dependencies
- pkg/processor: Update command parsing and response insertion
- pkg/parser: Modify command detection logic

## Acceptance Criteria
1. [x] Processor correctly identifies and ignores `-!` prefixed commands
2. [x] Commands are properly marked as processed with `-!` prefix
3. [x] No infinite processing loops occur
4. [x] Command history is preserved in files
5. [x] Whitespace and indentation are preserved
6. [x] Tests verify command invalidation behavior

## Impact Assessment

### File Changes (2-3 files)
1. pkg/processor/processor.go:
   - Update ProcessFile() to ignore `-!` commands
   - Modify response insertion to add `-!` prefix

2. pkg/parser/parser.go:
   - Update command detection regex
   - Add invalidation check

3. test/integration/integration_test.go (optional):
   - Add command invalidation test cases

### Test Changes (10-15 new test cases)
1. TestCommandInvalidation (5-6 cases):
   - Basic command invalidation
   - Whitespace preservation
   - Multiple commands in file
   - Already invalidated commands

2. TestProcessorBehavior (5-6 cases):
   - Command parsing with invalidation
   - Response insertion with prefix
   - Edge cases (empty commands, etc)

3. TestIntegration (3-4 cases):
   - End-to-end command processing
   - No infinite loops

### Level of Effort: Low
- Small, focused changes
- Clear implementation path
- Minimal dependencies
- Straightforward testing

### Risk Assessment
- Low: Simple prefix-based solution
- Low: No database/config changes
- Low: Easily reversible if issues
- Medium: Existing file compatibility

## Related
- Vision.md: Command processing
- Architecture.md: Processor design
