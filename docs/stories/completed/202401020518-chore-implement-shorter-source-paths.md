# Implement Shorter Source Paths in Logging (✓ Completed)

## Status
Completed on January 2, 2024 at 07:59
- Implemented shorter source paths in logging
- Updated HandlerOptions to include ReplaceAttr function
- Verified source paths are shortened in both text and JSON output formats
- Ensured line numbers are preserved

## Context

Users are reporting that log messages are too verbose due to the full file paths included in the source information. Currently, when source logging is enabled (the default), the full file path is shown (e.g. `/home/user/projects/skylark/pkg/logging/logger.go:42`).

This makes logs harder to read, especially when viewing multiple log lines. A shorter format showing just the filename and line number (e.g. `logger.go:42`) would improve readability while still maintaining the ability to trace log origins.

## Solution

Modify the logging package to use shorter source paths by leveraging slog's ReplaceAttr functionality:

1. Update HandlerOptions in pkg/logging/logger.go to include a ReplaceAttr function
2. Use the function to intercept source attributes
3. Replace full file paths with just the filename using filepath.Base
4. Preserve line numbers and other source information

Example implementation:
```go
handlerOpts := &slog.HandlerOptions{
    Level:     opts.Level,
    AddSource: opts.AddSource,
    ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
        if a.Key == slog.SourceKey {
            source, _ := a.Value.Any().(*slog.Source)
            if source != nil {
                source.File = filepath.Base(source.File)
            }
        }
        return a
    },
}
```

## Benefits

- Improved log readability
- Shorter log lines
- Maintains source tracing capability
- No configuration changes required
- Uses built-in slog functionality

## Testing

1. Verify source paths are shortened in both text and JSON output formats
2. Ensure line numbers are preserved
3. Check behavior with source logging enabled/disabled
4. Validate across different log levels

## Risks

- Low risk change confined to logging package
- No impact on log functionality or levels
- Only affects display format of source information
