package application_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/platforma-dev/platforma/application"
)

func TestNewHealth(t *testing.T) {
	t.Parallel()

	health := application.NewHealth()

	if health == nil {
		t.Fatal("expected non-nil health")
	}

	if health.Services == nil {
		t.Error("expected non-nil services map")
	}

	if len(health.Services) != 0 {
		t.Errorf("expected empty services map, got %d entries", len(health.Services))
	}

	// StartedAt should be zero value
	if !health.StartedAt.IsZero() {
		t.Error("expected zero StartedAt time")
	}
}

func TestStartService(t *testing.T) {
	t.Parallel()

	health := application.NewHealth()
	health.Services["test-service"] = &application.ServiceHealth{
		Status: application.ServiceStatusNotStarted,
	}

	beforeStart := time.Now()
	health.StartService("test-service")
	afterStart := time.Now()

	service := health.Services["test-service"]

	if service.Status != application.ServiceStatusStarted {
		t.Errorf("expected status %v, got %v", application.ServiceStatusStarted, service.Status)
	}

	if service.StartedAt == nil {
		t.Fatal("expected non-nil StartedAt")
	}

	if service.StartedAt.Before(beforeStart) || service.StartedAt.After(afterStart) {
		t.Error("StartedAt time should be between before and after timestamps")
	}

	if service.StoppedAt != nil {
		t.Error("expected nil StoppedAt for started service")
	}
}

func TestStartService_NonExistent(t *testing.T) {
	t.Parallel()

	health := application.NewHealth()

	// Starting a non-existent service should not panic
	health.StartService("nonexistent-service")

	// Verify it wasn't added
	if len(health.Services) != 0 {
		t.Error("expected no services to be added")
	}
}

func TestFailService(t *testing.T) {
	t.Parallel()

	health := application.NewHealth()
	health.Services["test-service"] = &application.ServiceHealth{
		Status: application.ServiceStatusStarted,
	}

	testErr := errors.New("service crashed")
	beforeFail := time.Now()
	health.FailService("test-service", testErr)
	afterFail := time.Now()

	service := health.Services["test-service"]

	if service.Status != application.ServiceStatusError {
		t.Errorf("expected status %v, got %v", application.ServiceStatusError, service.Status)
	}

	if service.Error != "service crashed" {
		t.Errorf("expected error message %q, got %q", "service crashed", service.Error)
	}

	if service.StoppedAt == nil {
		t.Fatal("expected non-nil StoppedAt")
	}

	if service.StoppedAt.Before(beforeFail) || service.StoppedAt.After(afterFail) {
		t.Error("StoppedAt time should be between before and after timestamps")
	}
}

func TestFailService_NonExistent(t *testing.T) {
	t.Parallel()

	health := application.NewHealth()

	// Failing a non-existent service should not panic
	testErr := errors.New("test error")
	health.FailService("nonexistent-service", testErr)

	// Verify it wasn't added
	if len(health.Services) != 0 {
		t.Error("expected no services to be added")
	}
}

func TestSetServiceData(t *testing.T) {
	t.Parallel()

	health := application.NewHealth()
	health.Services["test-service"] = &application.ServiceHealth{
		Status: application.ServiceStatusStarted,
	}

	testData := map[string]interface{}{
		"cpu":    "25%",
		"memory": "512MB",
	}

	health.SetServiceData("test-service", testData)

	service := health.Services["test-service"]
	if service.Data == nil {
		t.Fatal("expected non-nil Data")
	}

	data, ok := service.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected Data to be map[string]interface{}")
	}

	if data["cpu"] != "25%" {
		t.Errorf("expected cpu %q, got %q", "25%", data["cpu"])
	}

	if data["memory"] != "512MB" {
		t.Errorf("expected memory %q, got %q", "512MB", data["memory"])
	}
}

func TestSetServiceData_NonExistent(t *testing.T) {
	t.Parallel()

	health := application.NewHealth()

	// Setting data on non-existent service should not panic
	health.SetServiceData("nonexistent-service", "some data")

	// Verify it wasn't added
	if len(health.Services) != 0 {
		t.Error("expected no services to be added")
	}
}

func TestSetServiceData_NilData(t *testing.T) {
	t.Parallel()

	health := application.NewHealth()
	health.Services["test-service"] = &application.ServiceHealth{
		Status: application.ServiceStatusStarted,
	}

	health.SetServiceData("test-service", nil)

	service := health.Services["test-service"]
	if service.Data != nil {
		t.Error("expected Data to be nil")
	}
}

func TestStartApplication(t *testing.T) {
	t.Parallel()

	health := application.NewHealth()

	beforeStart := time.Now()
	health.StartApplication()
	afterStart := time.Now()

	if health.StartedAt.IsZero() {
		t.Error("expected non-zero StartedAt")
	}

	if health.StartedAt.Before(beforeStart) || health.StartedAt.After(afterStart) {
		t.Error("StartedAt should be between before and after timestamps")
	}
}

func TestHealthString(t *testing.T) {
	t.Parallel()

	health := application.NewHealth()
	health.StartApplication()

	startTime := time.Now()
	health.Services["service1"] = &application.ServiceHealth{
		Status:    application.ServiceStatusStarted,
		StartedAt: &startTime,
	}

	jsonStr := health.String()

	if jsonStr == "" {
		t.Error("expected non-empty JSON string")
	}

	// Verify it's valid JSON
	var unmarshaled application.Health
	err := json.Unmarshal([]byte(jsonStr), &unmarshaled)
	if err != nil {
		t.Errorf("expected valid JSON, got error: %v", err)
	}

	if len(unmarshaled.Services) != 1 {
		t.Errorf("expected 1 service in unmarshaled JSON, got %d", len(unmarshaled.Services))
	}
}

func TestHealthString_EmptyServices(t *testing.T) {
	t.Parallel()

	health := application.NewHealth()
	jsonStr := health.String()

	var unmarshaled application.Health
	err := json.Unmarshal([]byte(jsonStr), &unmarshaled)
	if err != nil {
		t.Errorf("expected valid JSON for empty health, got error: %v", err)
	}
}

func TestServiceHealth_JSONMarshaling(t *testing.T) {
	t.Parallel()

	startTime := time.Now()
	stopTime := time.Now().Add(time.Second)

	serviceHealth := &application.ServiceHealth{
		Status:    application.ServiceStatusError,
		StartedAt: &startTime,
		StoppedAt: &stopTime,
		Error:     "test error",
		Data:      map[string]string{"key": "value"},
	}

	jsonBytes, err := json.Marshal(serviceHealth)
	if err != nil {
		t.Fatalf("failed to marshal ServiceHealth: %v", err)
	}

	var unmarshaled application.ServiceHealth
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	if err != nil {
		t.Fatalf("failed to unmarshal ServiceHealth: %v", err)
	}

	if unmarshaled.Status != application.ServiceStatusError {
		t.Errorf("expected status %v, got %v", application.ServiceStatusError, unmarshaled.Status)
	}

	if unmarshaled.Error != "test error" {
		t.Errorf("expected error %q, got %q", "test error", unmarshaled.Error)
	}
}

func TestServiceStatus_Constants(t *testing.T) {
	t.Parallel()

	// Verify the constants have expected values
	if application.ServiceStatusNotStarted != "NOT_STARTED" {
		t.Errorf("expected ServiceStatusNotStarted to be %q, got %q", "NOT_STARTED", application.ServiceStatusNotStarted)
	}

	if application.ServiceStatusStarted != "STARTED" {
		t.Errorf("expected ServiceStatusStarted to be %q, got %q", "STARTED", application.ServiceStatusStarted)
	}

	if application.ServiceStatusError != "ERROR" {
		t.Errorf("expected ServiceStatusError to be %q, got %q", "ERROR", application.ServiceStatusError)
	}
}

func TestHealth_ServiceLifecycle(t *testing.T) {
	t.Parallel()

	health := application.NewHealth()

	// Initialize service
	health.Services["lifecycle-service"] = &application.ServiceHealth{
		Status: application.ServiceStatusNotStarted,
	}

	// Start service
	health.StartService("lifecycle-service")
	if health.Services["lifecycle-service"].Status != application.ServiceStatusStarted {
		t.Error("service should be started")
	}
	if health.Services["lifecycle-service"].StartedAt == nil {
		t.Error("service should have StartedAt time")
	}

	// Add health data
	health.SetServiceData("lifecycle-service", map[string]string{"status": "healthy"})
	if health.Services["lifecycle-service"].Data == nil {
		t.Error("service should have data")
	}

	// Fail service
	health.FailService("lifecycle-service", errors.New("crashed"))
	if health.Services["lifecycle-service"].Status != application.ServiceStatusError {
		t.Error("service should have error status")
	}
	if health.Services["lifecycle-service"].StoppedAt == nil {
		t.Error("service should have StoppedAt time")
	}
	if health.Services["lifecycle-service"].Error == "" {
		t.Error("service should have error message")
	}
}

func TestHealth_MultipleServices(t *testing.T) {
	t.Parallel()

	health := application.NewHealth()

	// Register multiple services with different states
	health.Services["service1"] = &application.ServiceHealth{Status: application.ServiceStatusNotStarted}
	health.Services["service2"] = &application.ServiceHealth{Status: application.ServiceStatusNotStarted}
	health.Services["service3"] = &application.ServiceHealth{Status: application.ServiceStatusNotStarted}

	health.StartService("service1")
	health.StartService("service2")
	health.FailService("service2", errors.New("service2 error"))

	// service1 should be started
	if health.Services["service1"].Status != application.ServiceStatusStarted {
		t.Error("service1 should be started")
	}

	// service2 should be in error state
	if health.Services["service2"].Status != application.ServiceStatusError {
		t.Error("service2 should be in error state")
	}

	// service3 should still be not started
	if health.Services["service3"].Status != application.ServiceStatusNotStarted {
		t.Error("service3 should be not started")
	}
}

func TestServiceHealth_OmitEmptyFields(t *testing.T) {
	t.Parallel()

	// Test that omitempty works correctly
	serviceHealth := &application.ServiceHealth{
		Status: application.ServiceStatusStarted,
	}

	jsonBytes, err := json.Marshal(serviceHealth)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(jsonBytes)

	// Should not contain stoppedAt or error fields when empty
	if contains(jsonStr, "stoppedAt") && !contains(jsonStr, "startedAt") {
		t.Error("JSON should omit stoppedAt when nil")
	}

	if contains(jsonStr, "error") && !contains(jsonStr, `"error":""`) {
		t.Error("JSON should omit error when empty")
	}
}

func TestHealth_ConcurrentModifications(t *testing.T) {
	t.Parallel()

	health := application.NewHealth()

	// Initialize services
	for i := 0; i < 10; i++ {
		serviceName := "service-" + string(rune('0'+i))
		health.Services[serviceName] = &application.ServiceHealth{
			Status: application.ServiceStatusNotStarted,
		}
	}

	// Concurrently modify services
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		serviceName := "service-" + string(rune('0'+i))
		go func(name string) {
			health.StartService(name)
			health.SetServiceData(name, map[string]string{"test": "data"})
			done <- true
		}(serviceName)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all services were started
	for i := 0; i < 10; i++ {
		serviceName := "service-" + string(rune('0'+i))
		if health.Services[serviceName].Status != application.ServiceStatusStarted {
			t.Errorf("service %s should be started", serviceName)
		}
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}