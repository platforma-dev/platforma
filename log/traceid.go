package log

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// TraceIDMiddleware adds a trace ID to request context and response headers.
type TraceIDMiddleware struct {
	contextKey any
	header     string
}

// NewTraceIDMiddleware returns a new TraceID middleware.
// If key is nil, TraceIDKey is used.
// If header is empty, "Platforma-Trace-Id" is used.
func NewTraceIDMiddleware(contextKey any, header string) *TraceIDMiddleware {
	if contextKey == nil {
		contextKey = TraceIDKey
	}

	if header == "" {
		header = "Platforma-Trace-Id"
	}

	return &TraceIDMiddleware{contextKey: contextKey, header: header}
}

// Wrap adds trace ID to requests.
func (m *TraceIDMiddleware) Wrap(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := uuid.NewString()
		ctx := context.WithValue(r.Context(), m.contextKey, traceID)
		r = r.WithContext(ctx)

		w.Header().Set(m.header, traceID)

		h.ServeHTTP(w, r)
	})
}
