package telemetry

import "time"

const defaultTelemetryValidity = 30 * 24 * time.Hour

type storeIf interface {
	Enabled() *bool
	SetEnabled(enabled *bool) error
	Validity() time.Duration
	SetValidity(d time.Duration) error
	DisabledDomains() []string
	SetDisabledDomains(disabledDomains []string) error
}
