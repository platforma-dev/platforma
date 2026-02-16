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
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey || a.Key == slog.MessageKey {
				return slog.Attr{}
			}
			return a
		},
	}

	var handler slog.Handler
	if loggerType == "json" {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		handler = slog.NewTextHandler(w, opts)
	}

	return &WideEventLogger{
		sampler: s,
		logger:  slog.New(&contextHandler{handler, contextKeys}),
	}
}

// WriteEvent finalizes event duration and conditionally writes it.
func (l *WideEventLogger) WriteEvent(ctx context.Context, e *Event) {
	e.Finish()

	if l.sampler.ShouldSample(ctx, e) {
		l.logger.LogAttrs(ctx, e.Level(), "", e.ToAttrs()...)
	}
}
