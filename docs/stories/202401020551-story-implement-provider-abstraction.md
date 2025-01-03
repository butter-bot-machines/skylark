# Implement Provider Abstraction

## Problem
Direct provider dependencies make testing difficult:

```go
// Direct OpenAI creation
p, err = openai.New("gpt-4", modelConfig)

// Direct HTTP calls
httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, body)
httpResp, err := p.client.Do(httpReq)

// Direct rate limiting
if err := p.rateLimits.Wait(ctx); err != nil {
    return nil, err
}
if err := p.rateLimits.AddTokens(resp.Usage.TotalTokens); err != nil {
    return nil, err
}

// Direct token management
messages = append(messages, map[string]any{
    "role": "assistant",
    "content": resp.Choices[0].Message.Content,
})
```

This means:
1. Tests need API keys
2. Tests need network access
3. Tests hit rate limits
4. Tests are slow and flaky

## Solution

1. Create Provider Interfaces:
```go
// pkg/provider/interface.go
type Provider interface {
    // Core operations
    Send(ctx context.Context, prompt string) (*Response, error)
    Close() error
    
    // Rate management
    SetRateLimiter(RateLimiter)
    GetRateLimiter() RateLimiter
}

type RateLimiter interface {
    Wait(ctx context.Context) error
    AddTokens(count int) error
    GetQuota() Quota
}

type HTTPClient interface {
    Do(req *http.Request) (*http.Response, error)
    CloseIdleConnections()
}

type TokenCounter interface {
    Count(text string) int
    Remaining() int
}

type Response struct {
    Content    string
    Usage      Usage
    ToolCalls  []ToolCall
    Error      error
}
```

2. Add Production Implementation:
```go
// pkg/provider/openai/provider.go
type OpenAIProvider struct {
    client     HTTPClient
    rateLimits RateLimiter
    counter    TokenCounter
    config     Config
}

func (p *OpenAIProvider) Send(ctx context.Context, prompt string) (*Response, error) {
    if err := p.rateLimits.Wait(ctx); err != nil {
        return nil, err
    }
    
    resp, err := p.doRequest(ctx, prompt)
    if err != nil {
        return nil, err
    }
    
    if err := p.rateLimits.AddTokens(resp.Usage.TotalTokens); err != nil {
        return nil, err
    }
    
    return resp, nil
}

// ... implement other methods
```

3. Add Test Implementation:
```go
// pkg/provider/memory/provider.go
type MemoryProvider struct {
    responses map[string]*Response
    calls     []Call
    mu        sync.RWMutex
}

type Call struct {
    Prompt string
    Time   time.Time
}

func (p *MemoryProvider) Send(ctx context.Context, prompt string) (*Response, error) {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    p.calls = append(p.calls, Call{
        Prompt: prompt,
        Time:   time.Now(),
    })
    
    resp, ok := p.responses[prompt]
    if !ok {
        return &Response{
            Content: "Test response",
            Usage: Usage{TotalTokens: 10},
        }, nil
    }
    return resp, nil
}

// ... implement other methods
```

4. Update Components:
```go
// pkg/processor/processor.go
type Processor struct {
    provider provider.Provider
    // ...
}

func New(cfg *config.Config, opts Options) (*Processor, error) {
    if opts.Provider == nil {
        p, err := openai.New(cfg.OpenAI)
        if err != nil {
            return nil, err
        }
        opts.Provider = p
    }
    
    return &Processor{
        provider: opts.Provider,
    }, nil
}

// Use interface instead of direct operations
func (p *Processor) processCommand(cmd *Command) (*Response, error) {
    return p.provider.Send(cmd.Context, cmd.Text)
}
```

## Benefits

1. Testing:
   - Use memory provider
   - No API keys needed
   - No rate limits
   - Fast execution

2. Production:
   - Same interface
   - No behavior changes
   - Better rate control
   - Error handling

3. Future:
   - Multiple providers
   - Provider switching
   - Better monitoring
   - Custom providers

## Implementation

1. Core Changes:
   - Create provider interfaces
   - Add implementations
   - Add rate limiting
   - Add tests

2. Component Updates:
   - Update processor
   - Update assistant
   - Update config
   - Update tests

3. Test Support:
   - Add test provider
   - Add test helpers
   - Update existing tests
   - Add examples

## Acceptance Criteria

1. Functionality:
   - [ ] All provider operations use interface
   - [ ] No direct HTTP calls
   - [ ] Production behavior unchanged
   - [ ] Proper rate limiting

2. Testing:
   - [ ] Tests use memory provider
   - [ ] No API keys in tests
   - [ ] No network access
   - [ ] Fast execution

3. Documentation:
   - [ ] Interface docs
   - [ ] Implementation docs
   - [ ] Test patterns
   - [ ] Examples
