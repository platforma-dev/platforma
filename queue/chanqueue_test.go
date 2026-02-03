package queue_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/platforma-dev/platforma/queue"
)

func TestChanQueue(t *testing.T) {
	t.Parallel()
	t.Run("simple enqueue", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		q := queue.NewChanQueue[job](3, time.Second)

		err := q.Open(ctx)
		if err != nil {
			t.Fatalf("expected no error, got: %s", err.Error())
		}
		defer q.Close(ctx)

		err = q.EnqueueJob(ctx, job{data: 1})
		if err != nil {
			t.Fatalf("expected no error, got: %s", err.Error())
		}

		ch, err := q.GetJobChan(ctx)
		if err != nil {
			t.Fatalf("expected no error, got: %s", err.Error())
		}

		select {
		case j := <-ch:
			if j.data != 1 {
				t.Fatalf("expected data to be 1, got: %d", j.data)
			}
		default:
			t.Fatalf("expected job to be received from channel")
		}
	})

	t.Run("enqueue timeout", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		q := queue.NewChanQueue[job](0, time.Second)

		err := q.Open(ctx)
		if err != nil {
			t.Fatalf("expected no error, got: %s", err.Error())
		}
		defer q.Close(ctx)

		err = q.EnqueueJob(ctx, job{data: 1})
		if !errors.Is(err, queue.ErrTimeout) {
			t.Fatalf("expected timeout error, got: %s", err.Error())
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		q := queue.NewChanQueue[job](0, 30*time.Second)

		err := q.Open(ctx)
		if err != nil {
			t.Fatalf("expected no error, got: %s", err.Error())
		}
		defer q.Close(ctx)

		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()

		err = q.EnqueueJob(ctx, job{data: 1})
		if !strings.Contains(err.Error(), "context cancelled") {
			t.Fatalf("expected context cancellation error, got: %s", err.Error())
		}
	})

	t.Run("enqueue to closed queue", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		q := queue.NewChanQueue[job](0, time.Second)

		err := q.EnqueueJob(ctx, job{data: 1})
		if !errors.Is(err, queue.ErrClosedQueue) {
			t.Fatalf("expected closed queue error, got: %s", err.Error())
		}
	})

}
