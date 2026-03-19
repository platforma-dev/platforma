package log_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	platformalog "github.com/platforma-dev/platforma/log"
)

func TestTraceIDMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("default params", func(t *testing.T) {
		t.Parallel()

		m := platformalog.NewTraceIDMiddleware(nil, "")
		wrappedHandler := m.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			i, ok := r.Context().Value(platformalog.TraceIDKey).(string)
			if ok {
				w.Header().Add("TraceIdFromContext", i)
			}
		}))

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, r)
		resp := w.Result()

		if len(resp.Header.Get("Platforma-Trace-Id")) == 0 {
			t.Fatalf("default trace id header expected, got: %s", resp.Header)
		}

		if len(resp.Header.Get("TraceIdFromContext")) == 0 {
			t.Fatalf("trace id from context expected, got: %s", resp.Header)
		}
	})
}
