package log2_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/platforma-dev/platforma/log"
	"github.com/platforma-dev/platforma/log2"
)

func TestEventFromContextMiss(t *testing.T) {
	t.Parallel()

	ev, ok := log2.EventFromContext(context.Background())
	if ok {
		t.Fatalf("expected ok=false, got true with event=%v", ev)
	}
	if ev != nil {
		t.Fatalf("expected nil event, got %v", ev)
	}
}

func TestEventFromContextWithRawContextValue(t *testing.T) {
	t.Parallel()

	logger := log2.New(log2.Config{Writer: &bytes.Buffer{}})
	ev := logger.Start(context.Background(), "ctx")

	ctx := context.WithValue(context.Background(), log2.LogEventContextKey, ev)
	got, ok := log2.EventFromContext(ctx)
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if got != ev {
		t.Fatalf("expected the same event pointer")
	}
}

func TestWithEvent(t *testing.T) {
	t.Parallel()

	logger := log2.New(log2.Config{Writer: &bytes.Buffer{}})
	ev := logger.Start(context.Background(), "ctx")

	ctx := log2.WithEvent(context.Background(), ev)
	got, ok := log2.EventFromContext(ctx)
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if got != ev {
		t.Fatalf("expected the same event pointer")
	}
}

func TestStartCollectsDefaultContextKeys(t *testing.T) {
	t.Parallel()

	buffer := &bytes.Buffer{}
	logger := log2.New(log2.Config{
		Writer:  buffer,
		Format:  "json",
		Level:   slog.LevelInfo,
		Sampler: samplerFunc(func(log2.EventView) log2.SamplingDecision { return log2.SamplingDecision{Keep: true, Reason: "forced"} }),
	})

	ctx := context.WithValue(context.Background(), log.TraceIDKey, "trace-123")
	ctx = context.WithValue(ctx, log.UserIDKey, "user-99")

	ev := logger.Start(ctx, "collect-keys")
	if err := ev.Finish(); err != nil {
		t.Fatalf("Finish() returned error: %v", err)
	}

	events := readEvents(t, buffer)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0]["traceId"] != "trace-123" {
		t.Fatalf("expected top-level traceId=trace-123, got %v", events[0]["traceId"])
	}

	attrs := requireMap(t, events[0], "attrs")
	if attrs["traceId"] != "trace-123" {
		t.Fatalf("expected attrs.traceId=trace-123, got %v", attrs["traceId"])
	}
	if attrs["userId"] != "user-99" {
		t.Fatalf("expected attrs.userId=user-99, got %v", attrs["userId"])
	}
}
