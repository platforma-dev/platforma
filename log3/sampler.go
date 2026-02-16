package log3

import (
	"context"
	"math/rand/v2"
	"time"
)

type Sampler interface {
	ShouldSample(ctx context.Context, e *Event) bool
}

type SamplerFunc func(ctx context.Context, e *Event) bool

func (f SamplerFunc) ShouldSample(ctx context.Context, e *Event) bool {
	return f(ctx, e)
}

type DefaultSampler struct {
	slowThreshold         time.Duration
	keepHttpStatusAtLeast int
	randomKeepRate        float64
}

func NewDefaultSampler(slowThreshold time.Duration, keepHttpStatusAtLeast int, randomKeepRate float64) *DefaultSampler {
	return &DefaultSampler{
		slowThreshold:         slowThreshold,
		keepHttpStatusAtLeast: keepHttpStatusAtLeast,
		randomKeepRate:        randomKeepRate,
	}
}

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

	if s.randomKeepRate <= rand.Float64() {
		return true
	}

	return false
}
