# 202401020100-story-implement-initial-project-setup.md

## Context
Users need a simple way to start using Skylark with minimal configuration while maintaining a clear path for advanced usage.

## Goals
1. User can start with minimal required configuration
2. Example tool demonstrates configuration pattern
3. Clear separation between provider, tool, and assistant configs

## Implementation

### 1. Initial Project Structure (my-project is optional, if not provided, use current directory)
When user runs `skylark init my-project`:
```
my-project/
 ├─ .skai/
 │   ├─ assistants/
 │   │   └─ default/
 │   │       └─ prompt.md    # Contains front-matter + prompt
 │   ├─ tools/
 │   │   └─ summarize/       # Example tool
 │   │       ├─ main.go
 │   │       └─ summarize    # Auto-compiled
 │   └─ config.yaml          # Minimal provider config
```

### 2. Default Assistant (prompt.md)
```yaml
---
description: Default assistant for general tasks
model:
  name: openai:gpt-4    # References provider:model
  temperature: 0.7      # Model-specific settings
tools:
  - name: summarize     # Available tools
---
You are a helpful assistant.
```

### 3. Initial Config (config.yaml)
```yaml
version: "1.0"
models:
  openai:
    api_key: ""    # Only required setting to start

tools:
  summarize:       # Example tool config
    env:          # Required by tool's --usage
      DATA_PATH: ""
```

### 4. Example Tool
Demonstrates:
- Required env vars via --usage
- How to configure in config.yaml
- How to reference in assistant front-matter

## User Flow
1. User runs `skylark init my-project`
2. Adds OpenAI API key to config.yaml
3. Sets tool's DATA_PATH in config.yaml
4. Can immediately use default assistant with example tool

## Advanced Usage
- Add custom assistants with their own model settings
- Add new tools and configure their env
- Override system settings via CLI flags

## Testing
1. Init creates valid structure
2. Config validation catches missing API key
3. Tool validates required env vars
4. Assistant properly references model and tool

## Success Criteria
- [ ] User can start with just API key
- [ ] Example tool demonstrates config pattern
- [ ] Assistant front-matter controls its behavior
- [ ] Clear path to adding more tools/assistants
