package meterelec

import (
	"fmt"
	"strings"

	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
)

type Reporter func(unit string) (float64, error)

type ExtendedReporter func() (map[string]float64, error)

type Service interface {
	adapter.Service

	// Report returns simplified HAN meter reporter based on input unit.
	Report(unit string) (float64, string, error)
	// ExtendedReport returns extended HAN meter reporter. Should return nil if extended reporter is not supported.
	ExtendedReport() (map[string]float64, error)
	// SupportedUnits returns units that are supported by the simplified meter reporter.
	SupportedUnits() []string
	// SupportedExtendedValues returns extended values that are supported by the extended meter reporter.
	SupportedExtendedValues() []string
	// SupportsExtendedReport returns true if meter supports the extended reporter.
	SupportsExtendedReport() bool
}

func NewService(
	inclusionReport *fimptype.Service,
	reporter Reporter,
	extendedReporter ExtendedReporter,
) Service {
	inclusionReport.Name = MeterElec

	return &service{
		Service:          adapter.NewService(inclusionReport),
		reporter:         reporter,
		extendedReporter: extendedReporter,
	}
}

type service struct {
	adapter.Service

	reporter         Reporter
	extendedReporter ExtendedReporter
}

func (s *service) Report(unit string) (float64, string, error) {
	normalizedUnit, ok := s.supportedUnit(unit)
	if !ok {
		return 0, "", fmt.Errorf("meter_elec: unit is unsupported: %s", unit)
	}

	value, err := s.reporter(unit)
	if err != nil {
		return 0, "", fmt.Errorf("meter_elec: failed to retrieve report: %w", err)
	}

	return value, normalizedUnit, nil
}

func (s *service) ExtendedReport() (map[string]float64, error) {
	if !s.SupportsExtendedReport() {
		return nil, fmt.Errorf("meter_elec: extended report is unsupported")
	}

	values, err := s.extendedReporter()
	if err != nil {
		return nil, fmt.Errorf("meter_elec: failed to retrieve extended report: %w", err)
	}

	return values, nil
}

func (s *service) SupportedUnits() []string {
	return s.propertyStrings("sup_units")
}

func (s *service) SupportedExtendedValues() []string {
	return s.propertyStrings("sup_extended_vals")
}

func (s *service) SupportsExtendedReport() bool {
	return s.extendedReporter != nil
}

func (s *service) supportedUnit(unit string) (string, bool) {
	for _, u := range s.SupportedUnits() {
		if strings.EqualFold(unit, u) {
			return u, true
		}
	}

	return "", false
}

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
