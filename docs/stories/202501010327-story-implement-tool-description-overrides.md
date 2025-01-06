# Story: Implement Tool Description Overrides

## Context
Currently, assistants can specify tool description overrides in their frontmatter:
```yaml
tools:
  - name: web_search
    description: Assistant-specific description override
```

However, this functionality is not fully implemented:
1. Assistant struct only loads tool names, not descriptions
2. Tool.Description field exists but isn't populated
3. OpenAI provider assumes Tool.Description is set

## Goal
Implement tool description overrides to allow assistants to customize tool descriptions for their specific use cases.

## Requirements
1. Assistant Loading:
   - Update Assistant struct to load tool descriptions
   - Parse tool overrides from frontmatter
   - Validate tool names exist

2. Tool Integration:
   - Pass description overrides to tools
   - Update Tool.Description field
   - Support fallback to Schema.Schema.Description

3. Provider Integration:
   - Use Tool.Description when set
   - Fall back to Schema.Schema.Description
   - Handle empty descriptions

Notes:
* Schema.Schema.Description may have been renamed by chore:202501010223-fix-tool-naming

## Technical Changes

1. Assistant Struct:
```go
// Tool override configuration
type ToolConfig struct {
    Name        string `yaml:"name"`
    Description string `yaml:"description,omitempty"`
}

// Assistant configuration
type Assistant struct {
    Name        string       `yaml:"name"`
    Description string       `yaml:"description"`
    Model       string       `yaml:"model"`
    Tools       []ToolConfig `yaml:"tools,omitempty"` // Now includes descriptions
    Prompt      string       `yaml:"-"`
}
```

2. Tool Loading:
```go
func (a *Assistant) executeTool(name string, input string) (string, error) {
    // Get tool
    tool, err := a.toolMgr.LoadTool(name)
    if err != nil {
        return "", fmt.Errorf("failed to load tool: %w", err)
    }

    // Apply description override if exists
    for _, cfg := range a.Tools {
        if cfg.Name == name && cfg.Description != "" {
            tool.Description = cfg.Description
            break
        }
    }

    // Rest of execution...
}
```

3. Provider Usage:
```go
func getToolDescription(t *tool.Tool) string {
    if t.Description != "" {
        return t.Description // Use override
    }
    return t.Schema.Schema.Description // Fall back to default
}
```

## Success Criteria
1. Assistant Configuration:
```yaml
tools:
  - name: summarize
    description: Condenses text into key points
  - name: search
    description: Finds relevant information
```

2. Tool Usage:
```markdown
!default What tools do you have?

> I have several tools available:
> - summarize: Condenses text into key points
> - search: Finds relevant information
```

3. Default Fallback:
```markdown
!other What tools do you have?

> I have several tools available:
> - summarize: Summarize text content (default description)
> - search: Search for information (default description)
```

## Non-Goals
1. Dynamic description updates
2. Multiple descriptions per tool
3. Description templates
4. Description inheritance
5. Backwards compatibility with existing assistants (there are none)

## Testing Plan
1. Unit Tests:
   - Assistant loading
   - Tool description overrides
   - Default fallbacks

2. Integration Tests:
   - End-to-end override flow
   - Multiple assistants
   - Provider integration

## Risks
1. Description validation
2. Provider assumptions

## Future Considerations
1. Description templates
2. Dynamic updates
3. Inheritance
4. Validation rules

## Acceptance Criteria
1. Configuration:
- Assistants can override tool descriptions
- Invalid tools are caught
- Empty descriptions fall back to defaults

2. Integration:
- Tools use correct descriptions
- Providers see overridden descriptions
- Multiple assistants work correctly with overrides

3. Validation:
- Invalid tools are detected
- Missing tools are handled
- Description format is checked

4. Logging:
```
time=2025-01-01T02:24:00Z level=INFO msg="loading assistant" name=research
time=2025-01-01T02:24:00Z level=DEBUG msg="tool override" tool=summarize description="Research-focused summarization"
time=2025-01-01T02:24:00Z level=INFO msg="assistant loaded" tools=2
