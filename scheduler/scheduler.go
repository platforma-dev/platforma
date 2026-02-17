// Package scheduler provides cron-based periodic task execution.
package scheduler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/platforma-dev/platforma/application"
	"github.com/platforma-dev/platforma/log"

	"github.com/google/uuid"
	cron "github.com/robfig/cron/v3"
)

var errEmptyCronExpression = errors.New("cron expression cannot be empty")

const cronParseOptions = cron.Minute |
	cron.Hour |
	cron.Dom |
	cron.Month |
	cron.Dow |
	cron.Descriptor

// Scheduler represents a periodic task runner that executes an action based on a cron expression.
type Scheduler struct {
	cronExpr string             // The cron expression
	runner   application.Runner // The runner to execute periodically
}

// New creates a new Scheduler instance with a cron expression.
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
//   - "@every 1s" - Every second (for intervals, use @every syntax)
//
// Returns an error if the cron expression is invalid.
func New(cronExpr string, runner application.Runner) (*Scheduler, error) {
	// Check for empty expression first to avoid parser errors
	if cronExpr == "" {
		return nil, fmt.Errorf("invalid cron expression %q: %w", cronExpr, errEmptyCronExpression)
	}

	parser := cron.NewParser(cronParseOptions)

	// Validate expression eagerly so errors are returned from constructor
	if _, err := parser.Parse(cronExpr); err != nil {
		return nil, fmt.Errorf("invalid cron expression %q: %w", cronExpr, err)
	}

	return &Scheduler{
		cronExpr: cronExpr,
		runner:   runner,
	}, nil
}

// Run starts the scheduler and executes the runner according to the cron schedule.
// The scheduler will continue running until the context is canceled.
func (s *Scheduler) Run(ctx context.Context) error {
	parser := cron.NewParser(cronParseOptions)

	cronScheduler := cron.New(
		cron.WithLocation(time.UTC),
		cron.WithParser(parser),
	)

	// Wrap runner to maintain consistent logging with trace IDs
	_, err := cronScheduler.AddFunc(s.cronExpr, func() {
		runCtx := context.WithValue(ctx, log.TraceIDKey, uuid.NewString())
		log.InfoContext(runCtx, "scheduler task started")

		err := s.runner.Run(runCtx)
		if err != nil {
			log.ErrorContext(runCtx, "error in scheduler", "error", err)
			return
		}

		log.InfoContext(runCtx, "scheduler task finished")
	})
	if err != nil {
		return fmt.Errorf("failed to add cron task: %w", err)
	}

	cronScheduler.Start()

	<-ctx.Done()

	stopCtx := cronScheduler.Stop()
	<-stopCtx.Done()

	return fmt.Errorf("scheduler context canceled: %w", ctx.Err())
}
