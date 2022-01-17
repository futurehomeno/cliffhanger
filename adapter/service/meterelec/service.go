package meterelec

import (
	"fmt"
	"strings"

	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
)

// Reporter is an interface representing an actual device reporting electricity meter values.
type Reporter interface {
	// Report returns simplified electricity meter report based on requested unit.
	Report(unit string) (float64, error)
	// ExtendedReport returns extended electricity meter report. Should return nil if extended reporter is not supported.
	ExtendedReport() (map[string]float64, error)
}

// Service is an interface representing a meter_elec FIMP service.
type Service interface {
	adapter.Service

	// Report returns simplified electricity meter report based on requested unit.
	Report(unit string) (float64, string, error)
	// ExtendedReport returns extended electricity meter report.
	ExtendedReport() (map[string]float64, error)
	// SupportedUnits returns units that are supported by the simplified meter report.
	SupportedUnits() []string
	// SupportedExtendedValues returns extended values that are supported by the extended meter report.
	SupportedExtendedValues() []string
	// SupportsExtendedReport returns true if meter supports the extended report.
	SupportsExtendedReport() bool
}

// NewService creates new instance of a meter_elec FIMP service.
func NewService(
	specification *fimptype.Service,
	reporter Reporter,
) Service {
	specification.Name = MeterElec

	return &service{
		Service:  adapter.NewService(specification),
		reporter: reporter,
	}
}

// service is a private implementation of a meter_elec FIMP service.
type service struct {
	adapter.Service

	reporter Reporter
}

// Report returns simplified electricity meter report based on requested unit.
func (s *service) Report(unit string) (float64, string, error) {
	normalizedUnit, ok := s.supportedUnit(unit)
	if !ok {
		return 0, "", fmt.Errorf("meter_elec: unit is unsupported: %s", unit)
	}

	value, err := s.reporter.Report(unit)
	if err != nil {
		return 0, "", fmt.Errorf("meter_elec: failed to retrieve report: %w", err)
	}

	return value, normalizedUnit, nil
}

// ExtendedReport returns extended electricity meter report.
func (s *service) ExtendedReport() (map[string]float64, error) {
	if !s.SupportsExtendedReport() {
		return nil, fmt.Errorf("meter_elec: extended report is unsupported")
	}

	values, err := s.reporter.ExtendedReport()
	if err != nil {
		return nil, fmt.Errorf("meter_elec: failed to retrieve extended report: %w", err)
	}

	return values, nil
}

// SupportedUnits returns units that are supported by the simplified meter report.
func (s *service) SupportedUnits() []string {
	return s.propertyStrings("sup_units")
}

// SupportedExtendedValues returns extended values that are supported by the extended meter report.
func (s *service) SupportedExtendedValues() []string {
	return s.propertyStrings("sup_extended_vals")
}

// SupportsExtendedReport returns true if meter supports the extended report.
func (s *service) SupportsExtendedReport() bool {
	return len(s.SupportedExtendedValues()) > 0
}

// supportedUnit checks if unit is supported and returns it's normalized form.
func (s *service) supportedUnit(unit string) (string, bool) {
	for _, u := range s.SupportedUnits() {
		if strings.EqualFold(unit, u) {
			return u, true
		}
	}

	return "", false
}

// propertyStrings extracts property settings out of the specification.
func (s *service) propertyStrings(name string) []string {
	value, ok := s.Service.Specification().Props[name]
	if !ok {
		return nil
	}

	values, ok := value.([]string)
	if !ok {
		return nil
	}

	return values
}
