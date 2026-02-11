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

	var counter atomic.Int32
	s, err := scheduler.New("@every 1s", application.RunnerFunc(func(ctx context.Context) error {
		counter.Add(1)
		return nil
	}))
	if err != nil {
		t.Fatalf("failed to create scheduler: %v", err)
	}

	go s.Run(context.TODO())

	time.Sleep(3500 * time.Millisecond)

	if counter.Load() != 3 {
		t.Errorf("wrong counter value. expected %v, got %v", 3, counter.Load())
	}
}

func TestErrorRun(t *testing.T) {
	t.Parallel()

	var counter atomic.Int32
	s, err := scheduler.New("@every 1s", application.RunnerFunc(func(ctx context.Context) error {
		counter.Add(1)
		return errors.New("some error")
	}))
	if err != nil {
		t.Fatalf("failed to create scheduler: %v", err)
	}

	go s.Run(context.TODO())

	time.Sleep(3500 * time.Millisecond)

	if counter.Load() != 3 {
		t.Errorf("wrong counter value. expected %v, got %v", 3, counter.Load())
	}
}

func TestContextDecline(t *testing.T) {
	t.Parallel()

	var counter atomic.Int32
	s, err := scheduler.New("@every 1s", application.RunnerFunc(func(ctx context.Context) error {
		counter.Add(1)
		return nil
	}))
	if err != nil {
		t.Fatalf("failed to create scheduler: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(3*time.Second + 10*time.Millisecond)
		cancel()
	}()

	runErr := s.Run(ctx)

	if counter.Load() != 3 {
		t.Errorf("wrong counter value. expected %v, got %v", 3, counter.Load())
	}

	if runErr == nil {
		t.Error("expected error, got nil")
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
		{"weekday mornings", "0 9 * * MON-FRI"},
		{"specific time", "30 14 * * *"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s, err := scheduler.New(tc.expr, application.RunnerFunc(func(ctx context.Context) error {
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

			s, err := scheduler.New(tc.expr, application.RunnerFunc(func(ctx context.Context) error {
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

	var counter atomic.Int32
	s, err := scheduler.New("@every 1s", application.RunnerFunc(func(ctx context.Context) error {
		counter.Add(1)
		return nil
	}))

	if err != nil {
		t.Fatalf("failed to create scheduler: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go s.Run(ctx)

	// Wait for approximately 3 executions (3.5 seconds to account for timing variations)
	time.Sleep(3500 * time.Millisecond)
	cancel()

	// Allow time for graceful shutdown
	time.Sleep(100 * time.Millisecond)

	// Should have executed 3 times (at ~1s, ~2s, ~3s)
	count := counter.Load()
	if count < 2 || count > 4 {
		t.Errorf("expected 3 executions (±1), got %v", count)
	}
}

func TestCronScheduling_ErrorHandling(t *testing.T) {
	t.Parallel()

	var counter atomic.Int32
	s, err := scheduler.New("@every 1s", application.RunnerFunc(func(ctx context.Context) error {
		counter.Add(1)
		return errors.New("task error")
	}))

	if err != nil {
		t.Fatalf("failed to create scheduler: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go s.Run(ctx)

	time.Sleep(3500 * time.Millisecond)
	cancel()

	// Allow time for graceful shutdown
	time.Sleep(100 * time.Millisecond)

	// Errors should not stop execution - should still run multiple times
	count := counter.Load()
	if count < 2 || count > 4 {
		t.Errorf("expected 3 executions (±1) despite errors, got %v", count)
	}
}

func TestCronScheduling_ContextCancellation(t *testing.T) {
	t.Parallel()

	var counter atomic.Int32
	s, err := scheduler.New("@every 1s", application.RunnerFunc(func(ctx context.Context) error {
		counter.Add(1)
		return nil
	}))

	if err != nil {
		t.Fatalf("failed to create scheduler: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(2500 * time.Millisecond)
		cancel()
	}()

	runErr := s.Run(ctx)

	// Should have executed 2-3 times before cancellation
	count := counter.Load()
	if count < 1 || count > 3 {
		t.Errorf("expected 2 executions (±1), got %v", count)
	}

	if runErr == nil {
		t.Error("expected error from context cancellation, got nil")
	}
}

func TestScheduling_HourlyDescriptor(t *testing.T) {
	t.Parallel()

	// This test validates that the @hourly descriptor is accepted
	// We won't wait an hour, just verify it's created successfully
	var executed atomic.Bool
	s, err := scheduler.New("@hourly", application.RunnerFunc(func(ctx context.Context) error {
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
