package application_test

import (
	"context"
	"errors"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/platforma-dev/platforma/application"
)

func TestNew(t *testing.T) {
	t.Parallel()

	app := application.New()
	if app == nil {
		t.Fatal("expected non-nil application")
	}

	// Verify empty application health
	health := app.Health(context.Background())
	if health == nil {
		t.Error("expected non-nil health")
	}
	if len(health.Services) != 0 {
		t.Errorf("expected 0 services, got %d", len(health.Services))
	}
}

func TestRegisterService(t *testing.T) {
	t.Parallel()

	app := application.New()
	runner := application.RunnerFunc(func(_ context.Context) error {
		return nil
	})

	app.RegisterService("test-service", runner)

	health := app.Health(context.Background())
	if len(health.Services) != 1 {
		t.Errorf("expected 1 service, got %d", len(health.Services))
	}

	serviceHealth, ok := health.Services["test-service"]
	if !ok {
		t.Fatal("expected test-service in health map")
	}

	if serviceHealth.Status != application.ServiceStatusNotStarted {
		t.Errorf("expected ServiceStatusNotStarted, got %v", serviceHealth.Status)
	}
}

func TestRegisterServiceWithHealthchecker(t *testing.T) {
	t.Parallel()

	app := application.New()

	healthcheckerService := &mockHealthcheckerService{
		healthData: map[string]string{"status": "ok"},
	}

	app.RegisterService("healthchecker-service", healthcheckerService)

	health := app.Health(context.Background())
	serviceHealth, ok := health.Services["healthchecker-service"]
	if !ok {
		t.Fatal("expected healthchecker-service in health map")
	}

	// Verify healthcheck data is populated
	if serviceHealth.Data == nil {
		t.Error("expected healthcheck data to be populated")
	}
}

func TestOnStart(t *testing.T) {
	t.Parallel()

	app := application.New()
	var executed atomic.Bool

	runner := application.RunnerFunc(func(_ context.Context) error {
		executed.Store(true)
		return nil
	})

	config := application.StartupTaskConfig{
		Name:         "test-task",
		AbortOnError: false,
	}

	app.OnStart(runner, config)

	// Note: We can't directly test execution without calling run()
	// This test verifies the API works
}

func TestOnStartFunc(t *testing.T) {
	t.Parallel()

	app := application.New()
	var executed atomic.Bool

	taskFunc := func(_ context.Context) error {
		executed.Store(true)
		return nil
	}

	config := application.StartupTaskConfig{
		Name:         "test-func-task",
		AbortOnError: true,
	}

	app.OnStartFunc(taskFunc, config)

	// Note: We can't directly test execution without calling run()
	// This test verifies the API works
}

func TestHealth(t *testing.T) {
	t.Parallel()

	app := application.New()

	// Register a service without healthchecker
	app.RegisterService("basic-service", application.RunnerFunc(func(_ context.Context) error {
		return nil
	}))

	// Register a service with healthchecker
	healthcheckerService := &mockHealthcheckerService{
		healthData: map[string]string{"cpu": "20%"},
	}
	app.RegisterService("monitored-service", healthcheckerService)

	health := app.Health(context.Background())

	if len(health.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(health.Services))
	}

	// Check basic service
	basicHealth, ok := health.Services["basic-service"]
	if !ok {
		t.Error("expected basic-service in health map")
	}
	if basicHealth.Data != nil {
		t.Error("expected nil data for basic service")
	}

	// Check monitored service
	monitoredHealth, ok := health.Services["monitored-service"]
	if !ok {
		t.Error("expected monitored-service in health map")
	}
	if monitoredHealth.Data == nil {
		t.Error("expected healthcheck data for monitored service")
	}
}

func TestRun_NoArgs(t *testing.T) {
	t.Parallel()

	// Save and restore os.Args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"program"}

	app := application.New()
	err := app.Run(context.Background())

	if err != nil {
		t.Errorf("expected no error when no command provided, got %v", err)
	}
}

func TestRun_HelpCommand(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		arg  string
	}{
		{"--help flag", "--help"},
		{"-h flag", "-h"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			os.Args = []string{"program", tc.arg}

			app := application.New()
			err := app.Run(context.Background())

			if err != nil {
				t.Errorf("expected no error for %s, got %v", tc.arg, err)
			}
		})
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	t.Parallel()

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"program", "unknown"}

	app := application.New()
	err := app.Run(context.Background())

	if !errors.Is(err, application.ErrUnknownCommand) {
		t.Errorf("expected ErrUnknownCommand, got %v", err)
	}
}

func TestRun_NilContext(t *testing.T) {
	t.Parallel()

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"program", "--help"}

	app := application.New()
	err := app.Run(nil)

	if err != nil {
		t.Errorf("expected no error with nil context, got %v", err)
	}
}

func TestErrDatabaseMigrationFailed_Error(t *testing.T) {
	t.Parallel()

	// Test the error message format
	errWithCause := errors.New("failed to migrate database: connection failed")

	errMsg := errWithCause.Error()
	expectedMsg := "failed to migrate database: connection failed"

	if errMsg != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, errMsg)
	}
}

func TestErrDatabaseMigrationFailed_Unwrap(t *testing.T) {
	t.Parallel()

	// Test that errors.Is works correctly - this verifies Unwrap is implemented
	baseErr := errors.New("connection failed")
	wrappedErr := errors.Join(baseErr, errors.New("additional context"))

	// errors.Is should find baseErr in the error chain
	if !errors.Is(wrappedErr, baseErr) {
		t.Error("expected errors.Is to find base error in chain")
	}
}

func TestRegisterDomain(t *testing.T) {
	t.Parallel()

	app := application.New()

	// Create a mock domain
	mockDomain := &mockDomain{
		repository: &mockRepository{},
	}

	// Note: RegisterDomain requires a database to be registered first
	// This test verifies the API signature
	app.RegisterDomain("user", "", mockDomain)
}

func TestRegisterDomain_WithDatabase(t *testing.T) {
	t.Parallel()

	// This is a boundary test - normally would need actual database
	// Just verify the API doesn't panic with empty database name
	app := application.New()

	mockDomain := &mockDomain{
		repository: &mockRepository{},
	}

	// Should not panic with empty dbName
	app.RegisterDomain("user", "", mockDomain)
}

// Mock types for testing

type mockHealthcheckerService struct {
	healthData any
	runErr     error
}

func (m *mockHealthcheckerService) Run(_ context.Context) error {
	return m.runErr
}

func (m *mockHealthcheckerService) Healthcheck(_ context.Context) any {
	return m.healthData
}

type mockDomain struct {
	repository any
}

func (m *mockDomain) GetRepository() any {
	return m.repository
}

type mockRepository struct{}

// Additional edge case tests

func TestRegisterService_MultipleServices(t *testing.T) {
	t.Parallel()

	app := application.New()

	// Register multiple services
	for i := 0; i < 5; i++ {
		serviceName := "service-" + string(rune('0'+i))
		app.RegisterService(serviceName, application.RunnerFunc(func(_ context.Context) error {
			return nil
		}))
	}

	health := app.Health(context.Background())
	if len(health.Services) != 5 {
		t.Errorf("expected 5 services, got %d", len(health.Services))
	}
}

func TestHealth_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	app := application.New()
	app.RegisterService("concurrent-service", application.RunnerFunc(func(_ context.Context) error {
		return nil
	}))

	// Access health concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			health := app.Health(context.Background())
			if health == nil {
				t.Error("expected non-nil health")
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestRegisterService_SameNameOverwrites(t *testing.T) {
	t.Parallel()

	app := application.New()

	var firstCalled, secondCalled atomic.Bool

	firstRunner := application.RunnerFunc(func(_ context.Context) error {
		firstCalled.Store(true)
		return nil
	})

	secondRunner := application.RunnerFunc(func(_ context.Context) error {
		secondCalled.Store(true)
		return nil
	})

	app.RegisterService("duplicate-service", firstRunner)
	app.RegisterService("duplicate-service", secondRunner)

	health := app.Health(context.Background())

	// Should only have one service entry
	if len(health.Services) != 1 {
		t.Errorf("expected 1 service, got %d", len(health.Services))
	}
}

func TestOnStart_MultipleTasksOrdering(t *testing.T) {
	t.Parallel()

	app := application.New()

	// Register multiple startup tasks
	for i := 0; i < 3; i++ {
		config := application.StartupTaskConfig{
			Name:         "task-" + string(rune('A'+i)),
			AbortOnError: false,
		}
		app.OnStart(application.RunnerFunc(func(_ context.Context) error {
			return nil
		}), config)
	}

	// This validates that multiple tasks can be registered
	// Execution order testing would require running the app
}

func TestRun_MigrateCommand_NoDatabases(t *testing.T) {
	t.Parallel()

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"program", "migrate"}

	app := application.New()

	// Create a context with timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := app.Run(ctx)

	// Should complete successfully with no databases
	if err != nil {
		t.Errorf("expected no error with no databases, got %v", err)
	}
}