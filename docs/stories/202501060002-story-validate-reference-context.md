# Reference Context Validation

## Context

The parser package (pkg/parser/parser.go) implements reference extraction and context handling with several key components:

1. Current Implementation:
   - Command parsing with assistant name handling
   - Block type detection (Header, List, Paragraph, Quote, Table, Code)
   - Reference extraction using regex pattern `#\s*([^#\n]+?)(?:\s*#|$)`
   - Context assembly with parent/sibling tracking
   - Size limits (maxCommandSize: 4000, maxTotalSize: 8000)

2. Areas Needing Validation:
   - Reference extraction accuracy
   - Block boundary detection
   - Context assembly logic
   - Context propagation to provider
   - Size limit enforcement

## Goals

1. Validate Reference Extraction:
   - Verify regex pattern handles all reference formats
   - Ensure proper handling of nested references
   - Validate whitespace and punctuation handling

2. Verify Block Handling:
   - Validate block type detection
   - Ensure proper block boundaries
   - Verify content preservation
   - Test nested block scenarios

3. Test Context Assembly:
   - Validate parent header collection
   - Verify sibling section handling
   - Test context size limits
   - Ensure proper content truncation

4. Implement Context Propagation:
   - Pass context separately to provider
   - Use proper delimiters for context sections
   - Format tools separately from context
   - Ensure model can distinguish instructions from context

## Technical Details

### Provider Interface Update
- Update pkg/provider/provider.go:
  ```go
  type RequestOptions struct {
      Model       string
      Temperature float64
      MaxTokens   int
      Tools       []Tool        // Available tools passed separately
      Context     []ContextBlock // Referenced context passed separately
  }

  type ContextBlock struct {
      Reference string // Original reference text
      Content   string // Block content
      Type      BlockType // Type of block
  }

  type Tool struct {
      Name        string
      Description string
      Schema      json.RawMessage
  }
  ```

### Context Propagation
- Update pkg/assistant/assistant.go Process:
  ```go
  func (a *Assistant) Process(cmd *parser.Command) (string, error) {
      // Build request options
      opts := &provider.RequestOptions{
          Model:       modelName,
          Temperature: 0.7,
          MaxTokens:   2000,
      }

      // Add available tools
      for _, toolName := range a.Tools {
          tool, err := a.toolMgr.LoadTool(toolName)
          if err != nil {
              return "", fmt.Errorf("failed to load tool: %w", err)
          }
          opts.Tools = append(opts.Tools, provider.Tool{
              Name:        tool.Name,
              Description: tool.Description,
              Schema:      tool.Schema,
          })
      }

      // Add referenced context blocks
      for ref, block := range cmd.Context {
          opts.Context = append(opts.Context, provider.ContextBlock{
              Reference: ref,
              Content:  block.Content,
              Type:     block.Type,
          })
      }

      // Get response from provider
      resp, err := p.Send(ctx, cmd.Text, opts)
      // ...
  }
  ```

### Reference Extraction Validation
- Test pkg/parser/parser.go ParseReferences:
  ```go
  func TestParseReferences(t *testing.T) {
      tests := []struct {
          name     string
          text     string
          expected []string
      }{
          {
              name: "single reference",
              text: "analyze # Section One",
              expected: []string{"Section One"},
          },
          {
              name: "nested references",
              text: "compare # Section One # with # Section Two",
              expected: []string{"Section One", "Section Two"},
          },
          {
              name: "whitespace handling",
              text: "check #  Spaced   Section  #",
              expected: []string{"Spaced   Section"},
          },
      }
      // ... test implementation
  }
  ```

### Block Detection Validation
- Test pkg/parser/parser.go ParseBlocks:
  ```go
  func TestParseBlocks(t *testing.T) {
      tests := []struct {
          name     string
          content  string
          expected []Block
      }{
          {
              name: "header hierarchy",
              content: `# H1
              ## H2
              Content
              ## H2-2`,
              expected: []Block{
                  {Type: Header, Level: 1, Content: "H1"},
                  {Type: Header, Level: 2, Content: "H2"},
                  {Type: Paragraph, Content: "Content"},
                  {Type: Header, Level: 2, Content: "H2-2"},
              },
          },
          // ... more test cases
      }
      // ... test implementation
  }
  ```

## Implementation Plan

1. Provider Interface Update
   - Add Tools and Context to RequestOptions
   - Update provider implementations
   - Add tests for new options

2. Context Propagation
   - Update assistant to pass tools separately
   - Update assistant to pass context separately
   - Add tests for proper propagation
   - Verify provider formatting

3. Reference Extraction Testing
   - Implement comprehensive test suite
   - Add edge case coverage
   - Test size limits

4. Block Detection Testing
   - Test each block type
   - Verify boundary detection
   - Test nested structures
   - Validate content preservation

## Acceptance Criteria

- [ ] Provider interface supports:
  - Separate tools section
  - Separate context section
  - Proper formatting for model

- [ ] Context propagation verifies:
  - Tools passed separately
  - Context blocks properly delimited
  - Size limits respected
  - Provider formats correctly

- [ ] Reference extraction tests verify:
  - Correct pattern matching
  - Nested reference handling
  - Whitespace normalization
  - Size limit enforcement

- [ ] Block detection tests verify:
  - All block types recognized
  - Proper boundary detection
  - Nested block handling
  - Content preservation

## Dependencies

- pkg/parser/parser.go implementation
- pkg/assistant/assistant.go implementation
- pkg/provider implementations
- pkg/logging for warning system

## Related

- Parser implementation in pkg/parser/parser.go
- Assistant implementation in pkg/assistant/assistant.go
- Provider interface in pkg/provider/provider.go
- Integration tests in test/integration/integration_test.go
