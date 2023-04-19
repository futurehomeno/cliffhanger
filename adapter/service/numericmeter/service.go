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

// Constants defining important properties specific for the service.
const (
	UnitKWh         = "kWh"
	UnitW           = "W"
	UnitA           = "A"
	UnitV           = "V"
	UnitKVAh        = "kVAh"
	UnitHz          = "Hz"
	UnitPowerFactor = "power_factor"
	UnitPulseCount  = "pulse_c"
	UnitCubicMeter  = "cub_m"
	UnitCubicFeet   = "cub_f"
	UnitGallon      = "gallon"

	ValueEnergyImport        = "e_import"
	ValueEnergyExport        = "e_export"
	ValueLastEnergyExport    = "last_e_export"
	ValueLastEnergyImport    = "last_e_import"
	ValuePowerImport         = "p_import"
	ValueReactivePowerImport = "p_import_react"
	ValueApparentPowerImport = "p_import_apparent"
	ValueAveragePowerImport  = "p_import_avg"
	ValueMinimumPowerImport  = "p_import_min"
	ValueMaximumPowerImport  = "p_import_max"
	ValuePowerExport         = "p_export"
	ValueReactivePowerExport = "p_export_react"
	ValueMinimumPowerExport  = "p_export_min"
	ValueMaximumPowerExport  = "p_export_max"
	ValuePowerFactor         = "p_factor"
	ValueFrequency           = "freq"
	ValueMinimumFrequency    = "freq_min"
	ValueMaximumFrequency    = "freq_max"
	ValueVoltagePhase1       = "u1"
	ValueVoltagePhase2       = "u2"
	ValueVoltagePhase3       = "u3"
	ValueCurrentPhase1       = "i1"
	ValueCurrentPhase2       = "i2"
	ValueCurrentPhase3       = "i3"
	ValueDCPower             = "dc_p"
	ValueMinimumDCPower      = "dc_p_min"
	ValueMaximumDCPower      = "dc_p_max"
	ValueDCVoltage           = "dc_u"
	ValueMinimumDCVoltage    = "dc_u_min"
	ValueMaximumDCVoltage    = "dc_u_max"
	ValueDCCurrent           = "dc_i"
	ValueMinimumDCCurrent    = "dc_i_min"
	ValueMaximumDCCurrent    = "dc_i_max"

	PropertyUnit                    = "unit"
	PropertySupportedUnits          = "sup_units"
	PropertySupportedExportUnits    = "sup_export_units"
	PropertySupportedExtendedValues = "sup_extended_vals"
	PropertyIsVirtual               = "is_virtual"
)

// DefaultReportingStrategy is the default reporting strategy used by the service for periodic reports.
var DefaultReportingStrategy = cache.ReportAtLeastEvery(30 * time.Minute)

// Reporter is an obligatory interface representing an actual device reporting meter values.
// In a polling scenario implementation might require some safeguards against excessive polling.
type Reporter interface {
	// MeterReport returns simplified meter report based on requested unit.
	MeterReport(unit string) (float64, error)
}

// ExportReporter is an optional interface representing an actual device reporting meter export values.
// In a polling scenario implementation might require some safeguards against excessive polling.
type ExportReporter interface {
	// MeterExportReport returns simplified meter export report based on requested unit.
	MeterExportReport(unit string) (float64, error)
}

// ExtendedReporter is an optional interface representing an actual device reporting meter extended values.
// In a polling scenario implementation might require some safeguards against excessive polling.
type ExtendedReporter interface {
	// MeterExtendedReport returns extended meter extended report for requested values.
	MeterExtendedReport(values []string) (map[string]float64, error)
}

// ResettableReporter is an interface representing an actual device supporting meter reset functionality.
type ResettableReporter interface {
	// MeterReset resets the meter.
	MeterReset() error
}

// Service is an interface representing a meter FIMP service.
type Service interface {
	adapter.Service

	// SendMeterReport sends a simplified meter report based on requested unit. Returns true if a report was sent.
	// Depending on a caching and reporting configuration the service might decide to skip a report.
	// To make sure report is being sent regardless of circumstances set the force argument to true.
	SendMeterReport(unit string, force bool) (bool, error)
	// SendMeterExportReport sends a simplified meter export report based on requested unit. Returns true if a report was sent.
	// Depending on a caching and reporting configuration the service might decide to skip a report.
	// To make sure report is being sent regardless of circumstances set the force argument to true.
	SendMeterExportReport(unit string, force bool) (bool, error)
	// SendMeterExtendedReport sends an extended meter report based on requested values. Returns true if a report was sent.
	// Depending on a caching and reporting configuration the service might decide to skip a report.
	// To make sure report is being sent regardless of circumstances set the force argument to true.
	SendMeterExtendedReport(values []string, force bool) (bool, error)
	// ResetMeter resets the meter.
	ResetMeter() error
	// SupportedUnits returns units that are supported by the simplified meter report.
	SupportedUnits() []string
	// SupportedExportUnits returns units that are supported by the simplified meter export report.
	SupportedExportUnits() []string
	// SupportedExtendedValues returns extended values that are supported by the extended meter report.
	SupportedExtendedValues() []string
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
	publisher adapter.Publisher,
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
// Depending on a caching and reporting configuration the service might decide to skip a report.
// To make sure report is being sent regardless of circumstances set the force argument to true.
func (s *service) SendMeterReport(unit string, force bool) (bool, error) {
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

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtMeterReport, normalizedUnit, value) {
		return false, nil
	}

	message := fimpgo.NewFloatMessage(
		EvtMeterReport,
		s.Name(),
		value,
		map[string]string{
			PropertyUnit:      normalizedUnit,
			PropertyIsVirtual: strconv.FormatBool(s.Specification().PropertyBool(PropertyIsVirtual)),
		},
		nil,
		nil,
	).WithStorageStrategy(fimpgo.StorageStrategyAggregate, normalizedUnit)

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("%s: failed to send meter report for unit %s: %w", s.Name(), normalizedUnit, err)
	}

	s.reportingCache.Reported(EvtMeterReport, normalizedUnit, value)

	return true, nil
}

// SendMeterExportReport sends a simplified meter export report based on requested unit. Returns true if a report was sent.
// Depending on a caching and reporting configuration the service might decide to skip a report.
// To make sure report is being sent regardless of circumstances set the force argument to true.
func (s *service) SendMeterExportReport(unit string, force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if !s.SupportsExportReport() {
		return false, fmt.Errorf("%s: export meter report is unsupported", s.Name())
	}

	exportReporter, ok := s.reporter.(ExportReporter)
	if !ok {
		return false, fmt.Errorf("%s: export meter report is unsupported", s.Name())
	}

	normalizedUnit, ok := s.normalizeUnit(unit, s.SupportedExportUnits())
	if !ok {
		return false, fmt.Errorf("%s: unit is unsupported: %s", s.Name(), unit)
	}

	value, err := exportReporter.MeterExportReport(unit)
	if err != nil {
		return false, fmt.Errorf("%s: failed to retrieve meter export report: %w", s.Name(), err)
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtMeterExportReport, normalizedUnit, value) {
		return false, nil
	}

	message := fimpgo.NewFloatMessage(
		EvtMeterExportReport,
		s.Name(),
		value,
		map[string]string{
			PropertyUnit:      normalizedUnit,
			PropertyIsVirtual: strconv.FormatBool(s.Specification().PropertyBool(PropertyIsVirtual)),
		},
		nil,
		nil,
	).WithStorageStrategy(fimpgo.StorageStrategyAggregate, normalizedUnit)

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("%s: failed to send meter report for unit %s: %w", s.Name(), normalizedUnit, err)
	}

	s.reportingCache.Reported(EvtMeterExportReport, normalizedUnit, value)

	return true, nil
}

// SendMeterExtendedReport sends an extended meter report. Returns true if a report was sent.
// Depending on a caching and reporting configuration the service might decide to skip a report.
// To make sure report is being sent regardless of circumstances set the force argument to true.
func (s *service) SendMeterExtendedReport(extendedValues []string, force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if !s.SupportsExtendedReport() {
		return false, fmt.Errorf("%s: extended meter report is unsupported", s.Name())
	}

	extendedReporter, ok := s.reporter.(ExtendedReporter)
	if !ok {
		return false, fmt.Errorf("%s: extended meter report is unsupported", s.Name())
	}

	normalizedExtendedValues, err := s.normalizeExtendedValues(extendedValues)
	if !ok {
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
		values,
		nil,
		nil,
		nil,
	).WithStorageStrategy(fimpgo.StorageStrategySplit, "")

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("%s: failed to send extended meter report: %w", s.Name(), err)
	}

	for extendedValue, value := range values {
		s.reportingCache.Reported(EvtMeterExtReport, extendedValue, value)
	}

	return true, nil
}

// ResetMeter resets the meter.
func (s *service) ResetMeter() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if !s.SupportsMeterReset() {
		return fmt.Errorf("%s: meter reset is unsupported", s.Name())
	}

	resettableMeter, ok := s.reporter.(ResettableReporter)
	if !ok {
		return fmt.Errorf("%s: meter reset is unsupported", s.Name())
	}

	err := resettableMeter.MeterReset()
	if err != nil {
		return fmt.Errorf("%s: failed to reset meter: %w", s.Name(), err)
	}

	return nil
}

// SupportedUnits returns units that are supported by the simplified meter report.
func (s *service) SupportedUnits() []string {
	return s.Specification().PropertyStrings(PropertySupportedUnits)
}

// SupportedExportUnits returns units that are supported by the simplified meter export report.
func (s *service) SupportedExportUnits() []string {
	return s.Specification().PropertyStrings(PropertySupportedExportUnits)
}

// SupportedExtendedValues returns extended values that are supported by the extended meter report.
func (s *service) SupportedExtendedValues() []string {
	return s.Specification().PropertyStrings(PropertySupportedExtendedValues)
}

// SupportsExportReport returns true if meter supports the export report.
func (s *service) SupportsExportReport() bool {
	_, ok := s.reporter.(ExportReporter)
	if !ok {
		return false
	}

	if len(s.SupportedExportUnits()) == 0 {
		return false
	}

	return true
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

// SupportsMeterReset returns true if meter supports the reset.
func (s *service) SupportsMeterReset() bool {
	_, ok := s.reporter.(ResettableReporter)

	return ok
}

// normalizeUnit checks if unit is supported and returns its normalized form.
func (s *service) normalizeUnit(unit string, units []string) (string, bool) {
	for _, u := range units {
		if strings.EqualFold(unit, u) {
			return u, true
		}
	}

	return "", false
}

// normalizeExtendedValues checks if all values are supported and returns their normalized form.
func (s *service) normalizeExtendedValues(values []string) ([]string, error) {
	normalizedValues := make([]string, len(values))

	for i, v := range values {
		normalizedValue, ok := s.normalizeUnit(v, s.SupportedExtendedValues())
		if !ok {
			return nil, fmt.Errorf("%s: extended value %s is unsupported", s.Name(), v)
		}

		normalizedValues[i] = normalizedValue
	}

	return normalizedValues, nil
}

// isReportRequired checks if a report is required for any of the given values.
func (s *service) isExtendedReportRequired(values map[string]float64) bool {
	for name, value := range values {
		if s.reportingCache.ReportRequired(s.reportingStrategy, EvtMeterExtReport, name, value) {
			return true
		}
	}

	return false
}
