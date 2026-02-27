// Package log provides structured logging functionality with context support.
package log //nolint:revive

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
)

type logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)

	DebugContext(ctx context.Context, msg string, args ...any)
	InfoContext(ctx context.Context, msg string, args ...any)
	WarnContext(ctx context.Context, msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)
}

// Logger is the default logger instance used by package-level logging functions.
var Logger logger = New(os.Stdout, "text", slog.LevelInfo, nil) //nolint:gochecknoglobals

// SetDefault sets the default logger used by the package-level logging functions.
func SetDefault(l logger) {
	Logger = l
}

type contextKey string

const (
	// DomainNameKey is the context key for domain name.
	DomainNameKey contextKey = "domainName"
	// TraceIDKey is the context key for trace ID.
	TraceIDKey contextKey = "traceId"
	// ServiceNameKey is the context key for service name.
	ServiceNameKey contextKey = "serviceName"
	// StartupTaskKey is the context key for startup task.
	StartupTaskKey contextKey = "startupTask"
	// UserIDKey is the context key for user ID.
	UserIDKey contextKey = "userId"
	// WorkerIDKey is the context key worker of queue processor.
	WorkerIDKey contextKey = "workerId"
)

type contextHandler struct {
	slog.Handler
	additionKeys map[string]any
}

// Handle processes the log record by adding context values before passing it to the underlying handler.
func (h *contextHandler) Handle(ctx context.Context, r slog.Record) error {
	var defaultKeys = []contextKey{
		DomainNameKey,
		TraceIDKey,
		ServiceNameKey,
		StartupTaskKey,
		UserIDKey,
		WorkerIDKey,
	}

	for _, key := range defaultKeys {
		if value, ok := ctx.Value(key).(string); ok {
			r.AddAttrs(slog.String(string(key), value))
		}
	}

	for keyString, key := range h.additionKeys {
		if value, ok := ctx.Value(key).(string); ok {
			r.AddAttrs(slog.String(keyString, value))
		}
	}

	err := h.Handler.Handle(ctx, r)
	if err != nil {
		return fmt.Errorf("failed to handle log record: %w", err)
	}
	return nil
}

// New creates a new slog.Logger with the specified type (json/text), log level, and additional context keys to include.
func New(w io.Writer, loggerType string, level slog.Level, contextKeys map[string]any) *slog.Logger {
	if loggerType == "json" {
		return slog.New(&contextHandler{slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level}), contextKeys})
	}

	return slog.New(&contextHandler{slog.NewTextHandler(w, &slog.HandlerOptions{Level: level}), contextKeys})
}

// Debug logs a message at Debug level.
func Debug(msg string, args ...any) {
	Logger.Debug(msg, args...)
}

// DebugContext logs a message at Debug level with context.
func DebugContext(ctx context.Context, msg string, args ...any) {
	Logger.DebugContext(ctx, msg, args...)
}

// Info logs a message at Info level.
func Info(msg string, args ...any) {
	Logger.Info(msg, args...)
}

// InfoContext logs a message at Info level with context.
func InfoContext(ctx context.Context, msg string, args ...any) {
	Logger.InfoContext(ctx, msg, args...)
}

// Warn logs a message at Warn level.
func Warn(msg string, args ...any) {
	Logger.Warn(msg, args...)
}

// WarnContext logs a message at Warn level with context.
func WarnContext(ctx context.Context, msg string, args ...any) {
	Logger.WarnContext(ctx, msg, args...)
}

// Error logs a message at Error level.
func Error(msg string, args ...any) {
	Logger.Error(msg, args...)
}

// ErrorContext logs a message at Error level with context.
func ErrorContext(ctx context.Context, msg string, args ...any) {
	Logger.ErrorContext(ctx, msg, args...)
}
