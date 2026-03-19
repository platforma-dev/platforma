package log

import (
	"context"
	"io"
	"log/slog"
	"slices"
	"time"
)

// WideEventLogger writes wide events with tail sampling.
type WideEventLogger struct {
	sampler          Sampler
	logger           *slog.Logger
	reservedAttrKeys []string
}

const (
	simpleLogEventName = "log.record"
)

var _ DefaultLogger = (*WideEventLogger)(nil)

// NewWideEventLogger creates a wide-event logger.
func NewWideEventLogger(w io.Writer, s Sampler, loggerType string, contextKeys map[string]any) *WideEventLogger {
	// If no sampler provided, use a keep-all sampler to prevent nil panics
	if s == nil {
		s = SamplerFunc(func(_ context.Context, _ *Event) bool { return true })
	}

	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			if a.Key == slog.MessageKey && a.Value.Kind() == slog.KindString && a.Value.String() == "" {
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
		sampler:          s,
		logger:           slog.New(&contextHandler{handler, contextKeys}),
		reservedAttrKeys: wideEventReservedAttrKeys(contextKeys),
	}
}

// Debug logs a message at Debug level.
func (l *WideEventLogger) Debug(msg string, args ...any) {
	l.DebugContext(context.Background(), msg, args...)
}

// Info logs a message at Info level.
func (l *WideEventLogger) Info(msg string, args ...any) {
	l.InfoContext(context.Background(), msg, args...)
}

// Warn logs a message at Warn level.
func (l *WideEventLogger) Warn(msg string, args ...any) {
	l.WarnContext(context.Background(), msg, args...)
}

// Error logs a message at Error level.
func (l *WideEventLogger) Error(msg string, args ...any) {
	l.ErrorContext(context.Background(), msg, args...)
}

// DebugContext logs a message at Debug level with context.
func (l *WideEventLogger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.writeSimpleLog(ctx, slog.LevelDebug, msg, args...)
}

// InfoContext logs a message at Info level with context.
func (l *WideEventLogger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.writeSimpleLog(ctx, slog.LevelInfo, msg, args...)
}

// WarnContext logs a message at Warn level with context.
func (l *WideEventLogger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.writeSimpleLog(ctx, slog.LevelWarn, msg, args...)
}

// ErrorContext logs a message at Error level with context.
func (l *WideEventLogger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.writeSimpleLog(ctx, slog.LevelError, msg, args...)
}

// WriteEvent finalizes event duration and conditionally writes it.
func (l *WideEventLogger) WriteEvent(ctx context.Context, e *Event) {
	e.Finish()

	if l.sampler.ShouldSample(ctx, e) {
		l.logger.LogAttrs(ctx, e.Level(), "", e.toAttrs(l.reservedAttrKeys)...)
	}
}

func (l *WideEventLogger) writeSimpleLog(ctx context.Context, level slog.Level, msg string, args ...any) {
	event := NewEvent(simpleLogEventName)
	event.SetLevel(level)
	event.AddAttrs(simpleLogEventAttrs(args...))
	event.Finish()

	if l.sampler.ShouldSample(ctx, event) {
		l.logger.LogAttrs(ctx, event.Level(), msg, event.toAttrs(l.reservedAttrKeys)...)
	}
}

func simpleLogEventAttrs(args ...any) map[string]any {
	attrs := map[string]any{}

	record := slog.NewRecord(time.Time{}, slog.LevelInfo, "", 0)
	record.Add(args...)
	record.Attrs(func(attr slog.Attr) bool {
		attrs[attr.Key] = attr.Value
		return true
	})

	return attrs
}

func wideEventReservedAttrKeys(contextKeys map[string]any) []string {
	reservedAttrKeys := append([]string{}, wideEventBuiltinAttrKeys()...)
	reservedAttrKeys = appendUnique(reservedAttrKeys, slog.LevelKey)
	reservedAttrKeys = appendUnique(reservedAttrKeys, string(DomainNameKey))
	reservedAttrKeys = appendUnique(reservedAttrKeys, string(TraceIDKey))
	reservedAttrKeys = appendUnique(reservedAttrKeys, string(ServiceNameKey))
	reservedAttrKeys = appendUnique(reservedAttrKeys, string(StartupTaskKey))
	reservedAttrKeys = appendUnique(reservedAttrKeys, string(UserIDKey))
	reservedAttrKeys = appendUnique(reservedAttrKeys, string(WorkerIDKey))
	for key := range contextKeys {
		reservedAttrKeys = appendUnique(reservedAttrKeys, key)
	}

	return reservedAttrKeys
}

func appendUnique(keys []string, key string) []string {
	if slices.Contains(keys, key) {
		return keys
	}

	return append(keys, key)
}
