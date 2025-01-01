package logging

import (
	"io"
	"log/slog"
	"os"
)

// Options configures the logger
type Options struct {
	// Level sets the minimum level to log
	Level slog.Level
	// AddSource adds source code information to log messages
	AddSource bool
	// Output sets the output destination (defaults to os.Stdout)
	Output io.Writer
	// JSON enables JSON output format
	JSON bool
}

// NewLogger creates a new logger with the given options
func NewLogger(opts *Options) *slog.Logger {
	if opts == nil {
		opts = &Options{
			Level:     slog.LevelInfo,
			AddSource: true,
			Output:    os.Stdout,
		}
	}

	if opts.Output == nil {
		opts.Output = os.Stdout
	}

	var handler slog.Handler
	handlerOpts := &slog.HandlerOptions{
		Level:     opts.Level,
		AddSource: opts.AddSource,
	}

	if opts.JSON {
		handler = slog.NewJSONHandler(opts.Output, handlerOpts)
	} else {
		handler = slog.NewTextHandler(opts.Output, handlerOpts)
	}

	return slog.New(handler)
}

// WithAttrs adds common attributes to a logger
func WithAttrs(logger *slog.Logger, attrs ...any) *slog.Logger {
	return logger.With(attrs...)
}

// WithGroup adds a group to a logger
func WithGroup(logger *slog.Logger, name string) *slog.Logger {
	return logger.WithGroup(name)
}
