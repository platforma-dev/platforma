package queue_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/platforma-dev/platforma/queue"
)

func TestProcessor(t *testing.T) {
	t.Parallel()

	t.Run("simple queue", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var res atomic.Int32

		q := &mockQueue[job]{
			jobChan: make(chan job, 10),
		}

		p := queue.New(queue.HandlerFunc[job](func(_ context.Context, job job) {
			res.Add(int32(job.data))
		}), q, 4, time.Microsecond)

		go p.Run(ctx)

		p.Enqueue(ctx, job{data: 1})
		p.Enqueue(ctx, job{data: 1})
		p.Enqueue(ctx, job{data: 1})

		// Wait with timeout for jobs to be processed
		deadline := time.Now().Add(5 * time.Second)
		for res.Load() != 3 && time.Now().Before(deadline) {
			time.Sleep(10 * time.Millisecond)
		}

		if res.Load() != 3 {
			t.Errorf("expected res to be 3, got %d", res.Load())
		}
	})

	t.Run("enqueue fail", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		res := 0

		var someErr = errors.New("some error")
		q := &mockQueue[job]{
			jobChan:    make(chan job, 10),
			enqueueJob: func(_ context.Context, _ job) error { return someErr },
		}

		p := queue.New(queue.HandlerFunc[job](func(_ context.Context, job job) {
			res += job.data
		}), q, 4, time.Microsecond)

		go p.Run(context.TODO())

		err := p.Enqueue(ctx, job{data: 1})
		if !errors.Is(err, someErr) {
			t.Fatalf("expected specific error, got: %s", err.Error())
		}
	})

	t.Run("run fail", func(t *testing.T) {
		t.Parallel()

		t.Run("open", func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			res := 0

			var someErr = errors.New("some error")
			q := &mockQueue[job]{
				jobChan: make(chan job, 10),
				open:    func(_ context.Context) error { return someErr },
			}

			p := queue.New(queue.HandlerFunc[job](func(_ context.Context, job job) {
				res += job.data
			}), q, 4, time.Microsecond)

			err := p.Run(ctx)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("close", func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(context.Background())
			res := 0

			var someErr = errors.New("some error")
			q := &mockQueue[job]{
				jobChan: make(chan job, 10),
				close:   func(_ context.Context) error { return someErr },
			}

			p := queue.New(queue.HandlerFunc[job](func(_ context.Context, job job) {
				res += job.data
			}), q, 4, time.Millisecond)

			go func() {
				time.Sleep(100 * time.Millisecond)
				cancel()
			}()

			err := p.Run(ctx)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	})
}

type job struct {
	data int
}

type mockQueue[T any] struct {
	jobChan    chan T
	enqueueJob func(ctx context.Context, job T) error
	open       func(ctx context.Context) error
	close      func(ctx context.Context) error
}

func (q *mockQueue[T]) Open(ctx context.Context) error {
	if q.open != nil {
		return q.open(ctx)
	}

	return nil
}

func (q *mockQueue[T]) Close(ctx context.Context) error {
	if q.close != nil {
		return q.close(ctx)
	}

	return nil
}

func (q *mockQueue[T]) EnqueueJob(ctx context.Context, job T) error {
	if q.enqueueJob != nil {
		return q.enqueueJob(ctx, job)
	}

	q.jobChan <- job
	return nil
}

func (q *mockQueue[T]) GetJobChan(_ context.Context) (chan T, error) {
	return q.jobChan, nil
}
