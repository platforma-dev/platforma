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
	keepHttpStatusAtLeast int
	randomKeepRate        float64
}

// NewDefaultSampler creates a rule-based sampler.
func NewDefaultSampler(slowThreshold time.Duration, keepHTTPStatusAtLeast int, randomKeepRate float64) *DefaultSampler {
	return &DefaultSampler{
		slowThreshold:         slowThreshold,
		keepHttpStatusAtLeast: keepHTTPStatusAtLeast,
		randomKeepRate:        randomKeepRate,
	}
}

// ShouldSample decides if event should be logged.
func (s *DefaultSampler) ShouldSample(ctx context.Context, e *Event) bool {
	if len(e.errors) > 0 {
		return true
	}

	if e.duration >= s.slowThreshold {
		return true
	}

	httpStatus := 0
	statusFromMap, exists := e.attrs["request.status"]
	if exists {
		success := false
		httpStatus, success = statusFromMap.(int)
		if !success {
			httpStatus = 0
		}
	}

	if httpStatus >= s.keepHttpStatusAtLeast {
		return true
	}

	if rand.Float64() < s.randomKeepRate {
		return true
	}

	return false
}
