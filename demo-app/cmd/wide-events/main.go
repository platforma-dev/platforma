package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/platforma-dev/platforma/log3"
)

func main() {
	logger := log3.NewWideEventLogger(
		os.Stdout,
		log3.NewDefaultSampler(3*time.Second, 200, 0.1),
		"text",
		nil,
	)

	ev := log3.NewEvent("test_event")

	ev.AddStep(slog.LevelInfo, "some step")
	ev.AddError(errors.New("some error"))
	ev.AddAttrs(map[string]any{
		"attr1": 1,
		"attr2": true,
	})

	logger.WriteEvent(context.Background(), ev)
}
