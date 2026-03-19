package httpserver_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/platforma-dev/platforma/httpserver"
)

// panicHandler is a test handler that panics with a specific message
type panicHandler struct {
	panicMessage string
}

func (h *panicHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {
	panic(h.panicMessage)
}

// normalHandler is a test handler that returns success
type normalHandler struct{}

func (h *normalHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Success"))
}

func TestRecoverMiddleware_NormalOperation(t *testing.T) {
	t.Parallel()

	// Setup
	middleware := httpserver.NewRecoverMiddleware()
	handler := &normalHandler{}
	wrappedHandler := middleware.Wrap(handler)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	// Execute
	wrappedHandler.ServeHTTP(w, req)

	// Verify
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	body := make([]byte, w.Body.Len())
	w.Body.Read(body)
	if string(body) != "Success" {
		t.Errorf("expected body 'Success', got '%s'", string(body))
	}
}

func TestRecoverMiddleware_PanicRecovery(t *testing.T) {
	t.Parallel()

	middleware := httpserver.NewRecoverMiddleware()
	handler := &panicHandler{panicMessage: "test panic"}
	wrappedHandler := middleware.Wrap(handler)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	// Execute - this should not panic due to the recovery middleware
	wrappedHandler.ServeHTTP(w, req)

	// Verify HTTP response
	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
	}

	body := make([]byte, w.Body.Len())
	w.Body.Read(body)
	expectedBody := "Internal Server Error"
	if string(body) != expectedBody {
		t.Errorf("expected body '%s', got '%s'", expectedBody, string(body))
	}
}

func TestRecoverMiddleware_ErrorResponse(t *testing.T) {
	t.Parallel()

	middleware := httpserver.NewRecoverMiddleware()
	handler := &panicHandler{panicMessage: "specific error for testing"}
	wrappedHandler := middleware.Wrap(handler)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	// Execute
	wrappedHandler.ServeHTTP(w, req)

	// Verify response
	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
	}

	// Verify response body content
	body := make([]byte, w.Body.Len())
	w.Body.Read(body)
	expectedBody := "Internal Server Error"
	if string(body) != expectedBody {
		t.Errorf("expected body '%s', got '%s'", expectedBody, string(body))
	}

	// Verify content type (should be text/plain by default)
	contentType := resp.Header.Get("Content-Type")
	if contentType != "" && contentType != "text/plain; charset=utf-8" {
		t.Errorf("unexpected content type: %s", contentType)
	}
}

func TestRecoverMiddleware_MultiplePanics(t *testing.T) {
	t.Parallel()

	middleware := httpserver.NewRecoverMiddleware()
	handler := &panicHandler{panicMessage: "first panic"}
	wrappedHandler := middleware.Wrap(handler)

	// Test multiple requests to ensure middleware continues to work
	for i := range 3 {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("request %d: expected status %d, got %d", i+1, http.StatusInternalServerError, resp.StatusCode)
		}
	}
}
