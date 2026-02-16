package log

import (
	"context"
	"fmt"
	"net/http"
)

const defaultWideEventName = "http.request"

// WideEventMiddleware creates and writes a request-wide event.
type WideEventMiddleware struct {
	logger     *WideEventLogger
	eventName  string
	contextKey any
}

// NewWideEventMiddleware creates middleware that stores a wide event in request context
// and writes it after request processing.
func NewWideEventMiddleware(logger *WideEventLogger, eventName string, contextKey any) *WideEventMiddleware {
	if eventName == "" {
		eventName = defaultWideEventName
	}

	if contextKey == nil {
		contextKey = WideEventKey
	}

	return &WideEventMiddleware{
		logger:     logger,
		eventName:  eventName,
		contextKey: contextKey,
	}
}

// Wrap creates request-wide event, stores it in context and writes event after handling.
func (m *WideEventMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		event := NewEvent(m.eventName)
		event.AddAttrs(map[string]any{
			"request.method":     r.Method,
			"request.path":       r.URL.Path,
			"request.remoteAddr": r.RemoteAddr,
		})

		ctx := context.WithValue(r.Context(), m.contextKey, event)
		r = r.WithContext(ctx)

		recorder := &statusResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		defer func() {
			recovered := recover()
			if recovered != nil {
				event.AddError(fmt.Errorf("panic: %v", recovered))
				if !recorder.wroteHeader {
					recorder.statusCode = http.StatusInternalServerError
				}
			}

			event.AddAttrs(map[string]any{
				"request.status": recorder.statusCode,
			})
			m.logger.WriteEvent(ctx, event)

			if recovered != nil {
				panic(recovered)
			}
		}()

		next.ServeHTTP(recorder, r)
	})
}

type statusResponseWriter struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func (w *statusResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *statusResponseWriter) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	return w.ResponseWriter.Write(p)
}
