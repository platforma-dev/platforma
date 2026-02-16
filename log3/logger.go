package log3

import (
	"context"
	"io"
	"log/slog"

	"github.com/platforma-dev/platforma/log"
)

type Logger struct {
	w       io.Writer
	sampler Sampler
	logger  *slog.Logger
}

func NewWideEventLogger(w io.Writer, s Sampler, loggerType string, contextKeys map[string]any) *Logger {
	return &Logger{
		sampler: s,
		logger:  log.New(w, loggerType, slog.LevelDebug, contextKeys),
	}
}

func (l *Logger) WriteEvent(ctx context.Context, e *Event) {
	e.Finish()

	if l.sampler.ShouldSample(ctx, e) {
		l.logger.LogAttrs(ctx, e.level, "", e.ToAttrs()...)
	}
}
