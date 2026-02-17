package log

import "context"

const (
	// WideEventKey is the default context key for request-wide events.
	WideEventKey contextKey = "wideEvent"
)

// EventFromContext returns a wide event from context when present.
func EventFromContext(ctx context.Context) *Event {
	event, ok := ctx.Value(WideEventKey).(*Event)
	if !ok {
		return nil
	}

	return event
}
