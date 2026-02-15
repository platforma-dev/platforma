package log2

import (
	"errors"
	"fmt"
	"net/http"
)

var errPanicRecovered = errors.New("panic recovered") //nolint:gochecknoglobals

// HTTPMiddlewareConfig configures wide-event HTTP middleware.
type HTTPMiddlewareConfig struct {
	EventName string
	RouteAttr string
}

// NewHTTPMiddleware creates middleware that builds one wide event per request.
func NewHTTPMiddleware(l *Logger, cfg HTTPMiddlewareConfig) func(http.Handler) http.Handler {
	logger := l
	if logger == nil {
		logger = Default
	}

	eventName := cfg.EventName
	if eventName == "" {
		eventName = "http_request"
	}

	routeAttr := cfg.RouteAttr
	if routeAttr == "" {
		routeAttr = "route"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ev := logger.Start(r.Context(), eventName)
			ctx := WithEvent(r.Context(), ev)
			r = r.WithContext(ctx)

			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			defer func() {
				panicValue := recover()

				route := r.Pattern
				if route == "" {
					route = r.URL.Path
				}

				requestAttrs := map[string]any{
					"method":     r.Method,
					"path":       r.URL.Path,
					"route":      route,
					"remoteAddr": r.RemoteAddr,
				}
				responseAttrs := map[string]any{
					"status": rec.status,
					"bytes":  rec.bytes,
				}

				ev.Add(
					"request", requestAttrs,
					"response", responseAttrs,
					routeAttr, route,
				)

				if panicValue != nil {
					ev.Error(fmt.Errorf("%w: %v", errPanicRecovered, panicValue))
				}

				_ = ev.Finish("status", rec.status)

				if panicValue != nil {
					panic(panicValue)
				}
			}()

			next.ServeHTTP(rec, r)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *statusRecorder) Write(body []byte) (int, error) {
	bytesWritten, err := r.ResponseWriter.Write(body)
	r.bytes += bytesWritten
	//nolint:wrapcheck // preserve exact error semantics from wrapped ResponseWriter.
	return bytesWritten, err
}
