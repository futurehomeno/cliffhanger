package meterelec

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/cache"
)

// Constants defining important properties specific for the service.
const (
	UnitKWh = "kWh"
	UnitW   = "W"
	UnitA   = "A"
	UnitV   = "V"

	PropertySupportedUnits          = "sup_units"
	PropertySupportedExtendedValues = "sup_extended_vals"
)

// DefaultReportingStrategy is the default reporting strategy used by the service for periodic reports.
var DefaultReportingStrategy = cache.ReportAtLeastEvery(30 * time.Minute)

// Reporter is an interface representing an actual device reporting electricity meter values.
// In a polling scenario implementation might require some safeguards against excessive polling.
type Reporter interface {
	// ElectricityMeterReport returns simplified electricity meter report based on requested unit.
	ElectricityMeterReport(unit string) (float64, error)
}

// ExtendedReporter is an interface representing an actual device reporting electricity meter values supporting extended reports.
// In a polling scenario implementation might require some safeguards against excessive polling.
type ExtendedReporter interface {
	Reporter

	// ElectricityMeterExtendedReport returns extended electricity meter report.
	ElectricityMeterExtendedReport() (map[string]float64, error)
}

// Service is an interface representing a meter_elec FIMP service.
type Service interface {
	adapter.Service

	// SendMeterReport sends a simplified electricity meter report based on requested unit. Returns true if a report was sent.
	// Depending on a caching and reporting configuration the service might decide to skip a report.
	// To make sure report is being sent regardless of circumstances set the force argument to true.
	SendMeterReport(unit string, force bool) (bool, error)
	// SendMeterExtendedReport sends an extended electricity meter report. Returns true if a report was sent.
	// Depending on a caching and reporting configuration the service might decide to skip a report.
	// To make sure report is being sent regardless of circumstances set the force argument to true.
	SendMeterExtendedReport(force bool) (bool, error)
	// SupportedUnits returns units that are supported by the simplified meter report.
	SupportedUnits() []string
	// SupportedExtendedValues returns extended values that are supported by the extended meter report.
	SupportedExtendedValues() []string
	// SupportsExtendedReport returns true if meter supports the extended report.
	SupportsExtendedReport() bool
}

// Config represents a service configuration.
type Config struct {
	Specification     *fimptype.Service
	Reporter          Reporter
	ReportingStrategy cache.ReportingStrategy
}

// NewService creates new instance of a meter_elec FIMP service.
func NewService(
	mqtt *fimpgo.MqttTransport,
	cfg *Config,
) Service {
	cfg.Specification.Name = MeterElec

	cfg.Specification.EnsureInterfaces(requiredInterfaces()...)

	if cfg.ReportingStrategy == nil {
		cfg.ReportingStrategy = DefaultReportingStrategy
	}

	s := &service{
		Service:           adapter.NewService(mqtt, cfg.Specification),
		reporter:          cfg.Reporter,
		lock:              &sync.Mutex{},
		reportingStrategy: cfg.ReportingStrategy,
		reportingCache:    cache.NewReportingCache(),
	}

	if s.SupportsExtendedReport() {
		cfg.Specification.EnsureInterfaces(extendedInterfaces()...)
	}

	return s
}

// service is a private implementation of a meter_elec FIMP service.
type service struct {
	adapter.Service

	reporter          Reporter
	lock              *sync.Mutex
	reportingCache    cache.ReportingCache
	reportingStrategy cache.ReportingStrategy
}

// SendMeterReport sends a simplified electricity meter report based on requested unit. Returns true if a report was sent.
// Depending on a caching and reporting configuration the service might decide to skip a report.
// To make sure report is being sent regardless of circumstances set the force argument to true.
func (s *service) SendMeterReport(unit string, force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	normalizedUnit, ok := s.normalizeUnit(unit)
	if !ok {
		return false, fmt.Errorf("%s: unit is unsupported: %s", s.Name(), unit)
	}

	value, err := s.reporter.ElectricityMeterReport(unit)
	if err != nil {
		return false, fmt.Errorf("%s: failed to retrieve meter report: %w", s.Name(), err)
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtMeterReport, normalizedUnit, value) {
		return false, nil
	}

	message := fimpgo.NewFloatMessage(
		EvtMeterReport,
		MeterElec,
		value,
		map[string]string{
			"unit": normalizedUnit,
		},
		nil,
		nil,
	)

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("%s: failed to send meter report for unit %s: %w", s.Name(), normalizedUnit, err)
	}

	s.reportingCache.Reported(EvtMeterReport, normalizedUnit, value)

	return true, nil
}

// SendMeterExtendedReport sends an extended electricity meter report. Returns true if a report was sent.
// Depending on a caching and reporting configuration the service might decide to skip a report.
// To make sure report is being sent regardless of circumstances set the force argument to true.
func (s *service) SendMeterExtendedReport(force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if !s.SupportsExtendedReport() {
		return false, fmt.Errorf("%s: extended meter report is unsupported", s.Name())
	}

	extendedReporter, ok := s.reporter.(ExtendedReporter)
	if !ok {
		return false, fmt.Errorf("%s: extended meter report is unsupported", s.Name())
	}

	values, err := extendedReporter.ElectricityMeterExtendedReport()
	if err != nil {
		return false, fmt.Errorf("%s: failed to retrieve extended meter report: %w", s.Name(), err)
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtMeterExtReport, "", values) {
		return false, nil
	}

	message := fimpgo.NewFloatMapMessage(
		EvtMeterExtReport,
		MeterElec,
		values,
		nil,
		nil,
		nil,
	)

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("meter_elec: failed to send extended meter report: %w", err)
	}

	s.reportingCache.Reported(EvtMeterExtReport, "", values)

	return true, nil
}

// SupportedUnits returns units that are supported by the simplified meter report.
func (s *service) SupportedUnits() []string {
	return s.Specification().PropertyStrings(PropertySupportedUnits)
}

// SupportedExtendedValues returns extended values that are supported by the extended meter report.
func (s *service) SupportedExtendedValues() []string {
	return s.Specification().PropertyStrings(PropertySupportedExtendedValues)
}

// SupportsExtendedReport returns true if meter supports the extended report.
func (s *service) SupportsExtendedReport() bool {
	_, ok := s.reporter.(ExtendedReporter)
	if !ok {
		return false
	}

	if len(s.SupportedExtendedValues()) == 0 {
		return false
	}

	return true
}

// normalizeUnit checks if unit is supported and returns its normalized form.
func (s *service) normalizeUnit(unit string) (string, bool) {
	for _, u := range s.SupportedUnits() {
		if strings.EqualFold(unit, u) {
			return u, true
		}
	}

	return "", false
}
