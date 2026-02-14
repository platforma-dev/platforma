package log2

import "context"

type contextKey string

// LogEventContextKey is used to store Event in context.
var LogEventContextKey any = contextKey("platformaLog2Event") //nolint:gochecknoglobals

// EventFromContext gets an event from context.
func EventFromContext(ctx context.Context) (*Event, bool) {
	if ctx == nil {
		return nil, false
	}

	ev, ok := ctx.Value(LogEventContextKey).(*Event)
	if !ok || ev == nil {
		return nil, false
	}

	return ev, true
}

// WithEvent stores an event in context.
func WithEvent(ctx context.Context, ev *Event) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	return context.WithValue(ctx, LogEventContextKey, ev)
}
