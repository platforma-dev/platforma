package log2_test

import (
	"testing"
	"time"

	"github.com/platforma-dev/platforma/log2"
)

func TestDefaultTailSamplerKeepsError(t *testing.T) {
	t.Parallel()

	sampler := log2.NewDefaultTailSampler(log2.DefaultTailSamplerConfig{
		RandomKeepRate: 0.01,
		RandomFloat:    func() float64 { return 0.99 },
	})

	decision := sampler.ShouldSample(log2.EventView{HasError: true})
	if !decision.Keep {
		t.Fatalf("expected keep for error")
	}
	if decision.Reason != "error" {
		t.Fatalf("expected reason=error, got %q", decision.Reason)
	}
}

func TestDefaultTailSamplerKeepsByStatus(t *testing.T) {
	t.Parallel()

	sampler := log2.NewDefaultTailSampler(log2.DefaultTailSamplerConfig{
		KeepStatusAtLeast: 500,
		RandomKeepRate:    0.01,
		RandomFloat:       func() float64 { return 0.99 },
	})

	decision := sampler.ShouldSample(log2.EventView{Status: 503})
	if !decision.Keep {
		t.Fatalf("expected keep for status")
	}
	if decision.Reason != "status" {
		t.Fatalf("expected reason=status, got %q", decision.Reason)
	}
}

func TestDefaultTailSamplerKeepsBySlowDuration(t *testing.T) {
	t.Parallel()

	sampler := log2.NewDefaultTailSampler(log2.DefaultTailSamplerConfig{
		SlowThreshold:  100 * time.Millisecond,
		RandomKeepRate: 0.01,
		RandomFloat:    func() float64 { return 0.99 },
	})

	decision := sampler.ShouldSample(log2.EventView{Duration: 200 * time.Millisecond})
	if !decision.Keep {
		t.Fatalf("expected keep for slow event")
	}
	if decision.Reason != "slow" {
		t.Fatalf("expected reason=slow, got %q", decision.Reason)
	}
}

func TestDefaultTailSamplerKeepsByRule(t *testing.T) {
	t.Parallel()

	t.Run("eq", func(t *testing.T) {
		t.Parallel()

		sampler := log2.NewDefaultTailSampler(log2.DefaultTailSamplerConfig{
			KeepRules:      []log2.KeepRule{{Field: "attrs.feature", Op: "eq", Value: "change_password"}},
			RandomKeepRate: 0.01,
			RandomFloat:    func() float64 { return 0.99 },
		})

		decision := sampler.ShouldSample(log2.EventView{Attrs: map[string]any{
			"attrs": map[string]any{"feature": "change_password"},
		}})
		if !decision.Keep || decision.Reason != "rule" {
			t.Fatalf("expected keep with reason=rule, got keep=%v reason=%q", decision.Keep, decision.Reason)
		}
	})

	t.Run("in", func(t *testing.T) {
		t.Parallel()

		sampler := log2.NewDefaultTailSampler(log2.DefaultTailSamplerConfig{
			KeepRules:      []log2.KeepRule{{Field: "queue", Op: "in", Value: []string{"billing", "emails"}}},
			RandomKeepRate: 0.01,
			RandomFloat:    func() float64 { return 0.99 },
		})

		decision := sampler.ShouldSample(log2.EventView{Attrs: map[string]any{"queue": "emails"}})
		if !decision.Keep || decision.Reason != "rule" {
			t.Fatalf("expected keep with reason=rule, got keep=%v reason=%q", decision.Keep, decision.Reason)
		}
	})

	t.Run("exists", func(t *testing.T) {
		t.Parallel()

		sampler := log2.NewDefaultTailSampler(log2.DefaultTailSamplerConfig{
			KeepRules:      []log2.KeepRule{{Field: "vip", Op: "exists"}},
			RandomKeepRate: 0.01,
			RandomFloat:    func() float64 { return 0.99 },
		})

		decision := sampler.ShouldSample(log2.EventView{Attrs: map[string]any{"vip": "yes"}})
		if !decision.Keep || decision.Reason != "rule" {
			t.Fatalf("expected keep with reason=rule, got keep=%v reason=%q", decision.Keep, decision.Reason)
		}
	})

	t.Run("true", func(t *testing.T) {
		t.Parallel()

		sampler := log2.NewDefaultTailSampler(log2.DefaultTailSamplerConfig{
			KeepRules:      []log2.KeepRule{{Field: "security.risk", Op: "true"}},
			RandomKeepRate: 0.01,
			RandomFloat:    func() float64 { return 0.99 },
		})

		decision := sampler.ShouldSample(log2.EventView{Attrs: map[string]any{"security": map[string]any{"risk": true}}})
		if !decision.Keep || decision.Reason != "rule" {
			t.Fatalf("expected keep with reason=rule, got keep=%v reason=%q", decision.Keep, decision.Reason)
		}
	})
}

func TestDefaultTailSamplerRandomPath(t *testing.T) {
	t.Parallel()

	t.Run("keep", func(t *testing.T) {
		t.Parallel()

		sampler := log2.NewDefaultTailSampler(log2.DefaultTailSamplerConfig{
			SlowThreshold:     5 * time.Second,
			KeepStatusAtLeast: 500,
			RandomKeepRate:    0.10,
			RandomFloat:       func() float64 { return 0.09 },
		})

		decision := sampler.ShouldSample(log2.EventView{})
		if !decision.Keep || decision.Reason != "random" {
			t.Fatalf("expected random keep, got keep=%v reason=%q", decision.Keep, decision.Reason)
		}
	})

	t.Run("drop", func(t *testing.T) {
		t.Parallel()

		sampler := log2.NewDefaultTailSampler(log2.DefaultTailSamplerConfig{
			SlowThreshold:     5 * time.Second,
			KeepStatusAtLeast: 500,
			RandomKeepRate:    0.10,
			RandomFloat:       func() float64 { return 0.11 },
		})

		decision := sampler.ShouldSample(log2.EventView{})
		if decision.Keep {
			t.Fatalf("expected drop")
		}
		if decision.Reason != "drop" {
			t.Fatalf("expected reason=drop, got %q", decision.Reason)
		}
	})
}
