# Story: Implement Tool Environment Handling

## Context
Tools can specify environment requirements in their --usage output:
```json
{
  "env": {
    "API_KEY": {
      "type": "string",
      "description": "API key for external service",
      "default": null
    }
  }
}
```

These are configured in config.yaml:
```yaml
tools:
  web_search:
    env:
      API_KEY: "key-123"
```

However, environment handling is not fully implemented:
1. Tool env requirements are loaded but not validated
2. Config values aren't passed to tool execution
3. Default values aren't used as fallbacks

## Goal
Implement complete environment handling for tools to enable secure and configurable external service integration.

## Requirements
1. Environment Loading:
   - Parse env requirements from --usage
   - Load env values from config.yaml
   - Support default values
   - Validate types and requirements

2. Environment Resolution:
   - Match config values to requirements
   - Apply default values when needed
   - Validate all required vars are set
   - Type check values

3. Tool Integration:
   - Pass resolved env to tool execution
   - Handle missing/invalid env vars
   - Support runtime env updates
   - Validate env health

## Technical Changes

1. Environment Resolution:
```go
type EnvResolver struct {
    config  *config.Config
    defaults map[string]map[string]interface{}
}

func (r *EnvResolver) ResolveToolEnv(tool *tool.Tool) (map[string]string, error) {
    // Get tool config
    cfg, ok := r.config.Tools[tool.Name]
    if !ok {
        cfg = &config.ToolConfig{
            Env: make(map[string]string),
        }
    }

    // Build env map
    env := make(map[string]string)
    for name, spec := range tool.Schema.Env {
        // Try config value
        if val, ok := cfg.Env[name]; ok {
            env[name] = val
            continue
        }

        // Try default value
        if spec.Default != nil {
            env[name] = fmt.Sprint(spec.Default)
            continue
        }

        // Required but missing
        return nil, fmt.Errorf("missing required env var: %s", name)
    }

    return env, nil
}
```

2. Tool Execution:
```go
func (t *Tool) Execute(input []byte, resolver *EnvResolver) ([]byte, error) {
    // Resolve environment
    env, err := resolver.ResolveToolEnv(t)
    if err != nil {
        return nil, fmt.Errorf("env resolution failed: %w", err)
    }

    // Execute with environment
    cmd := exec.Command(t.Path)
    cmd.Env = os.Environ() // Start with current env
    for k, v := range env {
        cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
    }

    // Rest of execution...
}
```

3. Health Checking:
```go
func (t *Tool) CheckHealth(resolver *EnvResolver) error {
    // Resolve environment first
    env, err := resolver.ResolveToolEnv(t)
    if err != nil {
        return fmt.Errorf("env resolution failed: %w", err)
    }

    // Run health check with env
    cmd := exec.Command(t.Path, "--health")
    cmd.Env = os.Environ()
    for k, v := range env {
        cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
    }

    // Check result...
}
```

## Success Criteria
1. Tool Configuration:
```yaml
tools:
  web_search:
    env:
      API_KEY: "key-123"
      TIMEOUT: 30
```

2. Tool Execution:
```go
// All env vars provided
result, err := tool.Execute(input, resolver)
if err != nil {
    log.Printf("execution failed: %v", err)
}

// Missing required var
result, err := tool.Execute(input, resolver)
if err != nil {
    log.Printf("missing API_KEY: %v", err)
}

// Using default value
result, err := tool.Execute(input, resolver)
if err != nil {
    log.Printf("using default timeout: 30s")
}
```

## Non-Goals
1. Dynamic env updates
2. Environment inheritance
3. Environment templates
4. Secret management
5. Environment validation beyond types

## Testing Plan
1. Unit Tests:
   - Environment resolution
   - Default handling
   - Type validation
   - Error cases

2. Integration Tests:
   - Tool execution with env
   - Health checks
   - Config loading
   - Default resolution

## Risks
1. Security implications
2. Type conversion errors
3. Default value handling
4. Environment pollution

## Future Considerations
1. Secret management
2. Environment inheritance
3. Dynamic updates
4. Validation rules

## Acceptance Criteria
1. Configuration:
- Tools can specify env requirements
- Config can provide env values
- Defaults are properly handled
- Types are validated

2. Integration:
- Tools receive correct env vars
- Missing vars are caught
- Defaults are applied
- Types are enforced

3. Security:
- Env vars are isolated
- Secrets are handled safely
- No env pollution
- Clear error messages

4. Logging:
```
time=2024-01-01T02:25:00Z level=INFO msg="resolving tool env" tool=web_search
time=2024-01-01T02:25:00Z level=DEBUG msg="using config value" var=API_KEY
time=2024-01-01T02:25:00Z level=DEBUG msg="using default value" var=TIMEOUT value=30
time=2024-01-01T02:25:00Z level=INFO msg="tool env resolved" vars=2
