package log

import "log/slog"

const (
	// LevelInfoForced keeps an event regardless of tail-sampling rules while rendering as INFO in output.
	LevelInfoForced slog.Level = slog.LevelInfo + 1
)

func formatLevel(level slog.Level) string {
	if level == LevelInfoForced {
		return slog.LevelInfo.String()
	}

	return level.String()
}

func replaceLevelAttr(a slog.Attr) slog.Attr {
	switch level := a.Value.Any().(type) {
	case slog.Level:
		return slog.String(a.Key, formatLevel(level))
	case slog.Leveler:
		return slog.String(a.Key, formatLevel(level.Level()))
	default:
		return a
	}
}
