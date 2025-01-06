# Skylark

Skylark is a Go-based system that transforms Markdown documents through AI-powered commands. It watches for file changes and processes inline commands that begin with `!`, routing them through assistants that can leverage specialized tools.

## Features

- **File Watching**: Automatically detects changes in Markdown files
- **Inline Commands**: Process commands starting with `!` directly in your Markdown files
- **Assistants**: Configurable AI assistants with specialized knowledge and capabilities
- **Tool System**: Extensible tool system with built-in and custom tools
- **Security**: Robust security features including:
  - API key management
  - File access controls
  - Audit logging
  - Resource limits

## Installation

```bash
go install github.com/butter-bot-machines/skylark@latest
```

Or build from source:

```bash
git clone https://github.com/butter-bot-machines/skylark.git
cd skylark
go build ./cmd/skylark
```

## Quick Start

1. Initialize a new project:
```bash
skai init my-project
```

2. Create a Markdown file (e.g., `notes.md`):
```markdown
# My Notes

!What's today's date?

# Research
!researcher Tell me about the current time in different timezones
```

3. Run Skylark:
```bash
skai watch
```

## Configuration

Skylark uses a `.skai` directory in your project root for configuration:

```
my_project/
 ├─ .skai/
 │   ├─ assistants/
 │   │   ├─ default/
 │   │   │   ├─ prompt.md
 │   │   │   └─ knowledge/
 │   │   └─ researcher/
 │   │       ├─ prompt.md
 │   │       └─ knowledge/
 │   ├─ tools/
 │   │   ├─ currentdatetime/  # Built-in tool
 │   │   │   ├─ main.go
 │   │   └─ url_lookup/       # Custom tool
 │   │       ├─ main.go
 │   └─ config.yml
 └─ ...
```

### config.yml Example

```yaml
environment:
  log_level: debug
  log_file: app.log

model:
  provider: openai
  name: gpt-4
  parameters:
    max_tokens: 2048
    temperature: 0.7

security:
  file_permissions:
    allowed_paths: ["."]
    max_file_size: 1048576  # 1MB
```

## Custom Tools

Skylark's tool system allows you to extend functionality through custom Go programs. Each tool lives in its own directory under `.skai/tools/` and is automatically compiled when modified.

### Tool Requirements

Each tool must implement two commands:

1. **--usage**: Describes the tool's capabilities and requirements
   - Returns a JSON schema defining input parameters
   - Specifies required environment variables
   - Used by Skylark to validate inputs and setup environment

2. **--health**: Verifies the tool is operational
   - Checks required dependencies and services
   - Returns a status indicating readiness
   - Called before tool execution

### Example Tool Structure

Here's a simple date/time tool (built-in `currentdatetime/main.go`):

```go
package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "os"
    "time"
)

// Command line flags
var (
    showUsage = flag.Bool("usage", false, "Show tool schema")
    checkHealth = flag.Bool("health", false, "Check tool status")
)

// Tool schema definition
var schema = map[string]interface{}{
    "schema": map[string]interface{}{
        "name": "currentdatetime",
        "description": "Returns current date and time",
        "parameters": map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "format": map[string]interface{}{
                    "type": "string",
                    "description": "Optional time format string (defaults to RFC3339)",
                },
            },
        },
    },
    "env": map[string]interface{}{}, // No environment variables needed
}

func main() {
    flag.Parse()

    // Handle --usage flag: return schema
    if *showUsage {
        json.NewEncoder(os.Stdout).Encode(schema)
        return
    }

    // Handle --health flag: check dependencies
    if *checkHealth {
        json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
            "status": true,
            "details": "Ready to process",
        })
        return
    }

    // Normal operation: process input
    var input struct {
        Format string `json:"format,omitempty"`
    }
    if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
        fmt.Fprintf(os.Stderr, "Invalid input: %v\n", err)
        os.Exit(1)
    }

    // Get current time
    now := time.Now()
    format := time.RFC3339
    if input.Format != "" {
        format = input.Format
    }

    // Return formatted time
    output := map[string]string{
        "datetime": now.Format(format),
    }
    json.NewEncoder(os.Stdout).Encode(output)
}
```

### Tool Configuration

Tools are configured in `config.yml` under the `tools` section:

```yaml
tools:
  # Built-in tool (no config needed)
  currentdatetime: {}

  # Custom tool with config
  web_search:
    env:
      API_KEY: "key-yyyyy"
      RATE_LIMIT: "60/hour"

  # Global tool settings
  defaults:
    timeout: "10s"
    retry_count: 3
```

### Tool Integration

1. For custom tools:
   - Create a new directory: `.skai/tools/<tool-name>/`
   - Implement `main.go` with required commands
2. Skylark automatically:
   - Extracts and compiles built-in tools
   - Compiles custom tools when source changes
   - Validates schema and environment
   - Manages tool lifecycle and execution
   - Handles errors and retries

Tools can be referenced in assistant configurations and used in Markdown commands:

```markdown
!What time is it?
```

## Security

Skylark includes comprehensive security features:

- **API Key Management**: Secure storage and rotation of API keys
- **File Access Control**: Path-based access restrictions and size limits
- **Audit Logging**: Detailed logging of security events
- **Resource Limits**: CPU and memory usage controls

## Contributing

We welcome contributions! Here's how you can help:

### Bug Reports & Feature Requests
- Use GitHub Issues to report bugs or suggest features
- Check existing issues before creating a new one
- Provide detailed information and steps to reproduce bugs

### Development
- Fork the repository and create your branch from `main`
- Follow Go coding standards and project conventions
- Add tests for new functionality
- Run tests with `go test ./...` before submitting changes
- Update documentation as needed

### Pull Requests
- Submit PRs against the `main` branch
- Describe your changes and the problem they solve
- Reference any related issues
- Ensure CI passes and tests are added/updated
- Be ready to address code review feedback

### Code of Conduct
We're committed to providing a welcoming and inclusive experience for everyone. Please be respectful and constructive in all interactions.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- OpenAI for GPT models
- The Go community for excellent libraries and tools
