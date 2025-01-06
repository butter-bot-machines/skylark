# Story: Complete Provider System Integration

## Status

Completed on January 1, 2025 at 07:34

- Implemented OpenAI provider with direct HTTP integration
- Added token bucket rate limiting
- Added comprehensive contract tests
- Added proper error handling and mapping

## Context

Currently, the Provider System implementation status:

1. ✓ Provider interface defined
2. ✓ Response types structured
3. ✗ Assistant to provider connection
4. ✓ Full OpenAI integration
   - ✓ Direct HTTP implementation
   - ✓ Tool integration
   - ✓ Message history
   - ✓ Contract tests
5. ✓ Error handling and rate limiting
   - ✓ Token bucket implementation
   - ✓ Error mapping
   - ✓ Rate limit tests

This creates a gap in our architecture where assistants cannot effectively communicate with AI models.

## Goal

Complete the Provider System integration to enable assistants to communicate with AI models through a robust, error-handled interface.

## Requirements

1. Provider System:

   - Complete provider interface implementation
   - Connect assistant manager to providers
   - Handle provider lifecycle
   - Manage provider state

2. Assistant Integration:

   - Route commands to providers
   - Handle provider responses
   - Manage conversation context
   - Support tool execution

3. Error Management:

   - Provider errors
   - Network issues
   - Rate limits
   - Context timeouts

4. Resource Control:
   - Token budgets
   - Request throttling
   - Connection pooling
   - Resource cleanup

## Implementation Note

While OpenAI provides an official strongly typed Go SDK, we'll use direct HTTP requests for better control and simplicity:

1. Tool Schema Integration:

   - Our tools already output OpenAI-compatible function schemas:

   ```json
   {
     "schema": {
       "name": "tool_name",
       "description": "...",
       "parameters": {
         "type": "object",
         "properties": {...},
         "required": [...]
       }
     }
   }
   ```

2. API Integration:

   - Direct HTTP POST to `/v1/chat/completions`
   - Request format:

   ```json
   {
     "model": "gpt-4",
     "messages": [...],
     "tools": [{
       "type": "function",
       "function": {
         "name": "tool_name",
         "description": "...",
         "parameters": {...}
       }
     }]
   }
   ```

3. Benefits:
   - Simpler implementation
   - Direct use of tool schemas
   - Better control over requests
   - Easier error handling
   - No SDK type complexity

## Technical Changes

1. Provider System:

```go
type ProviderSystem struct {
    providers map[string]Provider
    config    *Config
}

func (s *ProviderSystem) GetProvider(name string) (Provider, error) {
    provider, ok := s.providers[name]
    if !ok {
        return nil, fmt.Errorf("provider %s not found", name)
    }
    return provider, nil
}

func (s *ProviderSystem) Send(ctx context.Context, req *Request) (*Response, error) {
    // Get provider for model
    provider, err := s.GetProvider(req.Model)
    if err != nil {
        return nil, err
    }

    // Send with error handling
    resp, err := provider.Send(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("provider error: %w", err)
    }

    return resp, nil
}
```

2. Assistant Integration:

```go
func (a *Assistant) Process(cmd *Command) (*Response, error) {
    // Build provider request
    req := &Request{
        Model:   a.config.Model,
        Prompt:  cmd.Text,
        Context: a.buildContext(),
    }

    // Send through provider system
    resp, err := a.providers.Send(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("provider system: %w", err)
    }

    // Handle tool calls if needed
    if resp.HasToolCalls() {
        return a.handleToolCalls(resp)
    }

    return resp, nil
}
```

3. Error Handling:

```go
func (s *ProviderSystem) handleError(err error) error {
    var providerErr *ProviderError
    if errors.As(err, &providerErr) {
        switch providerErr.Code {
        case ErrRateLimit:
            // Handle rate limiting
            return s.handleRateLimit(providerErr)
        case ErrInvalidRequest:
            // Handle invalid requests
            return fmt.Errorf("invalid request: %w", err)
        default:
            // Handle other provider errors
            return fmt.Errorf("provider error: %w", err)
        }
    }
    return err
}
```

## Success Criteria

1. Assistant-Provider Flow:

```markdown
!default What can you do?

> I can help you with various tasks using tools like:
>
> - summarize: Summarize text content
> - search: Search for information
```

2. Tool Integration:

```markdown
!default Summarize this text

> Let me help you with that.
> Using the summarize tool...
> Here's the summary: ...
```

3. Error Handling:

```markdown
!default Process large text

> I apologize, but I've hit a token limit.
> Let me try to break this down into smaller chunks...
```

## Non-Goals

1. Supporting multiple providers
2. Custom provider implementations
3. Response caching
4. Provider-specific optimizations

## Testing Plan

1. Unit Tests:

   - Provider system
   - Assistant integration
   - Error handling
   - Resource management

2. Integration Tests:

   - End-to-end flows
   - Tool execution
   - Error scenarios
   - Resource limits

3. Performance Tests:
   - Concurrent requests
   - Rate limiting
   - Resource usage
   - Memory patterns

## Risks

1. Provider API changes
2. Rate limit complexity
3. Resource management
4. Error propagation

## Future Considerations

1. Multiple providers
2. Response caching
3. Custom providers
4. Advanced context management

## Acceptance Criteria

1. Assistant Integration:

- Commands routed to providers
- Responses handled correctly
- Context managed properly
- Tools executed successfully

2. Error Handling:

- Rate limits respected
- Network errors handled
- Invalid requests caught
- Resources cleaned up

3. Resource Management:

- Token budgets enforced
- Requests throttled
- Connections pooled
- Memory managed

4. Logging:

```
time=2025-01-01T02:21:00Z level=INFO msg="processing command" assistant=default
time=2025-01-01T02:21:01Z level=INFO msg="provider response" tokens=150
time=2025-01-01T02:21:01Z level=INFO msg="executing tool" name=summarize
time=2025-01-01T02:21:02Z level=INFO msg="command completed" duration=2s
```
