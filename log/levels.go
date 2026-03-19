package log

import "log/slog"

// Level is an alias for slog.Level representing log severity.
type Level = slog.Level

// Log level constants.
const (
	LevelDebug      Level = slog.LevelDebug
	LevelInfo       Level = slog.LevelInfo
	LevelWarn       Level = slog.LevelWarn
	LevelError      Level = slog.LevelError
	LevelInfoForced Level = 10
)
