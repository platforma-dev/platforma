package scheduler_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/platforma-dev/platforma/application"
	"github.com/platforma-dev/platforma/scheduler"
)

func TestSuccessRun(t *testing.T) {
	t.Parallel()

	// Test that scheduler can be created and started successfully
	s, err := scheduler.New("@hourly", application.RunnerFunc(func(_ context.Context) error {
		return nil
	}))
	if err != nil {
		t.Fatalf("failed to create scheduler: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Verify Run blocks until context is done
	runErr := s.Run(ctx)
	if runErr == nil {
		t.Error("expected context deadline error, got nil")
	}
}

func TestErrorRun(t *testing.T) {
	t.Parallel()

	// Test that scheduler handles runner errors without crashing
	s, err := scheduler.New("@hourly", application.RunnerFunc(func(_ context.Context) error {
		return errors.New("some error")
	}))
	if err != nil {
		t.Fatalf("failed to create scheduler: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Scheduler should run and handle context cancellation gracefully
	runErr := s.Run(ctx)
	if runErr == nil {
		t.Error("expected context deadline error, got nil")
	}
}

func TestContextDecline(t *testing.T) {
	t.Parallel()

	// Test that context cancellation stops the scheduler
	s, err := scheduler.New("@hourly", application.RunnerFunc(func(_ context.Context) error {
		return nil
	}))
	if err != nil {
		t.Fatalf("failed to create scheduler: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	runErr := s.Run(ctx)

	if runErr == nil {
		t.Error("expected error from context cancellation, got nil")
	}
}

// Cron functionality tests

func TestNew_ValidExpression(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		expr string
	}{
		{"standard cron every minute", "* * * * *"},
		{"every 5 minutes", "*/5 * * * *"},
		{"hourly descriptor", "@hourly"},
		{"daily descriptor", "@daily"},
		{"weekly descriptor", "@weekly"},
		{"monthly descriptor", "@monthly"},
		{"yearly descriptor", "@yearly"},
		{"every 30 seconds", "@every 30s"},
		{"every 5 minutes interval", "@every 5m"},
		{"every 2 hours interval", "@every 2h"},
		{"weekday mornings", "0 9 * * 1-5"},
		{"specific time", "30 14 * * *"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s, err := scheduler.New(tc.expr, application.RunnerFunc(func(_ context.Context) error {
				return nil
			}))

			if err != nil {
				t.Errorf("expected no error for valid expression %q, got: %v", tc.expr, err)
			}

			if s == nil {
				t.Error("expected non-nil scheduler")
			}
		})
	}
}

func TestNew_InvalidExpression(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		expr string
	}{
		{"empty expression", ""},
		{"invalid format", "invalid"},
		{"too many fields", "* * * * * * *"},
		{"invalid range", "60 * * * *"},
		{"invalid descriptor", "@invalid"},
		{"invalid interval", "@every abc"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s, err := scheduler.New(tc.expr, application.RunnerFunc(func(_ context.Context) error {
				return nil
			}))

			if err == nil {
				t.Errorf("expected error for invalid expression %q, got nil", tc.expr)
			}

			if s != nil {
				t.Error("expected nil scheduler for invalid expression")
			}
		})
	}
}

func TestCronScheduling_ExecutionTiming(t *testing.T) {
	t.Parallel()

	// Test that scheduler respects cron timing with @every syntax
	var counter atomic.Int32
	s, err := scheduler.New("@every 30s", application.RunnerFunc(func(_ context.Context) error {
		counter.Add(1)
		return nil
	}))

	if err != nil {
		t.Fatalf("failed to create scheduler: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Start scheduler - it won't execute within 100ms (first run is at 30s)
	s.Run(ctx)

	// Verify no execution happened yet (needs 30s for first run)
	count := counter.Load()
	if count != 0 {
		t.Errorf("expected 0 executions in 100ms, got %v", count)
	}
}

func TestCronScheduling_ErrorHandling(t *testing.T) {
	t.Parallel()

	// Test that scheduler can be created with error-returning runner
	s, err := scheduler.New("@daily", application.RunnerFunc(func(_ context.Context) error {
		return errors.New("task error")
	}))

	if err != nil {
		t.Fatalf("failed to create scheduler: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Scheduler should handle runner errors gracefully
	runErr := s.Run(ctx)
	if runErr == nil {
		t.Error("expected context timeout error, got nil")
	}
}

func TestCronScheduling_ContextCancellation(t *testing.T) {
	t.Parallel()

	// Test that context cancellation properly stops the scheduler
	s, err := scheduler.New("@every 30s", application.RunnerFunc(func(_ context.Context) error {
		return nil
	}))

	if err != nil {
		t.Fatalf("failed to create scheduler: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	runErr := s.Run(ctx)

	if runErr == nil {
		t.Error("expected error from context cancellation, got nil")
	}
}

func TestScheduling_HourlyDescriptor(t *testing.T) {
	t.Parallel()

	// This test validates that the @hourly descriptor is accepted
	// We won't wait an hour, just verify it's created successfully
	var executed atomic.Bool
	s, err := scheduler.New("@hourly", application.RunnerFunc(func(_ context.Context) error {
		executed.Store(true)
		return nil
	}))

	if err != nil {
		t.Errorf("expected no error for @hourly descriptor, got: %v", err)
	}

	if s == nil {
		t.Error("expected non-nil scheduler")
	}

	// Quick validation that it can start (but won't execute within test time)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	s.Run(ctx)

	// Should not have executed in 100ms
	if executed.Load() {
		t.Error("@hourly task should not execute within 100ms")
	}
}
