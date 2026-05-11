package telemetry

const (
	telemetryInterface      = "evt.telemetry.report"
	telemetryReportEvtTopic = "pt:j1/mt:rsp/rt:cloud/rn:backend-service/ad:telemetry"
)

// Predefined telemetry domains for cross-app events.
// Use these to keep domain naming consistent across apps so the cloud side
// can group and suppress events by stable keys.
const (
	// DomainPanic groups events emitted from recovered panics.
	DomainPanic = "panic"

	// DomainAuth groups authentication and authorization events.
	DomainAuth = "auth"

	// DomainShouldNeverHappen marks invariant violations: code reached a
	// branch the author believed was unreachable.
	DomainShouldNeverHappen = "should_never_happen"

	// DomainReboot groups reboot/restart events. Sample heavily — emit only
	// every restartMilestoneStep'th boot so a device stuck in a crash loop
	// does not flood the pipeline.
	DomainReboot = "reboot"

	EventLoggedOut = "logged_out"
)

const (
	EventRebootMilestone = "milestone"

	restartMilestoneStep = 500
)

// Event is the payload carried in the FIMP val field.
type Event struct {
	Event  string         `json:"event"`
	Domain string         `json:"domain,omitempty"`
	Data   map[string]any `json:"data,omitempty"`
}
