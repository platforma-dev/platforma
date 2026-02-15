package log3

import (
	"log/slog"
	"maps"
	"sync"
	"time"
)

type Event struct {
	mu sync.Mutex

	name      string
	timestamp time.Time
	level     slog.Level
	duration  time.Duration
	attrs     map[string]any
	steps     []stepRecord
	errors    []errorRecord
}

func NewEvent() *Event {
	return &Event{
		timestamp: time.Now(),
		level:     slog.LevelDebug,
		attrs:     map[string]any{},
	}
}

func (e *Event) SetLevel(level slog.Level) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if level > e.level {
		e.level = level
	}
}

func (e *Event) AddAttrs(attrs map[string]any) {
	e.mu.Lock()
	defer e.mu.Unlock()

	maps.Copy(e.attrs, attrs)
}

func (e *Event) AddStep(level slog.Level, name string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.SetLevel(level)

	e.steps = append(e.steps, stepRecord{
		timestamp: time.Now(),
		level:     level,
		name:      name,
	})
}

func (e *Event) AddError(err error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.SetLevel(slog.LevelError)

	e.errors = append(e.errors, errorRecord{
		timestamp: time.Now(),
		err:       err,
	})
}

func (e *Event) Finish() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.duration = time.Since(e.timestamp)
}

func (e *Event) ToAttrs() []slog.Attr {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.duration = time.Since(e.timestamp)

	return []slog.Attr{
		slog.String("name", e.name),
		slog.Time("timestamp", e.timestamp),
		slog.Int("level", int(e.level)),
		slog.Duration("duration", e.duration),
		slog.Any("attrs", e.attrs),
		slog.Any("steps", e.steps),
		slog.Any("errors", e.errors),
	}
}

type stepRecord struct {
	timestamp time.Time
	level     slog.Level
	name      string
}

type errorRecord struct {
	timestamp time.Time
	err       error
}
