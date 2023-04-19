package battery

import (
	"fmt"
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
	AlarmLowBatteryEvent  = "low_battery"
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

// FullReport represents value structure of a battery full report.
type FullReport struct {
	Level  int     `json:"lvl"`
	Health int     `json:"health"`
	State  string  `json:"state"`
	Temp   float64 `json:"temp_sensor"`
}

// Reporter is an interface representing an actual car charger device.
// In a polling scenario implementation might require some safeguards against excessive polling.
type Reporter interface {
	// BatteryLevelReport returns a current battery level.
	BatteryLevelReport() (level int64, state string, err error)
	// BatteryAlarmReport returns a current battery alarm state.
	BatteryAlarmReport() (AlarmReport, error)
	// BatteryFullReport returns a current battery state.
	BatteryFullReport() (FullReport, error)
}

// HealthReporter is an interface representing an actual device supporting health reports.
// In a polling scenario implementation might require some safeguards against excessive polling.
type HealthReporter interface {
	Reporter

	// BatteryHealthReport returns a current battery health state.
	BatteryHealthReport() (int64, error)
}

// SensorReporter is an interface representing an actual device supporting sensor reports.
// In a polling scenario implementation might require some safeguards against excessive polling.
type SensorReporter interface {
	Reporter

	// BatterySensorReport returns a current battery sensor state.
	BatterySensorReport() (sensorValue float64, unit string, err error)
}

// Service is an interface representing a battery FIMP service.
type Service interface {
	adapter.Service

	// SendBatteryLevelReport sends a battery level report. Returns true if a report was sent.
	// Depending on a caching and reporting configuration the service might decide to skip a report.
	// To make sure report is being sent regardless of circumstances set the force argument to true.
	SendBatteryLevelReport(force bool) (bool, error)
	// SendBatteryAlarmReport sends a battery alarm report. Returns true if a report was sent.
	// Depending on a caching and reporting configuration the service might decide to skip a report.
	// To make sure report is being sent regardless of circumstances set the force argument to true.
	SendBatteryAlarmReport(force bool) (bool, error)
	// SendBatteryHealthReport sends a battery health report. Returns true if a report was sent.
	// Depending on a caching and reporting configuration the service might decide to skip a report.
	// To make sure report is being sent regardless of circumstances set the force argument to true.
	SendBatteryHealthReport(force bool) (bool, error)
	// SendBatterySensorReport sends a battery sensor report. Returns true if a report was sent.
	// Depending on a caching and reporting configuration the service might decide to skip a report.
	// To make sure report is being sent regardless of circumstances set the force argument to true.
	SendBatterySensorReport(force bool) (bool, error)
	// SendBatteryFullReport sends a full battery report. Returns true if a report was sent.
	// Depending on a caching and reporting configuration the service might decide to skip a report.
	// To make sure report is being sent regardless of circumstances set the force argument to true.
	SendBatteryFullReport(force bool) (bool, error)

	// SupportsHealthReport returns true if the service supports battery health reports.
	SupportsHealthReport() bool
	// SupportsSensorReport returns true if the service supports battery sensor reports.
	SupportsSensorReport() bool
}

// Config represents a service configuration.
type Config struct {
	Specification     *fimptype.Service
	Reporter          Reporter
	ReportingStrategy cache.ReportingStrategy
}

// NewService creates a new instance of a battery FIMP service.
func NewService(
	publisher adapter.Publisher,
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

	if s.SupportsHealthReport() {
		cfg.Specification.EnsureInterfaces(healthInterfaces()...)
	}

	if s.SupportsSensorReport() {
		cfg.Specification.EnsureInterfaces(sensorInterfaces()...)
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

	level, state, err := s.reporter.BatteryLevelReport()
	if err != nil {
		return false, fmt.Errorf("failed to get battery level report: %w", err)
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtLevelReport, "", level) {
		return false, nil
	}

	props := fimpgo.Props{
		"state": state,
	}

	message := fimpgo.NewIntMessage(
		EvtLevelReport,
		s.Name(),
		level,
		props,
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

// SendBatteryAlarmReport sends a battery alarm report. Returns true if a report was sent.
// Depending on a caching and reporting configuration the service might decide to skip a report.
// To make sure report is being sent regardless of circumstances set the force argument to true.
func (s *service) SendBatteryAlarmReport(force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	alarm, err := s.reporter.BatteryAlarmReport()
	if err != nil {
		return false, fmt.Errorf("failed to get battery alarm report: %w", err)
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtAlarmReport, "", alarm) {
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

	s.reportingCache.Reported(EvtAlarmReport, "", alarm)

	return true, nil
}

// SendBatteryHealthReport sends a battery health report. Returns true if a report was sent.
// Depending on a caching and reporting configuration the service might decide to skip a report.
// To make sure report is being sent regardless of circumstances set the force argument to true.
func (s *service) SendBatteryHealthReport(force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if !s.SupportsHealthReport() {
		return false, fmt.Errorf("%s: battery health reports are not supported", s.Name())
	}

	healthReporter, ok := s.reporter.(HealthReporter)
	if !ok {
		return false, fmt.Errorf("%s: battery health reports are not supported", s.Name())
	}

	health, err := healthReporter.BatteryHealthReport()
	if err != nil {
		return false, fmt.Errorf("failed to get battery health report: %w", err)
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtHealthReport, "", health) {
		return false, nil
	}

	message := fimpgo.NewIntMessage(
		EvtHealthReport,
		s.Name(),
		health,
		nil,
		nil,
		nil,
	)

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("failed to send battery health report: %w", err)
	}

	s.reportingCache.Reported(EvtHealthReport, "", health)

	return true, nil
}

// SendBatterySensorReport sends a battery sensor report. Returns true if a report was sent.
// Depending on a caching and reporting configuration the service might decide to skip a report.
// To make sure report is being sent regardless of circumstances set the force argument to true.
func (s *service) SendBatterySensorReport(force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if !s.SupportsSensorReport() {
		return false, fmt.Errorf("%s: battery sensor reports are not supported", s.Name())
	}

	sensorReporter, ok := s.reporter.(SensorReporter)
	if !ok {
		return false, fmt.Errorf("%s: battery sensor reports are not supported", s.Name())
	}

	sensor, unit, err := sensorReporter.BatterySensorReport()
	if err != nil {
		return false, fmt.Errorf("failed to get battery sensor report: %w", err)
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtSensorReport, "", sensor) {
		return false, nil
	}

	props := fimpgo.Props{
		"unit": unit,
	}

	message := fimpgo.NewFloatMessage(
		EvtSensorReport,
		s.Name(),
		sensor,
		props,
		nil,
		nil,
	)

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("failed to send battery sensor report: %w", err)
	}

	s.reportingCache.Reported(EvtSensorReport, "", sensor)

	return true, nil
}

// SendBatteryFullReport sends a full battery report. Returns true if a report was sent.
// Depending on a caching and reporting configuration the service might decide to skip a report.
// To make sure report is being sent regardless of circumstances set the force argument to true.
func (s *service) SendBatteryFullReport(force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	full, err := s.reporter.BatteryFullReport()
	if err != nil {
		return false, fmt.Errorf("failed to get battery full report: %w", err)
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtBatteryReport, "", full) {
		return false, nil
	}

	message := fimpgo.NewObjectMessage(
		EvtBatteryReport,
		s.Name(),
		full,
		nil,
		nil,
		nil,
	)

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("failed to send battery full report: %w", err)
	}

	s.reportingCache.Reported(EvtBatteryReport, "", full)

	return true, nil
}

// SupportsHealthReport returns true if the battery supports health reports.
func (s *service) SupportsHealthReport() bool {
	_, ok := s.reporter.(HealthReporter)

	return ok
}

// SupportsSensorReport returns true if the battery supports sensor reports.
func (s *service) SupportsSensorReport() bool {
	_, ok := s.reporter.(SensorReporter)

	return ok
}
