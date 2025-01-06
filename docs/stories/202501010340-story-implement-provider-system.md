# Story: Implement Provider System

## Context
The OpenAI provider is implemented, but we need a system to manage providers and handle provider lifecycle:
```go
type ProviderSystem struct {
    providers map[string]Provider
    config    *Config
}
```

This system will:
1. Manage provider instances
2. Route requests to appropriate providers
3. Handle provider lifecycle
4. Manage provider state

## Goal
Implement a provider system that manages AI model providers and handles provider lifecycle.

## Requirements
1. Provider Management:
   - Load providers from config
   - Initialize provider instances
   - Handle provider cleanup
   - Manage provider state

2. Request Routing:
   - Route requests to correct provider
   - Handle provider errors
   - Support provider selection
   - Validate provider config

3. Lifecycle Management:
   - Initialize providers
   - Clean up resources
   - Handle reconnection
   - Monitor health

4. State Management:
   - Track provider status
   - Monitor rate limits
   - Handle provider errors
   - Cache provider instances

## Technical Changes

1. Provider System:
```go
// Provider interface
type Provider interface {
    Send(ctx context.Context, prompt string) (*Response, error)
    Close() error
}

// Provider system
type ProviderSystem struct {
    providers map[string]Provider
    config    *Config
    mu        sync.RWMutex
}

// Initialize system
func NewProviderSystem(cfg *Config) (*ProviderSystem, error) {
    system := &ProviderSystem{
        providers: make(map[string]Provider),
        config:    cfg,
    }
    return system.init()
}

// Get provider
func (s *ProviderSystem) GetProvider(name string) (Provider, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    provider, ok := s.providers[name]
    if !ok {
        return nil, fmt.Errorf("provider %s not found", name)
    }
    return provider, nil
}

// Send request
func (s *ProviderSystem) Send(ctx context.Context, req *Request) (*Response, error) {
    provider, err := s.GetProvider(req.Model)
    if err != nil {
        return nil, err
    }

    resp, err := provider.Send(ctx, req.Prompt)
    if err != nil {
        return nil, fmt.Errorf("provider error: %w", err)
    }

    return resp, nil
}

// Clean up
func (s *ProviderSystem) Close() error {
    s.mu.Lock()
    defer s.mu.Unlock()

    var errs []error
    for name, provider := range s.providers {
        if err := provider.Close(); err != nil {
            errs = append(errs, fmt.Errorf("failed to close %s: %w", name, err))
        }
    }

    if len(errs) > 0 {
        return fmt.Errorf("errors closing providers: %v", errs)
    }
    return nil
}
```

2. Provider Loading:
```go
func (s *ProviderSystem) init() error {
    // Load OpenAI provider
    if cfg, ok := s.config.Models["openai"]; ok {
        provider, err := openai.New(cfg)
        if err != nil {
            return fmt.Errorf("failed to init OpenAI: %w", err)
        }
        s.providers["openai"] = provider
    }

    return nil
}
```

## Success Criteria
1. Provider Management:
```go
system := NewProviderSystem(config)
defer system.Close()

provider, err := system.GetProvider("openai")
if err != nil {
    log.Printf("provider not found: %v", err)
}
```

2. Request Routing:
```go
resp, err := system.Send(ctx, &Request{
    Model: "openai:gpt-4",
    Prompt: "Hello",
})
if err != nil {
    log.Printf("request failed: %v", err)
}
```

3. Error Handling:
```go
resp, err := system.Send(ctx, &Request{
    Model: "unknown",
    Prompt: "Hello",
})
if err != nil {
    log.Printf("unknown provider: %v", err)
}
```

## Non-Goals
1. Multiple provider types
2. Provider discovery
3. Dynamic loading
4. Provider metrics

## Testing Plan
1. Unit Tests:
   - Provider loading
   - Request routing
   - Error handling
   - Resource cleanup

2. Integration Tests:
   - Provider lifecycle
   - Request flow
   - Error scenarios
   - Resource management

## Risks
1. Provider initialization
2. Resource leaks
3. Error propagation
4. Config validation

## Future Considerations
1. Provider discovery
2. Dynamic loading
3. Health monitoring
4. Provider metrics

## Acceptance Criteria
1. Provider Management:
- Providers loaded from config
- Provider instances managed
- Resources cleaned up
- State tracked properly

2. Request Routing:
- Requests routed correctly
- Errors handled properly
- Config validated
- Resources managed

3. Error Handling:
- Provider errors caught
- System errors handled
- Resources cleaned up
- Clear error messages

4. Logging:
```
time=2025-01-01T02:26:00Z level=INFO msg="initializing provider system"
time=2025-01-01T02:26:00Z level=DEBUG msg="loading provider" name=openai
time=2025-01-01T02:26:01Z level=INFO msg="provider system ready" providers=1
