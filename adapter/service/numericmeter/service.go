package numericmeter

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/cache"
)

// DefaultReportingStrategy is the default reporting strategy used by the service for periodic reports.
var DefaultReportingStrategy = cache.ReportAtLeastEvery(30 * time.Minute)

// Reporter is an obligatory interface representing an actual device reporting meter values.
// In a polling scenario implementation might require some safeguards against excessive polling.
type Reporter interface {
	// MeterReport returns simplified meter report based on requested unit.
	MeterReport(unit Unit) (float64, error)
}

// ExportReporter is an optional interface representing an actual device reporting meter export values.
// In a polling scenario implementation might require some safeguards against excessive polling.
type ExportReporter interface {
	// MeterExportReport returns simplified meter export report based on requested unit.
	MeterExportReport(unit Unit) (float64, error)
}

// ExtendedReporter is an optional interface representing an actual device reporting meter extended values.
// In a polling scenario implementation might require some safeguards against excessive polling.
type ExtendedReporter interface {
	// MeterExtendedReport returns extended meter extended report for requested values.
	MeterExtendedReport(values Values) (ValuesReport, error)
}

// ResettableReporter is an interface representing an actual device supporting meter reset functionality.
type ResettableReporter interface {
	// MeterReset resets the meter.
	MeterReset() error
}

// Service is an interface representing a meter FIMP service.
// Depending on a caching and reporting configuration the service might decide to skip a report.
// To make sure report is being sent regardless of circumstances set the force argument to true.
type Service interface {
	adapter.Service

	// SendMeterReport sends a simplified meter report based on requested unit. Returns true if a report was sent.
	SendMeterReport(unit Unit, force bool) (bool, error)
	// SendMeterExportReport sends a simplified meter export report based on requested unit. Returns true if a report was sent.
	SendMeterExportReport(unit Unit, force bool) (bool, error)
	// SendMeterExtendedReport sends an extended meter report based on requested values. Returns true if a report was sent.
	SendMeterExtendedReport(values Values, force bool) (bool, error)
	// ResetMeter resets the meter.
	ResetMeter() error
	// SupportedUnits returns units that are supported by the simplified meter report.
	SupportedUnits() Units
	// SupportedExportUnits returns units that are supported by the simplified meter export report.
	SupportedExportUnits() Units
	// SupportedExtendedValues returns extended values that are supported by the extended meter report.
	SupportedExtendedValues() Values
	// SupportsExportReport returns true if meter supports the export report.
	SupportsExportReport() bool
	// SupportsExtendedReport returns true if meter supports the extended report.
	SupportsExtendedReport() bool
}

// Config represents a service configuration.
type Config struct {
	Specification     *fimptype.Service
	Reporter          Reporter
	ReportingStrategy cache.ReportingStrategy
}

// NewService creates new instance of a meter FIMP service.
func NewService(
	publisher adapter.ServicePublisher,
	cfg *Config,
) Service {
	cfg.Specification.EnsureInterfaces(requiredInterfaces()...)

	if cfg.ReportingStrategy == nil {
		cfg.ReportingStrategy = DefaultReportingStrategy
	}

	s := &service{
		Service:           adapter.NewService(publisher, cfg.Specification),
		reporter:          cfg.Reporter,
		lock:              &sync.Mutex{},
		reportingStrategy: cfg.ReportingStrategy,
		reportingCache:    cache.NewReportingCache(),
	}

	if s.SupportsExportReport() {
		cfg.Specification.EnsureInterfaces(exportInterfaces()...)
	}

	if s.SupportsExtendedReport() {
		cfg.Specification.EnsureInterfaces(extendedInterfaces()...)
	}

	if s.SupportsMeterReset() {
		cfg.Specification.EnsureInterfaces(resetInterfaces()...)
	}

	return s
}

// service is a private implementation of a meter FIMP service.
type service struct {
	adapter.Service

	reporter          Reporter
	lock              *sync.Mutex
	reportingCache    cache.ReportingCache
	reportingStrategy cache.ReportingStrategy
}

// SendMeterReport sends a simplified meter report based on requested unit. Returns true if a report was sent.
func (s *service) SendMeterReport(unit Unit, force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	normalizedUnit, ok := s.normalizeUnit(unit, s.SupportedUnits())
	if !ok {
		return false, fmt.Errorf("%s: unit is unsupported: %s", s.Name(), unit)
	}

	value, err := s.reporter.MeterReport(unit)
	if err != nil {
		return false, fmt.Errorf("%s: failed to retrieve meter report: %w", s.Name(), err)
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtMeterReport, normalizedUnit.String(), value) {
		return false, nil
	}

	message := fimpgo.NewFloatMessage(
		EvtMeterReport,
		s.Name(),
		value,
		map[string]string{
			PropertyUnit:      normalizedUnit.String(),
			PropertyIsVirtual: strconv.FormatBool(s.Specification().PropertyBool(PropertyIsVirtual)),
		},
		nil,
		nil,
	).WithStorageStrategy(fimpgo.StorageStrategyAggregate, normalizedUnit.String())

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("%s: failed to send meter report for unit %s: %w", s.Name(), normalizedUnit, err)
	}

	s.reportingCache.Reported(EvtMeterReport, normalizedUnit.String(), value)

	return true, nil
}

// SendMeterExportReport sends a simplified meter export report based on requested unit. Returns true if a report was sent.
func (s *service) SendMeterExportReport(unit Unit, force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	exportReporter, err := s.exportReporter()
	if err != nil {
		return false, err
	}

	normalizedUnit, ok := s.normalizeUnit(unit, s.SupportedExportUnits())
	if !ok {
		return false, fmt.Errorf("%s: unit is unsupported: %s", s.Name(), unit)
	}

	value, err := exportReporter.MeterExportReport(unit)
	if err != nil {
		return false, fmt.Errorf("%s: failed to retrieve meter export report: %w", s.Name(), err)
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtMeterExportReport, normalizedUnit.String(), value) {
		return false, nil
	}

	message := fimpgo.NewFloatMessage(
		EvtMeterExportReport,
		s.Name(),
		value,
		map[string]string{
			PropertyUnit:      normalizedUnit.String(),
			PropertyIsVirtual: strconv.FormatBool(s.Specification().PropertyBool(PropertyIsVirtual)),
		},
		nil,
		nil,
	).WithStorageStrategy(fimpgo.StorageStrategyAggregate, normalizedUnit.String())

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("%s: failed to send meter report for unit %s: %w", s.Name(), normalizedUnit, err)
	}

	s.reportingCache.Reported(EvtMeterExportReport, normalizedUnit.String(), value)

	return true, nil
}

// SendMeterExtendedReport sends an extended meter report. Returns true if a report was sent.
func (s *service) SendMeterExtendedReport(extendedValues Values, force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	extendedReporter, err := s.extendedReporter()
	if err != nil {
		return false, err
	}

	normalizedExtendedValues, err := s.normalizeExtendedValues(extendedValues)
	if err != nil {
		return false, fmt.Errorf("%s: failed to normalize extended values: %w", s.Name(), err)
	}

	values, err := extendedReporter.MeterExtendedReport(normalizedExtendedValues)
	if err != nil {
		return false, fmt.Errorf("%s: failed to retrieve extended meter report: %w", s.Name(), err)
	}

	if !force && !s.isExtendedReportRequired(values) {
		return false, nil
	}

	message := fimpgo.NewFloatMapMessage(
		EvtMeterExtReport,
		s.Name(),
		values.Map(),
		nil,
		nil,
		nil,
	).WithStorageStrategy(fimpgo.StorageStrategySplit, "")

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("%s: failed to send extended meter report: %w", s.Name(), err)
	}

	for extendedValue, value := range values {
		s.reportingCache.Reported(EvtMeterExtReport, extendedValue.String(), value)
	}

	return true, nil
}

// ResetMeter resets the meter.
func (s *service) ResetMeter() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	resettableMeter, err := s.resettableReporter()
	if err != nil {
		return err
	}

	err = resettableMeter.MeterReset()
	if err != nil {
		return fmt.Errorf("%s: failed to reset meter: %w", s.Name(), err)
	}

	return nil
}

// SupportedUnits returns units that are supported by the simplified meter report.
func (s *service) SupportedUnits() Units {
	return NewUnits(s.Specification().PropertyStrings(PropertySupportedUnits)...)
}

// SupportedExportUnits returns units that are supported by the simplified meter export report.
func (s *service) SupportedExportUnits() Units {
	return NewUnits(s.Specification().PropertyStrings(PropertySupportedExportUnits)...)
}

// SupportedExtendedValues returns extended values that are supported by the extended meter report.
func (s *service) SupportedExtendedValues() Values {
	return NewValues(s.Specification().PropertyStrings(PropertySupportedExtendedValues)...)
}

// SupportsExportReport returns true if meter supports the export report.
func (s *service) SupportsExportReport() bool {
	_, err := s.exportReporter()

	return err == nil
}

// exportReporter returns the export reporter, if supported.
func (s *service) exportReporter() (ExportReporter, error) {
	reporter, ok := s.reporter.(ExportReporter)
	if !ok {
		return nil, fmt.Errorf("%s: export meter report is not supported", s.Name())
	}

	if len(s.SupportedExportUnits()) == 0 {
		return nil, fmt.Errorf("%s: export meter report is not supported", s.Name())
	}

	return reporter, nil
}

// SupportsExtendedReport returns true if meter supports the extended report.
func (s *service) SupportsExtendedReport() bool {
	_, err := s.extendedReporter()

	return err == nil
}

// extendedReporter returns the extended reporter, if supported.
func (s *service) extendedReporter() (ExtendedReporter, error) {
	reporter, ok := s.reporter.(ExtendedReporter)
	if !ok {
		return nil, fmt.Errorf("%s: extended meter report is not supported", s.Name())
	}

	if len(s.SupportedExtendedValues()) == 0 {
		return nil, fmt.Errorf("%s: extended meter report is not supported", s.Name())
	}

	return reporter, nil
}

// SupportsMeterReset returns true if meter supports the reset.
func (s *service) SupportsMeterReset() bool {
	_, err := s.resettableReporter()

	return err == nil
}

// resettableReporter returns the resettable reporter, if supported.
func (s *service) resettableReporter() (ResettableReporter, error) {
	reporter, ok := s.reporter.(ResettableReporter)
	if !ok {
		return nil, fmt.Errorf("%s: meter reset is not supported", s.Name())
	}

	return reporter, nil
}

// normalizeUnit checks if unit is supported and returns its normalized form.
func (s *service) normalizeUnit(unit Unit, units Units) (Unit, bool) {
	for _, u := range units {
		if strings.EqualFold(unit.String(), u.String()) {
			return u, true
		}
	}

	return "", false
}

// normalizeExtendedValues checks if all values are supported and returns their normalized form.
func (s *service) normalizeExtendedValues(values Values) (Values, error) {
	normalizedValues := make(Values, len(values))

	for i, v := range values {
		normalizedValue, ok := s.normalizeValue(v, s.SupportedExtendedValues())
		if !ok {
			return nil, fmt.Errorf("%s: extended value %s is unsupported", s.Name(), v)
		}

		normalizedValues[i] = normalizedValue
	}

	return normalizedValues, nil
}

// normalizeUnit checks if unit is supported and returns its normalized form.
func (s *service) normalizeValue(value Value, values Values) (Value, bool) {
	for _, u := range values {
		if strings.EqualFold(value.String(), u.String()) {
			return u, true
		}
	}

	return "", false
}

// isReportRequired checks if a report is required for any of the given values.
func (s *service) isExtendedReportRequired(values map[Value]float64) bool {
	for name, value := range values {
		if s.reportingCache.ReportRequired(s.reportingStrategy, EvtMeterExtReport, string(name), value) {
			return true
		}
	}

	return false
}
