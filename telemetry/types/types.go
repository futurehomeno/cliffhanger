package types

import "time"

// SuppressedEntry holds the per-service suppression rules sent by the cloud.
// When both Domains and Events are nil, suppress all telemetry. Otherwise,
// suppress any emit whose domain is listed in Domains or whose event is listed
// in Events. A nil *SuppressedEntry in TelemetryConfig means no suppression.
type SuppressedEntry struct {
	Domains []string `json:"domains"`
	Events  []string `json:"events"`
}

// TelemetryConfig is the persisted telemetry state. Embedded as an
// optional pointer in config.Default so a fresh config has no telemetry
// block at all.
//
// Suppression is controlled by the Suppressed field:
//   - Suppressed == nil: no suppression, all Emit calls publish.
//   - Suppressed != nil, both Domains and Events are nil: suppress everything.
//   - Otherwise: suppress if the domain is in Domains OR the event is in Events.
type TelemetryConfig struct {
	Enabled   bool             `json:"enabled,omitempty"`
	EnabledAt time.Time        `json:"enabled_at"`
	Validity  time.Duration    `json:"validity,omitempty"`
	Suppressed *SuppressedEntry `json:"suppressed,omitempty"`
}
