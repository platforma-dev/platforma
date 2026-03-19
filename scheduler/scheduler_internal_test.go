package scheduler

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/platforma-dev/platforma/application"
	"github.com/platforma-dev/platforma/log"
)

func TestRunTaskSuccess(t *testing.T) {
	t.Parallel()

	s, err := New("@hourly", application.RunnerFunc(func(ctx context.Context) error {
		traceID, ok := ctx.Value(log.TraceIDKey).(string)
		if !ok || traceID == "" {
			t.Fatalf("expected trace ID in run context, got %#v", ctx.Value(log.TraceIDKey))
		}

		return nil
	}))
	if err != nil {
		t.Fatalf("failed to create scheduler: %v", err)
	}

	runCtx, event := s.runTask(context.Background())

	if traceID, ok := runCtx.Value(log.TraceIDKey).(string); !ok || traceID == "" {
		t.Fatalf("expected trace ID in run context, got %#v", runCtx.Value(log.TraceIDKey))
	}

	if got := event.Name(); got != taskRunEventName {
		t.Fatalf("expected event name %q, got %q", taskRunEventName, got)
	}

	if got, ok := event.Attr("scheduler.cronExpr"); !ok || got != "@hourly" {
		t.Fatalf("expected scheduler.cronExpr attr, got %#v, exists=%v", got, ok)
	}

	if got := event.Level(); got != slog.LevelInfo {
		t.Fatalf("expected event level %v, got %v", slog.LevelInfo, got)
	}

	steps := eventSteps(t, event)
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}

	if steps[0]["name"] != "scheduler task started" {
		t.Fatalf("expected first step to be start, got %#v", steps[0])
	}

	if steps[1]["name"] != "scheduler task finished" {
		t.Fatalf("expected second step to be finish, got %#v", steps[1])
	}

	if errorsList := eventErrors(t, event); len(errorsList) != 0 {
		t.Fatalf("expected no errors, got %#v", errorsList)
	}
}

func TestRunTaskError(t *testing.T) {
	t.Parallel()

	s, err := New("@daily", application.RunnerFunc(func(_ context.Context) error {
		return errors.New("boom")
	}))
	if err != nil {
		t.Fatalf("failed to create scheduler: %v", err)
	}

	_, event := s.runTask(context.Background())

	if got := event.Level(); got != slog.LevelError {
		t.Fatalf("expected event level %v, got %v", slog.LevelError, got)
	}

	steps := eventSteps(t, event)
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}

	if steps[1]["name"] != "scheduler task failed" {
		t.Fatalf("expected failure step, got %#v", steps[1])
	}

	errorsList := eventErrors(t, event)
	if len(errorsList) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errorsList))
	}

	if errorsList[0]["error"] != "scheduler task failed: boom" {
		t.Fatalf("unexpected error payload: %#v", errorsList[0])
	}
}

func eventSteps(t *testing.T, event *log.Event) []map[string]any {
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

func eventErrors(t *testing.T, event *log.Event) []map[string]any {
	t.Helper()

	for _, attr := range event.ToAttrs() {
		if attr.Key == "errors" {
			errorsList, ok := attr.Value.Any().([]map[string]any)
			if !ok {
				t.Fatalf("expected []map[string]any for errors, got %T", attr.Value.Any())
			}

			return errorsList
		}
	}

	return nil
}
