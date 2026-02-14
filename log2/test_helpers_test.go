package log2_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/platforma-dev/platforma/log2"
)

type samplerFunc func(log2.EventView) log2.SamplingDecision

func (f samplerFunc) ShouldSample(view log2.EventView) log2.SamplingDecision {
	return f(view)
}

func readEvents(t *testing.T, buffer *bytes.Buffer) []map[string]any {
	t.Helper()

	trimmed := strings.TrimSpace(buffer.String())
	if trimmed == "" {
		return nil
	}

	lines := strings.Split(trimmed, "\n")
	events := make([]map[string]any, 0, len(lines))

	for _, line := range lines {
		event := make(map[string]any)
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("failed to parse event json: %v", err)
		}
		events = append(events, event)
	}

	return events
}

func requireMap(t *testing.T, source map[string]any, key string) map[string]any {
	t.Helper()

	value, ok := source[key]
	if !ok {
		t.Fatalf("expected key %q to exist", key)
	}

	asMap, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected %q to be a map, got %T", key, value)
	}

	return asMap
}

func requireSlice(t *testing.T, source map[string]any, key string) []any {
	t.Helper()

	value, ok := source[key]
	if !ok {
		t.Fatalf("expected key %q to exist", key)
	}

	asSlice, ok := value.([]any)
	if !ok {
		t.Fatalf("expected %q to be a slice, got %T", key, value)
	}

	return asSlice
}
