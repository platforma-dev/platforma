package log_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	platformalog "github.com/platforma-dev/platforma/log"
)

func TestDefaultSamplerForcedInfoLevel(t *testing.T) {
	t.Parallel()

	sampler := platformalog.NewDefaultSampler(time.Hour, 500, 0)

	t.Run("forced info is always sampled", func(t *testing.T) {
		t.Parallel()

		event := platformalog.NewEvent("background.job")
		event.SetLevel(platformalog.LevelInfoForced)

		if !sampler.ShouldSample(context.Background(), event) {
			t.Fatal("expected forced info event to be sampled")
		}
	})

	t.Run("regular info still depends on sampler rules", func(t *testing.T) {
		t.Parallel()

		event := platformalog.NewEvent("background.job")
		event.SetLevel(slog.LevelInfo)

		if sampler.ShouldSample(context.Background(), event) {
			t.Fatal("expected regular info event to be dropped when no sampler rule matches")
		}
	})
}

func TestEventToAttrsFormatsForcedInfoStep(t *testing.T) {
	t.Parallel()

	event := platformalog.NewEvent("background.job")
	event.AddStep(platformalog.LevelInfoForced, "marked for retention")

	attrs := event.ToAttrs()

	for _, attr := range attrs {
		if attr.Key != "steps" {
			continue
		}

		steps, ok := attr.Value.Any().([]map[string]any)
		if !ok {
			t.Fatalf("expected steps attr to be []map[string]any, got %T", attr.Value.Any())
		}

		if len(steps) != 1 {
			t.Fatalf("expected 1 step, got %d", len(steps))
		}

		level, ok := steps[0]["level"].(string)
		if !ok {
			t.Fatalf("expected step level to be string, got %T", steps[0]["level"])
		}

		if level != "INFO" {
			t.Fatalf("expected forced info step level to render as INFO, got %q", level)
		}

		return
	}

	t.Fatal("expected steps attr to be present")
}

func TestWideEventLoggerRendersForcedInfoLevelAsInfo(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := platformalog.NewWideEventLogger(&buf, platformalog.SamplerFunc(func(_ context.Context, _ *platformalog.Event) bool {
		return true
	}), "json", nil)

	event := platformalog.NewEvent("background.job")
	event.SetLevel(platformalog.LevelInfoForced)

	logger.WriteEvent(context.Background(), event)

	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal log output: %v", err)
	}

	level, ok := payload["level"].(string)
	if !ok {
		t.Fatalf("expected level field to be string, got %T", payload["level"])
	}

	if level != "INFO" {
		t.Fatalf("expected forced info event to render as INFO, got %q", level)
	}
}

func TestWideEventLoggerInfoContextUsesForcedInfoLevel(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := platformalog.NewWideEventLogger(
		&buf,
		platformalog.NewDefaultSampler(time.Hour, 500, 0),
		"json",
		nil,
	)

	logger.InfoContext(context.Background(), "forced info")

	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal log output: %v", err)
	}

	level, ok := payload["level"].(string)
	if !ok {
		t.Fatalf("expected level field to be string, got %T", payload["level"])
	}

	if level != "INFO" {
		t.Fatalf("expected forced info log to render as INFO, got %q", level)
	}
}

func TestNewLoggerRendersForcedInfoLevelAsInfo(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := platformalog.New(&buf, "json", slog.LevelDebug, nil)

	logger.Log(context.Background(), platformalog.LevelInfoForced, "forced info")

	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal log output: %v", err)
	}

	level, ok := payload["level"].(string)
	if !ok {
		t.Fatalf("expected level field to be string, got %T", payload["level"])
	}

	if level != "INFO" {
		t.Fatalf("expected forced info log to render as INFO, got %q", level)
	}
}
