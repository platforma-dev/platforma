// Package log2 provides wide-event logging with tail sampling.
package log2

import (
	"io"
	"log/slog"
	"os"
	"time"
)

const (
	defaultFormat         = "json"
	defaultKeepStatus     = 500
	defaultMaxSteps       = 100
	defaultRandomKeepRate = 0.05
	defaultSlowThreshold  = 2 * time.Second
)

// Config configures logger behavior.
type Config struct {
	Writer      io.Writer
	Format      string
	Level       slog.Level
	ContextKeys map[string]any
	Sampler     TailSampler
	MaxSteps    int
}

// DefaultConfig returns default logger configuration.
func DefaultConfig() Config {
	return Config{
		Writer: os.Stdout,
		Format: defaultFormat,
		Level:  slog.LevelInfo,
		Sampler: NewDefaultTailSampler(DefaultTailSamplerConfig{
			SlowThreshold:     defaultSlowThreshold,
			RandomKeepRate:    defaultRandomKeepRate,
			KeepStatusAtLeast: defaultKeepStatus,
		}),
		MaxSteps: defaultMaxSteps,
	}
}
