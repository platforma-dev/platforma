package queue

import (
	"context"
	"testing"
	"time"

	"github.com/platforma-dev/platforma/log"
)

func TestNewProcessorRunEvent(t *testing.T) {
	t.Parallel()

	event := newProcessorRunEvent(4, time.Second)

	if got := event.Name(); got != processorRunEventName {
		t.Fatalf("expected event name %q, got %q", processorRunEventName, got)
	}

	if got, ok := event.Attr("queue.workersAmount"); !ok || got != 4 {
		t.Fatalf("expected queue.workersAmount attr, got %#v, exists=%v", got, ok)
	}

	if got, ok := event.Attr("queue.shutdownTimeout"); !ok || got != time.Second {
		t.Fatalf("expected queue.shutdownTimeout attr, got %#v, exists=%v", got, ok)
	}
}

func TestRunWorkerEvent(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	q := &testQueue[workerJob]{
		jobChan: make(chan workerJob, 1),
	}

	q.jobChan <- workerJob{data: 1}

	processor := New(HandlerFunc[workerJob](func(_ context.Context, _ workerJob) {
		cancel()
	}), q, 1, time.Millisecond)

	event := processor.runWorker(ctx)

	if got := event.Name(); got != workerRunEventName {
		t.Fatalf("expected event name %q, got %q", workerRunEventName, got)
	}

	if got, ok := event.Attr("queue.processedJobs"); !ok || got != 1 {
		t.Fatalf("expected queue.processedJobs attr, got %#v, exists=%v", got, ok)
	}

	if got, ok := event.Attr("queue.drainedJobs"); !ok || got != 0 {
		t.Fatalf("expected queue.drainedJobs attr, got %#v, exists=%v", got, ok)
	}

	steps := workerEventSteps(t, event)
	if len(steps) < 5 {
		t.Fatalf("expected worker event steps, got %#v", steps)
	}
}

func workerEventSteps(t *testing.T, event *log.Event) []map[string]any {
	t.Helper()

	for _, attr := range event.ToAttrs() {
		if attr.Key == "steps" {
			steps, ok := attr.Value.Any().([]map[string]any)
			if !ok {
				t.Fatalf("expected []map[string]any for steps, got %T", attr.Value.Any())
			}

			return steps
		}
	}

	return nil
}

type workerJob struct {
	data int
}

type testQueue[T any] struct {
	jobChan chan T
}

func (q *testQueue[T]) Open(_ context.Context) error {
	return nil
}

func (q *testQueue[T]) Close(_ context.Context) error {
	return nil
}

func (q *testQueue[T]) EnqueueJob(_ context.Context, job T) error {
	q.jobChan <- job

	return nil
}

func (q *testQueue[T]) GetJobChan(_ context.Context) (chan T, error) {
	return q.jobChan, nil
}
