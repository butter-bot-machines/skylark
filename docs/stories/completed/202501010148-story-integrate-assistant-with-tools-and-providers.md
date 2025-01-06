# Story: Integrate Assistant with Tools and Providers (✓ Completed)

## Status
Completed on January 1, 2025 at 02:02
- Implemented tool integration with sandbox
- Added provider integration
- Added context management
- Added comprehensive tests

## Context

Currently, the assistant manager can load assistants and route commands, but it cannot:

1. Execute tools defined in `.skai/tools/`
2. Use AI providers (like OpenAI) for responses
3. Manage tool execution context

This is a critical gap in our architecture.md as shown in the diagram, where the assistant has unconnected relationships to both tools and providers.

## Goal

Enable assistants to execute tools and use AI providers to generate responses, making the system fully functional for command processing.

## Requirements

1. Assistant Tool Integration:

   - Load available tools from `.skai/tools/`
   - Execute tools in sandbox environment
   - Handle tool output in responses
   - Manage tool errors

2. Provider Integration:

   - Connect to configured providers (e.g., OpenAI)
   - Handle rate limiting and retries
   - Manage API keys and authentication
   - Process provider responses

3. Context Management:
   - Pass relevant context to tools
   - Include tool results in provider context
   - Handle context size limits
   - Manage token budgets

## Technical Changes

1. Assistant Manager:

```go
type Assistant struct {
    Name        string
    Tools       []Tool
    Provider    Provider
    Sandbox     *sandbox.Sandbox
}

func (a *Assistant) Process(cmd *Command) (string, error) {
    // Build context from command
    // Execute any tool commands
    // Get response from provider
    // Format and return response
}
```

2. Tool Integration:

```go
func (a *Assistant) ExecuteTool(name string, input []byte) ([]byte, error) {
    // Get tool from manager
    // Validate input against schema
    // Execute in sandbox
    // Handle and format result
}
```

3. Provider Integration:

```go
func (a *Assistant) GetResponse(prompt string) (string, error) {
    // Build provider request
    // Include tool results
    // Handle rate limits
    // Process response
}
```

## Success Criteria

1. Commands can use tools:

```markdown
!default Use the summarize tool on # Introduction

> Using summarize tool...
> Summary: The introduction covers key concepts...
```

2. Assistants use AI providers:

```markdown
!default What's the weather like?

> Let me check using the weather tool...
> According to the weather tool, it's currently 72°F...
```

3. Tools execute safely:

- Resource limits enforced
- Proper error handling
- Clean sandbox cleanup

4. Provider integration works:

- Rate limits respected
- Errors handled gracefully
- Responses properly formatted

## Non-Goals

1. Adding new tools or providers
2. Changing tool sandbox implementation
3. Modifying provider interfaces
4. Changing command syntax

## Testing Plan

1. Unit Tests:

   - Tool execution
   - Provider integration
   - Context management
   - Error handling

2. Integration Tests:

   - End-to-end command flow
   - Tool + provider interaction
   - Resource management
   - Error scenarios

3. Performance Tests:
   - Tool execution overhead
   - Provider response times
   - Memory usage patterns

## Risks

1. Tool execution security
2. Provider rate limits
3. Context size management
4. Error handling complexity

## Future Considerations

1. Tool result caching
2. Provider fallbacks
3. Parallel tool execution
4. Enhanced context management

## Acceptance Criteria

1. Tool Usage:

```markdown
# Test Tools

!default Summarize this text using the summarize tool

> Using summarize tool...
> Summary: The text discusses...

!default What's the weather in San Francisco?

> Using weather tool...
> Current conditions in San Francisco:
> Temperature: 65°F
> Conditions: Partly cloudy
```

2. Error Handling:

```markdown
!default Use missing_tool

> Error: Tool 'missing_tool' not found

!default Use weather_tool

> Error: Weather API rate limit exceeded, please try again in 30 seconds
```

3. Resource Management:

- Tools respect CPU/memory limits
- Provider rate limits honored
- Context sizes managed properly

4. Logging:

```
time=2025-01-01T12:00:00Z level=INFO msg="executing tool" name=summarize
time=2025-01-01T12:00:01Z level=INFO msg="tool completed" name=summarize duration=1.2s
time=2025-01-01T12:00:01Z level=INFO msg="getting provider response" provider=openai
```
