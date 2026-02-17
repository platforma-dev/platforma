package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/platforma-dev/platforma/log"
)

func main() {
	logger := log.NewWideEventLogger(
		os.Stdout,
		log.NewDefaultSampler(3*time.Second, 200, 0.1),
		"json",
		nil,
	)

	ev := log.NewEvent("test_event")

	ev.AddStep(slog.LevelInfo, "some step")
	ev.AddError(errors.New("some error"))
	ev.AddAttrs(map[string]any{
		"attr1": 1,
		"attr2": true,
	})

	logger.WriteEvent(context.Background(), ev)
}
