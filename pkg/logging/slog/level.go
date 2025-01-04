package slog

import (
	"log/slog"

	"github.com/butter-bot-machines/skylark/pkg/logging"
)

// levelToSlog converts our level to slog.Level
func levelToSlog(level logging.Level) slog.Level {
	switch level {
	case logging.LevelDebug:
		return slog.LevelDebug
	case logging.LevelInfo:
		return slog.LevelInfo
	case logging.LevelWarn:
		return slog.LevelWarn
	case logging.LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
