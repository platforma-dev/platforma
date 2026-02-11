package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/platforma-dev/platforma/application"
	"github.com/platforma-dev/platforma/log"

	"github.com/google/uuid"
	cron "github.com/pardnchiu/go-scheduler"
)

// scheduleMode represents the type of scheduling strategy.
type scheduleMode int

const (
	scheduleModeInterval scheduleMode = iota // Fixed interval-based scheduling
	scheduleModeCron                         // Cron expression-based scheduling
)

// Scheduler represents a periodic task runner that executes an action at fixed intervals or via cron expressions.
type Scheduler struct {
	period   time.Duration      // The interval between action executions (for interval mode)
	cronExpr string             // The cron expression (for cron mode)
	mode     scheduleMode       // The scheduling mode (interval or cron)
	runner   application.Runner // The runner to execute periodically
}

// New creates a new Scheduler instance with the specified period and action.
// The scheduler executes the runner at fixed intervals.
func New(period time.Duration, runner application.Runner) *Scheduler {
	return &Scheduler{
		period: period,
		runner: runner,
		mode:   scheduleModeInterval,
	}
}

// NewWithCron creates a new Scheduler instance with a cron expression.
// The scheduler executes the runner according to the cron schedule.
//
// Supported cron formats:
//   - Standard 5-field cron: "minute hour day month weekday" (e.g., "0 9 * * MON-FRI")
//   - Custom descriptors: @yearly, @monthly, @weekly, @daily, @hourly
//   - Interval syntax: @every 5m, @every 2h, @every 30s
//
// Examples:
//   - "*/5 * * * *" - Every 5 minutes
//   - "0 */2 * * *" - Every 2 hours at minute 0
//   - "0 9 * * MON-FRI" - 9 AM on weekdays
//   - "@daily" - Every day at midnight
//   - "@every 30m" - Every 30 minutes
//
// Returns an error if the cron expression is invalid.
func NewWithCron(cronExpr string, runner application.Runner) (*Scheduler, error) {
	// Validate the cron expression by attempting to create a scheduler
	testScheduler, err := cron.New(cron.Config{Location: time.UTC})
	if err != nil {
		return nil, fmt.Errorf("failed to create cron validator: %w", err)
	}

	// Attempt to add a test task to validate the expression
	_, err = testScheduler.Add(cronExpr, func() {})
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression %q: %w", cronExpr, err)
	}

	return &Scheduler{
		cronExpr: cronExpr,
		runner:   runner,
		mode:     scheduleModeCron,
	}, nil
}

// Run starts the scheduler and executes the runner at the configured interval or cron schedule.
// The scheduler will continue running until the context is canceled.
func (s *Scheduler) Run(ctx context.Context) error {
	switch s.mode {
	case scheduleModeInterval:
		return s.runInterval(ctx)
	case scheduleModeCron:
		return s.runCron(ctx)
	default:
		return fmt.Errorf("unknown schedule mode: %d", s.mode)
	}
}

// runInterval executes the scheduler using fixed interval timing.
func (s *Scheduler) runInterval(ctx context.Context) error {
	ticker := time.NewTicker(s.period)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			runCtx := context.WithValue(ctx, log.TraceIDKey, uuid.NewString())
			log.InfoContext(runCtx, "scheduler task started")

			err := s.runner.Run(runCtx)
			if err != nil {
				log.ErrorContext(runCtx, "error in scheduler", "error", err)
			}

			log.InfoContext(runCtx, "scheduler task finished")
		case <-ctx.Done():
			return fmt.Errorf("scheduler context canceled: %w", ctx.Err())
		}
	}
}

// runCron executes the scheduler using cron expression timing.
func (s *Scheduler) runCron(ctx context.Context) error {
	// Create a new cron scheduler
	cronScheduler, err := cron.New(cron.Config{Location: time.UTC})
	if err != nil {
		return fmt.Errorf("failed to create cron scheduler: %w", err)
	}

	// Add the task to the cron scheduler
	// Wrap the runner to maintain consistent logging with trace IDs
	_, err = cronScheduler.Add(s.cronExpr, func() error {
		runCtx := context.WithValue(ctx, log.TraceIDKey, uuid.NewString())
		log.InfoContext(runCtx, "scheduler task started")

		err := s.runner.Run(runCtx)
		if err != nil {
			log.ErrorContext(runCtx, "error in scheduler", "error", err)
		}

		log.InfoContext(runCtx, "scheduler task finished")
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to add cron task: %w", err)
	}

	// Start the cron scheduler
	cronScheduler.Start()

	// Wait for context cancellation
	<-ctx.Done()

	// Stop the cron scheduler and wait for tasks to complete
	stopCtx := cronScheduler.Stop()
	<-stopCtx.Done()

	return fmt.Errorf("scheduler context canceled: %w", ctx.Err())
}
