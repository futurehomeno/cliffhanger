package alarm

import (
	"fmt"
	"sync"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/cache"
	"github.com/futurehomeno/cliffhanger/utils"
)

// DefaultReportingStrategy is the default reporting strategy used by the service for periodic reports.
var DefaultReportingStrategy = cache.ReportOnChangeOnly()

// Constants defining important properties specific for the service.
const (
	PropertySupportedEvents = "sup_events"

	StatusActivate   = "activ"
	StatusDeactivate = "deactiv"
)

// Alarm events shared with the zigbee car charger, so that the hub renders faults
// reported by cloud adapters identically to the ones reported over zigbee.
const (
	EventGroundingFault = "GROUNDING_FAULT"
	EventPENFault       = "PEN_FAULT"
	EventOverTemp       = "OVER_TEMP_FAULT"
	EventRCDTrip        = "RCD_TRIP_FAULT"
	EventOverVoltage    = "OVER_VOLTAGE_FAULT"
	EventUnderVoltage   = "UNDER_VOLTAGE_FAULT"
	EventOverCurrent    = "OVER_CURRENT_FAULT"
	EventMeterComFault  = "METER_COM_FAULT"
	EventGridTypeFault  = "GRIDTYPE_CONFIG_FAULT"
	EventOtherChargeErr = "CP_OTHER_ERROR"
)

// Report represents value structure of an alarm report.
type Report struct {
	Event  string `json:"event"`
	Status string `json:"status"`
}

func (r *Report) ToStrMap() map[string]string {
	return map[string]string{
		"event":  r.Event,
		"status": r.Status,
	}
}

// Reporter is an interface representing an actual device.
// In a polling scenario implementation might require some safeguards against excessive polling.
type Reporter interface {
	// AlarmReport returns a current alarm state for the provided event.
	// Some devices will produce only ephemeral alerts of which state is not stored in the device.
	// If device does not support stateful events while no ephemeral alert is waiting in queue, it should return a nil report instead.
	AlarmReport(event string) (*Report, error)
}

// Service is an interface representing an alarm system FIMP service.
type Service interface {
	adapter.Service

	// SendAlarmReport sends an alarm report for provided event. Returns true if a report was sent.
	// Depending on a caching and reporting configuration the service might decide to skip a report.
	// To make sure report is being sent regardless of circumstances set the force argument to true.
	// Regardless report will not be sent if reporter does not support stateful events.
	SendAlarmReport(event string, force bool) (bool, error)
	// SupportedEvents returns events that are supported by the alarm.
	SupportedEvents() []string
}

// Config represents a service configuration.
type Config struct {
	Specification     *fimptype.Service
	Reporter          Reporter
	ReportingStrategy cache.ReportingStrategy
}

// NewService creates a new instance of an alarm system FIMP service.
func NewService(
	publisher adapter.ServicePublisher,
	cfg *Config,
) Service {
	cfg.Specification.Name = AlarmSystem

	cfg.Specification.EnsureInterfaces(requiredInterfaces()...)

	if cfg.ReportingStrategy == nil {
		cfg.ReportingStrategy = DefaultReportingStrategy
	}

	return &service{
		Service:           adapter.NewService(publisher, cfg.Specification),
		reporter:          cfg.Reporter,
		lock:              &sync.Mutex{},
		reportingCache:    cache.NewReportingCache(),
		reportingStrategy: cfg.ReportingStrategy,
	}
}

// service is a private implementation of an alarm system FIMP service.
type service struct {
	adapter.Service

	reporter          Reporter
	lock              *sync.Mutex
	reportingCache    cache.ReportingCache
	reportingStrategy cache.ReportingStrategy
}

// SendAlarmReport sends an alarm report for provided event. Returns true if a report was sent.
// Depending on a caching and reporting configuration the service might decide to skip a report.
// To make sure report is being sent regardless of circumstances set the force argument to true.
// Regardless report will not be sent if reporter does not support stateful events.
func (s *service) SendAlarmReport(event string, force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	normalizedEvent, ok := s.normalizeEvent(event)
	if !ok {
		return false, fmt.Errorf("%s: event is unsupported: %s", s.Name(), event)
	}

	report, err := s.reporter.AlarmReport(normalizedEvent)
	if err != nil {
		return false, fmt.Errorf("failed to get alarm report for event %s: %w", normalizedEvent, err)
	}

	// If device does not support stateful events we can't report anything.
	if report == nil {
		return false, nil
	}

	// The reporter is free to return any spelling, and may reuse the struct between calls.
	// Keys and payload must stay tied to the advertised event, or distinct alarms collapse
	// onto one cache key; the copy also keeps the cached snapshot immune to reporter reuse.
	normalized := *report
	normalized.Event = normalizedEvent
	report = &normalized

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtAlarmReport, report.Event, report) {
		return false, nil
	}

	message := fimpgo.NewStrMapMessage(
		EvtAlarmReport,
		s.Name(),
		report.ToStrMap(),
		nil,
		nil,
		nil,
	).WithStorageStrategy(fimpgo.StorageStrategyAggregate, report.Event)

	if err = s.SendMessage(message); err != nil {
		return false, fmt.Errorf("failed to send alarm report: %w", err)
	}

	s.reportingCache.Reported(EvtAlarmReport, report.Event, report)

	return true, nil
}

// SupportedEvents returns events that are supported by the alarm.
func (s *service) SupportedEvents() []string {
	return s.Service.Specification().PropertyStrings(PropertySupportedEvents)
}

// normalizeEvent checks if event is supported and returns its normalized form.
func (s *service) normalizeEvent(event string) (string, bool) {
	return utils.Normalize(event, s.SupportedEvents())
}
