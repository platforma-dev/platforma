package main

import (
	"context"
	"time"

	"github.com/platforma-dev/platforma/application"
	"github.com/platforma-dev/platforma/log"
	"github.com/platforma-dev/platforma/scheduler"
)

func scheduledTask(ctx context.Context) error {
	log.InfoContext(ctx, "scheduled task executed")
	return nil
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	s, err := scheduler.New("@every 1s", application.RunnerFunc(scheduledTask))
	if err != nil {
		log.ErrorContext(ctx, "failed to create scheduler", "error", err)
		return
	}

	go func() {
		time.Sleep(3500 * time.Millisecond)
		cancel()
	}()

	s.Run(ctx)
}
