# Chore: Fix Tool Package Type Naming (Chore)

## Context
The tool package has confusing naming patterns:
1. Schema is used both as type and field: `type Schema struct { Schema struct {...} }`
2. Tool contains Schema which contains Schema: `type Tool struct { Schema Schema }`
3. This creates nested confusion: `tool.Schema.Schema.Description`

## Goal
Improve code clarity by restructuring types in the tool package.

## Changes Needed
1. Rename and restructure types:
```go
// Before
type Tool struct {
    Name        string
    Path        string
    Version     string
    LastBuilt   time.Time
    Description string
    Schema      Schema
}

type Schema struct {
    Schema struct {
        Name        string
        Description string
        Parameters  map[string]interface{}
    }
    Env map[string]EnvVar
}

// After
type Tool struct {
    Name        string
    Path        string
    Version     string
    LastBuilt   time.Time
    Definition  ToolDefinition
}

type ToolDefinition struct {
    Name        string
    Description string
    Parameters  map[string]interface{}
    Environment map[string]EnvVar
}
```

2. Update references:
- Tool struct usage
- Function parameters
- Interface definitions
- Test cases
- Documentation

## Non-Goals
1. Changing functionality
2. Modifying validation logic
3. Changing file formats
4. Adding new features

## Risks
1. Breaking changes require coordination
2. Test coverage might miss usage
3. External tools might break
4. Config file compatibility

## Implementation Plan
1. Create new types
2. Update tool package internals
3. Update dependent packages
4. Run all tests
5. Update documentation
6. Remove old types

## Dependencies
- pkg/tool
- pkg/provider/openai
- pkg/assistant
- Any packages using Tool or Schema types

## Notes
This cleanup will make the code more maintainable and easier to understand. Should be done after current features are stable.
