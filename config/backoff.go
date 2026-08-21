package config

import (
	"github.com/futurehomeno/cliffhanger/backoff"
)

// BackoffConfig is a backoff specification persisted as part of application configuration.
type BackoffConfig struct {
	Initial       string `json:"initial"`
	Repeated      string `json:"repeated"`
	Final         string `json:"final"`
	InitialCount  uint32 `json:"initial_count"`
	RepeatedCount uint32 `json:"repeated_count"`
}

// Stateful builds a stateful backoff from the configuration, falling back to def for
// unset or unparsable durations and zero counts.
func (c BackoffConfig) Stateful(def BackoffConfig) backoff.Stateful {
	initialCount, repeatedCount := c.InitialCount, c.RepeatedCount
	if initialCount == 0 {
		initialCount = def.InitialCount
	}

	if repeatedCount == 0 {
		repeatedCount = def.RepeatedCount
	}

	return backoff.NewStateful(
		parseDurationOr(c.Initial, parseDurationOr(def.Initial, 0)),
		parseDurationOr(c.Repeated, parseDurationOr(def.Repeated, 0)),
		parseDurationOr(c.Final, parseDurationOr(def.Final, 0)),
		initialCount,
		repeatedCount,
	)
}
