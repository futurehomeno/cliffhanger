package battery

import (
	"fmt"
	"strings"
	"sync"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/cache"
)

// DefaultReportingStrategy is the default state reporting strategy used by the service for periodic reports of state changes.
var DefaultReportingStrategy = cache.ReportOnChangeOnly()

// Constants defining important properties specific for the service.
const (
	PropertySupportedEvents = "sup_events"

	AlarmEventLowBattery = "low_battery"

	AlarmStatusActivate   = "activ"
	AlarmStatusDeactivate = "deactiv"
)

// AlarmReport represents value structure of a battery alarm report.
type AlarmReport struct {
	Event  string `json:"event"`
	Status string `json:"status"`
}

func (r *AlarmReport) ToStrMap() map[string]string {
	return map[string]string{
		"event":  r.Event,
		"status": r.Status,
	}
}

// Reporter is an interface representing an actual car charger device.
// In a polling scenario implementation might require some safeguards against excessive polling.
type Reporter interface {
	// BatteryLevelReport returns a current battery level.
	BatteryLevelReport() (level int64, err error)
	// BatteryAlarmReport returns a current battery alarm state for the provided event.
	// Some devices will produce only ephemeral alerts of which state is not stored in the device.
	// If device does not support stateful events while no ephemeral alert is waiting in queue, it should return a nil report instead.
	BatteryAlarmReport(event string) (*AlarmReport, error)
}

// Service is an interface representing a battery FIMP service.
type Service interface {
	adapter.Service

	// SendBatteryLevelReport sends a battery level report. Returns true if a report was sent.
	// Depending on a caching and reporting configuration the service might decide to skip a report.
	// To make sure report is being sent regardless of circumstances set the force argument to true.
	SendBatteryLevelReport(force bool) (bool, error)
	// SendBatteryAlarmReport sends a battery alarm report for provided event. Returns true if a report was sent.
	// Depending on a caching and reporting configuration the service might decide to skip a report.
	// To make sure report is being sent regardless of circumstances set the force argument to true.
	// Regardless report will not be sent if reported does not support stateful events.
	SendBatteryAlarmReport(event string, force bool) (bool, error)
	// SupportedEvents returns events that are supported by the battery alarm.
	SupportedEvents() []string
}

// Config represents a service configuration.
type Config struct {
	Specification     *fimptype.Service
	Reporter          Reporter
	ReportingStrategy cache.ReportingStrategy
}

// NewService creates a new instance of a battery FIMP service.
func NewService(
	publisher adapter.ServicePublisher,
	cfg *Config,
) Service {
	cfg.Specification.Name = Battery

	cfg.Specification.EnsureInterfaces(requiredInterfaces()...)

	if cfg.ReportingStrategy == nil {
		cfg.ReportingStrategy = DefaultReportingStrategy
	}

	s := &service{
		Service:           adapter.NewService(publisher, cfg.Specification),
		reporter:          cfg.Reporter,
		lock:              &sync.Mutex{},
		reportingCache:    cache.NewReportingCache(),
		reportingStrategy: cfg.ReportingStrategy,
	}

	return s
}

// service is a private implementation of a battery FIMP service.
type service struct {
	adapter.Service

	reporter          Reporter
	lock              *sync.Mutex
	reportingCache    cache.ReportingCache
	reportingStrategy cache.ReportingStrategy
}

// SendBatteryLevelReport sends a battery level report. Returns true if a report was sent.
// Depending on a caching and reporting configuration the service might decide to skip a report.
// To make sure report is being sent regardless of circumstances set the force argument to true.
func (s *service) SendBatteryLevelReport(force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	level, err := s.reporter.BatteryLevelReport()
	if err != nil {
		return false, fmt.Errorf("failed to get battery level report: %w", err)
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtLevelReport, "", level) {
		return false, nil
	}

	message := fimpgo.NewIntMessage(
		EvtLevelReport,
		s.Name(),
		level,
		nil,
		nil,
		nil,
	)

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("failed to send battery level report: %w", err)
	}

	s.reportingCache.Reported(EvtLevelReport, "", level)

	return true, nil
}

// SendBatteryAlarmReport sends a battery alarm report for provided event. Returns true if a report was sent.
// Depending on a caching and reporting configuration the service might decide to skip a report.
// To make sure report is being sent regardless of circumstances set the force argument to true.
// Regardless report will not be sent if reported does not support stateful events.
func (s *service) SendBatteryAlarmReport(event string, force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	normalizedEvent, ok := s.normalizeEvent(event)
	if !ok {
		return false, fmt.Errorf("%s: event is unsupported: %s", s.Name(), event)
	}

	alarm, err := s.reporter.BatteryAlarmReport(normalizedEvent)
	if err != nil {
		return false, fmt.Errorf("failed to get battery alarm report for event %s: %w", normalizedEvent, err)
	}

	// If device does not support stateful events we can't report anything.
	if alarm == nil {
		return false, nil
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtAlarmReport, alarm.Event, alarm) {
		return false, nil
	}

	message := fimpgo.NewStrMapMessage(
		EvtAlarmReport,
		s.Name(),
		alarm.ToStrMap(),
		nil,
		nil,
		nil,
	).WithStorageStrategy(fimpgo.StorageStrategyAggregate, alarm.Event)

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("failed to send battery alarm report: %w", err)
	}

	s.reportingCache.Reported(EvtAlarmReport, alarm.Event, alarm)

	return true, nil
}

// SupportedEvents returns events that are supported by the battery alarm.
func (s *service) SupportedEvents() []string {
	return s.Service.Specification().PropertyStrings(PropertySupportedEvents)
}

// normalizeEvent checks if event is supported and returns its normalized form.
func (s *service) normalizeEvent(unit string) (string, bool) {
	for _, u := range s.SupportedEvents() {
		if strings.EqualFold(unit, u) {
			return u, true
		}
	}

	return "", false
}
