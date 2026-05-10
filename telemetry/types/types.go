package types

import "time"

// TelemetryConfig is the persisted telemetry state. Embedded as an
// optional pointer in config.Default so a fresh config has no telemetry
// block at all.
//
// Domain verbosity is encoded by the combination of Enabled and
// SuppressedDomains:
//   - Enabled=true and the domain is NOT in SuppressedDomains: both
//     Report/Emit and ReportRequired/EmitRequired publish.
//   - Enabled=true and the domain IS in SuppressedDomains: only
//     ReportRequired/EmitRequired publish; Report/Emit are dropped.
//   - Enabled=false: everything is dropped.
type TelemetryConfig struct {
	Enabled           bool          `json:"enabled,omitempty"`
	EnabledAt         time.Time     `json:"enabled_at"`
	Validity          time.Duration `json:"validity,omitempty"`
	SuppressedDomains []string      `json:"suppressed_domains,omitempty"`
}
