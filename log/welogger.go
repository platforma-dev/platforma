package log

import (
	"context"
	"io"
	"log/slog"
)

// WideEventLogger writes wide events with tail sampling.
type WideEventLogger struct {
	sampler Sampler
	logger  *slog.Logger
}

// NewWideEventLogger creates a wide-event logger.
func NewWideEventLogger(w io.Writer, s Sampler, loggerType string, contextKeys map[string]any) *WideEventLogger {
	return &WideEventLogger{
		sampler: s,
		logger:  New(w, loggerType, slog.LevelDebug, contextKeys),
	}
}

// WriteEvent finalizes event duration and conditionally writes it.
func (l *WideEventLogger) WriteEvent(ctx context.Context, e *Event) {
	e.Finish()

	if l.sampler.ShouldSample(ctx, e) {
		l.logger.LogAttrs(ctx, e.level, "", e.ToAttrs()...)
	}
}
