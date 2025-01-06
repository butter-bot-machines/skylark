# Story: Implement Assistant-Provider Integration

## Status

Completed on January 1, 2025 at 08:15

- Implemented assistant-provider integration
- Added tool execution support
- Added context management
- Added error handling and logging

## Context

Current Assistant-Provider integration status:

1. ✓ Assistant system implemented
2. ✓ Provider system completed
3. ✓ Tool system integrated
4. ✓ Command processing
   - ✓ Provider routing
   - ✓ Tool execution
   - ✓ Context building
   - ✓ Error handling
5. ✓ Resource management
   - ✓ Tool sandboxing
   - ✓ Error propagation
   - ✓ Structured logging

The integration between assistants and providers is now complete, enabling:
- Command routing to providers
- Tool execution through providers
- Context management
- Proper error handling
- Resource cleanup

## Goal

Implement the integration between assistants and providers to enable AI-powered command processing.

## Requirements

1. Command Processing:

   - Route commands to providers
   - Handle provider responses
   - Support conversation context
   - Execute tools when needed

2. Context Management:

   - Build provider context
   - Track conversation history
   - Handle tool results
   - Manage state

3. Tool Integration:

   - Register tools with providers
   - Handle tool calls
   - Process tool results
   - Validate tool usage

4. Error Handling:
   - Handle provider errors
   - Manage timeouts
   - Handle tool failures
   - Provide feedback

## Technical Changes

1. Assistant Integration:

```go
type Assistant struct {
    Name        string
    Description string
    Model       string
    Tools       []string
    Prompt      string
    provider    provider.Provider
    toolMgr     *tool.Manager
    logger      *slog.Logger
}

func (a *Assistant) Process(cmd *Command) (*Response, error) {
    // Build context
    ctx := context.Background()
    prompt := a.buildPrompt(cmd)

    // Send to provider
    resp, err := a.provider.Send(ctx, prompt)
    if err != nil {
        return nil, fmt.Errorf("provider error: %w", err)
    }

    // Handle tool calls
    if resp.HasToolCalls() {
        return a.handleToolCalls(ctx, resp)
    }

    return resp, nil
}
```

2. Context Building:

```go
func (a *Assistant) buildPrompt(cmd *Command) string {
    var b strings.Builder

    // Add system prompt
    b.WriteString(a.Prompt)
    b.WriteString("\n\n")

    // Add available tools
    if len(a.Tools) > 0 {
        b.WriteString("Available tools:\n")
        for _, tool := range a.Tools {
            b.WriteString(fmt.Sprintf("- %s\n", tool))
        }
        b.WriteString("\n")
    }

    // Add command
    b.WriteString("Command: ")
    b.WriteString(cmd.Text)

    return b.String()
}
```

3. Tool Handling:

```go
func (a *Assistant) handleToolCalls(ctx context.Context, resp *Response) (*Response, error) {
    for _, call := range resp.ToolCalls {
        // Get tool
        tool, err := a.toolMgr.GetTool(call.Name)
        if err != nil {
            return nil, fmt.Errorf("tool not found: %w", err)
        }

        // Execute tool
        result, err := tool.Execute(call.Arguments)
        if err != nil {
            return nil, fmt.Errorf("tool execution failed: %w", err)
        }

        // Add result to context
        resp.AddToolResult(call.Name, result)
    }

    return resp, nil
}
```

## Success Criteria

1. Command Processing:

```markdown
!default What can you do?

> I can help you with various tasks using tools like:
>
> - summarize: Summarize text content
> - search: Search for information
```

2. Tool Usage:

```markdown
!default Summarize this text

> Let me help you with that.
> Using the summarize tool...
> Here's the summary: ...
```

3. Error Handling:

```markdown
!default Use unknown tool

> I apologize, but I don't have access to that tool.
> Here are the tools I can use:
>
> - summarize: Summarize text content
> - search: Search for information
```

## Non-Goals

1. Multi-provider routing
2. Complex conversation history
3. Tool result caching
4. Provider fallbacks

## Testing Plan

1. Unit Tests:

   - Command routing
   - Context building
   - Tool handling
   - Error cases

2. Integration Tests:
   - End-to-end flow
   - Tool execution
   - Provider interaction
   - Error handling

## Risks

1. Context management
2. Tool execution
3. Error propagation
4. State handling

## Future Considerations

1. Conversation history
2. Tool result caching
3. Provider fallbacks
4. Multi-provider support

## Acceptance Criteria

1. Command Processing:

- Commands routed to providers
- Responses handled correctly
- Context managed properly
- Tools executed successfully

2. Tool Integration:

- Tools registered with providers
- Tool calls handled properly
- Results processed correctly
- Usage validated

3. Error Handling:

- Provider errors caught
- Tool errors handled
- Clear error messages
- Proper feedback

4. Logging:

```
time=2025-01-01T02:27:00Z level=INFO msg="processing command" assistant=default
time=2025-01-01T02:27:00Z level=DEBUG msg="building context" tools=2
time=2025-01-01T02:27:01Z level=INFO msg="executing tool" name=summarize
time=2025-01-01T02:27:02Z level=INFO msg="command completed" duration=2s
```
