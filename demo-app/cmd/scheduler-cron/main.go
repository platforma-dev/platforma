package main

import (
	"context"
	"fmt"
	"time"

	"github.com/platforma-dev/platforma/application"
	"github.com/platforma-dev/platforma/log"
	"github.com/platforma-dev/platforma/scheduler"
)

func dailyBackup(ctx context.Context) error {
	log.InfoContext(ctx, "executing daily backup task")
	return nil
}

func weekdayReport(ctx context.Context) error {
	log.InfoContext(ctx, "generating weekday report")
	return nil
}

func frequentHealthCheck(ctx context.Context) error {
	log.InfoContext(ctx, "performing health check")
	return nil
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Example 1: Using @every syntax - every 5 seconds
	s1, err := scheduler.NewWithCron("@every 5s", application.RunnerFunc(func(ctx context.Context) error {
		log.InfoContext(ctx, "@every syntax: every 5 seconds")
		return nil
	}))
	if err != nil {
		log.ErrorContext(ctx, "failed to create scheduler 1", "error", err)
		return
	}

	// Example 2: Using @every syntax - every 3 seconds
	s2, err := scheduler.NewWithCron("@every 3s", application.RunnerFunc(func(ctx context.Context) error {
		log.InfoContext(ctx, "@every syntax: every 3 seconds")
		return nil
	}))
	if err != nil {
		log.ErrorContext(ctx, "failed to create scheduler 2", "error", err)
		return
	}

	// Example 3: Daily task (would run at midnight, but won't execute in this demo)
	s3, err := scheduler.NewWithCron("@daily", application.RunnerFunc(dailyBackup))
	if err != nil {
		log.ErrorContext(ctx, "failed to create scheduler 3", "error", err)
		return
	}

	// Example 4: Weekday task (would run at 9 AM on weekdays, won't execute in this demo)
	s4, err := scheduler.NewWithCron("0 9 * * MON-FRI", application.RunnerFunc(weekdayReport))
	if err != nil {
		log.ErrorContext(ctx, "failed to create scheduler 4", "error", err)
		return
	}

	// Example 5: Hourly task (won't execute in this demo)
	s5, err := scheduler.NewWithCron("@hourly", application.RunnerFunc(frequentHealthCheck))
	if err != nil {
		log.ErrorContext(ctx, "failed to create scheduler 5", "error", err)
		return
	}

	fmt.Println("Starting cron scheduler demo...")
	fmt.Println("Active schedulers:")
	fmt.Println("  1. Every 5 seconds (@every 5s)")
	fmt.Println("  2. Every 3 seconds (@every 3s)")
	fmt.Println("  3. Daily at midnight (@daily) - won't execute in demo")
	fmt.Println("  4. Weekdays at 9 AM (0 9 * * MON-FRI) - won't execute in demo")
	fmt.Println("  5. Hourly (@hourly) - won't execute in demo")
	fmt.Println("\nWatch the logs for executions. Demo will run for 15 seconds.\n")

	// Start all schedulers in background
	go s1.Run(ctx)
	go s2.Run(ctx)
	go s3.Run(ctx)
	go s4.Run(ctx)
	go s5.Run(ctx)

	// Run for 15 seconds to demonstrate the frequent tasks
	time.Sleep(15 * time.Second)
	cancel()

	// Allow graceful shutdown
	time.Sleep(100 * time.Millisecond)

	fmt.Println("\nDemo completed!")
}
