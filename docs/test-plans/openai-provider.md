# OpenAI Provider Test Plan

## Components Under Test ✓

### 1. Contract Testing (via snapshots) ✓
- Request format matches OpenAI's API contract (verified in openai_contract_test.go)
- Response parsing for all variations:
  - Basic completion responses (testdata/responses/completion.json)
  - Tool call responses (testdata/responses/tool_call.json)
  - Error responses (testdata/responses/errors.json)
- Tool definitions match OpenAI's function calling format (verified in contract tests)

### 2. State Management (in openai.go) ✓
- Tool Registry
  - Thread-safe registration with mutex
  - Tools persist between requests
  - Tool schemas correctly formatted
- Message History
  - Assistant messages preserved in context
  - Tool results included in message chain
  - Context built properly during tool execution

### 3. Integration Points ✓
- HTTP Client (with mocks)
  - Headers and auth verified in contract tests
  - Timeout handling (30s default)
  - Response parsing and error mapping
- Tool Execution
  - Tool interface with Schema() and Execute()
  - Arguments passed as JSON
  - Results included in context
- Rate Limiter (rate_limiter_test.go)
  - Token bucket implementation
  - Request and token limits
  - Context cancellation

## Test Structure

### 1. Contract Tests (openai_contract_test.go)
- Use snapshot testing for request/response validation
- Store sample payloads in testdata/
- Cover all API interaction patterns:
  - Basic completions
  - Tool registration
  - Tool execution
  - Error responses

### 2. Rate Limiter Tests (rate_limiter_test.go)
- Test rate limiter implementation separately
- Focus on:
  - Token bucket behavior
  - Request rate limiting
  - Token usage limiting
  - Context cancellation

## Test Data Organization

```
pkg/provider/openai/
├── openai.go                 # Provider implementation
├── rate_limiter.go          # Rate limiter implementation
├── openai_test.go           # Provider behavior tests
├── openai_contract_test.go  # API contract tests
├── rate_limiter_test.go     # Rate limiter implementation tests
└── testdata/
    ├── requests/            # Sample API requests
    │   ├── basic.json
    │   ├── with_tools.json
    │   └── tool_result.json
    └── responses/           # Sample API responses
        ├── completion.json
        ├── tool_call.json
        └── errors.json
```

## Implementation Strategy

1. Create contract tests first
   - Define expected request/response formats
   - Create sample payloads
   - Implement snapshot testing

2. Maintain rate limiter tests
   - Keep focused on implementation details
   - Test edge cases thoroughly
   - Ensure proper resource cleanup
