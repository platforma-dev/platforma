package log2

import (
	"math/rand"
	"reflect"
	"strings"
	"time"
)

// TailSampler decides whether a finished event should be emitted.
type TailSampler interface {
	ShouldSample(EventView) SamplingDecision
}

// SamplingDecision is the sampler decision.
type SamplingDecision struct {
	Keep   bool
	Reason string
}

// EventView is immutable event information for sampling.
type EventView struct {
	Status   int
	Duration time.Duration
	HasError bool
	Attrs    map[string]any
}

// DefaultTailSamplerConfig configures the default tail sampler.
type DefaultTailSamplerConfig struct {
	SlowThreshold     time.Duration
	RandomKeepRate    float64
	KeepStatusAtLeast int
	KeepRules         []KeepRule
	RandomFloat       func() float64
}

// KeepRule defines a forced-keep rule.
type KeepRule struct {
	Field string
	Op    string
	Value any
}

type defaultTailSampler struct {
	slowThreshold     time.Duration
	randomKeepRate    float64
	keepStatusAtLeast int
	keepRules         []KeepRule
	randomFloat       func() float64
}

// NewDefaultTailSampler creates a rule-based tail sampler.
//
//nolint:iface // public API returns TailSampler to allow custom implementations.
func NewDefaultTailSampler(cfg DefaultTailSamplerConfig) TailSampler {
	slowThreshold := cfg.SlowThreshold
	if slowThreshold <= 0 {
		slowThreshold = defaultSlowThreshold
	}

	keepStatusAtLeast := cfg.KeepStatusAtLeast
	if keepStatusAtLeast <= 0 {
		keepStatusAtLeast = defaultKeepStatus
	}

	randomKeepRate := cfg.RandomKeepRate
	if randomKeepRate <= 0 {
		randomKeepRate = defaultRandomKeepRate
	}
	if randomKeepRate > 1 {
		randomKeepRate = 1
	}

	randomFloat := cfg.RandomFloat
	if randomFloat == nil {
		randomFloat = rand.Float64
	}

	keepRules := make([]KeepRule, len(cfg.KeepRules))
	copy(keepRules, cfg.KeepRules)

	return &defaultTailSampler{
		slowThreshold:     slowThreshold,
		randomKeepRate:    randomKeepRate,
		keepStatusAtLeast: keepStatusAtLeast,
		keepRules:         keepRules,
		randomFloat:       randomFloat,
	}
}

func (s *defaultTailSampler) ShouldSample(view EventView) SamplingDecision {
	if view.HasError {
		return SamplingDecision{Keep: true, Reason: "error"}
	}

	if view.Status >= s.keepStatusAtLeast {
		return SamplingDecision{Keep: true, Reason: "status"}
	}

	if view.Duration >= s.slowThreshold {
		return SamplingDecision{Keep: true, Reason: "slow"}
	}

	for _, rule := range s.keepRules {
		if ruleMatches(view.Attrs, rule) {
			return SamplingDecision{Keep: true, Reason: "rule"}
		}
	}

	if s.randomFloat() < s.randomKeepRate {
		return SamplingDecision{Keep: true, Reason: "random"}
	}

	return SamplingDecision{Keep: false, Reason: "drop"}
}

func ruleMatches(attrs map[string]any, rule KeepRule) bool {
	value, exists := lookupPath(attrs, rule.Field)

	switch strings.ToLower(rule.Op) {
	case "eq":
		if !exists {
			return false
		}
		return valuesEqual(value, rule.Value)
	case "in":
		if !exists {
			return false
		}
		return valueIn(value, rule.Value)
	case "exists":
		return exists
	case "true":
		if !exists {
			return false
		}
		boolValue, ok := value.(bool)
		return ok && boolValue
	default:
		return false
	}
}

func lookupPath(attrs map[string]any, path string) (any, bool) {
	if attrs == nil {
		return nil, false
	}

	if direct, ok := attrs[path]; ok {
		return direct, true
	}

	current := any(attrs)
	for _, part := range strings.Split(path, ".") {
		asMap, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}

		next, ok := asMap[part]
		if !ok {
			return nil, false
		}

		current = next
	}

	return current, true
}

func valueIn(value any, candidates any) bool {
	candidateValue := reflect.ValueOf(candidates)
	if !candidateValue.IsValid() {
		return false
	}

	kind := candidateValue.Kind()
	if kind != reflect.Slice && kind != reflect.Array {
		return false
	}

	for i := range candidateValue.Len() {
		if valuesEqual(value, candidateValue.Index(i).Interface()) {
			return true
		}
	}

	return false
}

func valuesEqual(a any, b any) bool {
	if reflect.DeepEqual(a, b) {
		return true
	}

	ai, aok := toInt(a)
	bi, bok := toInt(b)
	if aok && bok {
		return ai == bi
	}

	return false
}
