package log3

import (
	"context"
	"io"
	"log/slog"
)

type Logger struct {
	w       io.Writer
	sampler func(*Event) bool
	logger  *slog.Logger
}

func (l *Logger) WriteEvent(ctx context.Context, e *Event) {
	e.Finish()

	if l.sampler(e) {
		l.logger.LogAttrs(ctx, e.level, "", e.ToAttrs()...)
	}
}

func mapToAttrs(data map[string]any) []slog.Attr {
	attrs := make([]slog.Attr, 0, len(data))
	for k, v := range data {
		attrs = append(attrs, slog.Any(k, v))
	}
	return attrs
}
