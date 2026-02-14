package log2_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"

	"github.com/platforma-dev/platforma/log2"
)

func TestEventLifecycle(t *testing.T) {
	t.Parallel()

	buffer := &bytes.Buffer{}
	logger := log2.New(log2.Config{
		Writer:   buffer,
		Format:   "json",
		Level:    slog.LevelDebug,
		Sampler:  samplerFunc(func(log2.EventView) log2.SamplingDecision { return log2.SamplingDecision{Keep: true, Reason: "forced"} }),
		MaxSteps: 10,
	})

	ev := logger.Start(context.Background(), "auth_request", "component", "handler")
	ev.Add("userId", "u-1", "attempt", 2)
	ev.Step(slog.LevelInfo, "request accepted")
	ev.Error(errors.New("boom"), "phase", "validation")

	if err := ev.Finish("status", 500); err != nil {
		t.Fatalf("Finish() returned error: %v", err)
	}

	events := readEvents(t, buffer)
	if len(events) != 1 {
		t.Fatalf("expected 1 emitted event, got %d", len(events))
	}

	event := events[0]
	if event["event"] != "auth_request" {
		t.Fatalf("expected event name auth_request, got %v", event["event"])
	}
	if event["samplingReason"] != "forced" {
		t.Fatalf("expected sampling reason forced, got %v", event["samplingReason"])
	}
	if event["sampled"] != true {
		t.Fatalf("expected sampled=true, got %v", event["sampled"])
	}

	attrs := requireMap(t, event, "attrs")
	if attrs["component"] != "handler" {
		t.Fatalf("expected component=handler, got %v", attrs["component"])
	}
	if attrs["userId"] != "u-1" {
		t.Fatalf("expected userId=u-1, got %v", attrs["userId"])
	}
	if attrs["attempt"] != float64(2) {
		t.Fatalf("expected attempt=2, got %v", attrs["attempt"])
	}
	if attrs["status"] != float64(500) {
		t.Fatalf("expected status=500, got %v", attrs["status"])
	}

	steps := requireSlice(t, event, "steps")
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}

	errorsList := requireSlice(t, event, "errors")
	if len(errorsList) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errorsList))
	}
}

func TestEventAttributeOverwrite(t *testing.T) {
	t.Parallel()

	buffer := &bytes.Buffer{}
	logger := log2.New(log2.Config{
		Writer:  buffer,
		Format:  "json",
		Sampler: samplerFunc(func(log2.EventView) log2.SamplingDecision { return log2.SamplingDecision{Keep: true, Reason: "forced"} }),
	})

	ev := logger.Start(context.Background(), "overwrite")
	ev.Add("key", "first")
	ev.Add("key", "second")

	if err := ev.Finish(); err != nil {
		t.Fatalf("Finish() returned error: %v", err)
	}

	events := readEvents(t, buffer)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	attrs := requireMap(t, events[0], "attrs")
	if attrs["key"] != "second" {
		t.Fatalf("expected key=second, got %v", attrs["key"])
	}
}

func TestEventFinishCalledTwice(t *testing.T) {
	t.Parallel()

	logger := log2.New(log2.Config{
		Writer:  &bytes.Buffer{},
		Sampler: samplerFunc(func(log2.EventView) log2.SamplingDecision { return log2.SamplingDecision{Keep: true, Reason: "forced"} }),
	})

	ev := logger.Start(context.Background(), "double_finish")
	if err := ev.Finish(); err != nil {
		t.Fatalf("first Finish() returned error: %v", err)
	}

	err := ev.Finish()
	if !errors.Is(err, log2.ErrEventAlreadyFinished) {
		t.Fatalf("expected ErrEventAlreadyFinished, got %v", err)
	}
}

func TestEventConcurrentAccessAndStepCap(t *testing.T) {
	t.Parallel()

	buffer := &bytes.Buffer{}
	logger := log2.New(log2.Config{
		Writer:   buffer,
		Format:   "json",
		Level:    slog.LevelDebug,
		Sampler:  samplerFunc(func(log2.EventView) log2.SamplingDecision { return log2.SamplingDecision{Keep: true, Reason: "forced"} }),
		MaxSteps: 5,
	})

	ev := logger.Start(context.Background(), "concurrent")

	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ev.Add("counter", i)
			ev.Step(slog.LevelInfo, "work", "i", i)
			if i%3 == 0 {
				ev.Error(errors.New("worker-error"), "i", i)
			}
		}(i)
	}

	wg.Wait()

	if err := ev.Finish("status", 200); err != nil {
		t.Fatalf("Finish() returned error: %v", err)
	}

	events := readEvents(t, buffer)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	event := events[0]
	steps := requireSlice(t, event, "steps")
	if len(steps) != 5 {
		t.Fatalf("expected 5 steps (cap), got %d", len(steps))
	}

	dropped, ok := event["stepsDropped"].(float64)
	if !ok {
		t.Fatalf("expected stepsDropped to exist")
	}
	if dropped != float64(15) {
		t.Fatalf("expected stepsDropped=15, got %v", dropped)
	}
}

func TestEventDroppedBySampler(t *testing.T) {
	t.Parallel()

	buffer := &bytes.Buffer{}
	logger := log2.New(log2.Config{
		Writer:  buffer,
		Format:  "json",
		Sampler: samplerFunc(func(log2.EventView) log2.SamplingDecision { return log2.SamplingDecision{Keep: false, Reason: "drop"} }),
	})

	ev := logger.Start(context.Background(), "drop_me")
	ev.Add("k", "v")
	if err := ev.Finish(); err != nil {
		t.Fatalf("Finish() returned error: %v", err)
	}

	events := readEvents(t, buffer)
	if len(events) != 0 {
		t.Fatalf("expected no emitted events, got %d", len(events))
	}
}
