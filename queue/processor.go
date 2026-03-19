package queue

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/platforma-dev/platforma/log"
)

const (
	processorRunEventName = "queue.processor.run"
	workerRunEventName    = "queue.worker.run"
)

var errWorkerPanicRecovered = errors.New("worker panic recovered")

// Handler defines the interface for processing jobs.
type Handler[T any] interface {
	Handle(ctx context.Context, job T)
}

// HandlerFunc is an adapter to allow the use of ordinary functions as Handlers.
type HandlerFunc[T any] func(ctx context.Context, job T)

// Handle calls f(ctx, job).
func (f HandlerFunc[T]) Handle(ctx context.Context, job T) {
	f(ctx, job)
}

// Provider defines the interface for queue implementations.
type Provider[T any] interface {
	Open(ctx context.Context) error
	Close(ctx context.Context) error
	EnqueueJob(ctx context.Context, job T) error
	GetJobChan(ctx context.Context) (chan T, error)
}

// Processor manages a pool of workers to process jobs from a queue.
type Processor[T any] struct {
	handler         Handler[T]
	queue           Provider[T]
	wg              sync.WaitGroup
	workersAmount   int
	shutdownTimeout time.Duration
}

// New creates a new Processor with the specified handler, queue, and configuration.
func New[T any](handler Handler[T], queue Provider[T], workersAmount int, shutdownTimeout time.Duration) *Processor[T] {
	return &Processor[T]{handler: handler, queue: queue, workersAmount: workersAmount, shutdownTimeout: shutdownTimeout}
}

// Enqueue adds a job to the queue for processing.
func (p *Processor[T]) Enqueue(ctx context.Context, job T) error {
	err := p.queue.EnqueueJob(ctx, job)
	if err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}
	return nil
}

// Run starts the queue processor and blocks until all workers complete.
func (p *Processor[T]) Run(ctx context.Context) error {
	runEvent := newProcessorRunEvent(p.workersAmount, p.shutdownTimeout)
	defer log.WriteEvent(ctx, runEvent)

	runEvent.AddStep(slog.LevelInfo, "opening queue")

	err := p.queue.Open(ctx)
	if err != nil {
		runEvent.AddError(fmt.Errorf("failed to open queue: %w", err))
		return fmt.Errorf("failed to open queue: %w", err)
	}

	runEvent.AddStep(slog.LevelInfo, "starting workers")

	p.wg.Add(p.workersAmount)
	for range p.workersAmount {
		workerCtx := context.WithValue(ctx, log.WorkerIDKey, uuid.NewString())

		go p.worker(workerCtx)
	}

	p.wg.Wait()

	runEvent.AddStep(slog.LevelInfo, "all workers shut down")

	err = p.queue.Close(ctx)
	if err != nil {
		runEvent.AddError(fmt.Errorf("failed to close queue: %w", err))
		return fmt.Errorf("failed to close queue: %w", err)
	}

	runEvent.AddStep(slog.LevelInfo, "queue closed")

	return nil
}

func (p *Processor[T]) worker(ctx context.Context) {
	defer p.wg.Done()
	log.WriteEvent(ctx, p.runWorker(ctx))
}

func newProcessorRunEvent(workersAmount int, shutdownTimeout time.Duration) *log.Event {
	event := log.NewEvent(processorRunEventName)
	event.AddAttrs(map[string]any{
		"queue.workersAmount":   workersAmount,
		"queue.shutdownTimeout": shutdownTimeout,
	})

	return event
}

func newWorkerRunEvent(shutdownTimeout time.Duration) *log.Event {
	event := log.NewEvent(workerRunEventName)
	event.AddAttrs(map[string]any{
		"queue.shutdownTimeout": shutdownTimeout,
	})

	return event
}

func (p *Processor[T]) runWorker(ctx context.Context) (event *log.Event) {
	event = newWorkerRunEvent(p.shutdownTimeout)
	processedJobs := 0
	drainedJobs := 0

	defer func() {
		event.AddAttrs(map[string]any{
			"queue.processedJobs": processedJobs,
			"queue.drainedJobs":   drainedJobs,
		})
		event.AddStep(slog.LevelInfo, "worker finished")
	}()
	defer func() {
		if r := recover(); r != nil {
			event.AddStep(slog.LevelError, "worker panic recovered")
			event.AddError(fmt.Errorf("%w: %v", errWorkerPanicRecovered, r))
		}
	}()

	event.AddStep(slog.LevelInfo, "worker started")

	jobChan, err := p.queue.GetJobChan(ctx)
	if err != nil {
		event.AddError(fmt.Errorf("failed to get job chan: %w", err))
		return event
	}

	event.AddStep(slog.LevelInfo, "job channel opened")

	// we first check for ctx.Done() in separate select statement
	// because select statements choose randomly if both cases are ready
	for {
		breakLoop := false

		select {
		case <-ctx.Done():
			event.AddStep(slog.LevelInfo, "shutdown requested")
			breakLoop = true
		default:
			select {
			case job := <-jobChan:
				p.handler.Handle(ctx, job)
				processedJobs++

			case <-ctx.Done():
				event.AddStep(slog.LevelInfo, "shutdown requested")
				breakLoop = true
			}
		}

		if breakLoop {
			break
		}
	}

	// after context is cancelled we try to drain remaining jobs from channel
	// before shutdown time expired
	event.AddStep(slog.LevelInfo, "draining remaining jobs")

	shutdownCtx := context.WithoutCancel(ctx)
	shutdownCtx, cancel := context.WithTimeout(shutdownCtx, p.shutdownTimeout)
	defer cancel()

	// same logic with nested select statements as in main loop
	for {
		select {
		case <-shutdownCtx.Done():
			event.AddStep(slog.LevelInfo, "shutdown timeout expired")
			return event
		default:
			select {
			case job := <-jobChan:
				p.handler.Handle(shutdownCtx, job)
				drainedJobs++
			case <-shutdownCtx.Done():
				event.AddStep(slog.LevelInfo, "shutdown timeout expired")
				return event
			}
		}
	}
}
