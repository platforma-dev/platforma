package log2

import (
	"context"
	"fmt"
	"io"
	"log/slog"
)

// Logger writes sampled wide events.
type Logger struct {
	handler     slog.Handler
	sampler     TailSampler
	contextKeys map[string]any
	maxSteps    int
}

// Default is package-level default logger.
var Default = New(DefaultConfig()) //nolint:gochecknoglobals

// New creates a new wide-event logger.
func New(cfg Config) *Logger {
	writer := cfg.Writer
	if writer == nil {
		writer = io.Discard
	}

	var handler slog.Handler
	if cfg.Format == "text" {
		handler = slog.NewTextHandler(writer, &slog.HandlerOptions{Level: cfg.Level})
	} else {
		handler = slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: cfg.Level})
	}

	sampler := cfg.Sampler
	if sampler == nil {
		sampler = NewDefaultTailSampler(DefaultTailSamplerConfig{})
	}

	maxSteps := cfg.MaxSteps
	if maxSteps <= 0 {
		maxSteps = defaultMaxSteps
	}

	return &Logger{
		handler:     handler,
		sampler:     sampler,
		contextKeys: copyAttrs(cfg.ContextKeys),
		maxSteps:    maxSteps,
	}
}

// SetDefault sets the package-level default logger.
func SetDefault(l *Logger) {
	if l == nil {
		Default = New(DefaultConfig())
		return
	}

	Default = l
}

// Start creates a new event using the package-level default logger.
func Start(ctx context.Context, eventName string, attrs ...any) *Event {
	return Default.Start(ctx, eventName, attrs...)
}

// Start creates a new event using this logger.
func (l *Logger) Start(ctx context.Context, eventName string, attrs ...any) *Event {
	if l == nil {
		return Default.Start(ctx, eventName, attrs...)
	}

	baseAttrs := collectContextAttrs(ctx, l.contextKeys)
	mergeAttrs(baseAttrs, normalizeAttrs(attrs...))

	return &Event{
		logger:    l,
		eventName: eventName,
		startedAt: nowUTC(),
		attrs:     baseAttrs,
		steps:     make([]stepRecord, 0),
		errors:    make([]errorRecord, 0),
	}
}

func (l *Logger) emit(ctx context.Context, event eventPayload) error {
	if l == nil {
		return nil
	}

	if !l.handler.Enabled(ctx, event.level) {
		return nil
	}

	rec := slog.NewRecord(nowUTC(), event.level, event.eventName, 0)
	rec.AddAttrs(
		slog.String("event", event.eventName),
		slog.Int64("durationMs", event.durationMs),
		slog.Bool("sampled", event.sampled),
		slog.String("samplingReason", event.samplingReason),
	)

	if event.traceID != "" {
		rec.AddAttrs(slog.String("traceId", event.traceID))
	}

	rec.AddAttrs(
		slog.Any("attrs", event.attrs),
		slog.Any("steps", event.steps),
		slog.Any("errors", event.errors),
	)

	if event.stepsDropped > 0 {
		rec.AddAttrs(slog.Int("stepsDropped", event.stepsDropped))
	}

	if err := l.handler.Handle(ctx, rec); err != nil {
		return fmt.Errorf("failed to write log event: %w", err)
	}

	return nil
}
