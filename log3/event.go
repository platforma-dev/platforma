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

func NewEvent(name string) *Event {
	return &Event{
		name:      name,
		timestamp: time.Now(),
		level:     slog.LevelDebug,
		attrs:     map[string]any{},
	}
}

func (e *Event) SetLevel(level slog.Level) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.setLevelNoLock(level)
}

func (e *Event) setLevelNoLock(level slog.Level) {
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

	e.setLevelNoLock(level)

	e.steps = append(e.steps, stepRecord{
		Timestamp: time.Now(),
		Level:     level,
		Name:      name,
	})
}

func (e *Event) AddError(err error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.setLevelNoLock(slog.LevelError)

	e.errors = append(e.errors, errorRecord{
		Timestamp: time.Now(),
		Error:     err.Error(),
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
	steps := make([]map[string]any, 0, len(e.steps))
	for _, step := range e.steps {
		steps = append(steps, map[string]any{
			"timestamp": step.Timestamp,
			"level":     step.Level.String(),
			"name":      step.Name,
		})
	}
	eventErrors := make([]map[string]any, 0, len(e.errors))
	for _, eventError := range e.errors {
		eventErrors = append(eventErrors, map[string]any{
			"timestamp": eventError.Timestamp,
			"error":     eventError.Error,
		})
	}

	return []slog.Attr{
		slog.String("name", e.name),
		slog.Time("timestamp", e.timestamp),
		slog.Duration("duration", e.duration),
		slog.Any("attrs", e.attrs),
		slog.Any("steps", steps),
		slog.Any("errors", eventErrors),
	}
}

type stepRecord struct {
	Timestamp time.Time  `json:"timestamp"`
	Level     slog.Level `json:"level"`
	Name      string     `json:"name"`
}

func (r stepRecord) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Time("timestamp", r.Timestamp),
		slog.String("name", r.Name),
		slog.String("level", r.Level.String()),
	)
}

type errorRecord struct {
	Timestamp time.Time `json:"timestamp"`
	Error     string    `json:"error"`
}
