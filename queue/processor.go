package queue

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/platforma-dev/platforma/log"
)

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
	err := p.queue.Open(ctx)
	if err != nil {
		return fmt.Errorf("failed to open queue: %w", err)
	}

	p.wg.Add(p.workersAmount)
	for range p.workersAmount {
		workerCtx := context.WithValue(ctx, log.WorkerIDKey, uuid.NewString())

		go p.worker(workerCtx)
	}

	p.wg.Wait()

	log.InfoContext(ctx, "all workers shut down")

	err = p.queue.Close(ctx)
	if err != nil {
		return fmt.Errorf("failed to close queue: %w", err)
	}

	return nil
}

func (p *Processor[T]) worker(ctx context.Context) {
	defer p.wg.Done()
	defer log.InfoContext(ctx, "worker finished")
	defer func() {
		if r := recover(); r != nil {
			log.ErrorContext(ctx, "worker panic recovered", "panic", r)
		}
	}()

	log.InfoContext(ctx, "worker started")

	jobChan, err := p.queue.GetJobChan(ctx)
	if err != nil {
		log.ErrorContext(ctx, "failed to get job chan", "error", err)
		return
	}

	// we first check for ctx.Done() in separate select statement
	// because select statements choose randomly if both cases are ready
	for {
		breakLoop := false

		select {
		case <-ctx.Done():
			log.InfoContext(ctx, "skipping job due to shutdown")
			breakLoop = true
		default:
			select {
			case job := <-jobChan:
				p.handler.Handle(ctx, job)

			case <-ctx.Done():
				log.InfoContext(ctx, "shutting down worker")
				breakLoop = true
			}
		}

		if breakLoop {
			break
		}
	}

	// after context is cancelled we try to drain remaining jobs from channel
	// before shutdown time expired
	shutdownCtx := context.WithoutCancel(ctx)
	shutdownCtx, cancel := context.WithTimeout(shutdownCtx, p.shutdownTimeout)
	defer cancel()

	// same logic with nested select statements as in main loop
	for {
		select {
		case <-shutdownCtx.Done():
			log.InfoContext(shutdownCtx, "shutdown timeout expired")
			return
		default:
			select {
			case job := <-jobChan:
				p.handler.Handle(shutdownCtx, job)
			case <-shutdownCtx.Done():
				log.InfoContext(shutdownCtx, "shutdown timeout expired")
				return
			}
		}
	}
}
