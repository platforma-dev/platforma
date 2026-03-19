package log_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	platformalog "github.com/platforma-dev/platforma/log"
)

func TestWideEventLoggerCanBeSetAsDefaultLogger(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := platformalog.NewWideEventLogger(&buf, nil, "json", nil)

	platformalog.SetDefault(logger)
	defer platformalog.SetDefault(platformalog.New(io.Discard, "json", slog.LevelInfo, nil))

	platformalog.Info("startup", "service", "api")

	record := decodeSingleRecord(t, buf.String())
	if got := record["name"]; got != "log.record" {
		t.Fatalf("expected event name %q, got %v", "log.record", got)
	}
}

func TestWideEventLoggerWritesSimpleLogImmediately(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := platformalog.NewWideEventLogger(&buf, nil, "json", nil)

	logger.Debug("startup", "service", "api")

	record := decodeSingleRecord(t, buf.String())
	if got := record["level"]; got != "DEBUG" {
		t.Fatalf("expected level DEBUG, got %v", got)
	}

	if got := record["name"]; got != "log.record" {
		t.Fatalf("expected event name %q, got %v", "log.record", got)
	}

	if got := record["msg"]; got != "startup" {
		t.Fatalf("expected msg %q, got %v", "startup", got)
	}

	if got := record["service"]; got != "api" {
		t.Fatalf("expected service attr %q, got %v", "api", got)
	}

	assertNonZeroDuration(t, record)
}

func TestWideEventLoggerContextMethodsIncludeContextAttrs(t *testing.T) {
	t.Parallel()

	type tenantKey string

	const tenantIDKey tenantKey = "tenantID"

	var buf bytes.Buffer
	logger := platformalog.NewWideEventLogger(&buf, nil, "json", map[string]any{"tenantId": tenantIDKey})

	ctx := context.WithValue(context.Background(), platformalog.TraceIDKey, "trace-123")
	ctx = context.WithValue(ctx, platformalog.ServiceNameKey, "api")
	ctx = context.WithValue(ctx, tenantIDKey, "tenant-42")

	logger.InfoContext(ctx, "request received", "path", "/users")

	record := decodeSingleRecord(t, buf.String())
	if got := record[string(platformalog.TraceIDKey)]; got != "trace-123" {
		t.Fatalf("expected trace id %q, got %v", "trace-123", got)
	}

	if got := record[string(platformalog.ServiceNameKey)]; got != "api" {
		t.Fatalf("expected service name %q, got %v", "api", got)
	}

	if got := record["tenantId"]; got != "tenant-42" {
		t.Fatalf("expected tenant id %q, got %v", "tenant-42", got)
	}
}

func TestWideEventLoggerSimpleMethodsUseSampler(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := platformalog.NewWideEventLogger(&buf, platformalog.SamplerFunc(func(context.Context, *platformalog.Event) bool {
		return false
	}), "json", nil)

	logger.Info("startup")

	if buf.Len() != 0 {
		t.Fatalf("expected simple log output to be skipped by sampler, got %q", buf.String())
	}
}

func TestWideEventLoggerWriteEventStillUsesSampler(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := platformalog.NewWideEventLogger(&buf, platformalog.SamplerFunc(func(context.Context, *platformalog.Event) bool {
		return false
	}), "json", nil)

	logger.WriteEvent(context.Background(), platformalog.NewEvent("wide.request"))

	if buf.Len() != 0 {
		t.Fatalf("expected sampled wide event to be skipped, got %q", buf.String())
	}
}

func TestWideEventLoggerNormalizesSlogArgs(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := platformalog.NewWideEventLogger(&buf, nil, "json", nil)

	logger.Info(
		"startup",
		"service", "api",
		slog.String("component", "db"),
		slog.Group("http", slog.Int("status", 200)),
		"service", "worker",
	)

	record := decodeSingleRecord(t, buf.String())
	if got := record["service"]; got != "worker" {
		t.Fatalf("expected last service attr to win, got %v", got)
	}

	if got := record["component"]; got != "db" {
		t.Fatalf("expected component attr %q, got %v", "db", got)
	}

	httpGroup, ok := record["http"].(map[string]any)
	if !ok {
		t.Fatalf("expected http group to be an object, got %T", record["http"])
	}

	if got := httpGroup["status"]; got != float64(200) {
		t.Fatalf("expected http.status 200, got %v", got)
	}
}

func TestWideEventLoggerUsesSlogBadKeyForMalformedArgs(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := platformalog.NewWideEventLogger(&buf, nil, "json", nil)

	logger.Info("invalid args", "ok", 1, 2)

	record := decodeSingleRecord(t, buf.String())
	if got := record["ok"]; got != float64(1) {
		t.Fatalf("expected valid attr %v, got %v", 1, got)
	}

	if got := record["!BADKEY"]; got != float64(2) {
		t.Fatalf("expected !BADKEY attr to be 2, got %v", got)
	}
}

func TestWideEventLoggerErrorMethodsKeepErrorsAsAttrsOnly(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := platformalog.NewWideEventLogger(&buf, nil, "json", nil)

	logger.Error("failed", "error", errors.New("boom"))

	record := decodeSingleRecord(t, buf.String())
	if got := record["level"]; got != "ERROR" {
		t.Fatalf("expected level ERROR, got %v", got)
	}

	if _, ok := record["error"]; !ok {
		t.Fatal("expected error attr to be present")
	}

	if _, ok := record["errors"]; ok {
		t.Fatalf("expected no errors collection, got %v", record["errors"])
	}
}

func decodeSingleRecord(t *testing.T, output string) map[string]any {
	t.Helper()

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected a single log record, got %d in %q", len(lines), output)
	}

	var record map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &record); err != nil {
		t.Fatalf("unmarshal log record: %v", err)
	}

	return record
}

func assertNonZeroDuration(t *testing.T, record map[string]any) {
	t.Helper()

	duration, ok := record["duration"]
	if !ok {
		t.Fatal("expected duration attr to be present")
	}

	switch value := duration.(type) {
	case float64:
		if value <= 0 {
			t.Fatalf("expected duration to be > 0, got %v", value)
		}
	case string:
		if value == "" || value == "0s" {
			t.Fatalf("expected duration to be non-zero, got %q", value)
		}
	default:
		t.Fatalf("unexpected duration type %T", duration)
	}
}
