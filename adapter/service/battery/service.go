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

func (bar *AlarmReport) ToStrMap() (strMap map[string]string, err error) {
	strMap = map[string]string{
		"event":  bar.Event,
		"status": bar.Status,
	}

	return strMap, nil
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
	// BatteryHealthReport returns a current battery health state.
	BatteryHealthReport() (int64, error)
	// BatterySensorReport returns a current battery sensor state.
	BatterySensorReport() (sensorValue float64, unit string, err error)
	// BatteryFullReport returns a current battery state.
	BatteryFullReport() (FullReport, error)
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
}

// Config represents a service configuration.
type Config struct {
	Specification     *fimptype.Service
	Reporter          Reporter
	ReportingStrategy cache.ReportingStrategy
}

// NewService creates a new instance of a battery FIMP service.
func NewService(
	mqtt *fimpgo.MqttTransport,
	cfg *Config,
) Service {
	cfg.Specification.Name = Battery

	cfg.Specification.EnsureInterfaces(requiredInterfaces()...)

	if cfg.ReportingStrategy == nil {
		cfg.ReportingStrategy = DefaultReportingStrategy
	}

	s := &service{
		Service:           adapter.NewService(mqtt, cfg.Specification),
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

	alarmValue, err := alarm.ToStrMap()
	if err != nil {
		return false, fmt.Errorf("failed to map alarm value to map of strings: %w", err)
	}

	message := fimpgo.NewStrMapMessage(
		EvtAlarmReport,
		s.Name(),
		alarmValue,
		nil,
		nil,
		nil,
	)

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

	health, err := s.reporter.BatteryHealthReport()
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

	sensor, unit, err := s.reporter.BatterySensorReport()
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
