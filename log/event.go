package log

import (
	"log/slog"
	"maps"
	"slices"
	"sort"
	"sync"
	"time"
)

// Event is a mutable wide event.
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

// NewEvent creates a new wide event.
func NewEvent(name string) *Event {
	return &Event{
		name:      name,
		timestamp: time.Now(),
		level:     slog.LevelDebug,
		attrs:     map[string]any{},
	}
}

// SetLevel sets event level if it is higher than the current one.
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

// AddAttrs adds attributes to event data.
func (e *Event) AddAttrs(attrs map[string]any) {
	e.mu.Lock()
	defer e.mu.Unlock()

	maps.Copy(e.attrs, attrs)
}

// AddStep appends an event step and potentially escalates level.
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

// AddError appends an error and escalates event level to error.
func (e *Event) AddError(err error) {
	if err == nil {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.setLevelNoLock(slog.LevelError)

	e.errors = append(e.errors, errorRecord{
		Timestamp: time.Now(),
		Error:     err.Error(),
	})
}

// Finish stores current event duration.
func (e *Event) Finish() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.duration = time.Since(e.timestamp)
}

// HasErrors returns true if the event has errors.
func (e *Event) HasErrors() bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	return len(e.errors) > 0
}

// Duration returns the event duration.
func (e *Event) Duration() time.Duration {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.duration
}

// Level returns the event level.
func (e *Event) Level() slog.Level {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.level
}

// Name returns the event name.
func (e *Event) Name() string {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.name
}

// Attr returns an event attribute by key.
func (e *Event) Attr(key string) (any, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	value, ok := e.attrs[key]

	return value, ok
}

// ToAttrs converts event to slog attributes.
func (e *Event) ToAttrs() []slog.Attr {
	return e.toAttrs(nil)
}

func (e *Event) toAttrs(additionalReservedAttrKeys []string) []slog.Attr {
	e.mu.Lock()
	defer e.mu.Unlock()

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

	builtinAttrKeys := wideEventBuiltinAttrKeys()
	reservedAttrKeys := make([]string, 0, len(builtinAttrKeys)+len(additionalReservedAttrKeys))
	reservedAttrKeys = append(reservedAttrKeys, builtinAttrKeys...)
	for _, key := range additionalReservedAttrKeys {
		if slices.Contains(reservedAttrKeys, key) {
			continue
		}
		reservedAttrKeys = append(reservedAttrKeys, key)
	}

	attrs := make([]slog.Attr, 0, len(e.attrs)+len(builtinAttrKeys))
	attrs = append(attrs,
		slog.String("name", e.name),
		slog.Time("timestamp", e.timestamp),
		slog.Duration("duration", e.duration),
		slog.Any("steps", steps),
		slog.Any("errors", eventErrors),
	)

	customAttrKeys := make([]string, 0, len(e.attrs))
	for key := range e.attrs {
		if slices.Contains(reservedAttrKeys, key) {
			continue
		}

		customAttrKeys = append(customAttrKeys, key)
	}
	sort.Strings(customAttrKeys)

	for _, key := range customAttrKeys {
		attrs = append(attrs, slog.Any(key, e.attrs[key]))
	}

	return attrs
}

type stepRecord struct {
	Timestamp time.Time  `json:"timestamp"`
	Level     slog.Level `json:"level"`
	Name      string     `json:"name"`
}

type errorRecord struct {
	Timestamp time.Time `json:"timestamp"`
	Error     string    `json:"error"`
}

func wideEventBuiltinAttrKeys() []string {
	return []string{
		"name",
		"timestamp",
		"duration",
		"steps",
		"errors",
	}
}
