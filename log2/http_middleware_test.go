package log2_test

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/platforma-dev/platforma/log2"
)

func TestHTTPMiddlewareEmitsSingleWideEvent(t *testing.T) {
	t.Parallel()

	buffer := &bytes.Buffer{}
	logger := log2.New(log2.Config{
		Writer:  buffer,
		Format:  "json",
		Level:   slog.LevelInfo,
		Sampler: samplerFunc(func(log2.EventView) log2.SamplingDecision { return log2.SamplingDecision{Keep: true, Reason: "forced"} }),
	})

	middleware := log2.NewHTTPMiddleware(logger, log2.HTTPMiddlewareConfig{})
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ev, ok := log2.EventFromContext(r.Context())
		if !ok {
			t.Fatalf("expected event in request context")
		}
		ev.Add("handler", "users")

		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("hello"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	events := readEvents(t, buffer)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	event := events[0]
	if event["event"] != "http_request" {
		t.Fatalf("expected event=http_request, got %v", event["event"])
	}

	attrs := requireMap(t, event, "attrs")
	if attrs["status"] != float64(http.StatusTeapot) {
		t.Fatalf("expected status %d, got %v", http.StatusTeapot, attrs["status"])
	}
	if attrs["route"] != "/users/42" {
		t.Fatalf("expected fallback route /users/42, got %v", attrs["route"])
	}

	request := requireMap(t, attrs, "request")
	if request["method"] != http.MethodGet {
		t.Fatalf("expected method GET, got %v", request["method"])
	}
	if request["path"] != "/users/42" {
		t.Fatalf("expected path /users/42, got %v", request["path"])
	}

	response := requireMap(t, attrs, "response")
	if response["status"] != float64(http.StatusTeapot) {
		t.Fatalf("expected response.status=%d, got %v", http.StatusTeapot, response["status"])
	}
	if response["bytes"] != float64(5) {
		t.Fatalf("expected response.bytes=5, got %v", response["bytes"])
	}
}

func TestHTTPMiddlewarePanicPath(t *testing.T) {
	t.Parallel()

	buffer := &bytes.Buffer{}
	logger := log2.New(log2.Config{
		Writer:  buffer,
		Format:  "json",
		Level:   slog.LevelInfo,
		Sampler: samplerFunc(func(log2.EventView) log2.SamplingDecision { return log2.SamplingDecision{Keep: true, Reason: "forced"} }),
	})

	middleware := log2.NewHTTPMiddleware(logger, log2.HTTPMiddlewareConfig{})
	handler := middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	panicRecovered := false
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				panicRecovered = true
				if fmt.Sprint(recovered) != "boom" {
					t.Fatalf("expected panic value boom, got %v", recovered)
				}
			}
		}()
		handler.ServeHTTP(rec, req)
	}()

	if !panicRecovered {
		t.Fatalf("expected panic to be rethrown by middleware")
	}

	events := readEvents(t, buffer)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	event := events[0]
	if event["level"] != "ERROR" {
		t.Fatalf("expected level ERROR for panic path, got %v", event["level"])
	}

	errorsList := requireSlice(t, event, "errors")
	if len(errorsList) != 1 {
		t.Fatalf("expected exactly 1 recorded error, got %d", len(errorsList))
	}

	errorMap, ok := errorsList[0].(map[string]any)
	if !ok {
		t.Fatalf("expected errors[0] to be map, got %T", errorsList[0])
	}

	errorText, ok := errorMap["error"].(string)
	if !ok {
		t.Fatalf("expected errors[0].error string, got %T", errorMap["error"])
	}
	if !strings.Contains(errorText, "panic recovered: boom") {
		t.Fatalf("expected panic error text to include panic recovered: boom, got %q", errorText)
	}
}
