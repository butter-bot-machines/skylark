# Assistant-Provider Integration Test Plan

## Components Under Test

1. Assistant System
   - Command processing
   - Tool execution
   - Provider interaction
   - Response handling

2. Integration Points
   - Assistant -> Provider connection
   - Provider -> Tool invocation
   - Tool -> Provider result flow
   - Error propagation

## Test Categories

### 1. Direct Tool Usage
Test that assistants can handle direct tool commands from users:
```markdown
!default use summarize on this text
```
- Verify tool execution
- Verify result formatting
- Verify error handling

### 2. Provider Tool Calls
Test that assistants can handle tool calls from providers:
```json
{
  "content": "Let me help with that",
  "tool_calls": [
    {
      "id": "call_1",
      "function": {
        "name": "summarize",
        "arguments": "{\"text\":\"test\"}"
      }
    }
  ]
}
```
- Verify tool execution
- Verify result inclusion in next prompt
- Verify final response

### 3. Error Scenarios
Test error handling in various situations:
- Tool not found
- Tool execution failure
- Provider errors
- Invalid tool arguments

### 4. Context Management
Test how context is maintained:
- Tool results in provider context
- Multiple tool calls
- Context size limits
- Message history

## Test Structure

1. Test Fixtures
   - Sample provider responses
   - Tool execution results
   - Error cases

2. Mock Components
   - Provider mock with response queue
   - Tool mock with result verification
   - Sandbox mock for isolation

3. Test Cases
   ```go
   tests := []struct{
       name          string
       command       string    // Input command
       responses     []Response // Provider responses
       toolResults   []string   // Expected tool results
       wantExecuted  bool      // Should tool execute?
       wantRequests  int       // Expected provider requests
       wantResponse  string    // Final expected response
       wantError     bool      // Should error occur?
   }
   ```

## Success Criteria ✓

1. Direct Tool Usage ✓
   - Command parsed correctly (using parser.ParseCommand)
   - Tool executed properly (with real binaries)
   - Result formatted correctly (verified in tests)
   - Errors handled gracefully (execution errors, timeouts)

2. Provider Tool Calls ✓
   - Tool calls detected (in provider responses)
   - Tools executed correctly (with sandboxing)
   - Results included in context (verified in requests)
   - Final response correct (validated in tests)

3. Error Handling ✓
   - Tool not found errors
   - Tool execution failures
   - Provider errors
   - Context size limits
   - Tool timeouts (fast tests with 100ms limit)

4. Context Flow ✓
   - Tool results preserved in context
   - Multiple tool calls handled
   - Size limits enforced (4000 chars per command)
   - Results included in provider context

## Implementation Notes

1. Tool Execution
   - Real Go binaries compiled for tests
   - Sandbox with resource limits
   - Fast timeout tests (0.33s vs 30s)
   - Clean error propagation

2. Test Infrastructure
   - Test-specific resource limits
   - Mock provider with response queues
   - Real tool execution with sandboxing
   - Comprehensive test cases

3. Key Learnings
   - Mock at the right level (provider vs tools)
   - Test real behavior with controlled binaries
   - Keep tests fast with appropriate timeouts
   - Let each component handle its responsibilities

## Implementation Strategy

1. Create Test Infrastructure
   - Define mock interfaces
   - Create test fixtures
   - Set up helper functions

2. Implement Basic Tests
   - Direct tool usage
   - Simple provider calls
   - Basic error cases

3. Add Complex Scenarios
   - Multiple tool calls
   - Context management
   - Error combinations

4. Verify Edge Cases
   - Resource limits
   - Invalid inputs
   - Timeout handling
