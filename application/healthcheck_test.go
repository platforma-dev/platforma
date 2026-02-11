package application_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/platforma-dev/platforma/application"
)

func TestNewHealthCheckHandler(t *testing.T) {
	t.Parallel()

	app := application.New()
	handler := application.NewHealthCheckHandler(app)

	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestHealthCheckHandler_ServeHTTP_Success(t *testing.T) {
	t.Parallel()

	app := application.New()
	app.RegisterService("test-service", application.RunnerFunc(func(_ context.Context) error {
		return nil
	}))

	handler := application.NewHealthCheckHandler(app)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Check status code
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Check content type
	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type %q, got %q", "application/json", contentType)
	}

	// Check response body is valid JSON
	var health application.Health
	err := json.Unmarshal(rec.Body.Bytes(), &health)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Verify services in response
	if len(health.Services) != 1 {
		t.Errorf("expected 1 service, got %d", len(health.Services))
	}

	if _, ok := health.Services["test-service"]; !ok {
		t.Error("expected test-service in health response")
	}
}

func TestHealthCheckHandler_ServeHTTP_EmptyApp(t *testing.T) {
	t.Parallel()

	app := application.New()
	handler := application.NewHealthCheckHandler(app)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var health application.Health
	err := json.Unmarshal(rec.Body.Bytes(), &health)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(health.Services) != 0 {
		t.Errorf("expected 0 services, got %d", len(health.Services))
	}
}

func TestHealthCheckHandler_ServeHTTP_MultipleServices(t *testing.T) {
	t.Parallel()

	app := application.New()

	// Register multiple services
	app.RegisterService("service1", application.RunnerFunc(func(_ context.Context) error {
		return nil
	}))
	app.RegisterService("service2", application.RunnerFunc(func(_ context.Context) error {
		return nil
	}))
	app.RegisterService("service3", application.RunnerFunc(func(_ context.Context) error {
		return nil
	}))

	handler := application.NewHealthCheckHandler(app)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	var health application.Health
	err := json.Unmarshal(rec.Body.Bytes(), &health)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(health.Services) != 3 {
		t.Errorf("expected 3 services, got %d", len(health.Services))
	}
}

func TestHealthCheckHandler_ServeHTTP_WithHealthchecker(t *testing.T) {
	t.Parallel()

	app := application.New()

	// Register service with healthchecker
	healthcheckerService := &mockHealthcheckerService{
		healthData: map[string]string{
			"status":  "healthy",
			"uptime":  "120s",
			"version": "1.0.0",
		},
	}
	app.RegisterService("monitored-service", healthcheckerService)

	handler := application.NewHealthCheckHandler(app)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	var health application.Health
	err := json.Unmarshal(rec.Body.Bytes(), &health)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	serviceHealth, ok := health.Services["monitored-service"]
	if !ok {
		t.Fatal("expected monitored-service in response")
	}

	if serviceHealth.Data == nil {
		t.Error("expected healthcheck data to be populated")
	}
}

func TestHealthCheckHandler_ServeHTTP_POSTRequest(t *testing.T) {
	t.Parallel()

	app := application.New()
	handler := application.NewHealthCheckHandler(app)

	// Health endpoint should work with POST too (though GET is typical)
	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should still return 200 OK
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d for POST, got %d", http.StatusOK, rec.Code)
	}
}

func TestHealthCheckHandler_ServeHTTP_ContextPropagation(t *testing.T) {
	t.Parallel()

	app := application.New()

	// Use a custom healthchecker that checks context
	contextChecker := &contextCheckingHealthchecker{
		t: t,
	}
	app.RegisterService("context-service", contextChecker)

	handler := application.NewHealthCheckHandler(app)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if !contextChecker.contextReceived {
		t.Error("expected context to be passed to healthcheck")
	}
}

func TestHealthCheckHandler_ServeHTTP_JSONFormat(t *testing.T) {
	t.Parallel()

	app := application.New()
	app.RegisterService("json-test", application.RunnerFunc(func(_ context.Context) error {
		return nil
	}))

	handler := application.NewHealthCheckHandler(app)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	body := rec.Body.String()

	// Verify JSON structure
	if !strings.Contains(body, `"services"`) {
		t.Error("expected JSON to contain 'services' field")
	}

	if !strings.Contains(body, `"startedAt"`) {
		t.Error("expected JSON to contain 'startedAt' field")
	}

	// Verify it's valid JSON with proper structure
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(body), &parsed)
	if err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	if _, ok := parsed["services"]; !ok {
		t.Error("expected 'services' key in JSON")
	}

	if _, ok := parsed["startedAt"]; !ok {
		t.Error("expected 'startedAt' key in JSON")
	}
}

func TestHealthCheckHandler_ServeHTTP_ConcurrentRequests(t *testing.T) {
	t.Parallel()

	app := application.New()
	app.RegisterService("concurrent-service", application.RunnerFunc(func(_ context.Context) error {
		return nil
	}))

	handler := application.NewHealthCheckHandler(app)

	// Make concurrent requests
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
			}

			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestHealthCheckHandler_ServeHTTP_LargeResponse(t *testing.T) {
	t.Parallel()

	app := application.New()

	// Register many services
	for i := 0; i < 100; i++ {
		serviceName := "service-" + string(rune('0'+(i%10)))
		if i >= 10 {
			serviceName = serviceName + string(rune('0'+(i/10)))
		}
		app.RegisterService(serviceName, application.RunnerFunc(func(_ context.Context) error {
			return nil
		}))
	}

	handler := application.NewHealthCheckHandler(app)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Verify large response is valid JSON
	var health application.Health
	err := json.Unmarshal(rec.Body.Bytes(), &health)
	if err != nil {
		t.Fatalf("failed to unmarshal large response: %v", err)
	}
}

func TestHealthCheckHandler_ServeHTTP_WithRequestBody(t *testing.T) {
	t.Parallel()

	app := application.New()
	handler := application.NewHealthCheckHandler(app)

	// Health check should ignore request body
	body := strings.NewReader(`{"ignored": "data"}`)
	req := httptest.NewRequest(http.MethodGet, "/health", body)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestHealthCheckHandler_ServiceStates(t *testing.T) {
	t.Parallel()

	app := application.New()

	// Register services in different states
	app.RegisterService("not-started", application.RunnerFunc(func(_ context.Context) error {
		return nil
	}))

	handler := application.NewHealthCheckHandler(app)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	var health application.Health
	err := json.Unmarshal(rec.Body.Bytes(), &health)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	service := health.Services["not-started"]
	if service.Status != application.ServiceStatusNotStarted {
		t.Errorf("expected status %v, got %v", application.ServiceStatusNotStarted, service.Status)
	}
}

// Mock types for testing

type contextCheckingHealthchecker struct {
	t                *testing.T
	contextReceived  bool
}

func (c *contextCheckingHealthchecker) Run(_ context.Context) error {
	return nil
}

func (c *contextCheckingHealthchecker) Healthcheck(ctx context.Context) any {
	if ctx == nil {
		c.t.Error("expected non-nil context in healthcheck")
	} else {
		c.contextReceived = true
	}
	return map[string]string{"status": "ok"}
}