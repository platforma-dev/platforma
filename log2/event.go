package log2

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/platforma-dev/platforma/log"
)

// ErrEventAlreadyFinished means Finish was called more than once.
var ErrEventAlreadyFinished = errors.New("event already finished")

type stepRecord struct {
	ts    time.Time
	level string
	msg   string
	attrs map[string]any
}

type errorRecord struct {
	ts    time.Time
	err   string
	attrs map[string]any
}

type eventPayload struct {
	eventName      string
	level          slog.Level
	durationMs     int64
	sampled        bool
	samplingReason string
	traceID        string
	attrs          map[string]any
	steps          []map[string]any
	errors         []map[string]any
	stepsDropped   int
}

// Event is a mutable wide event.
type Event struct {
	mu sync.Mutex

	logger    *Logger
	eventName string
	startedAt time.Time

	attrs        map[string]any
	steps        []stepRecord
	errors       []errorRecord
	hasError     bool
	finished     bool
	stepsDropped int
}

// Add adds persistent attributes to the event.
func (e *Event) Add(attrs ...any) {
	if e == nil {
		return
	}

	normalized := normalizeAttrs(attrs...)

	e.mu.Lock()
	defer e.mu.Unlock()

	if e.finished {
		return
	}

	mergeAttrs(e.attrs, normalized)
}

// Step appends a timeline step to the event.
func (e *Event) Step(level slog.Level, msg string, attrs ...any) {
	if e == nil {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if e.finished {
		return
	}

	if len(e.steps) >= e.logger.maxSteps {
		e.stepsDropped++
		return
	}

	e.steps = append(e.steps, stepRecord{
		ts:    nowUTC(),
		level: level.String(),
		msg:   msg,
		attrs: normalizeAttrs(attrs...),
	})
}

// Error appends an error to the event.
func (e *Event) Error(err error, attrs ...any) {
	if e == nil || err == nil {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if e.finished {
		return
	}

	e.hasError = true
	e.errors = append(e.errors, errorRecord{
		ts:    nowUTC(),
		err:   err.Error(),
		attrs: normalizeAttrs(attrs...),
	})
}

// Finish finalizes and emits the event depending on sampling decision.
func (e *Event) Finish(attrs ...any) error {
	if e == nil {
		return nil
	}

	e.mu.Lock()
	if e.finished {
		e.mu.Unlock()
		return ErrEventAlreadyFinished
	}

	mergeAttrs(e.attrs, normalizeAttrs(attrs...))

	duration := nowUTC().Sub(e.startedAt)
	status := inferStatus(e.attrs)
	level := inferLevel(status, e.hasError)
	attrsCopy := copyAttrs(e.attrs)
	stepsCopy := copySteps(e.steps)
	errorsCopy := copyErrors(e.errors)
	stepsDropped := e.stepsDropped
	traceID := extractString(attrsCopy, string(log.TraceIDKey))

	decision := e.logger.sampler.ShouldSample(EventView{
		Status:   status,
		Duration: duration,
		HasError: e.hasError,
		Attrs:    attrsCopy,
	})

	e.finished = true
	logger := e.logger
	e.mu.Unlock()

	if !decision.Keep {
		return nil
	}

	return logger.emit(context.Background(), eventPayload{
		eventName:      e.eventName,
		level:          level,
		durationMs:     duration.Milliseconds(),
		sampled:        true,
		samplingReason: decision.Reason,
		traceID:        traceID,
		attrs:          attrsCopy,
		steps:          stepsCopy,
		errors:         errorsCopy,
		stepsDropped:   stepsDropped,
	})
}

func nowUTC() time.Time {
	return time.Now().UTC()
}

func collectContextAttrs(ctx context.Context, extraKeys map[string]any) map[string]any {
	attrs := map[string]any{}

	defaultKeys := []struct {
		name string
		key  any
	}{
		{name: string(log.DomainNameKey), key: log.DomainNameKey},
		{name: string(log.TraceIDKey), key: log.TraceIDKey},
		{name: string(log.ServiceNameKey), key: log.ServiceNameKey},
		{name: string(log.StartupTaskKey), key: log.StartupTaskKey},
		{name: string(log.UserIDKey), key: log.UserIDKey},
		{name: string(log.WorkerIDKey), key: log.WorkerIDKey},
	}

	for _, item := range defaultKeys {
		if value := ctx.Value(item.key); value != nil {
			attrs[item.name] = value
		}
	}

	for outputKey, ctxKey := range extraKeys {
		if value := ctx.Value(ctxKey); value != nil {
			attrs[outputKey] = value
		}
	}

	return attrs
}

func normalizeAttrs(attrs ...any) map[string]any {
	normalized := make(map[string]any, len(attrs)/2)

	for i := 0; i < len(attrs); i++ {
		if attr, ok := attrs[i].(slog.Attr); ok {
			normalized[attr.Key] = attr.Value.Any()
			continue
		}

		if i+1 >= len(attrs) {
			break
		}

		key := fmt.Sprint(attrs[i])
		normalized[key] = attrs[i+1]
		i++
	}

	return normalized
}

func mergeAttrs(dst map[string]any, src map[string]any) {
	for key, value := range src {
		dst[key] = value
	}
}

func copyAttrs(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}

	dst := make(map[string]any, len(src))
	for key, value := range src {
		dst[key] = value
	}

	return dst
}

func copySteps(src []stepRecord) []map[string]any {
	steps := make([]map[string]any, 0, len(src))
	for _, step := range src {
		entry := map[string]any{
			"ts":    step.ts,
			"level": step.level,
			"msg":   step.msg,
		}
		mergeAttrs(entry, step.attrs)
		steps = append(steps, entry)
	}

	return steps
}

func copyErrors(src []errorRecord) []map[string]any {
	errs := make([]map[string]any, 0, len(src))
	for _, item := range src {
		entry := map[string]any{
			"ts":    item.ts,
			"error": item.err,
		}
		mergeAttrs(entry, item.attrs)
		errs = append(errs, entry)
	}

	return errs
}

func inferLevel(status int, hasError bool) slog.Level {
	if hasError || status >= defaultKeepStatus {
		return slog.LevelError
	}

	if status >= 400 {
		return slog.LevelWarn
	}

	return slog.LevelInfo
}

func inferStatus(attrs map[string]any) int {
	if status, ok := toInt(attrs["status"]); ok {
		return status
	}

	if status, ok := toInt(attrs["statusCode"]); ok {
		return status
	}

	if status, ok := toInt(attrs["response.status"]); ok {
		return status
	}

	if responseAny, ok := attrs["response"]; ok {
		if response, okMap := responseAny.(map[string]any); okMap {
			if status, okStatus := toInt(response["status"]); okStatus {
				return status
			}
		}
	}

	return 0
}

func toInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int8:
		return int(typed), true
	case int16:
		return int(typed), true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case uint:
		if typed > uint(math.MaxInt) {
			return 0, false
		}
		return int(typed), true
	case uint8:
		return int(typed), true
	case uint16:
		return int(typed), true
	case uint32:
		return int(typed), true
	case uint64:
		if typed > uint64(math.MaxInt) {
			return 0, false
		}
		return int(typed), true
	case float32:
		return int(typed), true
	case float64:
		return int(typed), true
	case string:
		parsed, err := strconv.Atoi(typed)
		if err != nil {
			return 0, false
		}

		return parsed, true
	default:
		return 0, false
	}
}

func extractString(attrs map[string]any, key string) string {
	value, ok := attrs[key]
	if !ok {
		return ""
	}

	stringValue, ok := value.(string)
	if !ok {
		return ""
	}

	return stringValue
}
