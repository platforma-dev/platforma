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
	s := scheduler.New(1*time.Second, application.RunnerFunc(func(ctx context.Context) error {
		counter.Add(1)
		return nil
	}))

	go s.Run(context.TODO())

	time.Sleep(3500 * time.Millisecond)

	if counter.Load() != 3 {
		t.Errorf("wrong counter value. expected %v, got %v", 3, counter.Load())
	}
}

func TestErrorRun(t *testing.T) {
	t.Parallel()

	var counter atomic.Int32
	s := scheduler.New(1*time.Second, application.RunnerFunc(func(ctx context.Context) error {
		counter.Add(1)
		return errors.New("some error")
	}))

	go s.Run(context.TODO())

	time.Sleep(3500 * time.Millisecond)

	if counter.Load() != 3 {
		t.Errorf("wrong counter value. expected %v, got %v", 3, counter.Load())
	}
}

func TestContextDecline(t *testing.T) {
	t.Parallel()

	var counter atomic.Int32
	s := scheduler.New(1*time.Second, application.RunnerFunc(func(ctx context.Context) error {
		counter.Add(1)
		return nil
	}))

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(3*time.Second + 10*time.Millisecond)
		cancel()
	}()

	err := s.Run(ctx)

	if counter.Load() != 3 {
		t.Errorf("wrong counter value. expected %v, got %v", 3, counter.Load())
	}

	if err == nil {
		t.Error("expected error, got nil")
	}
}
