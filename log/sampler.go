package log

import (
	"context"
	"math/rand/v2"
	"time"
)

// Sampler decides whether a wide event should be emitted.
type Sampler interface {
	ShouldSample(ctx context.Context, e *Event) bool
}

// SamplerFunc is a function adapter for Sampler.
type SamplerFunc func(ctx context.Context, e *Event) bool

// ShouldSample implements Sampler.
func (f SamplerFunc) ShouldSample(ctx context.Context, e *Event) bool {
	return f(ctx, e)
}

// DefaultSampler samples by error, duration, status code, and random keep rate.
type DefaultSampler struct {
	slowThreshold         time.Duration
	keepHTTPStatusAtLeast int
	randomKeepRate        float64
}

// NewDefaultSampler creates a rule-based sampler.
func NewDefaultSampler(slowThreshold time.Duration, keepHTTPStatusAtLeast int, randomKeepRate float64) *DefaultSampler {
	return &DefaultSampler{
		slowThreshold:         slowThreshold,
		keepHTTPStatusAtLeast: keepHTTPStatusAtLeast,
		randomKeepRate:        randomKeepRate,
	}
}

// ShouldSample decides if event should be logged.
func (s *DefaultSampler) ShouldSample(_ context.Context, e *Event) bool {
	if e.HasErrors() {
		return true
	}

	if e.Duration() >= s.slowThreshold {
		return true
	}

	httpStatus := 0
	if statusFromMap, exists := e.Attr("request.status"); exists {
		if status, ok := statusFromMap.(int); ok {
			httpStatus = status
		}
	}

	if httpStatus >= s.keepHTTPStatusAtLeast {
		return true
	}

	//nolint:gosec // Non-cryptographic sampling is sufficient for log event retention.
	if rand.Float64() < s.randomKeepRate {
		return true
	}

	return false
}
